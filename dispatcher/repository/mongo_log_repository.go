package repository

import (
	"context"
	"dispatcher/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoLogRepository: LogStore interface'inin gerçek MongoDB implementasyonu.
type MongoLogRepository struct {
	collection *mongo.Collection
}

// NewMongoLogRepository: Dependency Injection constructor'ı.
// Dışarıya LogStore interface'i döner; somut tipi gizler.
func NewMongoLogRepository(collection *mongo.Collection) LogStore {
	return &MongoLogRepository{collection: collection}
}


func (r *MongoLogRepository) Insert(log models.TrafficLog) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := r.collection.InsertOne(ctx, log)
	return err
}


func (r *MongoLogRepository) GetRecent(limit int64) ([]models.TrafficLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().
		SetSort(bson.M{"timestamp": -1}).
		SetLimit(limit)

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}

	var logs []models.TrafficLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}