package mongodb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/gigliofr/mana-wise/config"
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
// If cfg.TLSCertFile is set, X.509 mutual-TLS authentication is configured
// using the PEM file (which must contain both the certificate and the private key).
func NewClient(ctx context.Context, cfg config.MongoDBConfig) (*Client, error) {
	opts := options.Client().
		ApplyURI(cfg.URI).
		SetConnectTimeout(15 * time.Second).
		SetServerSelectionTimeout(20 * time.Second)

	if cfg.TLSCertFile != "" {
		tlsCfg, err := loadX509TLSConfig(cfg.TLSCertFile)
		if err != nil {
			return nil, fmt.Errorf("mongodb tls config: %w", err)
		}
		opts.SetTLSConfig(tlsCfg)
	}

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb connect: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	if err = client.Ping(pingCtx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongodb ping: %w", err)
	}

	return &Client{
		inner: client,
		db:    client.Database(cfg.Database),
	}, nil
}

// loadX509TLSConfig builds a *tls.Config from a PEM file containing a client
// certificate and private key. The file may (optionally) also contain one or
// more CA certificates that are added to the trust pool.
func loadX509TLSConfig(pemFile string) (*tls.Config, error) {
	pemData, err := os.ReadFile(pemFile) // #nosec G304 — path comes from trusted config
	if err != nil {
		return nil, fmt.Errorf("read cert file %q: %w", pemFile, err)
	}

	// tls.X509KeyPair parses the first CERTIFICATE block and the first
	// PRIVATE KEY / RSA PRIVATE KEY block from the supplied PEM data.
	// Passing pemData to both arguments works when they are in the same file.
	cert, err := tls.X509KeyPair(pemData, pemData)
	if err != nil {
		return nil, fmt.Errorf("parse x509 key pair: %w", err)
	}

	// Optionally extend the system cert pool with any CA certs in the file.
	rootPool, err := x509.SystemCertPool()
	if err != nil {
		rootPool = x509.NewCertPool()
	}
	rootPool.AppendCertsFromPEM(pemData)

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      rootPool,
		MinVersion:   tls.VersionTLS12,
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
