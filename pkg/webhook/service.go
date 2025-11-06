package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/database"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
)

// Service handles webhook notifications
type Service struct {
	db     *database.DB
	client *http.Client
}

// NewService creates a new webhook service
func NewService(db *database.DB) *Service {
	return &Service{
		db: db,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// EventPayload represents the payload sent to webhook endpoints
type EventPayload struct {
	Event     string      `json:"event"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// TriggerEvent sends webhook notifications for a machine event
func (s *Service) TriggerEvent(eventType string, data interface{}) error {
	webhooks, err := s.db.GetWebhooksByEvent(eventType)
	if err != nil {
		log.Printf("Failed to get webhooks for event %s: %v", eventType, err)
		return err
	}

	if len(webhooks) == 0 {
		return nil // No webhooks configured for this event
	}

	payload := EventPayload{
		Event:     eventType,
		Timestamp: time.Now(),
		Data:      data,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Send webhooks asynchronously
	for _, webhook := range webhooks {
		go s.sendWebhook(webhook, payloadJSON)
	}

	return nil
}

func (s *Service) sendWebhook(webhook *models.Webhook, payload []byte) {
	delivery := &models.WebhookDelivery{
		WebhookID: webhook.ID,
		Event:     webhook.Events[0], // First event
		Payload:   string(payload),
		Attempts:  0,
		Success:   false,
	}

	maxRetries := webhook.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}

	timeout := time.Duration(webhook.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := &http.Client{
		Timeout: timeout,
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		delivery.Attempts = attempt

		req, err := http.NewRequest("POST", webhook.URL, bytes.NewReader(payload))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Metal-Enrollment-Webhook/1.0")

		// Add custom headers
		if webhook.Headers != nil {
			var headers map[string]string
			if err := json.Unmarshal(webhook.Headers, &headers); err == nil {
				for key, value := range headers {
					req.Header.Set(key, value)
				}
			}
		}

		// Add HMAC signature if secret is configured
		if webhook.Secret != "" {
			signature := s.generateSignature(payload, webhook.Secret)
			req.Header.Set("X-Webhook-Signature", signature)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("Webhook delivery attempt %d/%d failed for %s: %v", attempt, maxRetries, webhook.Name, err)

			// Exponential backoff
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
			}
			continue
		}

		// Read response
		responseBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		delivery.StatusCode = resp.StatusCode
		delivery.Response = string(responseBody)

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			delivery.Success = true
			now := time.Now()
			delivery.CompletedAt = &now

			// Update webhook last success
			s.db.UpdateWebhookDeliveryStatus(webhook.ID, true)

			log.Printf("Webhook delivered successfully to %s (attempt %d/%d)", webhook.Name, attempt, maxRetries)
			break
		} else {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(responseBody))
			log.Printf("Webhook delivery attempt %d/%d returned HTTP %d for %s", attempt, maxRetries, resp.StatusCode, webhook.Name)

			// Exponential backoff
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
			}
		}
	}

	if !delivery.Success {
		delivery.Error = lastErr.Error()
		now := time.Now()
		delivery.CompletedAt = &now

		// Update webhook last failure
		s.db.UpdateWebhookDeliveryStatus(webhook.ID, false)

		log.Printf("Webhook delivery failed after %d attempts to %s: %v", delivery.Attempts, webhook.Name, lastErr)
	}

	// Store delivery record
	if err := s.db.CreateWebhookDelivery(delivery); err != nil {
		log.Printf("Failed to store webhook delivery record: %v", err)
	}
}

func (s *Service) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
