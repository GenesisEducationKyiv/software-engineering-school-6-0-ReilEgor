package model

type OutboxMessage struct {
	ID      int64
	Queue   string
	Payload []byte
}
