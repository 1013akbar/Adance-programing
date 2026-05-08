package http

import (
	"errors"
	"net/http"
	"payment-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	paymentUC *usecase.PaymentService
}

func NewHandler(paymentUC *usecase.PaymentService) *Handler {
	return &Handler{paymentUC: paymentUC}
}

type createPaymentRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Amount  int64  `json:"amount" binding:"required"`
}

func (h *Handler) CreatePayment(c *gin.Context) {
	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := h.paymentUC.ProcessPayment(c.Request.Context(), req.OrderID, req.Amount)
	if errors.Is(err, usecase.ErrInvalidAmount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, usecase.ErrOrderNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, usecase.ErrAmountMismatch) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if errors.Is(err, usecase.ErrAlreadyPaid) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":             payment.ID,
		"order_id":       payment.OrderID,
		"transaction_id": payment.TransactionID,
		"amount":         payment.Amount,
		"status":         payment.Status,
	})
}

func (h *Handler) GetPaymentByOrderID(c *gin.Context) {
	orderID := c.Param("order_id")
	payment, err := h.paymentUC.GetByOrderID(c.Request.Context(), orderID)
	if errors.Is(err, usecase.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":             payment.ID,
		"order_id":       payment.OrderID,
		"transaction_id": payment.TransactionID,
		"amount":         payment.Amount,
		"status":         payment.Status,
	})
}
