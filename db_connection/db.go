package db_connection

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Connect(ctx context.Context, mongoURI, dbName string) (*mongo.Client, *mongo.Database, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, nil, fmt.Errorf("db_connection: failed to connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, fmt.Errorf("db_connection: failed to ping: %w", err)
	}

	database := client.Database(dbName)

	ttlIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "expiresAt", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0).SetSparse(true),
	}

	projects := database.Collection("projects")
	_, err = projects.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}},
		ttlIndex,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("db_connection: failed to create project indexes: %w", err)
	}

	subjects := database.Collection("subjects")
	_, err = subjects.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}},
		ttlIndex,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("db_connection: failed to create subject indexes: %w", err)
	}

	texts := database.Collection("texts")
	_, err = texts.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}},
		ttlIndex,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("db_connection: failed to create text indexes: %w", err)
	}

	return client, database, nil
}
