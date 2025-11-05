package database

import (
	"context"
	"database/sql"
)

// PostgresStore wraps a sql.DB and exposes methods matching api.DataStore
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

func (s *PostgresStore) Migrate() { Migrate(s.db) }

func (s *PostgresStore) ListRules(ctx context.Context, userEmail string) ([]Rule, error) {
	return ListRules(ctx, s.db, userEmail)
}
func (s *PostgresStore) CreateRule(ctx context.Context, arg CreateRuleParams) (Rule, error) {
	return CreateRule(ctx, s.db, arg)
}
func (s *PostgresStore) UpdateRule(ctx context.Context, ruleID, userEmail, ruleType, value, action string, ageDays int) (Rule, error) {
	return UpdateRule(ctx, s.db, ruleID, userEmail, ruleType, value, action, ageDays)
}
func (s *PostgresStore) DeleteRule(ctx context.Context, ruleID, userEmail string) error {
	return DeleteRule(ctx, s.db, ruleID, userEmail)
}

func (s *PostgresStore) ListEmails(ctx context.Context, userEmail string, page, pageSize int, filter string) ([]Email, int, error) {
	return ListEmails(ctx, s.db, userEmail, page, pageSize, filter)
}
func (s *PostgresStore) ListAllEmailsForUser(ctx context.Context, userEmail string) ([]Email, error) {
	return ListAllEmailsForUser(ctx, s.db, userEmail)
}
func (s *PostgresStore) DeleteEmail(ctx context.Context, id string) error {
	return DeleteEmail(ctx, id)
}
func (s *PostgresStore) UpsertEmails(ctx context.Context, emails []Email) error {
	return UpsertEmails(ctx, s.db, emails)
}

func (s *PostgresStore) CreateCleaningHistory(ctx context.Context, userID string, affectedEmails []string) (CleaningHistory, error) {
	return CreateCleaningHistory(ctx, s.db, userID, affectedEmails)
}
func (s *PostgresStore) ListCleaningHistory(ctx context.Context, userID string) ([]CleaningHistory, error) {
	return ListCleaningHistory(ctx, s.db, userID)
}

func (s *PostgresStore) GetTopSenders(ctx context.Context, userID string) ([]SenderAnalytic, error) {
	return GetTopSenders(ctx, s.db, userID)
}

func (s *PostgresStore) GetUserSettings(ctx context.Context, userID string) (UserSettings, error) {
	return GetUserSettings(ctx, s.db, userID)
}
func (s *PostgresStore) UpdateUserSettings(ctx context.Context, userID string, enabled bool, frequency string, timeOfDay string) (UserSettings, error) {
    return UpdateUserSettings(ctx, s.db, userID, enabled, frequency, timeOfDay)
}

func (s *PostgresStore) ListAutomatedUsers(ctx context.Context) ([]UserSettings, error) {
	return ListAutomatedUsers(ctx, s.db)
}

func (s *PostgresStore) SaveTrashOrigin(ctx context.Context, userID, emailID string, hadInbox bool) error {
    return SaveTrashOrigin(ctx, s.db, userID, emailID, hadInbox)
}

func (s *PostgresStore) GetTrashOrigin(ctx context.Context, userID, emailID string) (bool, bool, error) {
    return GetTrashOrigin(ctx, s.db, userID, emailID)
}

func (s *PostgresStore) DeleteTrashOrigin(ctx context.Context, userID, emailID string) error {
    return DeleteTrashOrigin(ctx, s.db, userID, emailID)
}

func (s *PostgresStore) ResetDB(ctx context.Context) error { return ResetDB(ctx, s.db) }

func (s *PostgresStore) CountEmails(ctx context.Context, userID string) (int, error) {
	return countEmails(ctx, s.db, userID)
}

func (s *PostgresStore) UpdateHistoryID(ctx context.Context, userID string, historyID uint64) error {
	return UpdateHistoryId(ctx, s.db, userID, historyID)
}
