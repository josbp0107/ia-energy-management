package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewRouter(database *gorm.DB, frontendOrigin string) *gin.Engine {
	h := &Handler{db: database}

	router := gin.Default()
	router.Use(corsMiddleware(frontendOrigin))

	router.GET("/health", h.health)

	api := router.Group("/api/v1")
	api.GET("/dashboard/summary", h.dashboardSummary)
	api.GET("/meters", h.listMeters)
	api.GET("/meters/:meterId", h.getMeter)
	api.GET("/meters/:meterId/readings", h.listMeterReadings)
	api.GET("/events", h.listEvents)

	return router
}

func (h *Handler) health(c *gin.Context) {
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.Ping() != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "down"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "up"})
}

func corsMiddleware(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func respondError(c *gin.Context, status int, message string, err error) {
	if err != nil {
		_ = c.Error(err)
	}
	c.JSON(status, gin.H{"error": message})
}
