package repository

import (
	"context"
	"errors"
	"event-service/models"
	"time"
 
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)
 
// Gerçek MongoDB implementasyonu
type EventRepository struct {
	collection *mongo.Collection
}
 
func NewEventRepository(collection *mongo.Collection) EventStore {
	return &EventRepository{collection: collection}
}

func (r *EventRepository) CreateEvent(event models.Event) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.collection.InsertOne(ctx, event)
	return err
}

func (r *EventRepository) GetAllEvents() ([]models.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
 
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	var events []models.Event
	err = cursor.All(ctx, &events)
	return events, err
}
 
func (r *EventRepository) GetEventByID(id string) (*models.Event, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
 
	var event models.Event
	err = r.collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&event)
	if err != nil {
		return nil, err
	}
	return &event, nil
}
 
func (r *EventRepository) UpdateEvent(id string, event models.Event) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
 
	update := bson.M{
		"$set": bson.M{
			"name":     event.Name,
			"location": event.Location,
			"capacity": event.Capacity,
			"date":     event.Date,
		},
	}
	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": objID}, update)
	return err
}
 
func (r *EventRepository) DeleteEvent(id string) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
 
	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": objID})
	return err
}
 
func (r *EventRepository) UpdateAvailableTickets(id string, amount int) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
 
	// $inc ile atomik güncelleme; available > 0 filtresi kontenjan taşmasını engeller.
	filter := bson.M{"_id": objID, "available": bson.M{"$gt": 0}}
	update := bson.M{"$inc": bson.M{"available": amount}}
 
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("etkinlik bulunamadı veya kontenjan yetersiz")
	}
	return nil
}