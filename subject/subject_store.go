package subject

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Strangebrewer/go-writer/utils/dumbwaiter"
	"github.com/google/uuid"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var ErrNotFound = errors.New("subject not found")

type subjectDoc struct {
	ID          string     `bson:"_id"`
	UserID      string     `bson:"userId"`
	Title       string     `bson:"title"`
	Description string     `bson:"description"`
	ProjectID   string     `bson:"projectId"`
	ExpiresAt   *time.Time `bson:"expiresAt,omitempty"`
	CreatedAt   time.Time  `bson:"createdAt"`
	UpdatedAt   time.Time  `bson:"updatedAt"`
}

func (d subjectDoc) toDomain() Subject {
	return Subject{
		ID:          d.ID,
		UserID:      d.UserID,
		Title:       d.Title,
		Description: d.Description,
		ProjectID:   d.ProjectID,
		ExpiresAt:   d.ExpiresAt,
	}
}

type Store struct {
	col *mongo.Collection
}

func NewStore(db *mongo.Database) *Store {
	return &Store{
		col: db.Collection("subjects"),
	}
}

func (s *Store) GetAllByProject(ctx context.Context, userID, projectID uuid.UUID) ([]Subject, error) {
	cursor, err := s.col.Find(ctx, bson.D{
		{Key: "projectId", Value: projectID.String()},
		{Key: "userId", Value: userID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("get texts by projectId: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []subjectDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode subjects: %w", err)
	}

	subjects := make([]Subject, len(docs))
	for i, d := range docs {
		subjects[i] = d.toDomain()
	}

	return subjects, nil
}

func (s *Store) GetByID(ctx context.Context, id, userId uuid.UUID) (Subject, error) {
	var doc subjectDoc
	err := s.col.FindOne(ctx, bson.D{{Key: "_id", Value: id.String()}}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Subject{}, ErrNotFound
		}
		return Subject{}, fmt.Errorf("get subject: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Create(ctx context.Context, userId uuid.UUID, req CreateSubjectRequest, expiresAt *time.Time) (Subject, error) {
	id, err := dumbwaiter.NewID()
	if err != nil {
		return Subject{}, fmt.Errorf("generate id: %w", err)
	}

	now := time.Now().UTC()
	doc := subjectDoc{
		ID:          id.String(),
		UserID:      userId.String(),
		Title:       req.Title,
		Description: req.Description,
		ProjectID:   req.ProjectID,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := s.col.InsertOne(ctx, doc); err != nil {
		return Subject{}, fmt.Errorf("create subject: %w", err)
	}

	return doc.toDomain(), nil
}
