package grpc

import (
	"order-service/internal/usecase"
	"sync"
	"time"

	"github.com/taubakabylnurlybek/ap2-generated/order"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderServiceServer struct {
	order.UnimplementedOrderServiceServer
	orderUC *usecase.OrderService
	mu      sync.RWMutex
	subs    map[string][]chan *order.OrderStatusUpdate
}

func NewOrderServiceServer(orderUC *usecase.OrderService) *OrderServiceServer {
	return &OrderServiceServer{
		orderUC: orderUC,
		subs:    make(map[string][]chan *order.OrderStatusUpdate),
	}
}

func (s *OrderServiceServer) SubscribeToOrderUpdates(req *order.OrderRequest, stream order.OrderService_SubscribeToOrderUpdatesServer) error {
	orderID := req.OrderId

	// Send initial status
	orderData, err := s.orderUC.GetOrder(stream.Context(), orderID)
	if err != nil {
		return err
	}

	update := &order.OrderStatusUpdate{
		OrderId:   orderData.ID,
		Status:    orderData.Status,
		UpdatedAt: timestamppb.New(time.Now()),
	}
	if err := stream.Send(update); err != nil {
		return err
	}

	// Subscribe to updates
	ch := make(chan *order.OrderStatusUpdate, 10)
	s.mu.Lock()
	s.subs[orderID] = append(s.subs[orderID], ch)
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		channels := s.subs[orderID]
		for i, c := range channels {
			if c == ch {
				s.subs[orderID] = append(channels[:i], channels[i+1:]...)
				break
			}
		}
		if len(s.subs[orderID]) == 0 {
			delete(s.subs, orderID)
		}
		s.mu.Unlock()
		close(ch)
	}()

	for {
		select {
		case update := <-ch:
			if err := stream.Send(update); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *OrderServiceServer) NotifyStatusUpdate(orderID, status string) {
	s.mu.RLock()
	channels := s.subs[orderID]
	s.mu.RUnlock()

	update := &order.OrderStatusUpdate{
		OrderId:   orderID,
		Status:    status,
		UpdatedAt: timestamppb.New(time.Now()),
	}

	for _, ch := range channels {
		select {
		case ch <- update:
		default:
			// Channel full, skip
		}
	}
}