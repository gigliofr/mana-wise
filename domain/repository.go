package domain

import "context"

// CardRepository defines persistence operations for cards.
type CardRepository interface {
	FindByID(ctx context.Context, id string) (*Card, error)
	FindByScryfallID(ctx context.Context, scryfallID string) (*Card, error)
	FindByName(ctx context.Context, name string) (*Card, error)
	FindByNames(ctx context.Context, names []string) ([]*Card, error)
	FindForEmbedding(ctx context.Context, limit int, onlyMissing bool) ([]*Card, error)
	Upsert(ctx context.Context, card *Card) error
	UpsertMany(ctx context.Context, cards []*Card) error
	UpdateEmbedding(ctx context.Context, id string, vector []float64) error
	FindWithEmbeddings(ctx context.Context, limit int) ([]*Card, error)
	CountAll(ctx context.Context) (int64, error)
}

// UserRepository defines persistence operations for users.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	// CheckAndIncrementDailyAnalyses atomically verifies the daily quota and
	// increments the counter in a single findOneAndUpdate operation.
	// Returns (true, nil) when the increment succeeded (quota was available),
	// (false, nil) when the user has exhausted their daily limit, or
	// (false, err) on a database error.
	CheckAndIncrementDailyAnalyses(ctx context.Context, userID, today string, limit int) (bool, error)
}

// DeckRepository defines persistence operations for decks.
type DeckRepository interface {
	FindByID(ctx context.Context, id string) (*Deck, error)
	FindByUserID(ctx context.Context, userID string) ([]*Deck, error)
	Create(ctx context.Context, deck *Deck) error
	Update(ctx context.Context, deck *Deck) error
	Delete(ctx context.Context, id string) error
}
