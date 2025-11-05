package api

import (
	"errors"
	"sync"
	"time"

	"backend/internal/auth"
	"backend/internal/fetcher"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// SyncProgress tracks the progress of sync operations
type SyncProgress struct {
	Type       string  `json:"type"`       // "full" or "quick"
	Stage      string  `json:"stage"`      // current stage
	Current    int     `json:"current"`    // current progress
	Total      int     `json:"total"`      // total items
	Percentage float64 `json:"percentage"` // percentage complete
	InProgress bool    `json:"in_progress"`
}

// Server represents the API server with its dependencies.
// It now depends on interfaces, not concrete types.
type Server struct {
	store         DataStore
	tokenStore    *auth.TokenStore
	syncProgress  map[string]*SyncProgress // user email -> progress
	progressMutex sync.RWMutex
}

// NewServer creates a new Server instance.
func NewServer(store DataStore, tokenStore *auth.TokenStore) *Server {
	return &Server{
		store:        store,
		tokenStore:   tokenStore,
		syncProgress: make(map[string]*SyncProgress),
	}
}

// updateSyncProgress updates the sync progress for a user
func (s *Server) updateSyncProgress(userEmail, syncType, stage string, current, total int) {
	s.progressMutex.Lock()
	defer s.progressMutex.Unlock()
	
	percentage := 0.0
	if total > 0 {
		percentage = float64(current) / float64(total) * 100
	}
	
	s.syncProgress[userEmail] = &SyncProgress{
		Type:       syncType,
		Stage:      stage,
		Current:    current,
		Total:      total,
		Percentage: percentage,
		InProgress: current < total,
	}
}

// getSyncProgress retrieves the sync progress for a user
func (s *Server) getSyncProgress(userEmail string) *SyncProgress {
	s.progressMutex.RLock()
	defer s.progressMutex.RUnlock()
	
	if progress, exists := s.syncProgress[userEmail]; exists {
		return progress
	}
	return &SyncProgress{InProgress: false}
}

// clearSyncProgress clears the sync progress for a user
func (s *Server) clearSyncProgress(userEmail string) {
	s.progressMutex.Lock()
	defer s.progressMutex.Unlock()
	delete(s.syncProgress, userEmail)
}

// getEmailService is a helper to create an EmailService instance for a given request.
// This encapsulates the logic of getting a user token and instantiating the fetcher.
func (s *Server) getEmailService(c *gin.Context) (EmailService, error) {
	email, _ := c.Cookie("session_user")
	if email == "" {
		return nil, errors.New("no user in session")
	}
	tok, err := s.tokenStore.Get(c.Request.Context(), email)
	if err != nil {
		return nil, err
	}
	if tok == nil {
		return nil, errors.New("no token for user")
	}

	// The concrete *fetcher.GmailFetcher type implicitly satisfies the EmailService interface.
	return fetcher.NewGmailFetcher(c.Request.Context(), option.WithTokenSource(oauth2.StaticTokenSource(tok)))
}

// Rule is now primarily defined in the database package.
// We keep this here for request/response bodies if needed, but db.Rule is the source of truth.
type Rule struct {
	ID    string `json:"id"`
	Type  string `json:"type"` // e.g., sender, subject, keyword
	Value string `json:"value"`
}

// CleaningHistory represents a single cleaning event log.
type CleaningHistory struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	AffectedEmails []string  `json:"affected_emails"`
}

// Note: The global variables 'Emails', 'Rules', and 'CleanHistory' have been removed
// to ensure the database is the single source of truth.
