package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Event struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Location    string             `json:"location" bson:"location"`
	Capacity    int                `json:"capacity" bson:"capacity"`
	Available   int                `json:"available" bson:"available"`
	Date        string             `json:"date" bson:"date"`
}

// TicketPatchRequest: PATCH /events/{id} isteğinin gövdesi
type TicketPatchRequest struct {
	Amount int `json:"amount"`
}
 