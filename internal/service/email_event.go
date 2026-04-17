package service

import (
	"encoding/json"
	"time"
)

type Email struct {
	FromAddress  string          `json:"from_address"`
	ReplyTo      *string         `json:"reply_to"`
	ToAddresses  []string        `json:"to_addresses"`
	CcAddresses  []string        `json:"cc_addresses"`
	BccAddresses []string        `json:"bcc_addresses"`
	Subject      string          `json:"subject"`
	BodyHtml     *string         `json:"body_html"`
	BodyText     *string         `json:"body_text"`
	Attachments  json.RawMessage `json:"attachments"`
	TemplateID   *string         `json:"template_id"`
	TemplateVars json.RawMessage `json:"template_vars"`
	Status       string          `json:"status"`
	ReceivedAt   time.Time       `json:"received_at"`
	ProcessedAt  *time.Time      `json:"processed_at"`
}

type EmailEventMetadata struct {
	EventType       string    `json:"event_type"`
	Timestamp       time.Time `json:"timestamp"`
	SourceServiceID string    `json:"source_service_id"`
	RequestID       string    `json:"request_id"`
}

type EmailEvent struct {
	Email Email
	Meta  EmailEventMetadata `json:"metadata"`
}
