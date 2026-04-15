package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gigliofr/mana-wise/api/middleware"
	"github.com/gigliofr/mana-wise/domain"
)

// AnalyticsHandler records user product events.
type AnalyticsHandler struct {
	tracker domain.AnalyticsTracker
}

// NewAnalyticsHandler creates analytics handler.
func NewAnalyticsHandler(tracker domain.AnalyticsTracker) *AnalyticsHandler {
	if tracker == nil {
		tracker = domain.NoopAnalyticsTracker{}
	}
	return &AnalyticsHandler{tracker: tracker}
}

// UpgradeClick handles POST /analytics/upgrade-click.
func (h *AnalyticsHandler) UpgradeClick(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Source string `json:"source"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	userID := middleware.UserIDFromContext(r.Context())
	plan := middleware.PlanFromContext(r.Context())
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "unknown"
	}
	_ = h.tracker.Track(r.Context(), userID, "upgrade_clicked", map[string]interface{}{
		"source": source,
		"plan":   plan,
	})
	jsonOK(w, map[string]string{"status": "ok"})
}
