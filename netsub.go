package main

import "github.com/google/uuid"

type Frame struct {
	ID     uuid.UUID
	Topic  string
	Source string
	Data   []byte
}
