package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/josbp0107/ia-energy-management/internal/db"
)

type DashboardSummary struct {
	TotalMeters         int64           `json:"total_meters"`
	TotalConsumptionKWh float64         `json:"total_consumption_kwh"`
	PeriodStart         *time.Time      `json:"period_start"`
	PeriodEnd           *time.Time      `json:"period_end"`
	AnomaliesDetected   int64           `json:"anomalies_detected"`
	HighPriority        int64           `json:"high_priority"`
	AvgConfidence       float64         `json:"avg_confidence"`
	LastAnalysis        *db.AnalysisRun `json:"last_analysis"`
}

// GET /dashboard/summary
func (h *Handler) dashboardSummary(c *gin.Context) {
	var summary DashboardSummary

	if err := h.db.Model(&db.Meter{}).Count(&summary.TotalMeters).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudo calcular el resumen", err)
		return
	}

	var totals struct {
		Total float64    `gorm:"column:total"`
		Start *time.Time `gorm:"column:period_start"`
		End   *time.Time `gorm:"column:period_end"`
	}
	err := h.db.Raw(`SELECT COALESCE(SUM(consumption_kwh), 0) AS total,
	                        MIN(timestamp) AS period_start, MAX(timestamp) AS period_end
	                 FROM readings`).Scan(&totals).Error
	if err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudo calcular el resumen", err)
		return
	}
	summary.TotalConsumptionKWh = totals.Total
	summary.PeriodStart = totals.Start
	summary.PeriodEnd = totals.End

	var run db.AnalysisRun
	err = h.db.Where("status = ?", db.RunStatusCompleted).Order("started_at DESC").First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusOK, summary)
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudo obtener el último análisis", err)
		return
	}
	summary.LastAnalysis = &run

	var ai struct {
		Anomalies     int64   `gorm:"column:anomalies"`
		HighPriority  int64   `gorm:"column:high_priority"`
		AvgConfidence float64 `gorm:"column:avg_confidence"`
	}
	err = h.db.Raw(`SELECT COUNT(*) AS anomalies,
	                       COUNT(*) FILTER (WHERE severity = ?) AS high_priority,
	                       COALESCE(AVG(confidence), 0) AS avg_confidence
	                FROM anomalies WHERE analysis_run_id = ?`, db.SeverityHigh, run.ID).Scan(&ai).Error
	if err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudo calcular el resumen", err)
		return
	}
	summary.AnomaliesDetected = ai.Anomalies
	summary.HighPriority = ai.HighPriority
	summary.AvgConfidence = ai.AvgConfidence

	c.JSON(http.StatusOK, summary)
}
