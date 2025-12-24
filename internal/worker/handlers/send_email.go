package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/amitbasuri/taskqueue-runner-go/internal/models"
)

// SendEmailHandler handles email sending tasks
type SendEmailHandler struct {
	rng *rand.Rand
}

// NewSendEmailHandler creates a new email handler
func NewSendEmailHandler() *SendEmailHandler {
	return &SendEmailHandler{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (h *SendEmailHandler) Type() models.TaskType {
	return models.TaskTypeSendEmail
}

func (h *SendEmailHandler) Execute(ctx context.Context, payload json.RawMessage) error {
	var req struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	if err := json.Unmarshal(payload, &req); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	// Validate required fields
	if req.To == "" {
		return fmt.Errorf("missing required field: to")
	}
	if req.Subject == "" {
		return fmt.Errorf("missing required field: subject")
	}

	// Check for cancellation before starting work
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// TODO: Integrate with actual email service (SendGrid, AWS SES, etc.)
	slog.Info("Sending email",
		"to", req.To,
		"subject", req.Subject,
		"body_length", len(req.Body),
	)

	// Simulate 25% failure rate
	if h.rng.Intn(4) == 0 {
		slog.Warn("Email sending failed (simulated)", "to", req.To)
		return fmt.Errorf("email delivery failed: SMTP connection timeout")
	}

	// Simulate email sending with cancellation support
	select {
	case <-time.After(3 * time.Second):
		slog.Info("Email sent successfully", "to", req.To)
		return nil
	case <-ctx.Done():
		slog.Warn("Email sending cancelled", "to", req.To, "error", ctx.Err())
		return ctx.Err()
	}
}
