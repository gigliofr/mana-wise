package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
)

// OTAHandler exposes secure OTA release operations.
type OTAHandler struct {
	uc *usecase.OTAUpdateUseCase
}

// NewOTAHandler creates OTA handler.
func NewOTAHandler(uc *usecase.OTAUpdateUseCase) *OTAHandler {
	return &OTAHandler{uc: uc}
}

// PublishRelease handles POST /ota/release.
func (h *OTAHandler) PublishRelease(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		jsonError(w, "ota use case not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		Version      string `json:"version"`
		Platform     string `json:"platform"`
		BinaryBase64 string `json:"binary_base64"`
		SHA256       string `json:"sha256"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := h.uc.PublishRelease(r.Context(), domain.OTAReleaseRequest{
		Version:      req.Version,
		Platform:     req.Platform,
		BinaryBase64: req.BinaryBase64,
		SHA256:       req.SHA256,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	jsonOK(w, result)
}

// ReportBoot handles POST /ota/report-boot.
func (h *OTAHandler) ReportBoot(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		jsonError(w, "ota use case not configured", http.StatusServiceUnavailable)
		return
	}
	var req struct {
		DeviceID string `json:"device_id"`
		Version  string `json:"version"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	result, err := h.uc.ReportBootResult(r.Context(), domain.OTABootReportRequest{
		DeviceID: req.DeviceID,
		Version:  req.Version,
		Status:   req.Status,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	jsonOK(w, result)
}

// Manifest handles GET /ota/manifest.
func (h *OTAHandler) Manifest(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		jsonError(w, "ota use case not configured", http.StatusServiceUnavailable)
		return
	}
	m, err := h.uc.GetManifest(r.Context())
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, m)
}
