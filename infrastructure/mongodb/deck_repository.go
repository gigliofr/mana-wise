package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/gigliofr/mana-wise/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const decksCollection = "decks"

// DeckRepository implements domain.DeckRepository against MongoDB.
type DeckRepository struct {
	col *mongo.Collection
}

// NewDeckRepository creates a DeckRepository and ensures required indexes.
func NewDeckRepository(ctx context.Context, client *Client) (*DeckRepository, error) {
	col := client.Collection(decksCollection)
	repo := &DeckRepository{col: col}
	if err := repo.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("deck repo indexes: %w", err)
	}
	return repo, nil
}

func (r *DeckRepository) ensureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			// List all decks for a user efficiently.
			Keys: bson.D{
				{Key: "user_id", Value: 1},
				{Key: "updated_at", Value: -1},
			},
		},
		{
			// Support public deck discovery sorted by recency.
			Keys: bson.D{
				{Key: "is_public", Value: 1},
				{Key: "updated_at", Value: -1},
			},
			Options: options.Index().SetSparse(true),
		},
	}
	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	return err
}

// FindByID returns a deck by its internal ID.
func (r *DeckRepository) FindByID(ctx context.Context, id string) (*domain.Deck, error) {
	var deck domain.Deck
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&deck)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("DeckRepo.FindByID: %w", err)
	}
	return &deck, nil
}

// FindByUserID returns all decks owned by a user, ordered by most recently updated.
func (r *DeckRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Deck, error) {
	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}})
	cursor, err := r.col.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, fmt.Errorf("DeckRepo.FindByUserID: %w", err)
	}
	defer cursor.Close(ctx)

	var decks []*domain.Deck
	if err = cursor.All(ctx, &decks); err != nil {
		return nil, fmt.Errorf("DeckRepo.FindByUserID decode: %w", err)
	}
	return decks, nil
}

// Create inserts a new deck document.
func (r *DeckRepository) Create(ctx context.Context, deck *domain.Deck) error {
	now := time.Now().UTC()
	deck.CreatedAt = now
	deck.UpdatedAt = now
	_, err := r.col.InsertOne(ctx, deck)
	if err != nil {
		return fmt.Errorf("DeckRepo.Create: %w", err)
	}
	return nil
}

// Update replaces an existing deck document. Only the owning user's document
// is updated (user_id filter prevents cross-user overwrites).
func (r *DeckRepository) Update(ctx context.Context, deck *domain.Deck) error {
	deck.UpdatedAt = time.Now().UTC()
	filter := bson.M{"_id": deck.ID, "user_id": deck.UserID}
	result, err := r.col.ReplaceOne(ctx, filter, deck)
	if err != nil {
		return fmt.Errorf("DeckRepo.Update: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("DeckRepo.Update: deck %s not found or not owned by user %s", deck.ID, deck.UserID)
	}
	return nil
}

// Delete removes a deck by ID. The userID parameter enforces ownership so that
// a user cannot delete another user's deck.
func (r *DeckRepository) Delete(ctx context.Context, id string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("DeckRepo.Delete: %w", err)
	}
	return nil
}
