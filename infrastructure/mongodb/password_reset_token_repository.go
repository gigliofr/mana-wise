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

const passwordResetTokensCollection = "password_reset_tokens"

// PasswordResetTokenRepository implements domain.PasswordResetTokenRepository against MongoDB.
type PasswordResetTokenRepository struct {
	col *mongo.Collection
}

// NewPasswordResetTokenRepository creates a repository and ensures indexes.
func NewPasswordResetTokenRepository(ctx context.Context, client *Client) (*PasswordResetTokenRepository, error) {
	col := client.Collection(passwordResetTokensCollection)
	repo := &PasswordResetTokenRepository{col: col}
	if err := repo.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("password reset token repo indexes: %w", err)
	}
	return repo, nil
}

func (r *PasswordResetTokenRepository) ensureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	}
	_, err := r.col.Indexes().CreateMany(ctx, indexes)
	return err
}

func (r *PasswordResetTokenRepository) Create(ctx context.Context, token *domain.PasswordResetToken) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}
	_, err := r.col.InsertOne(ctx, token)
	if err != nil {
		return fmt.Errorf("PasswordResetTokenRepo.Create: %w", err)
	}
	return nil
}

func (r *PasswordResetTokenRepository) Consume(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	now := time.Now().UTC()
	filter := bson.M{
		"_id":        token,
		"expires_at": bson.M{"$gt": now},
	}

	var out domain.PasswordResetToken
	err := r.col.FindOneAndDelete(ctx, filter).Decode(&out)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("PasswordResetTokenRepo.Consume: %w", err)
	}

	usedAt := now
	out.UsedAt = &usedAt
	return &out, nil
}
