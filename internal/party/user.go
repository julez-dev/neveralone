package party

import (
	"github.com/google/uuid"
	"github.com/goombaio/namegenerator"
	"time"
)

type User struct {
	ID   uuid.UUID
	Name string
}

func NewRandomUser() *User {
	seed := time.Now().UTC().UnixNano()
	generator := namegenerator.NewNameGenerator(seed)
	name := generator.Generate()

	id := uuid.New()

	return &User{
		ID:   id,
		Name: name,
	}
}
