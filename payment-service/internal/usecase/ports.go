package usecase

import (
	"context"
	"errors"
	"payment-service/internal/domain"
)

var (
	ErrInvalidAmount  = errors.New("amount must be greater than zero")
	ErrNotFound       = errors.New("payment not found")
	ErrOrderNotFound  = errors.New("order with this id was never created")
	ErrAmountMismatch = errors.New("amount is wrong")
	ErrAlreadyPaid    = errors.New("This payment already done before plz check again.")
)

type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
	GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error)
}

type OrderLookup interface {
	GetOrder(ctx context.Context, orderID string) (amount int64, found bool, err error)
}
