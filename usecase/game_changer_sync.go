package usecase

import (
	"context"
	"fmt"
	"time"
)

// GameChangerSource provides the official Game Changers list (e.g. Scryfall is:gamechanger).
type GameChangerSource interface {
	GameChangerNames(ctx context.Context) ([]string, error)
}

// Bounds used to reject an obviously wrong response (empty page, wrong query)
// instead of replacing a good list with it. The official list had 53 cards
// after the February 2026 update.
const (
	minGameChangers = 30
	maxGameChangers = 150
)

// SyncGameChangers fetches the official list and applies it to the bracket use
// case. On error, or if the result is implausible, the current list is kept.
func SyncGameChangers(ctx context.Context, src GameChangerSource, uc *CommanderBracketUseCase) (int, error) {
	if src == nil || uc == nil {
		return 0, fmt.Errorf("game changer sync: missing source or use case")
	}
	names, err := src.GameChangerNames(ctx)
	if err != nil {
		return 0, fmt.Errorf("game changer sync: %w", err)
	}
	if len(names) < minGameChangers || len(names) > maxGameChangers {
		return len(names), fmt.Errorf("game changer sync: implausible list size %d, keeping current list", len(names))
	}
	uc.SetGameChangers(names)
	return len(names), nil
}

// RunGameChangerSync syncs once immediately and then every interval until ctx
// is cancelled. logf receives one line per attempt.
func RunGameChangerSync(ctx context.Context, src GameChangerSource, uc *CommanderBracketUseCase, interval time.Duration, logf func(format string, args ...any)) {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	run := func() {
		callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		n, err := SyncGameChangers(callCtx, src, uc)
		if logf == nil {
			return
		}
		if err != nil {
			logf("⚠️  Game Changers sync failed (keeping current list): %v", err)
			return
		}
		logf("✅ Game Changers synced from Scryfall: %d cards", n)
	}
	run()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}
