package project

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Strangebrewer/go-writer/utils/dumbwaiter"
	"github.com/google/uuid"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrNotFound = errors.New("project not found")

type projectDoc struct {
	ID          string     `bson:"_id"`
	UserID      string     `bson:"userId"`
	Title       string     `bson:"title"`
	Description string     `bson:"description"`
	SortOrder   int        `bson:"sortOrder"`
	ExpiresAt   *time.Time `bson:"expiresAt,omitempty"`
	CreatedAt   time.Time  `bson:"createdAt"`
	UpdatedAt   time.Time  `bson:"updatedAt"`
}

func (d projectDoc) toDomain() Project {
	return Project{
		ID:          d.ID,
		UserID:      d.UserID,
		Title:       d.Title,
		Description: d.Description,
		SortOrder:   d.SortOrder,
		ExpiresAt:   d.ExpiresAt,
	}
}

type Store struct {
	col      *mongo.Collection
	subjects *mongo.Collection
	texts    *mongo.Collection
}

func NewStore(db *mongo.Database) *Store {
	return &Store{
		col:      db.Collection("projects"),
		subjects: db.Collection("subjects"),
		texts:    db.Collection("texts"),
	}
}

func (s *Store) GetAll(ctx context.Context, userID uuid.UUID) ([]Project, error) {
	cursor, err := s.col.Find(ctx, bson.D{{Key: "userId", Value: userID.String()}})
	if err != nil {
		return nil, fmt.Errorf("get all projects: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []projectDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode projects: %w", err)
	}

	projects := make([]Project, len(docs))
	for i, d := range docs {
		projects[i] = d.toDomain()
	}

	return projects, nil
}

func (s *Store) GetByID(ctx context.Context, id, userID uuid.UUID) (Project, error) {
	var doc projectDoc
	err := s.col.FindOne(ctx, bson.D{
		{Key: "_id", Value: id.String()},
		{Key: "userId", Value: userID.String()},
	}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Project{}, ErrNotFound
		}
		return Project{}, fmt.Errorf("get project: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Create(ctx context.Context, userId uuid.UUID, req CreateProjectRequest, expiresAt *time.Time) (Project, error) {
	id, err := dumbwaiter.NewID()
	if err != nil {
		return Project{}, fmt.Errorf("generate id: %w", err)
	}

	now := time.Now().UTC()
	doc := projectDoc{
		ID:          id.String(),
		UserID:      userId.String(),
		Title:       req.Title,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := s.col.InsertOne(ctx, doc); err != nil {
		return Project{}, fmt.Errorf("create project: %w", err)
	}

	return doc.toDomain(), nil
}

func (s *Store) Update(ctx context.Context, id, userID uuid.UUID, req UpdateProjectRequest) (Project, error) {
	filter := bson.D{
		{Key: "_id", Value: id.String()},
		{Key: "userId", Value: userID.String()},
	}

	update := bson.D{{Key: "updatedAt", Value: time.Now().UTC()}}
	if req.Title != nil {
		update = append(update, bson.E{Key: "title", Value: req.Title})
	}
	if req.Description != nil {
		update = append(update, bson.E{Key: "description", Value: req.Description})
	}
	if req.SortOrder != nil {
		update = append(update, bson.E{Key: "sortOrder", Value: req.SortOrder})
	}

	var doc projectDoc
	err := s.col.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Project{}, ErrNotFound
		}
		return Project{}, fmt.Errorf("update project: %w", err)
	}

	return doc.toDomain(), nil
}

func (s *Store) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := s.texts.DeleteMany(ctx, bson.D{
		{Key: "projectId", Value: id.String()},
		{Key: "userId", Value: userID.String()},
	})
	if err != nil {
		return fmt.Errorf("error deleting texts: %w", err)
	}

	_, err = s.subjects.DeleteMany(ctx, bson.D{
		{Key: "projectId", Value: id.String()},
		{Key: "userId", Value: userID.String()},
	})
	if err != nil {
		return fmt.Errorf("error deleting subjects: %w", err)
	}

	result, err := s.col.DeleteOne(ctx, bson.D{
		{Key: "_id", Value: id.String()},
		{Key: "userId", Value: userID.String()},
	})
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if result.DeletedCount == 0 {
		return ErrNotFound
	}

	return nil
}
