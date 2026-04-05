package models
 
type Booking struct {
	ID      string `json:"id,omitempty" bson:"_id,omitempty"`
	EventID string `json:"event_id"     bson:"event_id"`
	UserID  string `json:"user_id"      bson:"user_id"`
	Seats   int    `json:"seats"        bson:"seats"`
}
 