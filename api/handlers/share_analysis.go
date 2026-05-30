package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/infrastructure/notifications"
	"github.com/gigliofr/mana-wise/usecase"
)

type ShareAnalysisHandler struct {
	Repo    domain.SharedAnalysisLinkRepository
	Mailer  domain.EmailSender
	BaseURL string
}

type ShareAnalysisAPIRequest struct {
	DeckID    string `json:"deck_id"`
	Channel   string `json:"channel"`
	Recipient string `json:"recipient,omitempty"`
	Message   string `json:"message,omitempty"`
	TTL       int64  `json:"ttl_hours,omitempty"`
}

type ShareAnalysisAPIResponse struct {
	ShareURL  string    `json:"share_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func NewShareAnalysisHandler(repo domain.SharedAnalysisLinkRepository, mailer domain.EmailSender) *ShareAnalysisHandler {
	baseURL := os.Getenv("PUBLIC_BASE_URL")
	if baseURL == "" {
		baseURL = "https://mana-wise.app"
	}
	if mailer == nil {
		mailer = domain.NoopEmailSender{}
	}
	return &ShareAnalysisHandler{Repo: repo, Mailer: mailer, BaseURL: baseURL}
}

func (h *ShareAnalysisHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	var req ShareAnalysisAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIErrorFromMsg(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.DeckID) == "" || strings.TrimSpace(req.Channel) == "" {
		WriteAPIErrorFromMsg(w, "deck_id e channel sono obbligatori", http.StatusBadRequest)
		return
	}
	var ttl time.Duration
	if req.TTL > 0 {
		ttl = time.Duration(req.TTL) * time.Hour
	}
	shareReq := usecase.ShareAnalysisRequest{
		DeckID:    req.DeckID,
		Channel:   req.Channel,
		Recipient: req.Recipient,
		Message:   req.Message,
		UserID:    userIDString(userID),
		TTL:       ttl,
	}
	resp, err := usecase.ShareAnalysis(r.Context(), h.Repo, shareReq, h.publicBaseURL(r))
	if err != nil {
		WriteAPIErrorFromMsg(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If the user asked to share via email, attempt to send a short notification with the link.
	if strings.EqualFold(req.Channel, "email") && strings.TrimSpace(req.Recipient) != "" {
		tpl := notifications.ShareAnalysisTemplate(resp.ShareURL, req.Message)
		// Best-effort: do not fail the request on email send errors.
		_ = h.Mailer.Send(req.Recipient, tpl.Subject, tpl.TextBody, tpl.HtmlBody)
	}
	jsonOK(w, ShareAnalysisAPIResponse{
		ShareURL:  resp.ShareURL,
		ExpiresAt: resp.ExpiresAt,
	})
}

func (h *ShareAnalysisHandler) publicBaseURL(r *http.Request) string {
	if resolved := requestPublicBaseURL(r); resolved != "" {
		return resolved
	}
	fallback := strings.TrimSpace(h.BaseURL)
	if fallback != "" {
		return strings.TrimRight(fallback, "/")
	}
	return "https://mana-wise.app"
}

func requestPublicBaseURL(r *http.Request) string {
	if r == nil {
		return ""
	}
	proto := firstHeaderValue(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := firstHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = strings.TrimSpace(r.Host)
	}
	if host == "" {
		return ""
	}
	if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
		return strings.TrimRight(host, "/")
	}
	return proto + "://" + strings.TrimRight(host, "/")
}

func firstHeaderValue(raw string) string {
	parts := strings.Split(raw, ",")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func userIDString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
