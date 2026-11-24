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

const cardsCollection = "cards"

// CardRepository implements domain.CardRepository against MongoDB.
type CardRepository struct {
	col *mongo.Collection
	g   guard
}

// NewCardRepository creates a CardRepository and ensures required indexes.
func NewCardRepository(ctx context.Context, client *Client) (*CardRepository, error) {
	col := client.Collection(cardsCollection)
	repo := &CardRepository{col: col, g: newGuard()}
	if err := repo.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("card repo indexes: %w", err)
	}
	return repo, nil
}

func (r *CardRepository) ensureIndexes(ctx context.Context) error {
	ctx, cancel := WithIndexTimeout(ctx)
	defer cancel()
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "scryfall_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
		{
			Keys: bson.D{{Key: "name", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "legalities", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "edhrec_rank", Value: 1}},
		},
	}
	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	return err
}

// findOne decodes a single card; a missing document returns (nil, nil).
func (r *CardRepository) findOne(ctx context.Context, op string, filter bson.M) (*domain.Card, error) {
	var card domain.Card
	found := true
	err := r.g.read(ctx, func(ctx context.Context) error {
		err := r.col.FindOne(ctx, filter).Decode(&card)
		if err == mongo.ErrNoDocuments {
			found = false
			return nil
		}
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if !found {
		return nil, nil
	}
	return &card, nil
}

// findMany decodes all matching cards.
func (r *CardRepository) findMany(ctx context.Context, op string, filter interface{}, opts ...*options.FindOptions) ([]*domain.Card, error) {
	var cards []*domain.Card
	err := r.g.read(ctx, func(ctx context.Context) error {
		cursor, err := r.col.Find(ctx, filter, opts...)
		if err != nil {
			return err
		}
		defer cursor.Close(ctx)
		return cursor.All(ctx, &cards)
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return cards, nil
}

// FindByID returns a card by its internal ID.
func (r *CardRepository) FindByID(ctx context.Context, id string) (*domain.Card, error) {
	return r.findOne(ctx, "FindByID", bson.M{"_id": id})
}

// FindByScryfallID returns a card by its Scryfall UUID.
func (r *CardRepository) FindByScryfallID(ctx context.Context, scryfallID string) (*domain.Card, error) {
	return r.findOne(ctx, "FindByScryfallID", bson.M{"scryfall_id": scryfallID})
}

// FindByName returns a card by exact name (case-sensitive).
func (r *CardRepository) FindByName(ctx context.Context, name string) (*domain.Card, error) {
	return r.findOne(ctx, "FindByName", bson.M{"name": name})
}

// FindByNames returns cards matching any of the provided names.
func (r *CardRepository) FindByNames(ctx context.Context, names []string) ([]*domain.Card, error) {
	return r.findMany(ctx, "FindByNames", bson.M{"name": bson.M{"$in": names}})
}

// FindForEmbedding returns cards selected for embedding generation.
// When onlyMissing=true it returns only cards without an embedding_vector.
func (r *CardRepository) FindForEmbedding(ctx context.Context, limit int, onlyMissing bool) ([]*domain.Card, error) {
	if limit <= 0 {
		limit = 100
	}
	filter := bson.M{}
	if onlyMissing {
		filter = bson.M{
			"$or": []bson.M{
				{"embedding_vector": bson.M{"$exists": false}},
				{"embedding_vector": bson.M{"$size": 0}},
				{"embedding_vector": nil},
			},
		}
	}
	return r.findMany(ctx, "FindForEmbedding", filter, options.Find().SetLimit(int64(limit)))
}

// Upsert inserts or replaces a card by its _id.
func (r *CardRepository) Upsert(ctx context.Context, card *domain.Card) error {
	card.UpdatedAt = time.Now().UTC()
	opts := options.Replace().SetUpsert(true)
	err := r.g.write(ctx, func(ctx context.Context) error {
		_, err := r.col.ReplaceOne(ctx, bson.M{"_id": card.ID}, card, opts)
		return err
	})
	if err != nil {
		return fmt.Errorf("Upsert: %w", err)
	}
	return nil
}

// UpsertMany performs bulk upserts for a slice of cards.
func (r *CardRepository) UpsertMany(ctx context.Context, cards []*domain.Card) error {
	if len(cards) == 0 {
		return nil
	}
	models := make([]mongo.WriteModel, 0, len(cards))
	now := time.Now().UTC()
	for _, card := range cards {
		card.UpdatedAt = now
		models = append(models, mongo.NewReplaceOneModel().
			SetFilter(bson.M{"_id": card.ID}).
			SetReplacement(card).
			SetUpsert(true))
	}
	opts := options.BulkWrite().SetOrdered(false)
	err := r.g.write(ctx, func(ctx context.Context) error {
		_, err := r.col.BulkWrite(ctx, models, opts)
		return err
	})
	if err != nil {
		return fmt.Errorf("UpsertMany: %w", err)
	}
	return nil
}

// UpdateEmbedding sets the embedding_vector field for a card.
func (r *CardRepository) UpdateEmbedding(ctx context.Context, id string, vector []float64) error {
	err := r.g.write(ctx, func(ctx context.Context) error {
		_, err := r.col.UpdateOne(ctx,
			bson.M{"_id": id},
			bson.M{"$set": bson.M{
				"embedding_vector": vector,
				"updated_at":       time.Now().UTC(),
			}},
		)
		return err
	})
	if err != nil {
		return fmt.Errorf("UpdateEmbedding: %w", err)
	}
	return nil
}

// FindWithEmbeddings returns cards that have a non-nil embedding_vector.
func (r *CardRepository) FindWithEmbeddings(ctx context.Context, limit int) ([]*domain.Card, error) {
	return r.findMany(ctx, "FindWithEmbeddings",
		bson.M{"embedding_vector": bson.M{"$exists": true, "$not": bson.M{"$size": 0}}},
		options.Find().SetLimit(int64(limit)),
	)
}

// CountAll returns the total number of cards in the collection.
func (r *CardRepository) CountAll(ctx context.Context) (int64, error) {
	var n int64
	err := r.g.read(ctx, func(ctx context.Context) error {
		var err error
		n, err = r.col.CountDocuments(ctx, bson.D{})
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("CountAll: %w", err)
	}
	return n, nil
}
