package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/manawise/api/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const usersCollection = "users"

// UserRepository implements domain.UserRepository against MongoDB.
type UserRepository struct {
	col *mongo.Collection
}

// NewUserRepository creates a UserRepository and ensures required indexes.
func NewUserRepository(ctx context.Context, client *Client) (*UserRepository, error) {
	col := client.Collection(usersCollection)
	repo := &UserRepository{col: col}
	if err := repo.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("user repo indexes: %w", err)
	}
	return repo, nil
}

func (r *UserRepository) ensureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "discord_id", Value: 1}},
			Options: options.Index().SetSparse(true),
		},
	}
	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	return err
}

// FindByID returns a user by their internal ID.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("UserRepo.FindByID: %w", err)
	}
	return &user, nil
}

// FindByEmail returns a user by their email address.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.col.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("UserRepo.FindByEmail: %w", err)
	}
	return &user, nil
}

// Create inserts a new user document.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now
	_, err := r.col.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("UserRepo.Create: %w", err)
	}
	return nil
}

// Update replaces an existing user document.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	user.UpdatedAt = time.Now().UTC()
	_, err := r.col.ReplaceOne(ctx, bson.M{"_id": user.ID}, user)
	if err != nil {
		return fmt.Errorf("UserRepo.Update: %w", err)
	}
	return nil
}

// CheckAndIncrementDailyAnalyses atomically verifies that the user has quota
// remaining today and increments the counter in one findOneAndUpdate call,
// eliminating any TOCTOU race between concurrent requests.
//
// The filter matches only when quota is available:
//   - the stored day differs from today (new day → reset to 1 and allow), OR
//   - the stored day equals today and daily_analyses < limit (still has quota).
//
// Returns (true, nil) when the increment succeeded, (false, nil) when the
// daily limit is exhausted, or (false, err) on a database error.
func (r *UserRepository) CheckAndIncrementDailyAnalyses(ctx context.Context, userID, today string, limit int) (bool, error) {
	filter := bson.D{
		{Key: "_id", Value: userID},
		{Key: "$or", Value: bson.A{
			// New day: last_analysis_day is different, so the counter will reset.
			bson.D{{Key: "last_analysis_day", Value: bson.D{{Key: "$ne", Value: today}}}},
			// Same day but under the limit.
			bson.D{{Key: "daily_analyses", Value: bson.D{{Key: "$lt", Value: limit}}}},
		}},
	}

	// Aggregation pipeline update: evaluated against the current document values.
	//   • If last_analysis_day == today → increment daily_analyses by 1.
	//   • Otherwise (new day)           → reset daily_analyses to 1.
	pipeline := bson.A{
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "daily_analyses", Value: bson.D{
				{Key: "$cond", Value: bson.A{
					bson.D{{Key: "$eq", Value: bson.A{"$last_analysis_day", today}}},
					bson.D{{Key: "$add", Value: bson.A{"$daily_analyses", 1}}},
					1,
				}},
			}},
			{Key: "last_analysis_day", Value: today},
			{Key: "updated_at", Value: time.Now().UTC()},
		}}},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated struct{} // we only need to know if a document was matched
	err := r.col.FindOneAndUpdate(ctx, filter, pipeline, opts).Decode(&updated)
	if err == mongo.ErrNoDocuments {
		// No match: either user not found or quota exhausted.
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("UserRepo.CheckAndIncrementDailyAnalyses: %w", err)
	}
	return true, nil
}
