package repository

import "context"

//go:generate mockery --name Transactor --output ../../mocks --case underscore --outpkg mocks
type Transactor interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
