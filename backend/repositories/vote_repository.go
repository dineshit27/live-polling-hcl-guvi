package repositories

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"live-polling-backend/models"
)

var (
	ErrAlreadyVoted = errors.New("voter has already cast a vote on this poll")
)

type VoteRepository interface {
	CastVote(ctx context.Context, vote *models.Vote) error
	HasVoted(ctx context.Context, pollID, voterIdentifier string) (bool, error)
	FindUserVote(ctx context.Context, pollID, voterIdentifier string) (*models.Vote, error)
	FindByPollID(ctx context.Context, pollID string) ([]models.Vote, error)
	DeleteVote(ctx context.Context, id string) error
}

type MongoVoteRepository struct {
	collection *mongo.Collection
}

func NewVoteRepository(db *mongo.Database) VoteRepository {
	return &MongoVoteRepository{
		collection: db.Collection("votes"),
	}
}

func (r *MongoVoteRepository) CastVote(ctx context.Context, vote *models.Vote) error {
	now := time.Now().UTC()
	vote.CreatedAt = now
	if vote.ID == "" {
		vote.ID = bson.NewObjectID().Hex()
	}

	doc := bson.M{
		"_id":              vote.ID,
		"poll_id":          vote.PollID,
		"option_id":        vote.OptionID,
		"voter_identifier": vote.VoterIdentifier,
		"ip_hash":          vote.IPHash,
		"created_at":       vote.CreatedAt,
	}

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrAlreadyVoted
		}
		return err
	}
	return nil
}

func (r *MongoVoteRepository) HasVoted(ctx context.Context, pollID, voterIdentifier string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{
		"poll_id":          pollID,
		"voter_identifier": voterIdentifier,
	})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *MongoVoteRepository) FindUserVote(ctx context.Context, pollID, voterIdentifier string) (*models.Vote, error) {
	var vote models.Vote
	err := r.collection.FindOne(ctx, bson.M{
		"poll_id":          pollID,
		"voter_identifier": voterIdentifier,
	}).Decode(&vote)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &vote, nil
}

func (r *MongoVoteRepository) FindByPollID(ctx context.Context, pollID string) ([]models.Vote, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"poll_id": pollID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var votes []models.Vote
	if err := cursor.All(ctx, &votes); err != nil {
		return nil, err
	}
	if votes == nil {
		votes = []models.Vote{}
	}
	return votes, nil
}

func (r *MongoVoteRepository) DeleteVote(ctx context.Context, id string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

