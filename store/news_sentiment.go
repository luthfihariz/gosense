package store

import (
	"context"
	"gosense/entities"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type NewsSentimentStore struct {
	db      *mongo.Database
	context context.Context
}

func NewNewsSentimentStore(db *mongo.Database, context context.Context) *NewsSentimentStore {
	return &NewsSentimentStore{
		db:      db,
		context: context,
	}
}

func (ns *NewsSentimentStore) Search(keyword string) ([]entities.NewsSentiment, error) {
	filter := bson.D{{Key: "$text", Value: bson.D{{Key: "$search", Value: keyword}}}}
	cursor, err := ns.db.Collection("news_sentiment").Find(ns.context, filter)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ns.context)

	result := make([]entities.NewsSentiment, 0)
	for cursor.Next(ns.context) {
		var row entities.NewsSentiment
		err := cursor.Decode(&row)

		if err != nil {
			return nil, err
		}

		result = append(result, row)
	}

	return result, nil
}
