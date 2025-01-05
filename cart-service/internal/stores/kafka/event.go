package kafka

import "time"

const (
	Topic         = `user-service.account-created`
	ConsumerGroup = ``
)

// Representation of event that we would get in kafka

type StructureOfEvent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"` // Timestamp of creation
	UpdatedAt time.Time `json:"updated_at"` // Timestamp of last update
}
