package text

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

var ErrNotFound = errors.New("text not found")

type textDoc struct {
	ID          string     `bson:"_id"`
	UserID      string     `bson:"userId"`
	Title       string     `bson:"title"`
	Description string     `bson:"description"`
	Content     string     `bson:"content"`
	SubjectID   string     `bson:"subjectId"`
	ProjectID   string     `bson:"projectId"`
	ExpiresAt   *time.Time `bson:"expiresAt,omitempty"`
	CreatedAt   time.Time  `bson:"createdAt"`
	UpdatedAt   time.Time  `bson:"updatedAt"`
}

func (d textDoc) toDomain() Text {
	return Text{
		ID:          d.ID,
		UserID:      d.UserID,
		Title:       d.Title,
		Description: d.Description,
		Content:     d.Content,
		SubjectID:   d.SubjectID,
		ProjectID:   d.ProjectID,
		ExpiresAt:   d.ExpiresAt,
	}
}

type Store struct {
	col *mongo.Collection
}

func NewStore(db *mongo.Database) *Store {
	return &Store{
		col: db.Collection("texts"),
	}
}

func (s *Store) GetAllBySubject(ctx context.Context, userID, subjectID uuid.UUID) ([]Text, error) {
	cursor, err := s.col.Find(ctx, bson.D{
		{Key: "subjectId", Value: subjectID.String()},
		{Key: "userId", Value: userID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("get texts by subjectId: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []textDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decode texts: %w", err)
	}

	texts := make([]Text, len(docs))
	for i, d := range docs {
		texts[i] = d.toDomain()
	}
	return texts, nil
}

func (s *Store) GetByID(ctx context.Context, id, userId uuid.UUID) (Text, error) {
	var doc textDoc
	err := s.col.FindOne(ctx, bson.D{{Key: "_id", Value: id.String()}}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Text{}, ErrNotFound
		}
		return Text{}, fmt.Errorf("get text: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Create(ctx context.Context, userID uuid.UUID, req CreateTextRequest, expiresAt *time.Time) (Text, error) {
	id, err := dumbwaiter.NewID()
	if err != nil {
		return Text{}, fmt.Errorf("generate id: %w", err)
	}

	now := time.Now().UTC()
	doc := textDoc{
		ID:          id.String(),
		UserID:      userID.String(),
		Title:       req.Title,
		Description: req.Description,
		Content:     req.Content,
		SubjectID:   req.SubjectID,
		ProjectID:   req.ProjectID,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpiresAt:   expiresAt,
	}

	if _, err := s.col.InsertOne(ctx, doc); err != nil {
		return Text{}, fmt.Errorf("create text: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Update(ctx context.Context, id, userID uuid.UUID, req UpdateTextRequest) (Text, error) {
	filter := bson.D{{Key: "_id", Value: id.String()}, {Key: "userId", Value: userID.String()}}
	update := bson.D{{Key: "updatedAt", Value: time.Now().UTC()}}
	if req.Title != nil {
		update = append(update, bson.E{Key: "title", Value: req.Title})
	}
	if req.Description != nil {
		update = append(update, bson.E{Key: "description", Value: req.Description})
	}
	if req.Content != nil {
		update = append(update, bson.E{Key: "content", Value: req.Content})
	}
	if req.SubjectID != nil {
		update = append(update, bson.E{Key: "subjectId", Value: req.SubjectID})
	}
	if req.ProjectID != nil {
		update = append(update, bson.E{Key: "projectId", Value: req.ProjectID})
	}

	var doc textDoc
	err := s.col.FindOneAndUpdate(ctx, filter, bson.D{{Key: "$set", Value: update}},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Text{}, ErrNotFound
		}
		return Text{}, fmt.Errorf("update text: %w", err)
	}
	return doc.toDomain(), nil
}

func (s *Store) Delete(ctx context.Context, id, userID uuid.UUID) {
	// needs access to the subject to remove the id from textOrder
}
