package usecase

import (
	"context"
	"errors"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentService struct {
	repo        PaymentRepository
	orderLookup OrderLookup
}

func NewPaymentService(repo PaymentRepository, orderLookup OrderLookup) *PaymentService {
	return &PaymentService{repo: repo, orderLookup: orderLookup}
}

func (s *PaymentService) ProcessPayment(ctx context.Context, orderID string, amount int64) (*domain.Payment, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	orderAmount, found, err := s.orderLookup.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrOrderNotFound
	}
	if orderAmount != amount {
		return nil, ErrAmountMismatch
	}

	existing, err := s.repo.GetByOrderID(ctx, orderID)
	if err == nil {
		_ = existing
		return nil, ErrAlreadyPaid
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	status := domain.PaymentStatusAuthorized
	if amount > 100000 {
		status = domain.PaymentStatusDeclined
	}

	payment := &domain.Payment{
		ID:            uuid.NewString(),
		OrderID:       orderID,
		TransactionID: uuid.NewString(),
		Amount:        amount,
		Status:        status,
	}

	if err := s.repo.Create(ctx, payment); err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *PaymentService) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return s.repo.GetByOrderID(ctx, orderID)
}
