package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"order-service/internal/domain"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type OrderService struct {
	repo          OrderRepository
	paymentClient PaymentClient
	statusUpdater StatusUpdater
	redisClient   *redis.Client
}

func NewOrderService(repo OrderRepository, paymentClient PaymentClient, redisClient *redis.Client) *OrderService {
	return &OrderService{
		repo:          repo,
		paymentClient: paymentClient,
		redisClient:   redisClient,
	}
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
	// Try to get from cache first (Cache-aside pattern)
	if s.redisClient != nil {
		cacheKey := "order:" + orderID
		cachedData, err := s.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var order domain.Order
			if json.Unmarshal([]byte(cachedData), &order) == nil {
				return &order, nil
			}
		}
		// If cache miss or error, continue to database
	}

	// Get from database
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
					// Invalidate cache when status changes
					if s.redisClient != nil {
						s.redisClient.Del(ctx, "order:"+orderID)
					}
				}
			}
		}
	}

	// Cache the order (TTL: 5 minutes)
	if s.redisClient != nil {
		orderJSON, _ := json.Marshal(order)
		s.redisClient.Set(ctx, "order:"+orderID, orderJSON, 5*time.Minute)
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

	// Invalidate cache when status changes
	if s.redisClient != nil {
		s.redisClient.Del(ctx, "order:"+orderID)
	}

	return order, nil
}
