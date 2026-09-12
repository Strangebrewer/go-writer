package writer

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

var ErrSubjectNotFound = errors.New("subject not found")

type subjectDoc struct {
	ID          string     `bson:"_id"`
	UserID      string     `bson:"userId"`
	Title       string     `bson:"title"`
	Description string     `bson:"description"`
	SortOrder   int        `bson:"sortOrder"`
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
		SortOrder:   d.SortOrder,
		ProjectID:   d.ProjectID,
		ExpiresAt:   d.ExpiresAt,
	}
}

type SubjectStore struct {
	col      *mongo.Collection
	texts    *mongo.Collection
	projects *mongo.Collection
}

func NewSubjectStore(db *mongo.Database) *SubjectStore {
	return &SubjectStore{
		col:      db.Collection("subjects"),
		texts:    db.Collection("texts"),
		projects: db.Collection("projects"),
	}
}

func (s *SubjectStore) GetAllByProject(ctx context.Context, userID, projectID uuid.UUID) ([]Subject, error) {
	cursor, err := s.col.Find(ctx, bson.D{
		{Key: "projectId", Value: projectID.String()},
		{Key: "userId", Value: userID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("get subjects by projectId: %w", err)
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

func (s *SubjectStore) GetByID(ctx context.Context, id, userId uuid.UUID) (Subject, error) {
	var doc subjectDoc
	err := s.col.FindOne(ctx, bson.D{
		{Key: "_id", Value: id.String()},
		{Key: "userId", Value: userId.String()},
	}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Subject{}, ErrSubjectNotFound
		}
		return Subject{}, fmt.Errorf("get subject: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *SubjectStore) Create(ctx context.Context, userID uuid.UUID, req CreateSubjectRequest, expiresAt *time.Time) (Subject, error) {
	id, err := dumbwaiter.NewID()
	if err != nil {
		return Subject{}, fmt.Errorf("generate id: %w", err)
	}

	projects, err := s.projects.CountDocuments(ctx, bson.D{
		{Key: "_id", Value: req.ProjectID},
		{Key: "userId", Value: userID.String()},
	})
	if err != nil {
		return Subject{}, fmt.Errorf("verify project: %w", err)
	}
	if projects == 0 {
		return Subject{}, ErrProjectNotFound
	}

	now := time.Now().UTC()
	doc := subjectDoc{
		ID:          id.String(),
		UserID:      userID.String(),
		Title:       req.Title,
		Description: req.Description,
		SortOrder:   req.SortOrder,
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

func (s *SubjectStore) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	count, err := s.col.CountDocuments(ctx, bson.D{{Key: "userId", Value: userID.String()}})
	if err != nil {
		return 0, fmt.Errorf("count subjects: %w", err)
	}
	return count, nil
}

func (s *SubjectStore) Update(ctx context.Context, id, userID uuid.UUID, req UpdateSubjectRequest) (Subject, error) {
	filter := bson.D{
		{Key: "_id", Value: id.String()},
		{Key: "userId", Value: userID.String()},
	}

	update := bson.D{{Key: "updatedAt", Value: time.Now().UTC()}}
	if req.Description != nil {
		update = append(update, bson.E{Key: "description", Value: req.Description})
	}
	if req.Title != nil {
		update = append(update, bson.E{Key: "title", Value: req.Title})
	}
	if req.SortOrder != nil {
		update = append(update, bson.E{Key: "sortOrder", Value: req.SortOrder})
	}

	var doc subjectDoc
	err := s.col.FindOneAndUpdate(ctx, filter, bson.D{{Key: "$set", Value: update}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Subject{}, ErrSubjectNotFound
		}
		return Subject{}, fmt.Errorf("update subject: %w", err)
	}

	return doc.toDomain(), nil
}

func (s *SubjectStore) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := s.texts.DeleteMany(ctx, bson.D{
		{Key: "subjectId", Value: id.String()},
		{Key: "userId", Value: userID.String()},
	})
	if err != nil {
		return fmt.Errorf("error deleting texts: %w", err)
	}

	result, err := s.col.DeleteOne(ctx, bson.D{
		{Key: "_id", Value: id.String()},
		{Key: "userId", Value: userID.String()},
	})
	if err != nil {
		return fmt.Errorf("delete subject: %w", err)
	}
	if result.DeletedCount == 0 {
		return ErrSubjectNotFound
	}

	return nil
}
