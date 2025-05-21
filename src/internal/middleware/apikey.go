package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rfid/internal/config"
)

type APIKeyMiddleware struct {
	db     *gorm.DB
	config *config.Config
}

func NewAPIKeyMiddleware(db *gorm.DB, config *config.Config) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		db:     db,
		config: config,
	}
}

func (m *APIKeyMiddleware) APIKeyRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !m.config.APIKeyRequired {
			c.Next()
			return
		}

		apiKey := c.GetHeader("X-API-Key")

		if apiKey == "" {
			apiKey = c.Query("api_key")
		}

		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API kulcs szükséges"})
			c.Abort()
			return
		}

		validKey := false
		for _, key := range m.config.APIKeys {
			if apiKey == key {
				validKey = true
				break
			}
		}

		if !validKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Érvénytelen API kulcs"})
			c.Abort()
			return
		}

		c.Next()
	}
}