package idgen

import (
	"fmt"

	"github.com/google/uuid"
)

func New() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate uuidv7: %w", err)
	}

	return id.String(), nil
}
