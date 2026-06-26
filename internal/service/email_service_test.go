package service

import (
	"encoding/json"
	"testing"

	"github.com/resend/resend-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailToResendEmailRequest(t *testing.T) {
	tests := []struct {
		name        string
		email       Email
		expectError bool
		errorMsg    string
		validate    func(t *testing.T, req *resend.SendEmailRequest)
	}{
		{
			name: "valid email with html body",
			email: Email{
				FromAddress: "sender@posta.opencrafts.io",
				ToAddresses: []string{"recipient@example.com"},
				Subject:     "Test Subject",
				BodyHtml:    stringPtr("<h1>Hello</h1>"),
			},
			expectError: false,
			validate: func(t *testing.T, req *resend.SendEmailRequest) {
				assert.Equal(t, "sender@posta.opencrafts.io", req.From)
				assert.Equal(t, []string{"recipient@example.com"}, req.To)
				assert.Equal(t, "Test Subject", req.Subject)
				assert.Equal(t, "<h1>Hello</h1>", req.Html)
				assert.Equal(t, "", req.Text)
				assert.Nil(t, req.Template)
			},
		},
		{
			name: "valid email with text body",
			email: Email{
				FromAddress: "sender@posta.opencrafts.io",
				ToAddresses: []string{"recipient@example.com"},
				Subject:     "Test Subject",
				BodyText:    stringPtr("Hello World"),
			},
			expectError: false,
			validate: func(t *testing.T, req *resend.SendEmailRequest) {
				assert.Equal(t, "Hello World", req.Text)
				assert.Equal(t, "", req.Html)
			},
		},
		{
			name: "valid email with both html and text",
			email: Email{
				FromAddress: "sender@posta.opencrafts.io",
				ToAddresses: []string{"recipient@example.com"},
				Subject:     "Test Subject",
				BodyHtml:    stringPtr("<h1>Hello</h1>"),
				BodyText:    stringPtr("Hello"),
			},
			expectError: false,
			validate: func(t *testing.T, req *resend.SendEmailRequest) {
				assert.Equal(t, "<h1>Hello</h1>", req.Html)
				assert.Equal(t, "Hello", req.Text)
			},
		},
		{
			name: "valid email with template",
			email: Email{
				FromAddress:  "sender@posta.opencrafts.io",
				ToAddresses:  []string{"recipient@example.com"},
				Subject:      "Test Subject",
				TemplateID:   stringPtr("template_123"),
				TemplateVars: json.RawMessage(`{"name":"John","age":30}`),
			},
			expectError: false,
			validate: func(t *testing.T, req *resend.SendEmailRequest) {
				assert.NotNil(t, req.Template)
				assert.Equal(t, "template_123", req.Template.Id)
				assert.Equal(t, "John", req.Template.Variables["name"])
				assert.Equal(t, float64(30), req.Template.Variables["age"])
			},
		},
		{
			name: "email with cc and bcc",
			email: Email{
				FromAddress:  "sender@posta.opencrafts.io",
				ToAddresses:  []string{"recipient@example.com"},
				CcAddresses:  []string{"cc@example.com"},
				BccAddresses: []string{"bcc@example.com"},
				Subject:      "Test Subject",
				BodyHtml:     stringPtr("<h1>Hello</h1>"),
			},
			expectError: false,
			validate: func(t *testing.T, req *resend.SendEmailRequest) {
				assert.Equal(t, []string{"cc@example.com"}, req.Cc)
				assert.Equal(t, []string{"bcc@example.com"}, req.Bcc)
			},
		},
		{
			name: "email with reply-to",
			email: Email{
				FromAddress: "sender@posta.opencrafts.io",
				ToAddresses: []string{"recipient@example.com"},
				Subject:     "Test Subject",
				BodyHtml:    stringPtr("<h1>Hello</h1>"),
				ReplyTo:     stringPtr("reply@example.com"),
			},
			expectError: false,
			validate: func(t *testing.T, req *resend.SendEmailRequest) {
				assert.Equal(t, "reply@example.com", req.ReplyTo)
			},
		},
		{
			name: "email with attachments",
			email: Email{
				FromAddress: "sender@posta.opencrafts.io",
				ToAddresses: []string{"recipient@example.com"},
				Subject:     "Test Subject",
				BodyHtml:    stringPtr("<h1>Hello</h1>"),
				Attachments: json.RawMessage(
					`[{"filename":"test.txt","content":"dGVzdA=="}]`,
				),
			},
			expectError: false,
			validate: func(t *testing.T, req *resend.SendEmailRequest) {
				assert.NotNil(t, req.Attachments)
				assert.Len(t, req.Attachments, 1)
				assert.Equal(t, "test.txt", req.Attachments[0].Filename)
			},
		},
		{
			name: "missing from address",
			email: Email{
				FromAddress: "",
				ToAddresses: []string{"recipient@example.com"},
				Subject:     "Test Subject",
				BodyHtml:    stringPtr("<h1>Hello</h1>"),
			},
			expectError: true,
			errorMsg:    "from address is required",
		},
		{
			name: "missing to addresses",
			email: Email{
				FromAddress: "sender@posta.opencrafts.io",
				ToAddresses: []string{},
				Subject:     "Test Subject",
				BodyHtml:    stringPtr("<h1>Hello</h1>"),
			},
			expectError: true,
			errorMsg:    "at least one recipient is required",
		},
		{
			name: "missing subject",
			email: Email{
				FromAddress: "sender@posta.opencrafts.io",
				ToAddresses: []string{"recipient@example.com"},
				Subject:     "",
				BodyHtml:    stringPtr("<h1>Hello</h1>"),
			},
			expectError: true,
			errorMsg:    "subject is required",
		},
		{
			name: "missing body and template",
			email: Email{
				FromAddress: "sender@posta.opencrafts.io",
				ToAddresses: []string{"recipient@example.com"},
				Subject:     "Test Subject",
			},
			expectError: true,
			errorMsg:    "either template_id or body content (body_html/body_text) is required",
		},
		{
			name: "both body and template provided",
			email: Email{
				FromAddress: "sender@posta.opencrafts.io",
				ToAddresses: []string{"recipient@example.com"},
				Subject:     "Test Subject",
				BodyHtml:    stringPtr("<h1>Hello</h1>"),
				TemplateID:  stringPtr("template_123"),
			},
			expectError: true,
			errorMsg:    "cannot use both template and body content; provide only one",
		},
		{
			name: "invalid attachments json",
			email: Email{
				FromAddress: "sender@posta.opencrafts.io",
				ToAddresses: []string{"recipient@example.com"},
				Subject:     "Test Subject",
				BodyHtml:    stringPtr("<h1>Hello</h1>"),
				Attachments: json.RawMessage(`invalid json`),
			},
			expectError: true,
			errorMsg:    "invalid attachments format",
		},
		{
			name: "invalid template vars json",
			email: Email{
				FromAddress:  "sender@posta.opencrafts.io",
				ToAddresses:  []string{"recipient@example.com"},
				Subject:      "Test Subject",
				TemplateID:   stringPtr("template_123"),
				TemplateVars: json.RawMessage(`not valid json`),
			},
			expectError: true,
			errorMsg:    "invalid template_vars format",
		},
		{
			name: "empty template id with whitespace",
			email: Email{
				FromAddress: "sender@posta.opencrafts.io",
				ToAddresses: []string{"recipient@example.com"},
				Subject:     "Test Subject",
				BodyHtml:    stringPtr("<h1>Hello</h1>"),
				TemplateID:  stringPtr("   "),
			},
			expectError: false,
			validate: func(t *testing.T, req *resend.SendEmailRequest) {
				assert.Nil(t, req.Template)
			},
		},
	}

	es := &emailService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := es.emailToResendEmailRequest(tt.email)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, req)
			} else {
				require.NoError(t, err)
				require.NotNil(t, req)
				tt.validate(t, req)
			}
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
