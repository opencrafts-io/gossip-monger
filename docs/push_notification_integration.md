# Sending Push Notifications via gossip-monger

This document explains how to send push notifications through the `gossip-monger` service. Your service publishes a message to a RabbitMQ exchange, and `gossip-monger` picks it up and relays it to OneSignal.

---

## Overview

```
Your Service → RabbitMQ Exchange → gossip-monger → OneSignal → User Devices
```

You do not call any HTTP API directly. You publish a JSON message to RabbitMQ and gossip-monger handles the rest.

---

## Connection

You will be provided with RabbitMQ credentials by your platform team. Once connected, publish to:

| Property       | Value                        |
|----------------|------------------------------|
| Exchange       | `gossip.topic.exchange`      |
| Exchange Type  | `topic`                      |
| Routing Key    | `gossip.push.send`           |

---

## Message Structure

Every message must be valid JSON with two top-level fields: `metadata` and `notification`.

```json
{
  "metadata": {
    "event_type": "push.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.your-service-name",
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
  },
  "notification": {
    ...
  }
}
```

### `metadata` fields

| Field             | Type   | Required | Description                                                                 |
|-------------------|--------|----------|-----------------------------------------------------------------------------|
| `event_type`      | string | Yes      | Must be `"push.send"` for sending a notification                            |
| `timestamp`       | string | Yes      | ISO 8601 timestamp of when your service produced the event                  |
| `source_service_id` | string | Yes    | Your service's identifier. **Must start with `io.opencrafts.`**             |
| `request_id`      | string | Yes      | A unique ID for this request, used for tracing and logging                  |

### `notification` fields

#### Required

| Field      | Type            | Description                                                    |
|------------|-----------------|----------------------------------------------------------------|
| `headings` | object          | Notification title by language. **Must include `"en"`**        |
| `contents` | object          | Notification body by language. **Must include `"en"`**         |
| `target_user_id` | string (UUID) | The primary recipient's user ID. Always required.          |

At least one targeting field must also be present (see [Targeting](#targeting) below).

#### Optional content fields

| Field      | Type   | Description                                   |
|------------|--------|-----------------------------------------------|
| `subtitle` | object | Secondary line below the title (iOS, some Android). By language. |
| `big_picture` | string | URL of a large image shown in the notification |
| `large_icon`  | string | URL of the notification icon (Android)        |
| `small_icon`  | string | Small status bar icon resource name (Android) |
| `data`        | object | Custom key/value payload delivered silently alongside the notification |
| `buttons`     | array  | Action buttons (see [Buttons](#buttons))      |
| `url`         | string | URL to open when the notification is tapped   |
| `web_url`     | string | URL for web push notifications                |
| `app_url`     | string | Deep link URL for in-app navigation           |

#### Scheduling fields

| Field            | Type    | Description                                                          |
|------------------|---------|----------------------------------------------------------------------|
| `send_after`     | string  | ISO 8601 timestamp. Must be in the future. Schedules the notification. |
| `delayed_option` | string  | OneSignal delay strategy (e.g. `"last-active"`, `"timezone"`)        |
| `ttl`            | integer | Seconds before the notification expires. Must be between 1 and 2,592,000 (30 days). |
| `priority`       | integer | Delivery priority passed to OneSignal                               |

---

## Targeting

You must provide at least one of the following targeting fields, in addition to `target_user_id`.

| Field                     | Type     | Description                                      |
|---------------------------|----------|--------------------------------------------------|
| `included_segments`       | string[] | OneSignal segment names to target                |
| `excluded_segments`       | string[] | OneSignal segment names to exclude               |
| `include_external_user_ids` | string[] | OneSignal external user IDs                    |
| `include_email_tokens`    | string[] | Email addresses registered in OneSignal          |
| `include_phone_numbers`   | string[] | Phone numbers for SMS push                       |
| `include_ios_tokens`      | string[] | iOS device tokens                                |
| `include_android_reg_ids` | string[] | Android FCM registration IDs                    |
| `include_chrome_web_reg_ids` | string[] | Chrome Web Push registration IDs             |
| and others...             |          | See OneSignal docs for the full list             |

> **External user ID limit:** `target_user_id` is always added to the external user ID list internally. The combined total of `target_user_id` + `include_external_user_ids` must not exceed **2,000**.

---

## Buttons

`buttons` is a JSON array of action button objects. Each button appears below the notification text.

```json
"buttons": [
  { "id": "btn_accept", "text": "Accept", "icon": "" },
  { "id": "btn_decline", "text": "Decline", "icon": "" }
]
```

| Field  | Type   | Required | Description                         |
|--------|--------|----------|-------------------------------------|
| `id`   | string | Yes      | Unique identifier for the button    |
| `text` | string | Yes      | Label shown to the user             |
| `icon` | string | No       | Icon URL or resource name           |

---

## Examples

### Minimal valid message

The simplest possible notification — a heading, body, and a target user.

```json
{
  "metadata": {
    "event_type": "push.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.payments",
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
  },
  "notification": {
    "headings": { "en": "Payment received" },
    "contents": { "en": "Your payment of $50 has been processed." },
    "target_user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "included_segments": ["Active Users"]
  }
}
```

### Scheduled notification with custom data and buttons

```json
{
  "metadata": {
    "event_type": "push.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.orders",
    "request_id": "a3f1c2d4-b5e6-7890-abcd-ef1234567890"
  },
  "notification": {
    "headings": { "en": "Your order is ready" },
    "subtitle": { "en": "Tap to confirm pickup" },
    "contents": { "en": "Order #1042 is ready for collection at Store 5." },
    "target_user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "include_external_user_ids": ["user-9981"],
    "data": {
      "order_id": "1042",
      "store_id": "5"
    },
    "buttons": [
      { "id": "btn_confirm", "text": "Confirm Pickup" },
      { "id": "btn_later",   "text": "Remind me later" }
    ],
    "send_after": "2024-11-01T14:00:00Z",
    "ttl": 3600
  }
}
```

---

## Invalid message examples

### Wrong `source_service_id` prefix

```json
{
  "metadata": {
    "event_type": "push.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "payments-service",
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
  },
  "notification": { "..." : "..." }
}
```

**What went wrong:** `source_service_id` must start with `io.opencrafts.`. The value `"payments-service"` does not, so the message will be rejected before the notification is even attempted.

---

### Missing `"en"` in headings

```json
{
  "metadata": { "..." : "..." },
  "notification": {
    "headings": { "fr": "Paiement reçu" },
    "contents": { "en": "Your payment has been processed." },
    "target_user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "included_segments": ["Active Users"]
  }
}
```

**What went wrong:** The English heading `"en"` is mandatory. Other languages may be included alongside it, but `"en"` must always be present.

---

### No targeting specified

```json
{
  "metadata": { "..." : "..." },
  "notification": {
    "headings": { "en": "Hello" },
    "contents": { "en": "This is a test." },
    "target_user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  }
}
```

**What went wrong:** `target_user_id` alone is not sufficient targeting. You must also provide at least one of `included_segments`, `include_external_user_ids`, `include_email_tokens`, etc.

---

### `send_after` in the past

```json
{
  "metadata": { "..." : "..." },
  "notification": {
    "headings": { "en": "Flash sale!" },
    "contents": { "en": "50% off for the next hour." },
    "target_user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "included_segments": ["All Users"],
    "send_after": "2023-01-01T00:00:00Z"
  }
}
```

**What went wrong:** `send_after` must be a future date. A past timestamp will cause the message to be rejected.

---

### `ttl` out of range

```json
{
  "metadata": { "..." : "..." },
  "notification": {
    "headings": { "en": "Reminder" },
    "contents": { "en": "Don't forget your appointment." },
    "target_user_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "included_segments": ["Active Users"],
    "ttl": 0
  }
}
```

**What went wrong:** `ttl` must be a positive integer between 1 and 2,592,000 (30 days). Zero and negative values are rejected.

---

### Wrong `event_type`

```json
{
  "metadata": {
    "event_type": "notification.send",
    "timestamp": "2024-11-01T10:00:00Z",
    "source_service_id": "io.opencrafts.payments",
    "request_id": "550e8400-e29b-41d4-a716-446655440000"
  },
  "notification": { "..." : "..." }
}
```

**What went wrong:** The only currently supported `event_type` is `"push.send"`. Any other value will be logged as an error and the message will not be processed.

---

## Supported event types

| `event_type` | Routing Key        | Status              |
|--------------|--------------------|---------------------|
| `push.send`  | `gossip.push.send` | Supported           |

---

## Notes

- The `app_id` field on the notification object is ignored. The service uses its own configured OneSignal app ID.
- Successful and failed notifications are both persisted, including the raw OneSignal response and any error details.
- `request_id` is used for tracing only — include a unique value per message to make debugging easier.
