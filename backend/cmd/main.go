package main

import (
	"context"
	"fmt"
	"time"

	"backend/internal/api"
	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/fetcher"
	"backend/internal/rules"

	"github.com/go-co-op/gocron"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("Warning: No .env file found, reading from environment")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	db, err := database.ConnectDB(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("database connection error: %v", err)
	}

	store := database.NewPostgresStore(db)
	store.Migrate()

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("could not parse redis url: %v", err)
	}
	rdb := redis.NewClient(opt)

	tokenStore := auth.NewTokenStore(rdb)

	api.InitOAuthConfig(cfg)

	// Start the background scheduler with the store interface
	go startScheduler(store, tokenStore)

	router := api.NewRouter(cfg, store, tokenStore)

	log.Infof("MailCleaner starting on %s", cfg.HttpAddr)
	if err := router.Run(cfg.HttpAddr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// Track last execution to prevent duplicate runs in the same minute
var lastExecutionKey = make(map[string]string)

// NEW: Function to run the automated cleaning job
func startScheduler(store api.DataStore, tokenStore *auth.TokenStore) {
	log.Info("Starting automated cleaning scheduler...")
	s := gocron.NewScheduler(time.UTC)
	_, err := s.Every(1).Minute().Do(func() {
		log.Debug("Scheduler tick: checking for users to clean...")
		ctx := context.Background()
		userSettings, err := store.ListAutomatedUsers(ctx)
		if err != nil {
			log.Errorf("Scheduler error: could not list automated users: %v", err)
			return
		}

		if len(userSettings) == 0 {
			log.Debug("Scheduler: No users with automation enabled")
			return
		}

		log.Debugf("Scheduler: Checking %d users with automation enabled", len(userSettings))
		for _, settings := range userSettings {
			now := time.Now().UTC()
			loc, err := time.LoadLocation("Asia/Kolkata") // IST timezone
			if err != nil {
				log.Errorf("Scheduler: could not load IST location for user %s: %v", settings.UserID, err)
				continue
			}
			
			// Get current time in IST
			nowIST := now.In(loc)
			
			userTime, err := time.ParseInLocation("15:04", settings.AutomationTime, loc)
			if err != nil {
				log.Errorf("Scheduler: could not parse time for user %s: %v", settings.UserID, err)
				continue
			}

			// Check if it's the right time to run the job
			shouldRun := false
			executionKey := ""
			
			switch settings.AutomationFrequency {
			case "daily":
				if nowIST.Hour() == userTime.Hour() && nowIST.Minute() == userTime.Minute() {
					executionKey = fmt.Sprintf("%s-daily-%s", settings.UserID, nowIST.Format("2006-01-02-15:04"))
					shouldRun = true
				}
			case "weekly":
				// Weekly runs on the same weekday, at the specified time
				if nowIST.Hour() == userTime.Hour() && nowIST.Minute() == userTime.Minute() {
					// Run once per week on this weekday
					executionKey = fmt.Sprintf("%s-weekly-%s", settings.UserID, nowIST.Format("2006-W01-Mon-15:04"))
					shouldRun = true
				}
			case "hourly":
				if nowIST.Minute() == userTime.Minute() {
					executionKey = fmt.Sprintf("%s-hourly-%s", settings.UserID, nowIST.Format("2006-01-02-15:04"))
					shouldRun = true
				}
			}

			if shouldRun {
				// Prevent duplicate execution in the same minute
				if lastExecutionKey[settings.UserID] == executionKey {
					log.Debugf("Scheduler: Already executed for user %s at this time", settings.UserID)
					continue
				}
				
				lastExecutionKey[settings.UserID] = executionKey
				log.Infof("Scheduler: Triggering %s cleaning for user %s at %s IST", settings.AutomationFrequency, settings.UserID, nowIST.Format("15:04"))
				
				if err := executeCleanForUser(ctx, store, tokenStore, settings.UserID); err != nil {
					log.Errorf("Scheduler: failed to clean for user %s: %v", settings.UserID, err)
				} else {
					log.Infof("Scheduler: Successfully completed cleaning for user %s", settings.UserID)
				}
			}
		}
	})
	
	if err != nil {
		log.Errorf("Failed to schedule job: %v", err)
		return
	}
	
	s.StartAsync()
	log.Info("Scheduler started successfully")
}

// NEW: Standalone cleaning logic for the scheduler
func executeCleanForUser(ctx context.Context, store api.DataStore, tokenStore *auth.TokenStore, userEmail string) error {
	// This logic is a simplified, non-HTTP version of executeClean from clean.go
	dbRules, err := store.ListRules(ctx, userEmail)
	if err != nil {
		return fmt.Errorf("could not fetch rules: %w", err)
	}
	if len(dbRules) == 0 {
		return nil // No rules, nothing to do
	}

	dbEmails, err := store.ListAllEmailsForUser(ctx, userEmail)
	if err != nil {
		return fmt.Errorf("could not fetch emails: %w", err)
	}

	tok, err := tokenStore.Get(ctx, userEmail)
	if err != nil || tok == nil {
		// Attempt to refresh token if it's nil but might be refreshable in a real scenario
		return fmt.Errorf("could not get token for user: %w", err)
	}

	// The oauth2 library automatically handles token refreshes
	tokenSource := api.OAuthConfig().TokenSource(ctx, tok)

	gmailFetcher, err := fetcher.NewGmailFetcher(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return fmt.Errorf("failed to create gmail service: %w", err)
	}

	var affectedEmailIDs []string
	for _, dbEmail := range dbEmails {
		for _, dbRule := range dbRules {
			ruleEmail := rules.Email{Sender: dbEmail.Sender, Subject: dbEmail.Subject, Snippet: dbEmail.Snippet, Date: dbEmail.Date}
			ruleRule := rules.Rule{Type: dbRule.Type, Value: dbRule.Value, Action: dbRule.Action, AgeDays: dbRule.AgeDays}

			if rules.Match(ruleEmail, ruleRule) {
				var actionErr error
				switch ruleRule.Action {
				case "DELETE":
					actionErr = gmailFetcher.TrashMessage("me", dbEmail.ID)
				case "ARCHIVE":
					actionErr = gmailFetcher.ArchiveMessage("me", dbEmail.ID)
				case "MARK_READ":
					actionErr = gmailFetcher.MarkRead("me", dbEmail.ID)
				}

				if actionErr != nil {
					log.Errorf("Scheduler: action %s failed for email %s: %v", ruleRule.Action, dbEmail.ID, actionErr)
					// Don't stop for one failed action, just continue
				} else {
					// Only delete from local DB if API call was successful
					_ = store.DeleteEmail(context.Background(), dbEmail.ID)
					affectedEmailIDs = append(affectedEmailIDs, dbEmail.ID)
				}
				break // Move to the next email once one rule matches
			}
		}
	}

	if len(affectedEmailIDs) > 0 {
		if _, err := store.CreateCleaningHistory(ctx, userEmail, affectedEmailIDs); err != nil {
			log.Errorf("Scheduler: failed to log cleaning history for user %s: %v", userEmail, err)
		}
		log.Infof("Scheduler: successfully cleaned %d emails for user %s", len(affectedEmailIDs), userEmail)
	}
	return nil
}
