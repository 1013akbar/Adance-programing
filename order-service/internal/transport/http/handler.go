package http

import (
	"errors"
	"net/http"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	orderUC *usecase.OrderService
}

func NewHandler(orderUC *usecase.OrderService) *Handler {
	return &Handler{orderUC: orderUC}
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name" binding:"required"`
	Amount     int64  `json:"amount" binding:"required"`
}

func (h *Handler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")
	order, alreadyProcessed, err := h.orderUC.CreateOrder(
		c.Request.Context(),
		req.CustomerID,
		req.ItemName,
		req.Amount,
		idempotencyKey,
	)
	if errors.Is(err, usecase.ErrInvalidAmount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
		return
	}

	statusCode := http.StatusCreated
	if alreadyProcessed {
		statusCode = http.StatusOK
	}
	c.JSON(statusCode, order)
}

func (h *Handler) GetOrder(c *gin.Context) {
	order, err := h.orderUC.GetOrder(c.Request.Context(), c.Param("id"))
	if errors.Is(err, usecase.ErrOrderNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get order"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *Handler) CancelOrder(c *gin.Context) {
	order, err := h.orderUC.CancelOrder(c.Request.Context(), c.Param("id"))
	if errors.Is(err, usecase.ErrOrderNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	if errors.Is(err, usecase.ErrCannotCancel) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel order"})
		return
	}

	c.JSON(http.StatusOK, order)
}
