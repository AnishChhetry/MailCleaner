package rules

import (
	"strings"
	"time"
)

// Rule struct now includes action and age
type Rule struct {
	Type    string
	Value   string
	Action  string
	AgeDays int
}

// Email struct now includes the date for age checking
type Email struct {
	Sender  string
	Subject string
	Snippet string
	Date    time.Time
}

// Match checks if an email matches a rule, including the new age condition
func Match(email Email, rule Rule) bool {
	contentMatch := false
	switch rule.Type {
	case "sender":
		// Use Contains for partial matches e.g. "John Doe <john@example.com>"
		contentMatch = strings.Contains(strings.ToLower(email.Sender), strings.ToLower(rule.Value))
	case "subject":
		contentMatch = strings.Contains(strings.ToLower(email.Subject), strings.ToLower(rule.Value))
	case "keyword":
		contentMatch = strings.Contains(strings.ToLower(email.Subject), strings.ToLower(rule.Value)) || strings.Contains(strings.ToLower(email.Snippet), strings.ToLower(rule.Value))
	default:
		return false
	}

	// If no age rule, the result is just the content match
	if rule.AgeDays <= 0 {
		return contentMatch
	}

	// If there is an age rule, check if the email is older than specified
	isOldEnough := time.Since(email.Date).Hours() > float64(rule.AgeDays*24)

	return contentMatch && isOldEnough
}

// ParseQuery returns keyword tokens very simplistically.
func ParseQuery(q string) []string {
	return strings.Fields(q)
}
