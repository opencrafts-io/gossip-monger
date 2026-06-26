# Gossip Monger — Email Integration Guide

This guide explains how to integrate your service with Gossip Monger to send emails via the `gossip.emails.send` routing key.

> ⚠️ **Important: Emails cost real money.** Every message you publish that reaches the Resend API incurs a cost. Before publishing, make sure your integration is not running in a loop, not sending on every request when a single send would do, and that you have tested with a controlled payload first. If you are unsure, reach out to the Gossip team before going live.

---

## Prerequisites

Before you can send emails through Gossip Monger, confirm the following with the Gossip team:

- Your **service name** (e.g. `billing`, `auth`) — this determines your `from_address`
- Your **`source_service_id`** — assigned during your service onboarding, follows the `io.opencrafts.*` namespace (e.g. `io.opencrafts.billing`)
- Your **RabbitMQ access** to publish to `gossip.topic.exchange`
- Any **email templates** your service needs — request these from the Gossip team, who will provide you with a `template_id` and the available variable keys

---

## How It Works

Your service publishes a JSON message to the RabbitMQ topic exchange. Gossip Monger picks it up, validates it, records it in its database, and dispatches it to Resend.

- **Exchange:** `gossip.topic.exchange`
- **Routing key:** `gossip.emails.send`
- **Exchange type:** Topic

If the dispatch fails (e.g. Resend returns an error), Gossip Monger will mark the request as failed and retry it automatically when the system is free. There is no guaranteed retry time. **Do not republish the same message to force a retry** — if you publish again using the same `request_id`, it will be rejected automatically due to idempotency checks.

---

## Message Structure

Every message must be a JSON object with two top-level keys: `email` and `metadata`.

```json
{
  "email": {
    "from_address": "...",
    "reply_to": "...",
    "to_addresses": ["..."],
    "cc_addresses": ["..."],
    "bcc_addresses": ["..."],
    "subject": "...",
    "body_html": "...",
    "body_text": "...",
    "attachments": [...],
    "template_id": "...",
    "template_vars": {...}
  },
  "metadata": {
    "event_type": "email.send",
    "timestamp": "...",
    "source_service_id": "...",
    "request_id": "..."
  }
}
```

> **Note:** Do not include `status`, `received_at`, or `processed_at` in your message. These are managed internally by Gossip Monger and will be ignored or overwritten.

---

## Field Reference

### `metadata`

| Field | Type | Required | Description |
|---|---|---|---|
| `event_type` | string | Yes | Must always be `"email.send"` |
| `timestamp` | string (ISO 8601) | Yes | When your service generated the event |
| `source_service_id` | string | Yes | Your service's registered ID, e.g. `io.opencrafts.billing` |
| `request_id` | string (UUID) | Yes | A unique UUID for this request. Used for idempotency — never reuse a `request_id` |

### `email`

| Field | Type | Required | Description |
|---|---|---|---|
| `from_address` | string | Yes | Must be `<your-service-name>@posta.opencrafts.io` |
| `to_addresses` | array of strings | Yes | At least one recipient required |
| `subject` | string | Yes | Email subject line |
| `reply_to` | string | No | Optional reply-to address |
| `cc_addresses` | array of strings | No | CC recipients |
| `bcc_addresses` | array of strings | No | BCC recipients |
| `body_html` | string | No* | HTML email body |
| `body_text` | string | No* | Plain text email body |
| `template_id` | string | No* | ID of a Resend template (contact Gossip team to set up) |
| `template_vars` | object | No | Variable key/value pairs for the template |
| `attachments` | array of objects | No | File attachments — see Attachments section |

> \* You must provide either `template_id` **or** at least one of `body_html`/`body_text`. You cannot provide both. If neither is provided, the message will be rejected.

---

## Sending a Plain Email

Use this when you want to send a one-off email with HTML or plain text content directly in the message body.

### Valid Example

```json
{
  "email": {
    "from_address": "billing@posta.opencrafts.io",
    "to_addresses": ["customer@example.com"],
    "subject": "Your invoice is ready",
    "body_html": "<p>Hi, your invoice for this month is ready. Please log in to view it.</p>",
    "body_text": "Hi, your invoice for this month is ready. Please log in to view it."
  },
  "metadata": {
    "event_type": "email.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.billing",
    "request_id": "a3f1c2d4-11b2-4e5a-9c1d-000000000001"
  }
}
```

Valid because:
- `from_address` ends with `@posta.opencrafts.io`
- `to_addresses` has at least one entry
- `subject` is present
- Body content is provided and no `template_id` is set
- `source_service_id` is in the `io.opencrafts.*` namespace
- `request_id` is a UUID

---

## Invalid Examples

### Missing `from_address`

```json
{
  "email": {
    "to_addresses": ["customer@example.com"],
    "subject": "Your invoice is ready",
    "body_html": "<p>Hello</p>"
  },
  "metadata": {
    "event_type": "email.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.billing",
    "request_id": "a3f1c2d4-11b2-4e5a-9c1d-000000000002"
  }
}
```

Invalid because `from_address` is missing. Gossip Monger will reject this.

---

### Wrong `from_address` domain

```json
{
  "email": {
    "from_address": "billing@opencrafts.io",
    "to_addresses": ["customer@example.com"],
    "subject": "Your invoice is ready",
    "body_html": "<p>Hello</p>"
  },
  "metadata": {
    "event_type": "email.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.billing",
    "request_id": "a3f1c2d4-11b2-4e5a-9c1d-000000000003"
  }
}
```

Invalid because `from_address` must end with `@posta.opencrafts.io`, not `@opencrafts.io`.

---

### No body content and no template

```json
{
  "email": {
    "from_address": "billing@posta.opencrafts.io",
    "to_addresses": ["customer@example.com"],
    "subject": "Your invoice is ready"
  },
  "metadata": {
    "event_type": "email.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.billing",
    "request_id": "a3f1c2d4-11b2-4e5a-9c1d-000000000004"
  }
}
```

Invalid because neither `body_html`/`body_text` nor `template_id` is provided.

---

### Both body content and template provided

```json
{
  "email": {
    "from_address": "billing@posta.opencrafts.io",
    "to_addresses": ["customer@example.com"],
    "subject": "Your invoice is ready",
    "body_html": "<p>Hello</p>",
    "template_id": "tmpl_abc123"
  },
  "metadata": {
    "event_type": "email.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.billing",
    "request_id": "a3f1c2d4-11b2-4e5a-9c1d-000000000005"
  }
}
```

Invalid because you cannot provide both `template_id` and body content. Choose one.

---

### Wrong `source_service_id` namespace

```json
{
  "email": {
    "from_address": "billing@posta.opencrafts.io",
    "to_addresses": ["customer@example.com"],
    "subject": "Your invoice is ready",
    "body_html": "<p>Hello</p>"
  },
  "metadata": {
    "event_type": "email.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "billing-service",
    "request_id": "a3f1c2d4-11b2-4e5a-9c1d-000000000006"
  }
}
```

Invalid because `source_service_id` must be in the `io.opencrafts.*` namespace. Use the ID assigned to your service during onboarding.

---

### Wrong `event_type`

```json
{
  "email": {
    "from_address": "billing@posta.opencrafts.io",
    "to_addresses": ["customer@example.com"],
    "subject": "Your invoice is ready",
    "body_html": "<p>Hello</p>"
  },
  "metadata": {
    "event_type": "send.email",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.billing",
    "request_id": "a3f1c2d4-11b2-4e5a-9c1d-000000000007"
  }
}
```

Invalid because `event_type` must be exactly `"email.send"`.

---

## Sending with a Template

Templates let you decouple your email design from your service code. The Gossip team manages templates on Resend on your behalf. To use a template, contact the Gossip team and request:

- The template to be created or assigned to your service
- The `template_id` to use in your messages
- The list of available variable keys for that template

Once you have those, include `template_id` in your message and pass your variable values in `template_vars` as a flat key/value JSON object.

### Valid Example

Assume the Gossip team gave you template ID `tmpl_invoice_ready` with variables `customer_name` and `invoice_url`:

```json
{
  "email": {
    "from_address": "billing@posta.opencrafts.io",
    "to_addresses": ["customer@example.com"],
    "subject": "Your invoice is ready",
    "template_id": "tmpl_invoice_ready",
    "template_vars": {
      "customer_name": "Jane Doe",
      "invoice_url": "https://app.opencrafts.io/invoices/INV-0042"
    }
  },
  "metadata": {
    "event_type": "email.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.billing",
    "request_id": "a3f1c2d4-11b2-4e5a-9c1d-000000000008"
  }
}
```

Valid because:
- `template_id` is provided
- No `body_html` or `body_text` is present
- `template_vars` is a valid JSON object with the keys the template expects

### Notes on `template_vars`

- The keys must match exactly what the Gossip team documented for your template — unknown keys are silently ignored by Resend, and missing keys will result in blank placeholders in the rendered email
- All values must be strings or simple scalar types (numbers, booleans). Nested objects are not supported
- If your template has no variables, you can omit `template_vars` entirely

---

## Sending with Attachments

Attachments are passed as a JSON array in the `attachments` field. Each attachment object must follow the Resend attachment format with the file content base64-encoded.

### Attachment Object Structure

| Field | Type | Required | Description |
|---|---|---|---|
| `filename` | string | Yes | The filename shown to the recipient, e.g. `invoice.pdf` |
| `content` | string | Yes | Base64-encoded file content |

### Valid Example

```json
{
  "email": {
    "from_address": "billing@posta.opencrafts.io",
    "to_addresses": ["customer@example.com"],
    "subject": "Your invoice is attached",
    "body_html": "<p>Please find your invoice attached.</p>",
    "attachments": [
      {
        "filename": "invoice-october-2024.pdf",
        "content": "JVBERi0xLjQKJcOkw7zDtsOfCjIgMCBvYmoKPDwvTGVuZ3..."
      }
    ]
  },
  "metadata": {
    "event_type": "email.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.billing",
    "request_id": "a3f1c2d4-11b2-4e5a-9c1d-000000000009"
  }
}
```

Valid because `attachments` is a valid JSON array of objects each containing `filename` and base64-encoded `content`.

### Invalid Attachment Example

```json
{
  "attachments": [
    {
      "filename": "invoice.pdf"
    }
  ]
}
```

Invalid because `content` is missing. Gossip Monger will fail to parse the attachment and reject the message.

### Notes on Attachments

- The `content` field must be a valid base64-encoded string — not a file path or a URL
- Large attachments increase the size of your RabbitMQ message. Keep attachments reasonably sized
- Attachments can be combined with either direct body content or a template

---

## Idempotency and Retries

Every message requires a unique `request_id` (UUID). Gossip Monger uses this to prevent duplicate processing.

- **If a message is processed successfully**, publishing the same `request_id` again will be rejected
- **If a message fails** (e.g. Resend returns an error), Gossip Monger will retry it automatically. Do not republish — doing so with the same `request_id` will fail, and with a new `request_id` will result in a duplicate send attempt once the original retry also goes through
- Generate a fresh UUID per send event, not per session or per user

---

## Cost Reminder

Every message dispatched through Gossip Monger that reaches Resend costs money. Please make sure you:

- Are not publishing inside a loop without a deliberate throttle or deduplication check
- Are not sending on every polling cycle or background job tick when a single trigger-based send would be appropriate
- Have tested your integration end-to-end in a non-production environment before going live
- Have spoken to the Gossip team if you expect high send volumes

If something goes wrong and you suspect emails are being sent unintentionally, contact the Gossip team immediately so the queue can be paused.
