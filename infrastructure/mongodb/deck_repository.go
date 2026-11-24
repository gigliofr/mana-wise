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
	g   guard
}

// NewDeckRepository creates a DeckRepository and ensures required indexes.
func NewDeckRepository(ctx context.Context, client *Client) (*DeckRepository, error) {
	col := client.Collection(decksCollection)
	repo := &DeckRepository{col: col, g: newGuard()}
	if err := repo.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("deck repo indexes: %w", err)
	}
	return repo, nil
}

func (r *DeckRepository) ensureIndexes(ctx context.Context) error {
	ctx, cancel := WithIndexTimeout(ctx)
	defer cancel()
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
	found := true
	err := r.g.read(ctx, func(ctx context.Context) error {
		err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&deck)
		if err == mongo.ErrNoDocuments {
			found = false
			return nil
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("DeckRepo.FindByID: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &deck, nil
}

func (r *DeckRepository) findMany(ctx context.Context, filter bson.M, opts *options.FindOptions) ([]*domain.Deck, error) {
	var decks []*domain.Deck
	err := r.g.read(ctx, func(ctx context.Context) error {
		cursor, err := r.col.Find(ctx, filter, opts)
		if err != nil {
			return err
		}
		defer cursor.Close(ctx)
		return cursor.All(ctx, &decks)
	})
	return decks, err
}

// FindByUserID returns all decks owned by a user, ordered by most recently updated.
func (r *DeckRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.Deck, error) {
	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}})
	decks, err := r.findMany(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, fmt.Errorf("DeckRepo.FindByUserID: %w", err)
	}
	return decks, nil
}

// FindByUserIDPage returns a page of decks owned by a user and the total count.
func (r *DeckRepository) FindByUserIDPage(ctx context.Context, userID string, page, limit int) ([]*domain.Deck, int64, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	filter := bson.M{"user_id": userID}
	var total int64
	err := r.g.read(ctx, func(ctx context.Context) error {
		var err error
		total, err = r.col.CountDocuments(ctx, filter)
		return err
	})
	if err != nil {
		return nil, 0, fmt.Errorf("DeckRepo.FindByUserIDPage count: %w", err)
	}

	skip := int64((page - 1) * limit)
	opts := options.Find().
		SetSort(bson.D{{Key: "updated_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit))

	decks, err := r.findMany(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("DeckRepo.FindByUserIDPage find: %w", err)
	}
	return decks, total, nil
}

// Create inserts a new deck document.
func (r *DeckRepository) Create(ctx context.Context, deck *domain.Deck) error {
	now := time.Now().UTC()
	deck.CreatedAt = now
	deck.UpdatedAt = now
	err := r.g.write(ctx, func(ctx context.Context) error {
		_, err := r.col.InsertOne(ctx, deck)
		return err
	})
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
	var matched int64
	err := r.g.write(ctx, func(ctx context.Context) error {
		result, err := r.col.ReplaceOne(ctx, filter, deck)
		if err != nil {
			return err
		}
		matched = result.MatchedCount
		return nil
	})
	if err != nil {
		return fmt.Errorf("DeckRepo.Update: %w", err)
	}
	if matched == 0 {
		return fmt.Errorf("DeckRepo.Update: deck %s not found or not owned by user %s", deck.ID, deck.UserID)
	}
	return nil
}

// Delete removes a deck by ID. Ownership is checked by the caller.
func (r *DeckRepository) Delete(ctx context.Context, id string) error {
	err := r.g.write(ctx, func(ctx context.Context) error {
		_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
		return err
	})
	if err != nil {
		return fmt.Errorf("DeckRepo.Delete: %w", err)
	}
	return nil
}
