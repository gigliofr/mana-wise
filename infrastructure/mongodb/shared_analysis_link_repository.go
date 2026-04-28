package mongodb

import (
	"context"
	"time"

	"github.com/gigliofr/mana-wise/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// SharedAnalysisLinkRepositoryMongo implementa domain.SharedAnalysisLinkRepository su MongoDB.
type SharedAnalysisLinkRepositoryMongo struct {
	col *mongo.Collection
}

func NewSharedAnalysisLinkRepositoryMongo(db *mongo.Database) *SharedAnalysisLinkRepositoryMongo {
	return &SharedAnalysisLinkRepositoryMongo{
		col: db.Collection("shared_analysis_links"),
	}
}

func (r *SharedAnalysisLinkRepositoryMongo) Create(ctx context.Context, link *domain.SharedAnalysisLink) error {
	_, err := r.col.InsertOne(ctx, link)
	return err
}

func (r *SharedAnalysisLinkRepositoryMongo) FindByID(ctx context.Context, id string) (*domain.SharedAnalysisLink, error) {
	var link domain.SharedAnalysisLink
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&link)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &link, err
}

func (r *SharedAnalysisLinkRepositoryMongo) Delete(ctx context.Context, id string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (r *SharedAnalysisLinkRepositoryMongo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	res, err := r.col.DeleteMany(ctx, bson.M{"expires_at": bson.M{"$lte": now}})
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

// IncrementVisit incrementa il contatore visite e aggiorna last_visit.
func (r *SharedAnalysisLinkRepositoryMongo) IncrementVisit(ctx context.Context, id string, at time.Time) error {
	update := bson.M{
		"$inc": bson.M{"visit_count": 1},
		"$set": bson.M{"last_visit": at},
	}
	_, err := r.col.UpdateByID(ctx, id, update)
	return err
}
