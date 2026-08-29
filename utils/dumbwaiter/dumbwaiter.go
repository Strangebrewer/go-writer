package dumbwaiter

import "github.com/google/uuid"

func NewID() (uuid.UUID, error) {
	return uuid.NewV7()
}
