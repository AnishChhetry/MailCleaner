package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var DB *sql.DB

// ConnectDB connects to the database, now with sslmode disabled for compatibility.
func ConnectDB(dsn string) (*sql.DB, error) {
	// Only append sslmode=disable if it's not already in the DSN.
	dbURL := dsn
	if !strings.Contains(dsn, "sslmode") {
		dbURL = fmt.Sprintf("%s sslmode=disable", dsn)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db
	log.Println("Successfully connected to the database!")
	return db, nil
}

// Migrate now includes the new user_settings table and updated rules table
func Migrate(db *sql.DB) {
	log.Println("Running database migrations...")
	migrationSQL := `
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";

	CREATE TABLE IF NOT EXISTS rules (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		type TEXT NOT NULL,
		value TEXT NOT NULL,
		action TEXT NOT NULL DEFAULT 'DELETE',
		age_days INT DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS emails (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		sender TEXT,
		subject TEXT,
		snippet TEXT,
		date TIMESTAMPTZ,
		read BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS cleaning_history (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id TEXT NOT NULL,
		timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		affected_emails TEXT[],
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

    CREATE TABLE IF NOT EXISTS user_settings (
        user_id TEXT PRIMARY KEY,
        automation_enabled BOOLEAN NOT NULL DEFAULT FALSE,
        automation_frequency TEXT NOT NULL DEFAULT 'daily',
        automation_time TEXT DEFAULT '00:00',
        automation_runs_per_day INT NOT NULL DEFAULT 1,
        last_history_id BIGINT,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

	-- Remember origin state when moving to trash
	CREATE TABLE IF NOT EXISTS trash_state (
		user_id TEXT NOT NULL,
		email_id TEXT NOT NULL,
		had_inbox BOOLEAN NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		PRIMARY KEY (user_id, email_id)
	);

    ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS automation_time TEXT DEFAULT '00:00';
    ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS automation_runs_per_day INT NOT NULL DEFAULT 1;
    ALTER TABLE user_settings ADD COLUMN IF NOT EXISTS last_history_id BIGINT;

	`
	_, err := db.Exec(migrationSQL)
	if err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	log.Println("Database migrations completed successfully.")
}

// UpsertEmails (Optimized for Large Batches)
func UpsertEmails(ctx context.Context, db *sql.DB, emails []Email) error {
	if len(emails) == 0 {
		return nil
	}

	log.Infof("Starting upsert of %d emails", len(emails))

	// Process emails in batches to avoid timeouts
	batchSize := 50 // Smaller batches for better reliability
	var totalProcessed int

	for i := 0; i < len(emails); i += batchSize {
		end := i + batchSize
		if end > len(emails) {
			end = len(emails)
		}
		batch := emails[i:end]

		log.Infof("Processing batch %d-%d of %d emails", i+1, end, len(emails))

		if err := upsertEmailBatch(ctx, db, batch); err != nil {
			log.Errorf("Failed to upsert batch %d-%d: %v", i+1, end, err)
			return fmt.Errorf("failed to upsert batch %d-%d: %w", i+1, end, err)
		}

		totalProcessed += len(batch)
		log.Infof("Successfully processed batch %d-%d (%d total processed)", i+1, end, totalProcessed)
	}

	log.Infof("Successfully upserted all %d emails", totalProcessed)
	return nil
}

// upsertEmailBatch processes a single batch of emails
func upsertEmailBatch(ctx context.Context, db *sql.DB, emails []Email) error {
	if len(emails) == 0 {
		return nil
	}

	// Create a longer context for database operations
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	tx, err := db.BeginTx(dbCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(dbCtx, `
		INSERT INTO emails (id, user_id, sender, subject, snippet, date, read)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			sender = EXCLUDED.sender,
			subject = EXCLUDED.subject,
			snippet = EXCLUDED.snippet,
			date = EXCLUDED.date,
			read = EXCLUDED.read,
			updated_at = NOW();
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, email := range emails {
		if _, err := stmt.ExecContext(dbCtx, email.ID, email.UserID, email.Sender, email.Subject, email.Snippet, email.Date, email.Read); err != nil {
			log.Errorf("Database error on email ID %s: %v", email.ID, err)
			return fmt.Errorf("failed to execute insert for email ID %s: %w", email.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type Email struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Sender    string    `json:"sender"`
	Subject   string    `json:"subject"`
	Snippet   string    `json:"snippet"`
	Date      time.Time `json:"date"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type Rule struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"`
	Value     string    `json:"value"`
	Action    string    `json:"action"`
	AgeDays   int       `json:"age_days"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type CleaningHistory struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Timestamp      time.Time `json:"timestamp"`
	AffectedEmails []string  `json:"affected_emails"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
type UserSettings struct {
	UserID              string    `json:"user_id"`
	AutomationEnabled   bool      `json:"automation_enabled"`
	AutomationFrequency string    `json:"automation_frequency"`
	AutomationTime      string    `json:"automation_time"`
	LastHistoryID       uint64    `json:"last_history_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type TrashState struct {
    UserID   string    `json:"user_id"`
    EmailID  string    `json:"email_id"`
    HadInbox bool      `json:"had_inbox"`
    CreatedAt time.Time `json:"created_at"`
}

type SenderAnalytic struct {
	Sender string `json:"sender"`
	Count  int    `json:"count"`
}

type CreateRuleParams struct {
	ID      string `json:"id"`
	UserID  string `json:"user_id"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Action  string `json:"action"`
	AgeDays int    `json:"age_days"`
}

func DeleteEmail(ctx context.Context, id string) error {
	result, err := DB.ExecContext(ctx, `DELETE FROM emails WHERE id=$1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("email not found")
	}
	return nil
}

func ListEmails(ctx context.Context, db *sql.DB, userEmail string, page, pageSize int, filter string) ([]Email, int, error) {
	offset := (page - 1) * pageSize
	listArgs := []interface{}{userEmail, pageSize, offset}
	listQuery := `SELECT id, user_id, sender, subject, snippet, date, read, created_at, updated_at FROM emails WHERE user_id=$1`
	if filter != "" {
		listQuery += " AND (subject ILIKE $4 OR sender ILIKE $4)"
		listArgs = append(listArgs, "%"+filter+"%")
	}
	listQuery += " ORDER BY date DESC LIMIT $2 OFFSET $3"
	rows, err := db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var emails []Email
	for rows.Next() {
		var e Email
		if err := rows.Scan(&e.ID, &e.UserID, &e.Sender, &e.Subject, &e.Snippet, &e.Date, &e.Read, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		emails = append(emails, e)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	countArgs := []interface{}{userEmail}
	countQuery := `SELECT COUNT(*) FROM emails WHERE user_id=$1`
	if filter != "" {
		countQuery += " AND (subject ILIKE $2 OR sender ILIKE $2)"
		countArgs = append(countArgs, "%"+filter+"%")
	}
	var total int
	if err := db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return emails, total, nil
}
func ListAllEmailsForUser(ctx context.Context, db *sql.DB, userEmail string) ([]Email, error) {
	query := `SELECT id, user_id, sender, subject, snippet, date, read, created_at, updated_at
			  FROM emails WHERE user_id=$1 ORDER BY date DESC`
	rows, err := db.QueryContext(ctx, query, userEmail)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var emails []Email
	for rows.Next() {
		var e Email
		err := rows.Scan(&e.ID, &e.UserID, &e.Sender, &e.Subject, &e.Snippet, &e.Date, &e.Read, &e.CreatedAt, &e.UpdatedAt)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, nil
}

func ListRules(ctx context.Context, db *sql.DB, userEmail string) ([]Rule, error) {
	query := `SELECT id, user_id, type, value, action, age_days, created_at, updated_at FROM rules WHERE user_id=$1 ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, userEmail)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []Rule
	for rows.Next() {
		var r Rule
		if err := rows.Scan(&r.ID, &r.UserID, &r.Type, &r.Value, &r.Action, &r.AgeDays, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

func CreateRule(ctx context.Context, db *sql.DB, arg CreateRuleParams) (Rule, error) {
	query := `
		INSERT INTO rules (id, user_id, type, value, action, age_days)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, type, value, action, age_days, created_at, updated_at
	`
	var rule Rule
	err := db.QueryRowContext(ctx, query, arg.ID, arg.UserID, arg.Type, arg.Value, arg.Action, arg.AgeDays).Scan(
		&rule.ID, &rule.UserID, &rule.Type, &rule.Value, &rule.Action, &rule.AgeDays, &rule.CreatedAt, &rule.UpdatedAt,
	)
	return rule, err
}

func UpdateRule(ctx context.Context, db *sql.DB, ruleID, userEmail, ruleType, value, action string, ageDays int) (Rule, error) {
	query := `
		UPDATE rules
		SET type = $3, value = $4, action = $5, age_days = $6, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, type, value, action, age_days, created_at, updated_at
	`
	var rule Rule
	err := db.QueryRowContext(ctx, query, ruleID, userEmail, ruleType, value, action, ageDays).Scan(
		&rule.ID, &rule.UserID, &rule.Type, &rule.Value, &rule.Action, &rule.AgeDays, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Rule{}, errors.New("rule not found or not owned by user")
		}
		return Rule{}, err
	}
	return rule, nil
}

func DeleteRule(ctx context.Context, db *sql.DB, ruleID, userEmail string) error {
	query := `DELETE FROM rules WHERE id = $1 AND user_id = $2`
	res, err := db.ExecContext(ctx, query, ruleID, userEmail)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("rule not found or not owned by user")
	}
	return nil
}

func CreateCleaningHistory(ctx context.Context, db *sql.DB, userID string, affectedEmails []string) (CleaningHistory, error) {
	query := `
		INSERT INTO cleaning_history (user_id, affected_emails, timestamp)
		VALUES ($1, $2, NOW())
		RETURNING id, user_id, timestamp, affected_emails, created_at, updated_at
	`
	var history CleaningHistory
	err := db.QueryRowContext(ctx, query, userID, pq.Array(affectedEmails)).Scan(
		&history.ID, &history.UserID, &history.Timestamp, pq.Array(&history.AffectedEmails), &history.CreatedAt, &history.UpdatedAt,
	)
	return history, err
}

func ListCleaningHistory(ctx context.Context, db *sql.DB, userID string) ([]CleaningHistory, error) {
	query := `
		SELECT id, user_id, timestamp, affected_emails, created_at, updated_at
		FROM cleaning_history
		WHERE user_id = $1
		ORDER BY timestamp DESC
		LIMIT 100
	`
	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []CleaningHistory
	for rows.Next() {
		var h CleaningHistory
		if err := rows.Scan(&h.ID, &h.UserID, &h.Timestamp, pq.Array(&h.AffectedEmails), &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		histories = append(histories, h)
	}
	return histories, rows.Err()
}

func GetTopSenders(ctx context.Context, db *sql.DB, userID string) ([]SenderAnalytic, error) {
	query := `
		SELECT sender, COUNT(*) as email_count
		FROM emails
		WHERE user_id = $1 AND sender <> ''
		GROUP BY sender
		ORDER BY email_count DESC
		LIMIT 10;
	`
	rows, err := db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var analytics []SenderAnalytic
	for rows.Next() {
		var sa SenderAnalytic
		if err := rows.Scan(&sa.Sender, &sa.Count); err != nil {
			return nil, err
		}
		analytics = append(analytics, sa)
	}
	return analytics, rows.Err()
}

func GetUserSettings(ctx context.Context, db *sql.DB, userID string) (UserSettings, error) {
	var settings UserSettings
	// Upsert followed by Select to ensure a settings row always exists for a user.
	upsertQuery := `INSERT INTO user_settings (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING;`
	_, err := db.ExecContext(ctx, upsertQuery, userID)
	if err != nil {
		return UserSettings{}, err
	}

	selectQuery := `
        SELECT user_id, automation_enabled, automation_frequency, automation_time, COALESCE(last_history_id, 0), created_at, updated_at
        FROM user_settings WHERE user_id = $1;
    `
	err = db.QueryRowContext(ctx, selectQuery, userID).Scan(
		&settings.UserID, &settings.AutomationEnabled, &settings.AutomationFrequency, &settings.AutomationTime, &settings.LastHistoryID, &settings.CreatedAt, &settings.UpdatedAt,
	)
	return settings, err
}

func SaveTrashOrigin(ctx context.Context, db *sql.DB, userID, emailID string, hadInbox bool) error {
    _, err := db.ExecContext(ctx, `
        INSERT INTO trash_state (user_id, email_id, had_inbox)
        VALUES ($1, $2, $3)
        ON CONFLICT (user_id, email_id) DO UPDATE SET had_inbox = EXCLUDED.had_inbox, created_at = NOW();
    `, userID, emailID, hadInbox)
    return err
}

func GetTrashOrigin(ctx context.Context, db *sql.DB, userID, emailID string) (bool, bool, error) {
    var hadInbox bool
    err := db.QueryRowContext(ctx, `SELECT had_inbox FROM trash_state WHERE user_id=$1 AND email_id=$2`, userID, emailID).Scan(&hadInbox)
    if err != nil {
        if err == sql.ErrNoRows {
            return false, false, nil
        }
        return false, false, err
    }
    return hadInbox, true, nil
}

func DeleteTrashOrigin(ctx context.Context, db *sql.DB, userID, emailID string) error {
    _, err := db.ExecContext(ctx, `DELETE FROM trash_state WHERE user_id=$1 AND email_id=$2`, userID, emailID)
    return err
}

func UpdateUserSettings(ctx context.Context, db *sql.DB, userID string, enabled bool, frequency string, timeOfDay string) (UserSettings, error) {
	var settings UserSettings
	query := `
        UPDATE user_settings
        SET automation_enabled = $2, automation_frequency = $3, automation_time = $4, updated_at = NOW()
        WHERE user_id = $1
        RETURNING user_id, automation_enabled, automation_frequency, automation_time, COALESCE(last_history_id, 0), created_at, updated_at
	`
	err := db.QueryRowContext(ctx, query, userID, enabled, frequency, timeOfDay).Scan(
		&settings.UserID, &settings.AutomationEnabled, &settings.AutomationFrequency, &settings.AutomationTime, &settings.LastHistoryID, &settings.CreatedAt, &settings.UpdatedAt,
	)
	return settings, err
}

func ListAutomatedUsers(ctx context.Context, db *sql.DB) ([]UserSettings, error) {
	rows, err := db.QueryContext(ctx, `
        SELECT user_id, automation_enabled, automation_frequency, automation_time, COALESCE(last_history_id, 0), created_at, updated_at
        FROM user_settings WHERE automation_enabled = TRUE
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settingsList []UserSettings
	for rows.Next() {
		var s UserSettings
		if err := rows.Scan(&s.UserID, &s.AutomationEnabled, &s.AutomationFrequency, &s.AutomationTime, &s.LastHistoryID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		settingsList = append(settingsList, s)
	}
	return settingsList, rows.Err()
}

func ResetDB(ctx context.Context, db *sql.DB) error {
	// The order matters here due to foreign key constraints if they existed.
	// It's good practice to drop tables in the reverse order of creation.
    tables := []string{
		"user_settings",
        "trash_state",
		"cleaning_history",
		"emails",
		"rules",
	}

	for _, table := range tables {
		log.Infof("Dropping table: %s", table)
		_, err := db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", table))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	log.Info("All tables dropped. Re-running migrations...")
	// Re-run migrations to create the tables again
	Migrate(db)

	return nil
}
func countEmails(ctx context.Context, db *sql.DB, userID string) (int, error) {
	var total int
	query := `SELECT COUNT(*) FROM emails WHERE user_id=$1`
	if err := db.QueryRowContext(ctx, query, userID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func UpdateHistoryId(ctx context.Context, db *sql.DB, userID string, historyID uint64) error {
	query := `UPDATE user_settings SET last_history_id = $1, updated_at = NOW() WHERE user_id = $2`
	_, err := db.ExecContext(ctx, query, historyID, userID)
	return err
}