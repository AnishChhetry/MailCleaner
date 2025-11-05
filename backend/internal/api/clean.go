package api

import (
	"net/http"

	"backend/internal/fetcher"
	"backend/internal/rules"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// CleanPreviewHandler performs a dry run of the cleaning process.
func (s *Server) CleanPreviewHandler(c *gin.Context) {
	s.executeClean(c, true)
}

// CleanHandler performs the actual cleaning of emails.
func (s *Server) CleanHandler(c *gin.Context) {
	s.executeClean(c, false)
}

// executeClean contains the shared logic for both preview and actual cleaning.
func (s *Server) executeClean(c *gin.Context, dryRun bool) {
	userEmail := getUserEmail(c)
	ctx := c.Request.Context()

	// 1. Fetch user's rules from the database.
	dbRules, err := s.store.ListRules(ctx, userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch rules"})
		return
	}
	if len(dbRules) == 0 {
		msg := "No rules defined. Nothing to clean."
		if dryRun {
			msg = "No rules defined. Nothing to preview."
		}
		c.JSON(http.StatusOK, gin.H{"message": msg, "affected": []gin.H{}})
		return
	}

	// 2. Fetch all user's emails from the database.
	dbEmails, err := s.store.ListAllEmailsForUser(ctx, userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch emails"})
		return
	}

    // 3. Apply rules and determine which emails to delete/archive.
	var affectedEmails []gin.H
	for _, dbEmail := range dbEmails {
		for _, dbRule := range dbRules {
			// Convert database types to our domain types for the rule engine.
			ruleEmail := rules.Email{Sender: dbEmail.Sender, Subject: dbEmail.Subject, Snippet: dbEmail.Snippet, Date: dbEmail.Date}
			ruleRule := rules.Rule{Type: dbRule.Type, Value: dbRule.Value, Action: dbRule.Action, AgeDays: dbRule.AgeDays}

			if rules.Match(ruleEmail, ruleRule) {
				affectedEmails = append(affectedEmails, gin.H{
					"id":      dbEmail.ID,
					"sender":  dbEmail.Sender,
					"subject": dbEmail.Subject,
					"date":    dbEmail.Date,
					"action":  dbRule.Action, // Include the action in the preview
				})
				break // Move to the next email once one rule matches
			}
		}
	}

    if len(affectedEmails) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No emails matched your rules.", "affected": []gin.H{}})
		return
	}

    // 4a. If client provided an allowlist of IDs, filter to those only.
    var request struct {
        IDs              []string `json:"ids"`
        PermanentDelete  bool     `json:"permanentDelete"`
    }
    if err := c.ShouldBindJSON(&request); err == nil && len(request.IDs) > 0 {
        allowed := make(map[string]struct{}, len(request.IDs))
        for _, id := range request.IDs {
            allowed[id] = struct{}{}
        }
        filtered := make([]gin.H, 0, len(request.IDs))
        for _, e := range affectedEmails {
            if _, ok := allowed[e["id"].(string)]; ok {
                filtered = append(filtered, e)
            }
        }
        affectedEmails = filtered
    }

    if len(affectedEmails) == 0 {
        c.JSON(http.StatusOK, gin.H{"message": "No emails selected.", "affected": []gin.H{}})
        return
    }

    // 4b. If this is a dry run, return the list of emails that would be affected.
	if dryRun {
		c.JSON(http.StatusOK, gin.H{
			"message":  "Preview generated successfully.",
			"affected": affectedEmails,
		})
		return
	}

	// 5. If not a dry run, perform the actual actions via Gmail API.
	tok, err := s.getUserToken(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token is invalid"})
		return
	}
	gmailFetcher, err := fetcher.NewGmailFetcher(ctx, option.WithTokenSource(oauth2.StaticTokenSource(tok)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Gmail service"})
		return
	}

    var successfullyProcessedIDs []string
	for _, emailData := range affectedEmails {
		emailID := emailData["id"].(string)
		action := emailData["action"].(string)
		var actionErr error

        switch action {
		case "DELETE":
            if request.PermanentDelete {
                actionErr = gmailFetcher.DeleteMessagePermanently("me", emailID)
            } else {
                actionErr = gmailFetcher.TrashMessage("me", emailID)
            }
		case "ARCHIVE":
			actionErr = gmailFetcher.ArchiveMessage("me", emailID)
		case "MARK_READ":
			actionErr = gmailFetcher.MarkRead("me", emailID)
		}

		if actionErr != nil {
			c.Error(actionErr)
			continue
		}

		_ = s.store.DeleteEmail(ctx, emailID)
		successfullyProcessedIDs = append(successfullyProcessedIDs, emailID)
	}

	// 6. Log the cleaning event to the database.
	if _, err := s.store.CreateCleaningHistory(ctx, userEmail, successfullyProcessedIDs); err != nil {
		c.Error(err) // Log error but don't fail the whole request
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Cleaning complete",
		"affected_count": len(successfullyProcessedIDs),
		"affected_ids":   successfullyProcessedIDs,
	})
}

// GetCleanHistoryHandler fetches the cleaning history from the database.
func (s *Server) GetCleanHistoryHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	history, err := s.store.ListCleaningHistory(c.Request.Context(), userEmail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cleaning history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"history": history})
}
