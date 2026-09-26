package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/josbp0107/ia-energy-management/internal/analysis"
	"github.com/josbp0107/ia-energy-management/internal/db"
)

type MeterSummary struct {
	MeterID        string     `gorm:"column:meter_id" json:"meter_id"`
	Name           string     `gorm:"column:name" json:"name"`
	Location       string     `gorm:"column:location" json:"location"`
	TotalKWh       float64    `gorm:"column:total_kwh" json:"total_kwh"`
	LastDayKWh     float64    `gorm:"column:last_day_kwh" json:"last_day_kwh"`
	BaselineKWhDay *float64   `gorm:"column:baseline_kwh_day" json:"baseline_kwh_day"`
	VariationPct   *float64   `gorm:"column:variation_pct" json:"variation_pct"`
	ReadingsCount  int        `gorm:"column:readings_count" json:"readings_count"`
	LastReadingAt  *time.Time `gorm:"column:last_reading_at" json:"last_reading_at"`
	Status         string     `gorm:"column:status" json:"status"` // NORMAL | ALERT | CRITICAL
	AnomalyID      *uint      `gorm:"column:anomaly_id" json:"anomaly_id"`
	AnomalyType    *string    `gorm:"column:anomaly_type" json:"anomaly_type"`
	Severity       *string    `gorm:"column:severity" json:"severity"`
	PriorityScore  *float64   `gorm:"column:priority_score" json:"priority_score"`
	AnomalyStatus  *string    `gorm:"column:anomaly_status" json:"anomaly_status"`
}

const meterSummarySQL = `
WITH last_run AS (
    SELECT id FROM analysis_runs WHERE status = @run_completed ORDER BY started_at DESC LIMIT 1
),
daily AS (
    SELECT meter_id, date_trunc('day', timestamp) AS day, SUM(consumption_kwh) AS kwh
    FROM readings
    GROUP BY meter_id, date_trunc('day', timestamp)
),
baseline AS (
    SELECT meter_id, percentile_cont(0.5) WITHIN GROUP (ORDER BY kwh) AS kwh
    FROM daily
    WHERE day < (SELECT date_trunc('day', MIN(timestamp)) FROM readings) + make_interval(days => @baseline_days)
    GROUP BY meter_id
),
last_day AS (
    SELECT DISTINCT ON (meter_id) meter_id, kwh FROM daily ORDER BY meter_id, day DESC
),
totals AS (
    SELECT meter_id, SUM(consumption_kwh) AS total_kwh, COUNT(*) AS readings_count, MAX(timestamp) AS last_reading_at
    FROM readings
    GROUP BY meter_id
)
SELECT m.meter_id, m.name, m.location,
       COALESCE(t.total_kwh, 0) AS total_kwh,
       COALESCE(ld.kwh, 0) AS last_day_kwh,
       ROUND(b.kwh::numeric, 1)::float8 AS baseline_kwh_day,
       CASE WHEN b.kwh > 0 THEN ROUND(((ld.kwh / b.kwh - 1) * 100)::numeric, 1)::float8 END AS variation_pct,
       COALESCE(t.readings_count, 0) AS readings_count,
       t.last_reading_at,
       CASE
           WHEN a.is_anomaly AND a.severity = @severity_high THEN @status_critical
           WHEN a.is_anomaly THEN @status_alert
           ELSE @status_normal
       END AS status,
       a.id AS anomaly_id, a.type AS anomaly_type, a.severity, a.priority_score, a.status AS anomaly_status
FROM meters m
LEFT JOIN totals t    ON t.meter_id = m.meter_id
LEFT JOIN last_day ld ON ld.meter_id = m.meter_id
LEFT JOIN baseline b  ON b.meter_id = m.meter_id
LEFT JOIN anomalies a ON a.meter_id = m.meter_id AND a.analysis_run_id = (SELECT id FROM last_run)
WHERE (@meter_id = '' OR m.meter_id = @meter_id)
ORDER BY m.meter_id`

// queryMeterSummaries
func (h *Handler) queryMeterSummaries(meterID string) ([]MeterSummary, error) {
	meters := make([]MeterSummary, 0)
	err := h.db.Raw(meterSummarySQL, map[string]any{
		"run_completed":   db.RunStatusCompleted,
		"baseline_days":   analysis.BaselineDays,
		"severity_high":   db.SeverityHigh,
		"status_critical": db.MeterStatusCritical,
		"status_alert":    db.MeterStatusAlert,
		"status_normal":   db.MeterStatusNormal,
		"meter_id":        meterID,
	}).Scan(&meters).Error
	return meters, err
}

// GET /meters
func (h *Handler) listMeters(c *gin.Context) {
	meters, err := h.queryMeterSummaries("")
	if err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudieron obtener los medidores", err)
		return
	}
	c.JSON(http.StatusOK, meters)
}

// GET /meters/:meterId
func (h *Handler) getMeter(c *gin.Context) {
	meters, err := h.queryMeterSummaries(c.Param("meterId"))
	if err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudo obtener el medidor", err)
		return
	}
	if len(meters) == 0 {
		respondError(c, http.StatusNotFound, "medidor no encontrado", nil)
		return
	}
	c.JSON(http.StatusOK, meters[0])
}

// GET /meters/:meterId/readings
func (h *Handler) listMeterReadings(c *gin.Context) {
	meterID := c.Param("meterId")

	var count int64
	if err := h.db.Model(&db.Meter{}).Where("meter_id = ?", meterID).Count(&count).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudo obtener el medidor", err)
		return
	}
	if count == 0 {
		respondError(c, http.StatusNotFound, "medidor no encontrado", nil)
		return
	}

	readings := make([]db.Reading, 0)
	err := h.db.Where("meter_id = ?", meterID).Order("timestamp").Find(&readings).Error
	if err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudieron obtener las lecturas", err)
		return
	}
	c.JSON(http.StatusOK, readings)
}

// GET /meters/:meterId/baseline
func (h *Handler) getMeterBaseline(c *gin.Context) {
	meterID := c.Param("meterId")

	var readings []db.Reading
	if err := h.db.Where("meter_id = ?", meterID).Order("timestamp").Find(&readings).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudieron obtener las lecturas", err)
		return
	}
	if len(readings) == 0 {
		respondError(c, http.StatusNotFound, "medidor no encontrado o sin lecturas", nil)
		return
	}

	end := analysis.BaselineEnd(readings)
	c.JSON(http.StatusOK, gin.H{
		"meter_id":      meterID,
		"baseline_from": end.AddDate(0, 0, -analysis.BaselineDays),
		"baseline_to":   end,
		"threshold_pct": analysis.DeviationThreshold * 100,
		"hours":         analysis.HourlyBaseline(readings),
	})
}

// GET /events?meter_id=M-109
func (h *Handler) listEvents(c *gin.Context) {
	query := h.db.Order("event_timestamp")
	if meterID := c.Query("meter_id"); meterID != "" {
		query = query.Where("meter_id = ?", meterID)
	}

	events := make([]db.Event, 0)
	if err := query.Find(&events).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudieron obtener los eventos", err)
		return
	}
	c.JSON(http.StatusOK, events)
}
