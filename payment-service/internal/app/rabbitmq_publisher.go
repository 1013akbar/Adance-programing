package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/rabbitmq/amqp091-go"
)

type PaymentEvent struct {
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"`
	CustomerEmail string `json:"customer_email"`
	Status        string `json:"status"`
}

type RabbitMQPublisher struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	queue   amqp091.Queue
}

func NewRabbitMQPublisher(url, queueName string) (*RabbitMQPublisher, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Declare durable queue
	q, err := ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &RabbitMQPublisher{
		conn:    conn,
		channel: ch,
		queue:   q,
	}, nil
}

func (p *RabbitMQPublisher) PublishPaymentEvent(ctx context.Context, orderID string, amount int64, status string) error {
	// Check if channel is available
	if p == nil || p.channel == nil {
		return fmt.Errorf("rabbitmq channel is not available")
	}

	event := PaymentEvent{
		OrderID:       orderID,
		Amount:        amount,
		CustomerEmail: fmt.Sprintf("%s@example.com", orderID), // Placeholder; replace with real customer email from order service
		Status:        status,
	}

	body, err := json.Marshal(event)
	if err != nil {
		return err
	}

	err = p.channel.PublishWithContext(ctx,
		"",           // exchange
		p.queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent, // Make message persistent
		})
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		return err
	}

	log.Printf("Published payment event for order %s", orderID)
	return nil
}

func (p *RabbitMQPublisher) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
