package models

import (
	"time"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TrafficLog struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Method      string             `bson:"method" json:"method"`
	Path        string             `bson:"path" json:"path"`
	Status      int                `bson:"status" json:"status"`
	Duration    int64              `bson:"duration_ms" json:"duration_ms"`
	IP          string             `bson:"ip" json:"ip"`
	Timestamp   time.Time          `bson:"timestamp" json:"timestamp"`
	Message     string             `bson:"message,omitempty" json:"message,omitempty"`
}