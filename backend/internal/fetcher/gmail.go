package fetcher

import (
	"context"
	"encoding/base64"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type GmailFetcher struct {
	srv *gmail.Service
}

func NewGmailFetcher(ctx context.Context, tokenSource option.ClientOption) (*GmailFetcher, error) {
	srv, err := gmail.NewService(ctx, tokenSource)
	if err != nil {
		return nil, err
	}
	return &GmailFetcher{srv: srv}, nil
}

// GetMessageDetails (Concurrent Version with Rate Limiting)
func (g *GmailFetcher) GetMessageDetails(userID string, ids []string) ([]*gmail.Message, error) {
	var results []*gmail.Message
	var wg sync.WaitGroup
	var mu sync.Mutex
	errChan := make(chan error, len(ids))

	// Create a semaphore to limit concurrency.
	concurrencyLimit := 10
	semaphore := make(chan struct{}, concurrencyLimit)

	// Process in chunks to avoid overwhelming the API
	chunkSize := 100
	for i := 0; i < len(ids); i += chunkSize {
		end := i + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[i:end]
		
		log.Infof("Processing chunk %d-%d of %d message IDs", i+1, end, len(ids))
		
		for _, id := range chunk {
			wg.Add(1)
			go func(msgID string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				msg, err := g.srv.Users.Messages.Get(userID, msgID).Format("metadata").Fields("id", "snippet", "payload/headers", "labelIds").Do()
				if err != nil {
					log.Errorf("Failed to get details for message ID %s: %v", msgID, err)
					errChan <- err
					return
				}
				mu.Lock()
				results = append(results, msg)
				mu.Unlock()
			}(id)
		}
		
		// Small delay between chunks to be respectful to the API
		if end < len(ids) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	wg.Wait()
	close(errChan)

	// Check if any goroutine reported an error
	if err := <-errChan; err != nil {
		return nil, err
	}

	log.Infof("Successfully fetched details for %d messages", len(results))
	return results, nil
}

// NEW functions for soft delete and undo
func (g *GmailFetcher) TrashMessage(userID, id string) error {
	_, err := g.srv.Users.Messages.Trash(userID, id).Do()
	return err
}

func (g *GmailFetcher) UntrashMessage(userID, id string) error {
	_, err := g.srv.Users.Messages.Untrash(userID, id).Do()
	return err
}

func (g *GmailFetcher) MarkRead(userID, id string) error {
	_, err := g.srv.Users.Messages.Modify(userID, id, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}).Do()
	return err
}

func (g *GmailFetcher) MarkUnread(userID, id string) error {
	_, err := g.srv.Users.Messages.Modify(userID, id, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{"UNREAD"},
	}).Do()
	return err
}

func (g *GmailFetcher) ArchiveMessage(userID, id string) error {
	_, err := g.srv.Users.Messages.Modify(userID, id, &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"INBOX"},
	}).Do()
	return err
}

// GetFullMessage fetches a single message with its full payload (body).
func (g *GmailFetcher) GetFullMessage(userID, messageID string) (*gmail.Message, error) {
	msg, err := g.srv.Users.Messages.Get(userID, messageID).Format("full").Do()
	if err != nil {
		return nil, err
	}

	// The body can be in parts (e.g., plain text and HTML)
	// This logic finds the best part and decodes it.
	if msg.Payload != nil {
		body, err := parseMessagePart(msg.Payload)
		if err == nil {
			// Replace the raw payload with our decoded, clean body
			msg.Snippet = body // Using Snippet field to hold the decoded body
		}
	}
	return msg, nil
}

// Helper function to parse the complex payload structure of a Gmail message.
func parseMessagePart(part *gmail.MessagePart) (string, error) {
	var plainText, htmlText string
	
	// First pass: collect both plain text and HTML
	if part.MimeType == "text/plain" && part.Body != nil && part.Body.Size > 0 {
		decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			plainText = string(decoded)
		}
	}

	if part.MimeType == "text/html" && part.Body != nil && part.Body.Size > 0 {
		decoded, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			htmlText = string(decoded)
		}
	}

	if strings.HasPrefix(part.MimeType, "multipart/") {
		for _, subPart := range part.Parts {
			body, err := parseMessagePart(subPart)
			if err == nil && body != "" {
				// Determine content type based on MIME type
				if strings.Contains(subPart.MimeType, "text/plain") {
					plainText = body
				} else if strings.Contains(subPart.MimeType, "text/html") {
					htmlText = body
				}
			}
		}
	}

	// Prefer plain text for readability, fall back to HTML if no plain text
	if plainText != "" {
		return plainText, nil
	}
	if htmlText != "" {
		return htmlText, nil
	}
	
	return "", nil
}

// ListAllMessageIDs fetches all message IDs with pagination support
func (g *GmailFetcher) ListAllMessageIDs(userID, query string, labelIDs []string) ([]string, error) {
	var allIDs []string
	pageToken := ""
	
	for {
		call := g.srv.Users.Messages.List(userID).MaxResults(500) // Gmail API max per page
		if query != "" {
			call.Q(query)
		}
		if len(labelIDs) > 0 {
			call.LabelIds(labelIDs...)
		}
		if pageToken != "" {
			call.PageToken(pageToken)
		}

		r, err := call.Do()
		if err != nil {
			return nil, err
		}

		// Add IDs from this page
		for _, m := range r.Messages {
			allIDs = append(allIDs, m.Id)
		}

		// Check if there are more pages
		if r.NextPageToken == "" {
			break
		}
		pageToken = r.NextPageToken
		
		log.Infof("Fetched %d message IDs so far, continuing...", len(allIDs))
	}
	
	log.Infof("Total message IDs fetched: %d", len(allIDs))
	return allIDs, nil
}

// ListMessageIDs can now list messages with a specific label or query (kept for backward compatibility)
func (g *GmailFetcher) ListMessageIDs(userID, query string, labelIDs []string, max int64) ([]string, error) {
	call := g.srv.Users.Messages.List(userID).MaxResults(max)
	if query != "" {
		call.Q(query)
	}
	if len(labelIDs) > 0 {
		call.LabelIds(labelIDs...)
	}

	r, err := call.Do()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(r.Messages))
	for _, m := range r.Messages {
		ids = append(ids, m.Id)
	}
	return ids, nil
}

func (g *GmailFetcher) DeleteMessagePermanently(userID, id string) error {
	return g.srv.Users.Messages.Delete(userID, id).Do()
}

func (g *GmailFetcher) UnarchiveMessage(userID, id string) error {
	_, err := g.srv.Users.Messages.Modify(userID, id, &gmail.ModifyMessageRequest{
		AddLabelIds: []string{"INBOX"},
	}).Do()
	return err
}

// HasInboxLabel returns whether the message currently has the INBOX label.
func (g *GmailFetcher) HasInboxLabel(userID, id string) (bool, error) {
    msg, err := g.srv.Users.Messages.Get(userID, id).Format("minimal").Do()
    if err != nil {
        return false, err
    }
    for _, lid := range msg.LabelIds {
        if lid == "INBOX" {
            return true, nil
        }
    }
    return false, nil
}

// Add these new functions to internal/fetcher/gmail.go

func (g *GmailFetcher) GetLabelMessageCount(userID string, labelID string) (int, error) {
	label, err := g.srv.Users.Labels.Get(userID, labelID).Fields("messagesTotal").Do()
	if err != nil {
		return 0, err
	}
	return int(label.MessagesTotal), nil
}

func (g *GmailFetcher) CountArchivedMessages(userID string) (int, error) {
	// Get the actual count of archived messages by fetching all message IDs
	// This ensures consistency with how archived emails are actually displayed
	ids, err := g.ListAllMessageIDs(userID, "-in:inbox -in:spam -in:trash", nil)
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

// SendMessage sends an email message via Gmail API
func (g *GmailFetcher) SendMessage(userID string, message *gmail.Message) (*gmail.Message, error) {
	return g.srv.Users.Messages.Send(userID, message).Do()
}
