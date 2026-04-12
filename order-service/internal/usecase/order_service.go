package usecase

import (
	"context"
	"errors"
	"order-service/internal/domain"
	"time"

	"github.com/google/uuid"
)

type OrderService struct {
	repo          OrderRepository
	paymentClient PaymentClient
	statusUpdater StatusUpdater
}

func NewOrderService(repo OrderRepository, paymentClient PaymentClient) *OrderService {
	return &OrderService{repo: repo, paymentClient: paymentClient}
}

func (s *OrderService) SetStatusUpdater(updater StatusUpdater) {
	s.statusUpdater = updater
}

func (s *OrderService) CreateOrder(ctx context.Context, customerID, itemName string, amount int64, idempotencyKey string) (*domain.Order, bool, error) {
	if amount <= 0 {
		return nil, false, ErrInvalidAmount
	}

	if idempotencyKey != "" {
		existingOrderID, err := s.repo.GetOrderIDByIdempotencyKey(ctx, idempotencyKey)
		if err == nil {
			existingOrder, getErr := s.repo.GetByID(ctx, existingOrderID)
			if getErr != nil {
				return nil, false, getErr
			}
			return existingOrder, true, nil
		}
		if !errors.Is(err, ErrOrderNotFound) {
			return nil, false, err
		}
	}

	order := &domain.Order{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     domain.OrderStatusPending,
		CreatedAt:  time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, false, err
	}

	if idempotencyKey != "" {
		if err := s.repo.SaveIdempotencyKey(ctx, idempotencyKey, order.ID); err != nil {
			return nil, false, err
		}
	}
	return order, false, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// Keep the order ownership in this service, but sync payment outcome from Payment Service.
	if order.Status == domain.OrderStatusPending {
		paymentStatus, found, payErr := s.paymentClient.GetPaymentStatus(ctx, order.ID)
		if payErr == nil && found {
			nextStatus := order.Status
			if paymentStatus == "Authorized" {
				nextStatus = domain.OrderStatusPaid
			} else if paymentStatus == "Declined" {
				nextStatus = domain.OrderStatusFailed
			}

			if nextStatus != order.Status {
				if err := s.repo.UpdateStatus(ctx, order.ID, nextStatus); err == nil {
					order.Status = nextStatus
					if s.statusUpdater != nil {
						s.statusUpdater.NotifyStatusUpdate(order.ID, nextStatus)
					}
				}
			}
		}
	}

	return order, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	order, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.Status != domain.OrderStatusPending {
		return nil, ErrCannotCancel
	}

	if err := s.repo.UpdateStatus(ctx, orderID, domain.OrderStatusCancelled); err != nil {
		return nil, err
	}

	order.Status = domain.OrderStatusCancelled
	if s.statusUpdater != nil {
		s.statusUpdater.NotifyStatusUpdate(orderID, domain.OrderStatusCancelled)
	}
	return order, nil
}
