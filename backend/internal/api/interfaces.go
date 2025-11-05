package api

import (
	"context"

	"backend/internal/database"

	"google.golang.org/api/gmail/v1"
)

// DataStore describes all database operations the API needs.
type DataStore interface {
	// Rule methods
	ListRules(ctx context.Context, userEmail string) ([]database.Rule, error)
	CreateRule(ctx context.Context, arg database.CreateRuleParams) (database.Rule, error)
	UpdateRule(ctx context.Context, ruleID, userEmail, ruleType, value, action string, ageDays int) (database.Rule, error)
	DeleteRule(ctx context.Context, ruleID, userEmail string) error

	// Email methods
	ListEmails(ctx context.Context, userEmail string, page, pageSize int, filter string) ([]database.Email, int, error)
	ListAllEmailsForUser(ctx context.Context, userEmail string) ([]database.Email, error)
	DeleteEmail(ctx context.Context, id string) error
	UpsertEmails(ctx context.Context, emails []database.Email) error

	// History methods
	CreateCleaningHistory(ctx context.Context, userID string, affectedEmails []string) (database.CleaningHistory, error)
	ListCleaningHistory(ctx context.Context, userID string) ([]database.CleaningHistory, error)

	// Analytics methods
	GetTopSenders(ctx context.Context, userID string) ([]database.SenderAnalytic, error)

    // Settings methods
    GetUserSettings(ctx context.Context, userID string) (database.UserSettings, error)
    UpdateUserSettings(ctx context.Context, userID string, enabled bool, frequency string, timeOfDay string) (database.UserSettings, error)
    UpdateHistoryID(ctx context.Context, userID string, historyID uint64) error

	// Automation methods
	ListAutomatedUsers(ctx context.Context) ([]database.UserSettings, error)

    // Trash origin methods
    SaveTrashOrigin(ctx context.Context, userID, emailID string, hadInbox bool) error
    GetTrashOrigin(ctx context.Context, userID, emailID string) (bool, bool, error)
    DeleteTrashOrigin(ctx context.Context, userID, emailID string) error

	// Debug methods
	ResetDB(ctx context.Context) error
	CountEmails(ctx context.Context, userID string) (int, error)
}

// EmailService describes all actions for an email provider.
type EmailService interface {
	GetFullMessage(userID, messageID string) (*gmail.Message, error)
	GetMessageDetails(userID string, ids []string) ([]*gmail.Message, error)
	// CORRECTED: Added []string for labelIDs to match the implementation
	ListMessageIDs(userID, query string, labelIDs []string, max int64) ([]string, error)
	ListAllMessageIDs(userID, query string, labelIDs []string) ([]string, error)
	TrashMessage(userID, id string) error
	UntrashMessage(userID, id string) error
	// NEW: Added the missing method
	UnarchiveMessage(userID, id string) error
	ArchiveMessage(userID, id string) error
	MarkRead(userID, id string) error
	MarkUnread(userID, id string) error
	DeleteMessagePermanently(userID, id string) error

	CountArchivedMessages(userID string) (int, error)
	GetLabelMessageCount(userID, labelID string) (int, error)
    HasInboxLabel(userID, id string) (bool, error)
	SendMessage(userID string, message *gmail.Message) (*gmail.Message, error)
}
