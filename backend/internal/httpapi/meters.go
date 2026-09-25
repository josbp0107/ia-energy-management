package httpapi

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/josbp0107/ia-energy-management/internal/db"
)

type MeterSummary struct {
	MeterID       string     `gorm:"column:meter_id" json:"meter_id"`
	Name          string     `gorm:"column:name" json:"name"`
	Location      string     `gorm:"column:location" json:"location"`
	TotalKWh      float64    `gorm:"column:total_kwh" json:"total_kwh"`
	LastDayKWh    float64    `gorm:"column:last_day_kwh" json:"last_day_kwh"`
	ReadingsCount int        `gorm:"column:readings_count" json:"readings_count"`
	LastReadingAt *time.Time `gorm:"column:last_reading_at" json:"last_reading_at"`
}

const meterSummarySQL = `
SELECT m.meter_id, m.name, m.location,
       COALESCE(SUM(r.consumption_kwh), 0) AS total_kwh,
       COALESCE(SUM(r.consumption_kwh) FILTER (WHERE r.timestamp >= date_trunc('day', last.max_ts)), 0) AS last_day_kwh,
       COUNT(r.id) AS readings_count,
       last.max_ts AS last_reading_at
FROM meters m
LEFT JOIN readings r ON r.meter_id = m.meter_id
LEFT JOIN (SELECT meter_id, MAX(timestamp) AS max_ts FROM readings GROUP BY meter_id) last
       ON last.meter_id = m.meter_id
WHERE (? = '' OR m.meter_id = ?)
GROUP BY m.meter_id, m.name, m.location, last.max_ts
ORDER BY m.meter_id`

func (h *Handler) queryMeterSummaries(meterID string) ([]MeterSummary, error) {
	var meters []MeterSummary
	err := h.db.Raw(meterSummarySQL, meterID, meterID).Scan(&meters).Error
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
