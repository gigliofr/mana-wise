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
	g   guard
}

func NewSharedAnalysisLinkRepositoryMongo(db *mongo.Database) *SharedAnalysisLinkRepositoryMongo {
	return &SharedAnalysisLinkRepositoryMongo{
		col: db.Collection("shared_analysis_links"),
		g:   newGuard(),
	}
}

func (r *SharedAnalysisLinkRepositoryMongo) Create(ctx context.Context, link *domain.SharedAnalysisLink) error {
	return r.g.write(ctx, func(ctx context.Context) error {
		_, err := r.col.InsertOne(ctx, link)
		return err
	})
}

func (r *SharedAnalysisLinkRepositoryMongo) FindByID(ctx context.Context, id string) (*domain.SharedAnalysisLink, error) {
	var link domain.SharedAnalysisLink
	found := true
	err := r.g.read(ctx, func(ctx context.Context) error {
		err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&link)
		if err == mongo.ErrNoDocuments {
			found = false
			return nil
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &link, nil
}

func (r *SharedAnalysisLinkRepositoryMongo) Delete(ctx context.Context, id string) error {
	return r.g.write(ctx, func(ctx context.Context) error {
		_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
		return err
	})
}

func (r *SharedAnalysisLinkRepositoryMongo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	var deleted int64
	err := r.g.write(ctx, func(ctx context.Context) error {
		res, err := r.col.DeleteMany(ctx, bson.M{"expires_at": bson.M{"$lte": now}})
		if err != nil {
			return err
		}
		deleted = res.DeletedCount
		return nil
	})
	return deleted, err
}

// IncrementVisit incrementa il contatore visite e aggiorna last_visit.
func (r *SharedAnalysisLinkRepositoryMongo) IncrementVisit(ctx context.Context, id string, at time.Time) error {
	update := bson.M{
		"$inc": bson.M{"visit_count": 1},
		"$set": bson.M{"last_visit": at},
	}
	return r.g.write(ctx, func(ctx context.Context) error {
		_, err := r.col.UpdateByID(ctx, id, update)
		return err
	})
}
