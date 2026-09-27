package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/josbp0107/ia-energy-management/internal/ai"
	"github.com/josbp0107/ia-energy-management/internal/analysis"
	"github.com/josbp0107/ia-energy-management/internal/db"
)

type analysisRunResponse struct {
	db.AnalysisRun
	Steps []string `json:"steps"`
}

// POST /ai/analyze
func (h *Handler) startAnalysis(c *gin.Context) {
	run := db.AnalysisRun{
		Status:      db.RunStatusRunning,
		CurrentStep: db.StepReadings,
		StartedAt:   time.Now().UTC(),
	}
	if err := h.db.Create(&run).Error; err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudo crear el análisis", err)
		return
	}

	go h.runAnalysis(run.ID)

	c.JSON(http.StatusAccepted, analysisRunResponse{AnalysisRun: run, Steps: db.AnalysisSteps})
}

// GET /analysis/rules
func (h *Handler) getRules(c *gin.Context) {
	c.JSON(http.StatusOK, analysis.CurrentRules())
}

// GET /ai/analysis/:id
func (h *Handler) getAnalysis(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var run db.AnalysisRun
	err := h.db.First(&run, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respondError(c, http.StatusNotFound, "análisis no encontrado", nil)
		return
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, "no se pudo obtener el análisis", err)
		return
	}
	c.JSON(http.StatusOK, analysisRunResponse{AnalysisRun: run, Steps: db.AnalysisSteps})
}

// runAnalysis
func (h *Handler) runAnalysis(runID uint) {

	defer func() {
		if r := recover(); r != nil {
			h.failRun(runID, fmt.Errorf("panic: %v", r))
		}
	}()

	if err := h.executeAnalysis(runID); err != nil {
		h.failRun(runID, err)
	}
}

func (h *Handler) executeAnalysis(runID uint) error {
	if err := h.setStep(runID, db.StepReadings); err != nil {
		return err
	}
	var readings []db.Reading
	if err := h.db.Order("timestamp").Find(&readings).Error; err != nil {
		return fmt.Errorf("leer lecturas: %w", err)
	}
	var events []db.Event
	if err := h.db.Order("event_timestamp").Find(&events).Error; err != nil {
		return fmt.Errorf("leer eventos: %w", err)
	}

	if err := h.setStep(runID, db.StepBaseline); err != nil {
		return err
	}
	results := analysis.Analyze(readings, events)
	for _, step := range []string{db.StepDetection, db.StepCorrelation, db.StepEvents} {
		if err := h.setStep(runID, step); err != nil {
			return err
		}
	}

	if err := h.setStep(runID, db.StepExplanation); err != nil {
		return err
	}
	var flagged []analysis.Result
	for _, r := range results {
		if r.Type != db.TypeNormal {
			flagged = append(flagged, r)
		}
	}
	explanations := h.explainAll(flagged)

	var anomalies []db.Anomaly
	for i, r := range flagged {
		evidence, err := json.Marshal(r.Evidence)
		if err != nil {
			return fmt.Errorf("serializar evidencia de %s: %w", r.MeterID, err)
		}
		explanation := explanations[i]
		anomalies = append(anomalies, db.Anomaly{
			AnalysisRunID:     runID,
			MeterID:           r.MeterID,
			IsAnomaly:         r.Anomaly,
			Type:              r.Type,
			Severity:          r.Severity,
			Confidence:        r.Confidence,
			Reason:            explanation.Reason,
			RecommendedAction: explanation.RecommendedAction,
			ExplanationSource: explanation.Source,
			PriorityScore:     r.PriorityScore,
			VariationPct:      r.VariationPct,
			BaselineKWhDay:    r.BaselineKWhDay,
			CurrentKWhDay:     r.CurrentKWhDay,
			Evidence:          evidence,
			Status:            db.AnomalyStatusOpen,
		})
	}

	if err := h.setStep(runID, db.StepRecommendation); err != nil {
		return err
	}
	return h.db.Transaction(func(tx *gorm.DB) error {
		if len(anomalies) > 0 {
			if err := tx.Create(&anomalies).Error; err != nil {
				return fmt.Errorf("guardar anomalías: %w", err)
			}
		}
		return tx.Model(&db.AnalysisRun{}).Where("id = ?", runID).Updates(map[string]any{
			"status":      db.RunStatusCompleted,
			"finished_at": time.Now().UTC(),
		}).Error
	})
}

func (h *Handler) explainAll(results []analysis.Result) []ai.Explanation {
	explanations := make([]ai.Explanation, len(results))
	var wg sync.WaitGroup
	for i, r := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			explanations[i] = h.explainer.Explain(context.Background(), r)
		}()
	}
	wg.Wait()
	return explanations
}

func (h *Handler) setStep(runID uint, step string) error {
	err := h.db.Model(&db.AnalysisRun{}).Where("id = ?", runID).Update("current_step", step).Error
	if err != nil {
		return fmt.Errorf("actualizar paso %s: %w", step, err)
	}
	time.Sleep(h.stepDelay)
	return nil
}

func (h *Handler) failRun(runID uint, cause error) {
	log.Printf("análisis %d falló: %v", runID, cause)
	err := h.db.Model(&db.AnalysisRun{}).Where("id = ?", runID).Updates(map[string]any{
		"status":      db.RunStatusFailed,
		"error":       cause.Error(),
		"finished_at": time.Now().UTC(),
	}).Error
	if err != nil {
		log.Printf("no se pudo marcar el análisis %d como FAILED: %v", runID, err)
	}
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respondError(c, http.StatusBadRequest, "id inválido", nil)
		return 0, false
	}
	return uint(id), true
}
