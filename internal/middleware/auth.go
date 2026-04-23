package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type CustomClaims struct {
	UserID string   `json:"sub"`
	Roles  []string `json:"roles"`
	Type   string   `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// RequireAuth validates access token from Authorization header
func RequireAuth(secret string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("missing authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			logger.Warn("invalid authorization header format")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := authHeader[len(bearerPrefix):]

		claims := &CustomClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			logger.Warn("invalid token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Verify token type
		if claims.Type != "access" {
			logger.Warn("invalid token type", zap.String("type", claims.Type))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
			c.Abort()
			return
		}

		// Inject into context
		c.Set("userID", claims.UserID)
		c.Set("roles", claims.Roles)

		c.Next()
	}
}

// RequireRefreshToken validates refresh token from Authorization header
func RequireRefreshToken(secret string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Warn("missing authorization header")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			logger.Warn("invalid authorization header format")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := authHeader[len(bearerPrefix):]

		claims := &CustomClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			logger.Warn("invalid token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Verify token type
		if claims.Type != "refresh" {
			logger.Warn("invalid token type", zap.String("type", claims.Type))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
			c.Abort()
			return
		}

		// Inject into context
		c.Set("userID", claims.UserID)

		c.Next()
	}
}

// GetUserID retrieves userID from context
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get("userID")
	if !exists {
		return ""
	}
	if id, ok := userID.(string); ok {
		return id
	}
	return ""
}

// GetRoles retrieves roles from context
func GetRoles(c *gin.Context) []string {
	roles, exists := c.Get("roles")
	if !exists {
		return []string{}
	}
	if r, ok := roles.([]string); ok {
		return r
	}
	return []string{}
}
