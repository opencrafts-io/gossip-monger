package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opencrafts-io/gossip-monger/internal/repository"
	"github.com/opencrafts-io/gossip-monger/internal/resilience"
	"github.com/resend/resend-go/v3"
)

type EmailService interface {
	Send(ctx context.Context, emailEvent EmailEvent) error
}

type emailService struct {
	pool                 *pgxpool.Pool
	logger               *slog.Logger
	emailClient          *resend.Client
	allowedSenderDomains []string
	breaker              resilience.Breaker[*resend.SendEmailResponse]
}

func NewEmailService(
	pool *pgxpool.Pool,
	emailClient *resend.Client,
	allowedSenderDomains []string,
	breakerSettings resilience.Settings,
	logger *slog.Logger,
) EmailService {
	return &emailService{
		pool:                 pool,
		emailClient:          emailClient,
		allowedSenderDomains: allowedSenderDomains,
		breaker:              resilience.New[*resend.SendEmailResponse]("resend", breakerSettings, logger),
		logger:               logger,
	}
}

func (es *emailService) Send(ctx context.Context, emailEvent EmailEvent) error {
	commited := false
	conn, err := es.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if !commited {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				es.logger.Error(
					"failed to rollback transaction",
					"error",
					rollbackErr,
				)
			}
		}
	}()

	repo := repository.New(tx)

	// A retry (DLX redelivery) and an external duplicate republish of the
	// same request_id are indistinguishable at this point — both must be
	// safe. If this request_id already reached Resend successfully, skip
	// resending: the upsert below would otherwise happily retry a send
	// that already went through.
	if existing, err := repo.GetEmailRequestByQueueMessageID(
		ctx,
		emailEvent.Meta.RequestID,
	); err == nil {
		if existing.Status == "dispatched" {
			es.logger.Info("duplicate request_id already dispatched, skipping resend",
				"request_id", emailEvent.Meta.RequestID,
			)
			return nil
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check for duplicate email request: %w", err)
	}

	if _, err := repo.UpsertService(ctx, repository.UpsertServiceParams{
		ID:   emailEvent.Meta.SourceServiceID,
		Name: emailEvent.Meta.SourceServiceID,
	}); err != nil {
		return fmt.Errorf("failed to upsert service: %w", err)
	}

	now := time.Now()
	emailReq, err := repo.UpsertEmailRequest(
		ctx,
		repository.UpsertEmailRequestParams{
			ServiceID:      emailEvent.Meta.SourceServiceID,
			QueueMessageID: emailEvent.Meta.RequestID,
			Exchange:       "gossip.topic.exchange",
			RoutingKey:     "gossip.emails.send",
			FromAddress:    emailEvent.Email.FromAddress,
			ReplyTo:        emailEvent.Email.ReplyTo,
			ToAddresses:    emailEvent.Email.ToAddresses,
			CcAddresses:    emailEvent.Email.CcAddresses,
			BccAddresses:   emailEvent.Email.BccAddresses,
			Subject:        emailEvent.Email.Subject,
			BodyHtml:       emailEvent.Email.BodyHtml,
			BodyText:       emailEvent.Email.BodyText,
			Attachments:    emailEvent.Email.Attachments,
			TemplateID:     emailEvent.Email.TemplateID,
			TemplateVars:   emailEvent.Email.TemplateVars,
			ProcessedAt:    &now,
			Status:         "received",
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create email request: %w", err)
	}

	resendRequest, err := es.emailToResendEmailRequest(emailEvent.Email)
	if err != nil {
		return fmt.Errorf("failed to convert email to resend request: %w", err)
	}

	sent, resendErr := es.breaker.Execute(func() (*resend.SendEmailResponse, error) {
		return es.emailClient.Emails.Send(resendRequest)
	})

	resendPayload, err := json.Marshal(resendRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal resend payload: %w", err)
	}

	// Determine dispatch status based on Resend API response. circuit_open
	// is distinct from failed so operators can tell "the provider was
	// already known-down, we didn't even try" apart from "we tried and
	// Resend rejected this payload."
	dispatchStatus := "failed"
	if resendErr == nil {
		dispatchStatus = "sent"
	} else if resilience.Open(resendErr) {
		dispatchStatus = "circuit_open"
	}

	dispatchParams := repository.CreateEmailDispatchParams{
		EmailRequestID: emailReq.ID,
		Status:         dispatchStatus,
		ResendPayload:  resendPayload,
	}

	if resendErr != nil {
		errString := resendErr.Error()
		dispatchParams.ResendError = &errString
		var statusCode int32 = 400
		dispatchParams.HttpStatusCode = &statusCode
		es.logger.Error("resend delivery failed",
			"email_request_id", emailReq.ID,
			"status", dispatchStatus,
			"error", resendErr,
		)
	} else {
		var statusCode int32 = 200
		dispatchParams.ResendEmailID = &sent.Id
		dispatchParams.HttpStatusCode = &statusCode
		es.logger.Info("resend delivery ok!",
			"email_request_id", emailReq.ID,
		)

	}

	_, err = repo.CreateEmailDispatch(ctx, dispatchParams)
	if err != nil {
		return fmt.Errorf("failed to persist email dispatch record: %w", err)
	}

	finalStatus := "dispatched"
	if resendErr != nil {
		finalStatus = dispatchStatus // "failed" or "circuit_open"
	}
	_, err = repo.UpdateEmailRequestStatusByID(
		ctx,
		repository.UpdateEmailRequestStatusByIDParams{
			ID:     emailReq.ID,
			Status: finalStatus,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to update email request status: %w", err)
	}

	// Commit the transaction so the attempt is recorded regardless of
	// outcome, then propagate resendErr so the consumer nacks the message
	// for a retry instead of acking a failed send as if it succeeded.
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	commited = true

	if resendErr != nil {
		return fmt.Errorf("failed to send email via resend: %w", resendErr)
	}

	return nil
}

func (es *emailService) emailToResendEmailRequest(
	email Email,
) (*resend.SendEmailRequest, error) {
	if strings.TrimSpace(email.FromAddress) == "" {
		return nil, fmt.Errorf("from address is required")
	}

	allowed := false
	for _, domain := range es.allowedSenderDomains {
		if strings.HasSuffix(email.FromAddress, domain) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf(
			"from address must end with one of: %s",
			strings.Join(es.allowedSenderDomains, ", "),
		)
	}

	if len(email.ToAddresses) == 0 && len(email.CcAddresses) == 0 &&
		len(email.BccAddresses) == 0 {
		return nil, fmt.Errorf("at least one recipient is required")
	}

	if strings.TrimSpace(email.Subject) == "" {
		return nil, fmt.Errorf("subject is required")
	}

	// Validate body - either template OR body content must be provided
	hasTemplate := email.TemplateID != nil &&
		strings.TrimSpace(*email.TemplateID) != ""
	hasBodyContent := email.BodyHtml != nil || email.BodyText != nil

	if !hasTemplate && !hasBodyContent {
		return nil, fmt.Errorf(
			"either template_id or body content (body_html/body_text) is required",
		)
	}

	if hasTemplate && hasBodyContent {
		return nil, fmt.Errorf(
			"cannot use both template and body content; provide only one",
		)
	}

	// Parse attachments
	var attachments []*resend.Attachment
	if len(email.Attachments) > 0 {
		var parsedAttachments []resend.Attachment
		if err := json.Unmarshal(email.Attachments, &parsedAttachments); err != nil {
			return nil, fmt.Errorf("invalid attachments format: %w", err)
		}

		attachments = make([]*resend.Attachment, len(parsedAttachments))
		for i := range parsedAttachments {
			attachments[i] = &parsedAttachments[i]
		}
	}

	// Build the base request
	request := &resend.SendEmailRequest{
		From:        email.FromAddress,
		To:          email.ToAddresses,
		Subject:     email.Subject,
		Bcc:         email.BccAddresses,
		Cc:          email.CcAddresses,
		ReplyTo:     derefString(email.ReplyTo),
		Attachments: attachments,
	}

	// Set body content OR template
	if hasTemplate {
		// Parse template variables if provided
		var templateVars map[string]any
		if len(email.TemplateVars) > 0 {
			if err := json.Unmarshal(email.TemplateVars, &templateVars); err != nil {
				return nil, fmt.Errorf("invalid template_vars format: %w", err)
			}
		}

		request.Template = &resend.EmailTemplate{
			Id:        *email.TemplateID,
			Variables: templateVars,
		}
	} else {
		request.Html = derefString(email.BodyHtml)
		request.Text = derefString(email.BodyText)
	}

	return request, nil
}
