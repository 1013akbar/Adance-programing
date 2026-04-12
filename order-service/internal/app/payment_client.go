package app

import (
	"context"
	"fmt"

	"github.com/taubakabylnurlybek/ap2-generated/payment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

type GRPCPaymentClient struct {
	client payment.PaymentServiceClient
}

func NewGRPCPaymentClient(conn *grpc.ClientConn) *GRPCPaymentClient {
	return &GRPCPaymentClient{client: payment.NewPaymentServiceClient(conn)}
}

func (c *GRPCPaymentClient) AuthorizePayment(ctx context.Context, orderID string, amount int64) (string, string, error) {
	req := &payment.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	}

	resp, err := c.client.ProcessPayment(ctx, req)
	if err != nil {
		st, ok := grpcstatus.FromError(err)
		if ok {
			switch st.Code() {
			case codes.InvalidArgument:
				return "", "", fmt.Errorf("invalid payment request: %s", st.Message())
			case codes.FailedPrecondition:
				return "", "", fmt.Errorf("payment failed: %s", st.Message())
			case codes.Unavailable:
				return "", "", err
			default:
				return "", "", err
			}
		}
		return "", "", err
	}

	return resp.Status, resp.TransactionId, nil
}

func (c *GRPCPaymentClient) GetPaymentStatus(ctx context.Context, orderID string) (statusStr string, found bool, err error) {
	req := &payment.PaymentStatusRequest{
		OrderId: orderID,
	}

	resp, err := c.client.GetPaymentStatus(ctx, req)
	if err != nil {
		st, ok := grpcstatus.FromError(err)
		if ok && st.Code() == codes.NotFound {
			return "", false, nil
		}
		return "", false, err
	}

	return resp.Status, resp.Found, nil
}
