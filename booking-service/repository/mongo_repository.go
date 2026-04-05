package repository

import (
	"booking-service/models"
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// BookingStore interface'inin gerçek MongoDB implementasyonu.
type MongoBookingRepository struct {
	collection *mongo.Collection
}

func NewMongoBookingRepository(db *mongo.Database) BookingStore {
	return &MongoBookingRepository{
		collection: db.Collection("bookings"),
	}
}

func (r *MongoBookingRepository) CreateBooking(booking models.Booking) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := r.collection.InsertOne(ctx, booking)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", res.InsertedID), nil
}

func (r *MongoBookingRepository) GetBookingsByUser(userID string) ([]models.Booking, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}

	var bookings []models.Booking
	if err := cursor.All(ctx, &bookings); err != nil {
		return nil, err
	}
	return bookings, nil
}