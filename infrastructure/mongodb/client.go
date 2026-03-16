package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Client wraps the MongoDB driver client.
type Client struct {
	inner *mongo.Client
	db    *mongo.Database
}

// NewClient creates and verifies a MongoDB connection.
func NewClient(ctx context.Context, uri, dbName string) (*Client, error) {
	opts := options.Client().
		ApplyURI(uri).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(10 * time.Second)

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb connect: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err = client.Ping(pingCtx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("mongodb ping: %w", err)
	}

	return &Client{
		inner: client,
		db:    client.Database(dbName),
	}, nil
}

// Database returns the underlying *mongo.Database.
func (c *Client) Database() *mongo.Database {
	return c.db
}

// Collection returns a named collection from the database.
func (c *Client) Collection(name string) *mongo.Collection {
	return c.db.Collection(name)
}

// Disconnect closes the client connection.
func (c *Client) Disconnect(ctx context.Context) error {
	return c.inner.Disconnect(ctx)
}
