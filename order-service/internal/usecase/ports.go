package usecase

import (
	"context"
	"errors"
	"order-service/internal/domain"
)

var (
	ErrInvalidAmount      = errors.New("amount must be greater than zero")
	ErrOrderNotFound      = errors.New("order not found")
	ErrCannotCancel       = errors.New("only pending orders can be cancelled")
	ErrPaymentUnavailable = errors.New("payment service unavailable")
)

type OrderRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByID(ctx context.Context, orderID string) (*domain.Order, error)
	UpdateStatus(ctx context.Context, orderID, status string) error
	SaveIdempotencyKey(ctx context.Context, key, orderID string) error
	GetOrderIDByIdempotencyKey(ctx context.Context, key string) (string, error)
}

type PaymentClient interface {
	AuthorizePayment(ctx context.Context, orderID string, amount int64) (statusStr, transactionID string, err error)
	GetPaymentStatus(ctx context.Context, orderID string) (statusStr string, found bool, err error)
}

type StatusUpdater interface {
	NotifyStatusUpdate(orderID, status string)
}
