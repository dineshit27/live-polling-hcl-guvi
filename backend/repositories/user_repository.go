package repositories

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"live-polling-backend/models"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user with this email already exists")
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id string) (*models.User, error)
}

type MongoUserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *mongo.Database) UserRepository {
	return &MongoUserRepository{
		collection: db.Collection("users"),
	}
}

func (r *MongoUserRepository) Create(ctx context.Context, user *models.User) error {
	now := time.Now().UTC()
	user.CreatedAt = now
	user.UpdatedAt = now

	// If no ID is pre-set, assign a new ObjectID
	if user.ID == "" {
		user.ID = bson.NewObjectID().Hex()
	}

	doc := bson.M{
		"_id":           user.ID,
		"name":          user.Name,
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
	}

	_, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrUserAlreadyExists
		}
		return err
	}

	return nil
}

func (r *MongoUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *MongoUserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
