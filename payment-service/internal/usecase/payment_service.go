package usecase

import (
	"context"
	"errors"
	"log"
	"payment-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentService struct {
	repo           PaymentRepository
	orderLookup    OrderLookup
	eventPublisher EventPublisher
}

type EventPublisher interface {
	PublishPaymentEvent(ctx context.Context, orderID string, amount int64, status string) error
}

func NewPaymentService(repo PaymentRepository, orderLookup OrderLookup, eventPublisher EventPublisher) *PaymentService {
	return &PaymentService{repo: repo, orderLookup: orderLookup, eventPublisher: eventPublisher}
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

	// Publish event after successful payment
	if payment.Status == domain.PaymentStatusAuthorized {
		if s.eventPublisher != nil {
			if err := s.eventPublisher.PublishPaymentEvent(ctx, payment.OrderID, payment.Amount, string(payment.Status)); err != nil {
				// Log error but don't fail the payment
				// In production, you might want to implement retry or dead letter queue
				log.Printf("Failed to publish payment event: %v", err)
			}
		} else {
			log.Printf("Event publisher is not available, payment event for order %s was not published", payment.OrderID)
		}
	}

	return payment, nil
}

func (s *PaymentService) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	return s.repo.GetByOrderID(ctx, orderID)
}
