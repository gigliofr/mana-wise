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

// IncrementDailyAnalyses atomically increments the daily analysis counter.
// If the day has changed it resets the counter to 1.
func (r *UserRepository) IncrementDailyAnalyses(ctx context.Context, userID, today string) error {
	// Fetch current document to decide whether to reset or increment.
	user, err := r.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user %s not found", userID)
	}

	var update bson.M
	if user.LastAnalysisDay != today {
		update = bson.M{"$set": bson.M{
			"daily_analyses":    1,
			"last_analysis_day": today,
			"updated_at":        time.Now().UTC(),
		}}
	} else {
		update = bson.M{
			"$inc": bson.M{"daily_analyses": 1},
			"$set": bson.M{"updated_at": time.Now().UTC()},
		}
	}

	_, err = r.col.UpdateOne(ctx, bson.M{"_id": userID}, update)
	if err != nil {
		return fmt.Errorf("UserRepo.IncrementDailyAnalyses: %w", err)
	}
	return nil
}
