package api

import (
	"encoding/json"
	"net/http"
	"os"

	"backend/internal/auth"
	"backend/internal/config"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var oauthConf *oauth2.Config

// Expose the config for the scheduler
func OAuthConfig() *oauth2.Config {
	return oauthConf
}

func InitOAuthConfig(cfg *config.Config) {
	oauthConf = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8080/auth/google/callback",
		// Use the single, all-encompassing mail scope for simplicity
		Scopes:   []string{"https://mail.google.com/", "openid", "email", "profile"},
		Endpoint: google.Endpoint,
	}
}

func sessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := c.Cookie("session_user")
		if err != nil && c.Request.URL.Path != "/auth/google/login" && c.Request.URL.Path != "/auth/google/callback" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			return
		}
		c.Next()
	}
}

func NewRouter(cfg *config.Config, store DataStore, tokenStore *auth.TokenStore) *gin.Engine {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	server := NewServer(store, tokenStore)

	r.GET("/auth/google/login", func(c *gin.Context) {
		url := oauthConf.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
		c.Redirect(http.StatusTemporaryRedirect, url)
	})

	r.GET("/auth/google/callback", func(c *gin.Context) {
		code := c.Query("code")
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No code in request"})
			return
		}
		tok, err := oauthConf.Exchange(c, code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Token exchange failed"})
			return
		}

		client := oauthConf.Client(c, tok)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get userinfo"})
			return
		}
		defer resp.Body.Close()
		var u struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode userinfo"})
			return
		}

		if err := tokenStore.Save(c, u.Email, tok); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save token"})
			return
		}

		http.SetCookie(c.Writer, &http.Cookie{
			Name:     "session_user",
			Value:    u.Email,
			Path:     "/",
			MaxAge:   3600 * 24, // Extend cookie to 24 hours
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false, // Set to true in production with HTTPS
		})
		c.Redirect(http.StatusTemporaryRedirect, "http://localhost:3000/")
	})

	authGroup := r.Group("/")
	authGroup.Use(sessionMiddleware())
	{
		authGroup.POST("/logout", func(c *gin.Context) {
			c.SetCookie("session_user", "", -1, "/", "", false, true)
			c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
		})

		authGroup.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

		// --- Email Routes (Corrected Order) ---
		// Static routes MUST be defined before parameterized routes.

		authGroup.GET("/emails", server.GetEmailsHandler)
		authGroup.GET("/emails/paginated", server.GetEmailsPaginatedHandler)
		authGroup.POST("/emails/sync", server.SyncEmailsHandler)
		authGroup.POST("/emails/sync-history", server.SyncHistoryHandler)
		authGroup.GET("/emails/sync/progress", server.GetSyncProgressHandler)
		authGroup.GET("/emails/trash", server.GetTrashEmailsHandler)
		authGroup.GET("/emails/archived", server.GetArchivedEmailsHandler)

		// Bulk action routes
		authGroup.POST("/emails/bulk/read", server.BulkMarkReadHandler)
		authGroup.POST("/emails/bulk/unread", server.BulkMarkUnreadHandler)
		authGroup.POST("/emails/bulk/delete", server.BulkDeleteHandler)
		authGroup.POST("/emails/bulk/archive", server.BulkArchiveHandler)

		// Parameterized routes come after all static routes with the same prefix.
		authGroup.GET("/emails/:id", server.GetEmailDetailsHandler)
		authGroup.DELETE("/emails/:id", server.DeleteEmailHandler)
		authGroup.POST("/emails/:id/archive", server.ArchiveEmailHandler)
		authGroup.POST("/emails/:id/read", server.MarkEmailReadHandler)
		authGroup.POST("/emails/:id/unread", server.MarkEmailUnreadHandler)
		authGroup.POST("/emails/:id/untrash", server.UntrashEmailHandler)
		authGroup.POST("/emails/:id/unarchive", server.UnarchiveEmailHandler)
		authGroup.DELETE("/emails/trash/:id", server.DeletePermanentHandler)

		// --- Rule Routes ---
		authGroup.GET("/rules", server.GetRulesHandler)
		authGroup.POST("/rules", server.CreateRuleHandler)
		authGroup.DELETE("/rules/:id", server.DeleteRuleHandler)
		authGroup.PUT("/rules/:id", server.UpdateRuleHandler)
		authGroup.PATCH("/rules/:id", server.UpdateRuleHandler)

		// --- Clean Routes ---
		authGroup.POST("/clean", server.CleanHandler)
		authGroup.POST("/clean/preview", server.CleanPreviewHandler)
		authGroup.GET("/clean/history", server.GetCleanHistoryHandler)

		// --- Other Feature Routes ---
		authGroup.POST("/block-sender", server.BlockSenderHandler)
		authGroup.POST("/unsubscribe-newsletter", server.UnsubscribeFromNewsletterHandler)
		authGroup.GET("/analytics/top-senders", server.GetSenderAnalyticsHandler)
		authGroup.GET("/analytics/subscribed-senders", server.GetSubscribedSendersHandler)
		authGroup.GET("/settings", server.GetSettingsHandler)
		authGroup.POST("/settings", server.UpdateSettingsHandler)

		authGroup.GET("/stats", server.GetStatsHandler)

		// --- IMAP Route ---
		authGroup.DELETE("/imap/emails", server.IMAPDeleteHandler)
	}

	debugGroup := r.Group("/debug")
	debugGroup.Use(sessionMiddleware())
	{
		debugGroup.POST("/reset-db", server.ResetDBHandler)
	}

	return r
}
