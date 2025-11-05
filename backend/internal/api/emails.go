package api

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"backend/internal/database"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"

	"backend/internal/imap"

	"github.com/google/uuid"
)

func (s *Server) SyncEmailsHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	
	// Create a longer context for sync operations (5 minutes)
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()
	defer s.clearSyncProgress(userEmail)

	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token is invalid"})
		return
	}

	log.Infof("Starting full email sync for user: %s", userEmail)
	s.updateSyncProgress(userEmail, "full", "Listing emails", 0, 100)

    // Use the new method to fetch only INBOX message IDs
    ids, err := emailService.ListAllMessageIDs("me", "", []string{"INBOX"})
	if err != nil {
		log.Errorf("Failed to list Gmail messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list Gmail messages"})
		return
	}

	log.Infof("Found %d total emails to sync", len(ids))
	s.updateSyncProgress(userEmail, "full", "Fetching email details", 10, 100)

	messages, err := emailService.GetMessageDetails("me", ids)
	if err != nil {
		log.Errorf("Failed to get Gmail message details: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Gmail message details"})
		return
	}

	log.Infof("Successfully fetched details for %d messages", len(messages))
	s.updateSyncProgress(userEmail, "full", "Processing emails", 50, 100)

	var emailsToUpsert []database.Email
	for _, msg := range messages {
		var sender, subject, dateStr string
		for _, h := range msg.Payload.Headers {
			switch h.Name {
			case "From":
				sender = h.Value
			case "Subject":
				subject = h.Value
			case "Date":
				dateStr = h.Value
			}
		}

		date, err := parseGmailDate(dateStr)
		if err != nil {
			continue // Skip emails with a date format we can't parse
		}

		isRead := true
		for _, labelId := range msg.LabelIds {
			if labelId == "UNREAD" {
				isRead = false
				break
			}
		}

		cleanSender := strings.ReplaceAll(sender, "\x00", "")
		cleanSubject := strings.ReplaceAll(subject, "\x00", "")
		cleanSnippet := strings.ReplaceAll(msg.Snippet, "\x00", "")

		emailsToUpsert = append(emailsToUpsert, database.Email{
			ID:      msg.Id,
			UserID:  userEmail,
			Sender:  cleanSender,
			Subject: cleanSubject,
			Snippet: cleanSnippet,
			Date:    date,
			Read:    isRead,
		})
	}

	log.Infof("Processing %d emails for database upsert", len(emailsToUpsert))
	s.updateSyncProgress(userEmail, "full", "Saving to database", 75, 100)

	if err := s.store.UpsertEmails(ctx, emailsToUpsert); err != nil {
		log.Errorf("Failed to save emails to database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save emails to database"})
		return
	}

	// Initialize history ID for future quick syncs using the History API
	if len(messages) > 0 && messages[0].HistoryId > 0 {
		if err := s.store.UpdateHistoryID(ctx, userEmail, messages[0].HistoryId); err != nil {
			log.Errorf("Failed to initialize history ID: %v", err)
		} else {
			log.Infof("Initialized history ID to %d for efficient quick syncs", messages[0].HistoryId)
		}
	}

	s.updateSyncProgress(userEmail, "full", "Complete", 100, 100)
	log.Infof("Successfully synced %d emails for user %s", len(emailsToUpsert), userEmail)
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Successfully synced %d emails.", len(emailsToUpsert)),
		"total":   len(emailsToUpsert),
	})
}

// BulkMarkReadHandler marks multiple emails as read
func (s *Server) BulkMarkReadHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}

	var request struct {
		EmailIDs []string `json:"emailIds"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(request.EmailIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No email IDs provided"})
		return
	}

	log.Infof("Bulk marking %d emails as read", len(request.EmailIDs))

	var successCount int
	var errors []string

	for _, id := range request.EmailIDs {
		if err := emailService.MarkRead("me", id); err != nil {
			log.Errorf("Failed to mark email %s as read: %v", id, err)
			errors = append(errors, fmt.Sprintf("Failed to mark %s as read", id))
		} else {
			successCount++
		}
	}

	if len(errors) > 0 {
		c.JSON(http.StatusPartialContent, gin.H{
			"message":      fmt.Sprintf("Marked %d emails as read", successCount),
			"successCount": successCount,
			"errors":       errors,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message":      fmt.Sprintf("Successfully marked %d emails as read", successCount),
			"successCount": successCount,
		})
	}
}

// BulkMarkUnreadHandler marks multiple emails as unread
func (s *Server) BulkMarkUnreadHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}

	var request struct {
		EmailIDs []string `json:"emailIds"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(request.EmailIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No email IDs provided"})
		return
	}

	log.Infof("Bulk marking %d emails as unread", len(request.EmailIDs))

	var successCount int
	var errors []string

	for _, id := range request.EmailIDs {
		if err := emailService.MarkUnread("me", id); err != nil {
			log.Errorf("Failed to mark email %s as unread: %v", id, err)
			errors = append(errors, fmt.Sprintf("Failed to mark %s as unread", id))
		} else {
			successCount++
		}
	}

	if len(errors) > 0 {
		c.JSON(http.StatusPartialContent, gin.H{
			"message":      fmt.Sprintf("Marked %d emails as unread", successCount),
			"successCount": successCount,
			"errors":       errors,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message":      fmt.Sprintf("Successfully marked %d emails as unread", successCount),
			"successCount": successCount,
		})
	}
}

// BulkDeleteHandler moves multiple emails to trash
func (s *Server) BulkDeleteHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}

	var request struct {
		EmailIDs []string `json:"emailIds"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(request.EmailIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No email IDs provided"})
		return
	}

	log.Infof("Bulk deleting %d emails", len(request.EmailIDs))

	var successCount int
	var errors []string
	userEmail := getUserEmail(c)
	ctx := c.Request.Context()

	for _, id := range request.EmailIDs {
		// Save origin inbox state for each email
		hadInbox, _ := emailService.HasInboxLabel("me", id)
		_ = s.store.SaveTrashOrigin(ctx, userEmail, id, hadInbox)
		if err := emailService.TrashMessage("me", id); err != nil {
			log.Errorf("Failed to delete email %s: %v", id, err)
			errors = append(errors, fmt.Sprintf("Failed to delete %s", id))
		} else {
			successCount++
			// Remove from local cache so it disappears immediately from Inbox view
			_ = s.store.DeleteEmail(ctx, id)
		}
	}

	if len(errors) > 0 {
		c.JSON(http.StatusPartialContent, gin.H{
			"message":      fmt.Sprintf("Moved %d emails to trash", successCount),
			"successCount": successCount,
			"errors":       errors,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message":      fmt.Sprintf("Successfully moved %d emails to trash", successCount),
			"successCount": successCount,
		})
	}
}

// BulkArchiveHandler archives multiple emails
func (s *Server) BulkArchiveHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}

	var request struct {
		EmailIDs []string `json:"emailIds"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(request.EmailIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No email IDs provided"})
		return
	}

	log.Infof("Bulk archiving %d emails", len(request.EmailIDs))

    var successCount int
	var errors []string
    ctx := c.Request.Context()

    for _, id := range request.EmailIDs {
        if err := emailService.ArchiveMessage("me", id); err != nil {
			log.Errorf("Failed to archive email %s: %v", id, err)
			errors = append(errors, fmt.Sprintf("Failed to archive %s", id))
        } else {
            // ensure INBOX removed
            for i := 0; i < 2; i++ {
                hasInbox, chkErr := emailService.HasInboxLabel("me", id)
                if chkErr == nil && !hasInbox {
                    break
                }
                _ = emailService.ArchiveMessage("me", id)
                time.Sleep(200 * time.Millisecond)
            }
			successCount++
            // remove from local inbox cache so it doesn't reappear
            _ = s.store.DeleteEmail(ctx, id)
		}
	}

	if len(errors) > 0 {
		c.JSON(http.StatusPartialContent, gin.H{
			"message":      fmt.Sprintf("Archived %d emails", successCount),
			"successCount": successCount,
			"errors":       errors,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message":      fmt.Sprintf("Successfully archived %d emails", successCount),
			"successCount": successCount,
		})
	}
}

// parseGmailDate handles multiple potential date formats from the Gmail API.
func parseGmailDate(dateStr string) (time.Time, error) {
	if idx := strings.LastIndex(dateStr, " ("); idx != -1 {
		dateStr = dateStr[:idx]
	}

	layouts := []string{
		time.RFC1123Z,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		time.RFC3339,
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, dateStr)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse date: %s", dateStr)
}

func getUserEmail(c *gin.Context) string {
	email, _ := c.Cookie("session_user")
	return email
}

func (s *Server) getUserToken(c *gin.Context) (*oauth2.Token, error) {
	email := getUserEmail(c)
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
	return tok, nil
}

func (s *Server) GetEmailsHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	emails, total, err := s.store.ListEmails(c.Request.Context(), userEmail, 1, 25, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list emails from database"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"emails": emails, "total": total, "user": userEmail})
}

func (s *Server) DeleteEmailHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
    ctx := c.Request.Context()
    id := c.Param("id")
    // Record whether this email was in INBOX prior to trashing, so we can restore accordingly.
    userEmail := getUserEmail(c)
    hadInbox, _ := emailService.HasInboxLabel("me", id)
    _ = s.store.SaveTrashOrigin(ctx, userEmail, id, hadInbox)
	err = emailService.TrashMessage("me", id)
	if err != nil {
		log.Errorf("Failed to trash email %s from Gmail: %v", id, err)
		if gerr, ok := err.(*googleapi.Error); ok {
			if gerr.Code == http.StatusNotFound {
				_ = s.store.DeleteEmail(ctx, id)
				c.JSON(http.StatusOK, gin.H{"message": "Email already gone from Gmail; removed from local cache.", "id": id})
				return
			}
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move email to trash: " + err.Error()})
		return
	}
	_ = s.store.DeleteEmail(ctx, id)
	c.JSON(http.StatusOK, gin.H{"message": "Email moved to trash", "id": id})
}

// NEW: Handler for untrashing an email
func (s *Server) UntrashEmailHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
    id := c.Param("id")
    userEmail := getUserEmail(c)
    err = emailService.UntrashMessage("me", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to untrash email"})
		return
	}
    // Determine original location and enforce it
    if hadInbox, ok, _ := s.store.GetTrashOrigin(c.Request.Context(), userEmail, id); ok {
        if hadInbox {
            // Ensure in inbox
            _ = emailService.UnarchiveMessage("me", id)
        } else {
            // Ensure not in inbox (archived)
            _ = emailService.ArchiveMessage("me", id)
        }
        _ = s.store.DeleteTrashOrigin(c.Request.Context(), userEmail, id)
    }
    c.JSON(http.StatusOK, gin.H{"message": "Email restored"})
}

func (s *Server) MarkEmailReadHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
	id := c.Param("id")
	err = emailService.MarkRead("me", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Email marked as read"})
}

func (s *Server) MarkEmailUnreadHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
	id := c.Param("id")
	err = emailService.MarkUnread("me", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as unread"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Email marked as unread"})
}

func (s *Server) ArchiveEmailHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No token"})
		return
	}
    id := c.Param("id")
    err = emailService.ArchiveMessage("me", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to archive email"})
		return
	}
    // Ensure INBOX label is actually removed; retry briefly if needed
    for i := 0; i < 3; i++ {
        hasInbox, chkErr := emailService.HasInboxLabel("me", id)
        if chkErr == nil && !hasInbox {
            break
        }
        // try to remove again and wait a moment
        _ = emailService.ArchiveMessage("me", id)
        time.Sleep(300 * time.Millisecond)
    }
    // Remove from local inbox cache so it disappears immediately from Inbox view
    _ = s.store.DeleteEmail(c.Request.Context(), id)
    c.JSON(http.StatusOK, gin.H{"message": "Email archived"})
}

func (s *Server) GetEmailsPaginatedHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	page := 1
	pageSize := 10
	var filter string

	if v := c.Query("page"); v != "" {
		fmt.Sscanf(v, "%d", &page)
	}
	if v := c.Query("pageSize"); v != "" {
		fmt.Sscanf(v, "%d", &pageSize)
	}
	filter = c.Query("filter")

	// The call to s.store.ListEmails now returns a total count
	emails, total, err := s.store.ListEmails(c.Request.Context(), userEmail, page, pageSize, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list paginated emails from database"})
		return
	}

	// Include the total in the response
	c.JSON(http.StatusOK, gin.H{"emails": emails, "total": total, "user": userEmail})
}

func (s *Server) BlockSenderHandler(c *gin.Context) {
	userEmail := getUserEmail(c)
	var req struct {
		Sender string `json:"sender"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Sender == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sender"})
		return
	}
	params := database.CreateRuleParams{
		ID:     uuid.NewString(),
		UserID: userEmail,
		Type:   "sender",
		Value:  req.Sender,
		Action: "DELETE", // Default action for blocking sender
	}
	rule, err := s.store.CreateRule(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create block rule"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Block rule created for sender", "rule": rule})
}

func (s *Server) UnsubscribeFromNewsletterHandler(c *gin.Context) {
	var req struct {
		UnsubscribeHeader string `json:"unsubscribe_header"`
		Sender            string `json:"sender"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UnsubscribeHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Parse the List-Unsubscribe header
	// Format can be: <mailto:unsubscribe@example.com>, <http://example.com/unsubscribe>
	// or both separated by commas
	header := strings.TrimSpace(req.UnsubscribeHeader)
	
	// Try to find mailto or HTTP URL
	var mailtoAddr, httpURL string
	
	// Split by comma in case there are multiple options
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Remove angle brackets
		part = strings.Trim(part, "<>")
		
		if strings.HasPrefix(part, "mailto:") {
			mailtoAddr = strings.TrimPrefix(part, "mailto:")
		} else if strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
			httpURL = part
		}
	}

	// Try HTTP unsubscribe first (more common and reliable)
	if httpURL != "" {
		resp, err := http.Post(httpURL, "application/x-www-form-urlencoded", nil)
		if err != nil {
			// Try GET if POST fails
			resp, err = http.Get(httpURL)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unsubscribe via HTTP", "details": err.Error()})
				return
			}
		}
		defer resp.Body.Close()
		
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			c.JSON(http.StatusOK, gin.H{
				"message": "Successfully unsubscribed via HTTP",
				"method":  "http",
				"url":     httpURL,
			})
			return
		}
	}

	// Try mailto unsubscribe
	if mailtoAddr != "" {
		emailService, err := s.getEmailService(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get email service"})
			return
		}

		// Create unsubscribe email
		message := fmt.Sprintf("To: %s\r\n"+
			"Subject: Unsubscribe\r\n"+
			"Content-Type: text/plain; charset=utf-8\r\n\r\n"+
			"Please unsubscribe me from this mailing list.", mailtoAddr)

		// Encode to base64
		encodedMessage := base64.URLEncoding.EncodeToString([]byte(message))

		// Send via Gmail API
		_, err = emailService.SendMessage("me", &gmail.Message{
			Raw: encodedMessage,
		})
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send unsubscribe email", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Unsubscribe email sent successfully",
			"method":  "mailto",
			"to":      mailtoAddr,
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "No valid unsubscribe method found in header"})
}

func (s *Server) IMAPDeleteHandler(c *gin.Context) {
	var req struct {
		Host     string   `json:"host"`
		Username string   `json:"username"`
		Password string   `json:"password"`
		Uids     []uint32 `json:"uids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Host == "" || req.Username == "" || req.Password == "" || len(req.Uids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid IMAP request"})
		return
	}
	err := imap.Delete(req.Host, req.Username, req.Password, req.Uids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "IMAP emails deleted"})
}

// GetEmailDetailsHandler now fetches the full email body.
func (s *Server) GetEmailDetailsHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get email service: " + err.Error()})
		return
	}
	id := c.Param("id")
	// Use the new fetcher method to get the full body
	details, err := emailService.GetFullMessage("me", id)
	if err != nil || details == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get email details"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"email": details})
}

// NEW: Handler to get emails from the trash.
func (s *Server) GetTrashEmailsHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get email service: " + err.Error()})
		return
	}
	
	// Get pagination parameters
	page := 1
	pageSize := 50
	var filter string

	if v := c.Query("page"); v != "" {
		fmt.Sscanf(v, "%d", &page)
	}
	if v := c.Query("pageSize"); v != "" {
		fmt.Sscanf(v, "%d", &pageSize)
	}
	filter = c.Query("filter")
	
	// Calculate offset for pagination
	offset := (page - 1) * pageSize
	
	// List messages with the TRASH label
	ids, err := emailService.ListMessageIDs("me", "", []string{"TRASH"}, int64(pageSize+offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list trash messages"})
		return
	}
	
	// Apply pagination
	totalEmails := len(ids)
	if offset >= totalEmails {
		c.JSON(http.StatusOK, gin.H{"emails": []string{}, "total": totalEmails, "page": page, "pageSize": pageSize})
		return
	}
	
	// Get the page slice
	endIdx := offset + pageSize
	if endIdx > totalEmails {
		endIdx = totalEmails
	}
	pageIds := ids[offset:endIdx]
	
	if len(pageIds) == 0 {
		c.JSON(http.StatusOK, gin.H{"emails": []string{}, "total": totalEmails, "page": page, "pageSize": pageSize})
		return
	}
	
	messages, err := emailService.GetMessageDetails("me", pageIds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get trash message details"})
		return
	}
	
	// Apply filter if provided
	if filter != "" {
		var filteredMessages []interface{}
		for _, msg := range messages {
			// Check if sender or subject contains the filter
			var sender, subject string
			for _, h := range msg.Payload.Headers {
				switch h.Name {
				case "From":
					sender = h.Value
				case "Subject":
					subject = h.Value
				}
			}
			if strings.Contains(strings.ToLower(sender), strings.ToLower(filter)) ||
			   strings.Contains(strings.ToLower(subject), strings.ToLower(filter)) {
				filteredMessages = append(filteredMessages, msg)
			}
		}
		c.JSON(http.StatusOK, gin.H{"emails": filteredMessages, "total": len(filteredMessages), "page": page, "pageSize": pageSize})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"emails": messages, "total": totalEmails, "page": page, "pageSize": pageSize})
}

// NEW: Handler to get archived emails.
func (s *Server) GetArchivedEmailsHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get email service: " + err.Error()})
		return
	}
	
	// Get pagination parameters
	page := 1
	pageSize := 50
	var filter string

	if v := c.Query("page"); v != "" {
		fmt.Sscanf(v, "%d", &page)
	}
	if v := c.Query("pageSize"); v != "" {
		fmt.Sscanf(v, "%d", &pageSize)
	}
	filter = c.Query("filter")
	
	// Calculate offset for pagination
	offset := (page - 1) * pageSize
	
	// Use a query to find messages that are not in INBOX, SPAM, or TRASH
	ids, err := emailService.ListMessageIDs("me", "-in:inbox -in:spam -in:trash", nil, int64(pageSize+offset))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list archived messages"})
		return
	}
	
	// Apply pagination
	totalEmails := len(ids)
	if offset >= totalEmails {
		c.JSON(http.StatusOK, gin.H{"emails": []string{}, "total": totalEmails, "page": page, "pageSize": pageSize})
		return
	}
	
	// Get the page slice
	endIdx := offset + pageSize
	if endIdx > totalEmails {
		endIdx = totalEmails
	}
	pageIds := ids[offset:endIdx]
	
	if len(pageIds) == 0 {
		c.JSON(http.StatusOK, gin.H{"emails": []string{}, "total": totalEmails, "page": page, "pageSize": pageSize})
		return
	}
	
	messages, err := emailService.GetMessageDetails("me", pageIds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get archived message details"})
		return
	}
	
	// Apply filter if provided
	if filter != "" {
		var filteredMessages []interface{}
		for _, msg := range messages {
			// Check if sender or subject contains the filter
			var sender, subject string
			for _, h := range msg.Payload.Headers {
				switch h.Name {
				case "From":
					sender = h.Value
				case "Subject":
					subject = h.Value
				}
			}
			if strings.Contains(strings.ToLower(sender), strings.ToLower(filter)) ||
			   strings.Contains(strings.ToLower(subject), strings.ToLower(filter)) {
				filteredMessages = append(filteredMessages, msg)
			}
		}
		c.JSON(http.StatusOK, gin.H{"emails": filteredMessages, "total": len(filteredMessages), "page": page, "pageSize": pageSize})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"emails": messages, "total": totalEmails, "page": page, "pageSize": pageSize})
}

// NEW: Handler to permanently delete an email.
func (s *Server) DeletePermanentHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get email service: " + err.Error()})
		return
	}
	id := c.Param("id")
	err = emailService.DeleteMessagePermanently("me", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to permanently delete email: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Email permanently deleted"})
}

// NEW: Handler to move an email from archive back to inbox.
func (s *Server) UnarchiveEmailHandler(c *gin.Context) {
	emailService, err := s.getEmailService(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to get email service: " + err.Error()})
		return
	}
	id := c.Param("id")
	err = emailService.UnarchiveMessage("me", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move email to inbox: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Email moved to inbox"})
}
