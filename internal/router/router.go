package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/uncle3dev/velotrax-core-go/internal/middleware"
)

// Setup initializes HTTP routes used for health and operations.
func Setup(engine *gin.Engine, logger *zap.Logger) {
	// Apply global middleware
	engine.Use(middleware.Logger(logger))
	engine.Use(middleware.Recovery(logger))
	engine.Use(middleware.CORS())

	// ── Health check ───────────────────────────────────────────────────
	engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "velotrax-core-go",
			"status":  "ok",
		})
	})
}
