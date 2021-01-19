package submission

import (
	"github.com/google/uuid"
	"log"
)

// Enclave represents a temporary on-disk file structure and an interface to manipulate said data.
type Enclave struct {
	ID uuid.UUID
}

// When we first initialize the Enclave struct,
func (e *Enclave) init() {
	e.ID = generateUUID()
}

func genUUID() uuid.UUID {
	id, err := uuid.NewUUID()
	if err != nil {
		log.Fatalln(err)
	}
	return id
}
