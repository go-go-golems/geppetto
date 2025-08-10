package openai

import (
    "github.com/google/uuid"
)

// turnsID provides a generic unique ID for step metadata without depending on conversation package.
func turnsID() uuid.UUID {
    return uuid.New()
}


