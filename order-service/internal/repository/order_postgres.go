package repository

import (
	"context"
	"database/sql"
	"errors"
	"order-service/internal/domain"
	"order-service/internal/usecase"
)

type OrderPostgresRepository struct {
	db *sql.DB
}

func NewOrderPostgresRepository(db *sql.DB) *OrderPostgresRepository {
	return &OrderPostgresRepository{db: db}
}

func (r *OrderPostgresRepository) Create(ctx context.Context, order *domain.Order) error {
	query := `
		INSERT INTO orders (id, customer_id, item_name, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query, order.ID, order.CustomerID, order.ItemName, order.Amount, order.Status, order.CreatedAt)
	return err
}

func (r *OrderPostgresRepository) GetByID(ctx context.Context, orderID string) (*domain.Order, error) {
	query := `
		SELECT id, customer_id, item_name, amount, status, created_at
		FROM orders
		WHERE id = $1
	`

	order := &domain.Order{}
	err := r.db.QueryRowContext(ctx, query, orderID).Scan(
		&order.ID,
		&order.CustomerID,
		&order.ItemName,
		&order.Amount,
		&order.Status,
		&order.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, usecase.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}

	return order, nil
}

func (r *OrderPostgresRepository) UpdateStatus(ctx context.Context, orderID, status string) error {
	query := `UPDATE orders SET status = $1 WHERE id = $2`
	result, err := r.db.ExecContext(ctx, query, status, orderID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return usecase.ErrOrderNotFound
	}

	return nil
}

func (r *OrderPostgresRepository) SaveIdempotencyKey(ctx context.Context, key, orderID string) error {
	query := `
		INSERT INTO order_idempotency (idempotency_key, order_id)
		VALUES ($1, $2)
	`
	_, err := r.db.ExecContext(ctx, query, key, orderID)
	return err
}

func (r *OrderPostgresRepository) GetOrderIDByIdempotencyKey(ctx context.Context, key string) (string, error) {
	query := `
		SELECT order_id
		FROM order_idempotency
		WHERE idempotency_key = $1
	`
	var orderID string
	err := r.db.QueryRowContext(ctx, query, key).Scan(&orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", usecase.ErrOrderNotFound
	}
	if err != nil {
		return "", err
	}
	return orderID, nil
}
