package rabbitmq

import (
	"testing"

	"github.com/stretchr/testify/mock"

	amqp "github.com/rabbitmq/amqp091-go"
)

type mockAck struct {
	mock.Mock
}

var _ amqp.Acknowledger = (*mockAck)(nil)

func newMockAck(t *testing.T) *mockAck {
	t.Helper()
	m := &mockAck{}
	m.Test(t)
	t.Cleanup(func() { m.AssertExpectations(t) })
	return m
}

func (m *mockAck) Ack(tag uint64, multiple bool) error {
	return m.Called(tag, multiple).Error(0) //nolint:wrapcheck
}

func (m *mockAck) Nack(tag uint64, multiple, requeue bool) error {
	return m.Called(tag, multiple, requeue).Error(0) //nolint:wrapcheck
}

func (m *mockAck) Reject(tag uint64, requeue bool) error {
	return m.Called(tag, requeue).Error(0) //nolint:wrapcheck
}
