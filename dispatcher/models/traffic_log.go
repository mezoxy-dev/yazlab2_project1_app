package models

import "time"

type TrafficLog struct {
	Method      string     `bson:"method" json:"method"`
	Path        string     `bson:"path" json:"path"`
	Status      int        `bson:"status" json:"status"`
	Duration    int64      `bson:"duration_ms" json:"duration_ms"`
	IP          string     `bson:"ip" json:"ip"`
	Timestamp   time.Time  `bson:"timestamp" json:"timestamp"`
}