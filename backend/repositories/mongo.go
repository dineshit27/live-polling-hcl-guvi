package repositories

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type MongoClient struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// ConnectMongoDB connects to MongoDB with retries and exponential backoff.
func ConnectMongoDB(ctx context.Context, uri, dbName string) (*MongoClient, error) {
	clientOptions := options.Client().ApplyURI(uri)

	var client *mongo.Client
	var err error

	maxRetries := 5
	backoff := 500 * time.Millisecond

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[MongoDB] Attempting connection to %s (attempt %d/%d)...", uri, attempt, maxRetries)
		
		connCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		client, err = mongo.Connect(clientOptions)
		if err == nil {
			err = client.Ping(connCtx, readpref.Primary())
		}
		cancel()

		if err == nil {
			log.Printf("[MongoDB] Connected successfully to database '%s'", dbName)
			break
		}

		log.Printf("[MongoDB] Connection attempt %d failed: %v", attempt, err)
		if attempt < maxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB after %d attempts: %w", maxRetries, err)
	}

	db := client.Database(dbName)
	mongoClient := &MongoClient{
		Client:   client,
		Database: db,
	}

	// Ensure essential collections and indexes exist
	if err := mongoClient.EnsureIndexes(ctx); err != nil {
		log.Printf("[MongoDB] Warning: failed to ensure indexes: %v", err)
		return nil, fmt.Errorf("failed to initialize collection indexes: %w", err)
	}

	return mongoClient, nil
}

// EnsureIndexes creates required indexes including compound and unique constraints.
func (m *MongoClient) EnsureIndexes(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 1. Users collection: Unique Email index
	userCol := m.Database.Collection("users")
	_, err := userCol.Indexes().CreateOne(timeoutCtx, mongo.IndexModel{
		Keys: bson.D{{Key: "email", Value: 1}},
		Options: options.Index().
			SetUnique(true).
			SetName("uniq_user_email"),
	})
	if err != nil {
		return fmt.Errorf("failed to create users.email index: %w", err)
	}

	// 2. Polls collection: OwnerID and CreatedAt indexes
	pollCol := m.Database.Collection("polls")
	_, err = pollCol.Indexes().CreateMany(timeoutCtx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "owner_id", Value: 1}},
			Options: options.Index().SetName("idx_poll_owner"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_poll_created_at"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create poll indexes: %w", err)
	}

	// 3. Votes collection: Compound unique index (poll_id + voter_identifier)
	// Ensures a voter cannot cast more than one vote on the same poll
	voteCol := m.Database.Collection("votes")
	_, err = voteCol.Indexes().CreateMany(timeoutCtx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "poll_id", Value: 1},
				{Key: "voter_identifier", Value: 1},
			},
			Options: options.Index().
				SetUnique(true).
				SetName("uniq_poll_voter"),
		},
		{
			Keys:    bson.D{{Key: "poll_id", Value: 1}},
			Options: options.Index().SetName("idx_vote_poll_id"),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create vote indexes: %w", err)
	}

	log.Printf("[MongoDB] All collections and unique/compound indexes verified successfully.")
	return nil
}

// Close gracefully disconnects the MongoDB client.
func (m *MongoClient) Close(ctx context.Context) error {
	if m.Client != nil {
		return m.Client.Disconnect(ctx)
	}
	return nil
}
