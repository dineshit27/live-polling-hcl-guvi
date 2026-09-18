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
	ErrPollNotFound = errors.New("poll not found")
)

type PollRepository interface {
	Create(ctx context.Context, poll *models.Poll) error
	FindByID(ctx context.Context, id string) (*models.Poll, error)
	FindByOwnerID(ctx context.Context, ownerID string) ([]models.Poll, error)
	FindAllActive(ctx context.Context, limit, offset int64) ([]models.Poll, error)
	Update(ctx context.Context, id string, update bson.M) error
	Delete(ctx context.Context, id string) error
	IncrementOptionVote(ctx context.Context, pollID string, optionID string) error
}

type MongoPollRepository struct {
	collection *mongo.Collection
}

func NewPollRepository(db *mongo.Database) PollRepository {
	return &MongoPollRepository{
		collection: db.Collection("polls"),
	}
}

func (r *MongoPollRepository) Create(ctx context.Context, poll *models.Poll) error {
	now := time.Now().UTC()
	poll.CreatedAt = now
	poll.UpdatedAt = now
	if poll.Status == "" {
		poll.Status = "active"
	}
	if poll.ID == "" {
		poll.ID = bson.NewObjectID().Hex()
	}

	for i := range poll.Options {
		if poll.Options[i].ID == "" {
			poll.Options[i].ID = bson.NewObjectID().Hex()
		}
	}

	doc := bson.M{
		"_id":         poll.ID,
		"owner_id":    poll.OwnerID,
		"title":       poll.Title,
		"description": poll.Description,
		"status":      poll.Status,
		"options":     poll.Options,
		"total_votes": poll.TotalVotes,
		"created_at":  poll.CreatedAt,
		"updated_at":  poll.UpdatedAt,
	}

	_, err := r.collection.InsertOne(ctx, doc)
	return err
}

func (r *MongoPollRepository) FindByID(ctx context.Context, id string) (*models.Poll, error) {
	var poll models.Poll
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&poll)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrPollNotFound
		}
		return nil, err
	}
	return &poll, nil
}

func (r *MongoPollRepository) FindByOwnerID(ctx context.Context, ownerID string) ([]models.Poll, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.collection.Find(ctx, bson.M{"owner_id": ownerID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		return nil, err
	}
	if polls == nil {
		polls = []models.Poll{}
	}
	return polls, nil
}

func (r *MongoPollRepository) FindAllActive(ctx context.Context, limit, offset int64) ([]models.Poll, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(offset)

	cursor, err := r.collection.Find(ctx, bson.M{"status": "active"}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var polls []models.Poll
	if err := cursor.All(ctx, &polls); err != nil {
		return nil, err
	}
	if polls == nil {
		polls = []models.Poll{}
	}
	return polls, nil
}

func (r *MongoPollRepository) Update(ctx context.Context, id string, update bson.M) error {
	update["updated_at"] = time.Now().UTC()
	res, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrPollNotFound
	}
	return nil
}

func (r *MongoPollRepository) Delete(ctx context.Context, id string) error {
	res, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrPollNotFound
	}
	return nil
}

// IncrementOptionVote atomically increments the targeted option's vote count and the total poll votes.
func (r *MongoPollRepository) IncrementOptionVote(ctx context.Context, pollID string, optionID string) error {
	filter := bson.M{
		"_id":        pollID,
		"options.id": optionID,
		"status":     "active",
	}
	update := bson.M{
		"$inc": bson.M{
			"total_votes":        1,
			"options.$.vote_count": 1,
		},
		"$set": bson.M{
			"updated_at": time.Now().UTC(),
		},
	}

	res, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrPollNotFound
	}
	return nil
}
