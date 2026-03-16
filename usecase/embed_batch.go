package usecase

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/manawise/api/domain"
)

// EmbeddingProvider creates vector embeddings for input text.
type EmbeddingProvider interface {
	EmbedText(ctx context.Context, input string) ([]float64, error)
}

// EmbedBatchRequest is the input for the batch embedding pipeline.
type EmbedBatchRequest struct {
	Limit int  `json:"limit"`
	Force bool `json:"force"`
}

// EmbedBatchResult summarizes the pipeline execution.
type EmbedBatchResult struct {
	Processed  int   `json:"processed"`
	Updated    int   `json:"updated"`
	Skipped    int   `json:"skipped"`
	Failed     int   `json:"failed"`
	DurationMs int64 `json:"duration_ms"`
}

// EmbedBatchUseCase generates embeddings and persists them in cards.embedding_vector.
type EmbedBatchUseCase struct {
	cardRepo domain.CardRepository
	embedder EmbeddingProvider
	poolSize int
}

// NewEmbedBatchUseCase creates a new EmbedBatchUseCase.
func NewEmbedBatchUseCase(cardRepo domain.CardRepository, embedder EmbeddingProvider, poolSize int) *EmbedBatchUseCase {
	if poolSize <= 0 {
		poolSize = 20
	}
	return &EmbedBatchUseCase{cardRepo: cardRepo, embedder: embedder, poolSize: poolSize}
}

// Execute selects cards, generates embeddings and saves vectors to MongoDB.
func (uc *EmbedBatchUseCase) Execute(ctx context.Context, req EmbedBatchRequest) (*EmbedBatchResult, error) {
	if uc.embedder == nil {
		return nil, fmt.Errorf("embedding provider is not configured")
	}
	if req.Limit <= 0 {
		req.Limit = 100
	}

	start := time.Now()
	cards, err := uc.cardRepo.FindForEmbedding(ctx, req.Limit, !req.Force)
	if err != nil {
		return nil, fmt.Errorf("load cards for embedding: %w", err)
	}
	if len(cards) == 0 {
		return &EmbedBatchResult{DurationMs: time.Since(start).Milliseconds()}, nil
	}

	var updated int64
	var failed int64
	var skipped int64

	results := WorkerPool(ctx, uc.poolSize, cards,
		func(ctx context.Context, card *domain.Card) (bool, error) {
			if !req.Force && len(card.EmbeddingVector) > 0 {
				atomic.AddInt64(&skipped, 1)
				return false, nil
			}

			input := cardEmbeddingText(card)
			if strings.TrimSpace(input) == "" {
				atomic.AddInt64(&skipped, 1)
				return false, nil
			}

			vector, err := uc.embedder.EmbedText(ctx, input)
			if err != nil {
				return false, err
			}
			if len(vector) == 0 {
				return false, fmt.Errorf("empty embedding for card %s", card.ID)
			}

			if err = uc.cardRepo.UpdateEmbedding(ctx, card.ID, vector); err != nil {
				return false, err
			}

			atomic.AddInt64(&updated, 1)
			return true, nil
		},
	)

	for _, r := range results {
		if r.Err != nil {
			atomic.AddInt64(&failed, 1)
		}
	}

	return &EmbedBatchResult{
		Processed:  len(cards),
		Updated:    int(updated),
		Skipped:    int(skipped),
		Failed:     int(failed),
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func cardEmbeddingText(card *domain.Card) string {
	parts := []string{
		card.Name,
		card.ManaCost,
		card.TypeLine,
		card.OracleText,
		strings.Join(card.Keywords, " "),
		strings.Join(card.Colors, " "),
		strings.Join(card.ColorIdentity, " "),
	}
	for _, f := range card.Faces {
		parts = append(parts, f.Name, f.TypeLine, f.OracleText)
	}
	return strings.TrimSpace(strings.Join(parts, " | "))
}
