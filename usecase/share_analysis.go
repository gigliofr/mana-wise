package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/gigliofr/mana-wise/domain"
)

const DefaultShareLinkTTL = 24 * time.Hour

// ShareAnalysisRequest rappresenta la richiesta per condividere un'analisi.
type ShareAnalysisRequest struct {
	DeckID    string
	Channel   string // email, whatsapp, etc
	Recipient string // opzionale
	Message   string // opzionale
	UserID    string // opzionale
	TTL       time.Duration // opzionale, default 24h
}

type ShareAnalysisResponse struct {
	ShareURL  string
	ExpiresAt time.Time
}

// ShareAnalysis genera un link pubblico temporaneo per la condivisione dell'analisi.
func ShareAnalysis(ctx context.Context, repo domain.SharedAnalysisLinkRepository, req ShareAnalysisRequest, baseURL string) (*ShareAnalysisResponse, error) {
	if req.DeckID == "" || req.Channel == "" {
		return nil, errors.New("deck_id e channel sono obbligatori")
	}
	ttl := req.TTL
	if ttl == 0 {
		ttl = DefaultShareLinkTTL
	}
	expiresAt := time.Now().Add(ttl)
	token, err := generateShareToken()
	if err != nil {
		return nil, err
	}
	link := &domain.SharedAnalysisLink{
		ID:        token,
		DeckID:    req.DeckID,
		UserID:    req.UserID,
		Channel:   req.Channel,
		Recipient: req.Recipient,
		Message:   req.Message,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
	if err := repo.Create(ctx, link); err != nil {
		return nil, err
	}
	shareURL := strings.TrimRight(baseURL, "/") + "/share/" + token
	return &ShareAnalysisResponse{
		ShareURL:  shareURL,
		ExpiresAt: expiresAt,
	}, nil
}

func generateShareToken() (string, error) {
	b := make([]byte, 12)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
