package notification

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// EmailSender interface for the Adapter Pattern
type EmailSender interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}

// SimulatedEmailSender implements EmailSender with simulated behavior
type SimulatedEmailSender struct {
	failureRate float64 // 0.0 to 1.0
	baseDelay   time.Duration
}

func NewSimulatedEmailSender(failureRate float64, baseDelay time.Duration) *SimulatedEmailSender {
	return &SimulatedEmailSender{
		failureRate: failureRate,
		baseDelay:   baseDelay,
	}
}

func (s *SimulatedEmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	// Simulate network delay
	time.Sleep(s.baseDelay + time.Duration(rand.Intn(500))*time.Millisecond)

	// Simulate random failures
	if rand.Float64() < s.failureRate {
		return fmt.Errorf("simulated email service failure")
	}

	fmt.Printf("[SIMULATED] Email sent to %s: %s\n", to, subject)
	return nil
}

// SMTPEmailSender implements EmailSender using SMTP
type SMTPEmailSender struct {
	smtpHost string
	smtpPort string
	username string
	password string
	from     string
}

func NewSMTPEmailSender(smtpHost, smtpPort, username, password, from string) *SMTPEmailSender {
	return &SMTPEmailSender{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPEmailSender) SendEmail(ctx context.Context, to, subject, body string) error {
	// In a real implementation, this would use net/smtp or a library like go-mail
	// For now, we'll just simulate it
	fmt.Printf("[SMTP] Would send email to %s via %s:%s\n", to, s.smtpHost, s.smtpPort)
	return nil
}
