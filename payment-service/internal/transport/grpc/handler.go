package grpc

import (
	"context"
	"errors"
	"payment-service/internal/usecase"

	"github.com/taubakabylnurlybek/ap2-generated/payment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentServiceServer struct {
	payment.UnimplementedPaymentServiceServer
	paymentUC *usecase.PaymentService
}

func NewPaymentServiceServer(paymentUC *usecase.PaymentService) *PaymentServiceServer {
	return &PaymentServiceServer{paymentUC: paymentUC}
}

func (s *PaymentServiceServer) ProcessPayment(ctx context.Context, req *payment.PaymentRequest) (*payment.PaymentResponse, error) {
	p, err := s.paymentUC.ProcessPayment(ctx, req.OrderId, req.Amount)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidAmount) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if errors.Is(err, usecase.ErrOrderNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if errors.Is(err, usecase.ErrAmountMismatch) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		if errors.Is(err, usecase.ErrAlreadyPaid) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, "failed to process payment")
	}

	return &payment.PaymentResponse{
		Id:            p.ID,
		OrderId:       p.OrderID,
		TransactionId: p.TransactionID,
		Amount:        p.Amount,
		Status:        p.Status,
		CreatedAt:     timestamppb.Now(),
	}, nil
}

func (s *PaymentServiceServer) GetPaymentStatus(ctx context.Context, req *payment.PaymentStatusRequest) (*payment.PaymentStatusResponse, error) {
	p, err := s.paymentUC.GetByOrderID(ctx, req.OrderId)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			return &payment.PaymentStatusResponse{Status: "", Found: false}, nil
		}
		return nil, status.Error(codes.Internal, "failed to get payment status")
	}

	return &payment.PaymentStatusResponse{Status: p.Status, Found: true}, nil
}