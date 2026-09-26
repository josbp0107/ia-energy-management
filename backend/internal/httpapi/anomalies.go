package httpapi

import (
	"errors"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/josbp0107/ia-energy-management/internal/db"
)

// GET /anomalies?meter_id=&status=&severity=
func (h *Handler) listAnomalies(c *gin.Context) {
	anomalies := make([]db.Anomaly, 0)

	var run db.AnalysisRun
	err := h.db.Where("status = ?", db.RunStatusCompleted).Order("started_at DESC").First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, anomalies)
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudo obtener el último análisis", err)
		return
	}

	query := h.db.Where("analysis_run_id = ?", run.ID).Order("priority_score DESC")
	for _, filter := range []string{"meter_id", "status", "severity"} {
		if value := c.Query(filter); value != "" {
			query = query.Where(filter+" = ?", value)
		}
	}
	if err := query.Find(&anomalies).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudieron obtener las anomalías", err)
		return
	}
	c.JSON(http.StatusOK, anomalies)
}

// GET /anomalies/:id
func (h *Handler) getAnomaly(c *gin.Context) {
	anomaly, ok := h.findAnomaly(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, anomaly)
}

// PATCH /anomalies/:id  body: {"status": "INVESTIGATING"}
func (h *Handler) updateAnomalyStatus(c *gin.Context) {
	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "body JSON inválido", nil)
		return
	}
	if !slices.Contains(db.AnomalyStatuses, body.Status) {
		respondError(c, http.StatusBadRequest, "status inválido; valores permitidos: OPEN, INVESTIGATING, RESOLVED, DISMISSED", nil)
		return
	}

	anomaly, ok := h.findAnomaly(c)
	if !ok {
		return
	}

	if err := h.db.Model(&anomaly).Update("status", body.Status).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudo actualizar la anomalía", err)
		return
	}
	c.JSON(http.StatusOK, anomaly)
}

// busca la anomalía de :id
func (h *Handler) findAnomaly(c *gin.Context) (db.Anomaly, bool) {
	var anomaly db.Anomaly
	id, ok := parseID(c)
	if !ok {
		return anomaly, false
	}

	err := h.db.First(&anomaly, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respondError(c, http.StatusNotFound, "anomalía no encontrada", nil)
		return anomaly, false
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudo obtener la anomalía", err)
		return anomaly, false
	}
	return anomaly, true
}
