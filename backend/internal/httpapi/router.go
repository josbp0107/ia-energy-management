package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/josbp0107/ia-energy-management/internal/ai"
	"github.com/josbp0107/ia-energy-management/internal/config"
)

type Handler struct {
	db        *gorm.DB
	stepDelay time.Duration
	explainer ai.Explainer
}

func NewRouter(database *gorm.DB, cfg config.Config, explainer ai.Explainer) (*gin.Engine, error) {
	h := &Handler{db: database, stepDelay: cfg.AnalysisStepDelay, explainer: explainer}
	auth, err := newAuth(cfg.Auth)
	if err != nil {
		return nil, err
	}

	router := gin.Default()
	router.Use(corsMiddleware(cfg.FrontendOrigin))

	router.GET("/health", h.health)

	api := router.Group("/api/v1")
	api.POST("/auth/login", auth.login)

	api = api.Group("", auth.requireToken)
	api.GET("/dashboard/summary", h.dashboardSummary)
	api.GET("/meters", h.listMeters)
	api.GET("/meters/:meterId", h.getMeter)
	api.GET("/meters/:meterId/readings", h.listMeterReadings)
	api.GET("/meters/:meterId/baseline", h.getMeterBaseline)
	api.GET("/events", h.listEvents)

	api.POST("/ai/analyze", h.startAnalysis)
	api.GET("/ai/analysis/:id", h.getAnalysis)
	api.GET("/anomalies", h.listAnomalies)
	api.GET("/anomalies/:id", h.getAnomaly)
	api.PATCH("/anomalies/:id", h.updateAnomalyStatus)

	return router, nil
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
