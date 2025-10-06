
# Gossip Monger: Notification Service

**Gossip Monger** is a reliable, standalone notification proxy service for the Academia platform. It acts as a central hub for sending information to users across various communication channels, primarily:

* **Push Notifications**
* **Emails**

The service acts as an intermediary (a proxy) between other internal Academia services and external providers such as **Onesignal** (for push notifications) and **Resend** (for emails).

---

## How Gossip Monger Works

Gossip Monger is a **consumer-only** service. It does not expose a traditional API (HTTP, gRPC, etc.); it **only** communicates via a secure, single-entry event bus provided by **RabbitMQ**.

This approach ensures a decoupled and secure mode of communication, where requesting services publish events to a known exchange, and Gossip Monger processes them asynchronously.

### RabbitMQ Integration Details

To send a notification request, an external service must publish a message to the specified RabbitMQ exchange and routing key.

| Component       | Type      | Name                                                 | Description                                                                                                              |
| :-------------- | :-------- | :--------------------------------------------------- | :----------------------------------------------------------------------------------------------------------------------- |
| **Exchange**    | `direct`  | `gossip-monger.exchange`                             | The main exchange that requesting services must bind to and publish messages on.                                         |
| **Queue**       | `classic` | `io.opencrafts.gossip-monger.notification.requested` | The queue that Gossip Monger listens to for incoming requests.                                                           |
| **Routing Key** | `string`  | `notification.requested`                             | The routing key that must be used when publishing a message to `gossip-monger.exchange` to be consumed by Gossip Monger. |

**Gossip Monger** binds the queue `io.opencrafts.gossip-monger.notification.requested` to the exchange `gossip-monger.exchange` using the routing key `notification.requested` and continuously acts as a consumer.

---

## Notification Request Payload

All messages sent to the `gossip-monger.exchange` with the routing key `notification.requested` must be a valid JSON object conforming to the following structure.

### Sample Payload

```json
{
    "notification": {
        "app_id": "88ca0bb7-c0d7-4e36-b9e6-ea0e29213593",
        "headings": {
            "en": "Ping!"
        },
        "contents": {
            "en": "Notification Content here"
        },	
        "target_user_id": "f71a-2b57-4678-8776-9708c92d8dd1", // Primary user to recieve the notification
        "include_external_user_ids": [
            "3a3-bb78-4a76-918e-875778053c70",
            "664-a407-4271-87f2-eb6efbfeb1ea"
        ], // Other users to recieve the notification
        "subtitle": {
            "en": "This is a notification subtitle"
        },
        "android_channel_id": "60023d0b-dcd4-41ae-8e58-7eabbf382c8c" , // refer to @erick for other possible valies
        "ios_sound": "pay",
        "big_picture": "https://images.com/image.png",
        "large_icon": "https://images.com/image.png",
        "small_icon": "https://images.com/image.png",
        "url": "https://opencrafts.io", // The url to launch
        "buttons": [
            {
                "id": "id-01", // Random id
                "text": "Pay Now",
                "icon":"ic" // Refere to mobile team for possible values
            },
            {
                "id": "id-02",
                "text": "Pay Later",
                "icon":""
            }
        ]
    },
    "meta": {
        "event_type": "notification.requested",
        "source_service_id": "io.opencrafts.verisafe",
        "request_id": "00000000-0000-0000-0000-000000000000"
    }
}

```

## Payload Specification

The top-level object must contain two mandatory keys: `notification` and `meta`.

### 1. notification Object (Required)

This object contains all the details required to construct and send the user notification (primarily for Push Notifications via Onesignal).


> Check the comments on the json payload for more info where unclear

### 2. `meta` Object (Required)

This object provides context and tracing information for the request.

| Key                 | Type   | Required | Description                                                                                                          |
| ------------------- | ------ | -------- | -------------------------------------------------------------------------------------------------------------------- |
| `event_type`        | string | `true`   | Must be set to notification.requested. Used by Gossip Monger to validate the action.                                 |
| `source_service_id` | string | true     | A unique identifier for the service making the request (e.g., io.opencrafts.verisafe).                               |
| request_id          | uuid   | true     | A unique UUID generated by the source service. This will be used by Gossip Monger for internal tracking and logging. |


## To Wrap things up
1.  **Gossip Monger's Setup (Internal):**
    * Gossip Monger creates the exchange `gossip-monger.exchange` (type `direct`).
    * Gossip Monger creates the queue `io.opencrafts.gossip-monger.notification.requested`.
    * Gossip Monger **binds** its queue to the exchange using the routing key `notification.requested`.

2.  **Requesting Service's Action (External):**
    * The requesting service connects to the RabbitMQ broker.
    * The service **publishes** a JSON message (the payload) to the exchange named `gossip-monger.exchange`.
    * Critically, the message **must** be published with the routing key set to `notification.requested`.

### 3. Publishing to Gossip Monger

The requesting service does **not** need to declare the queue or the binding; it only needs to know the Exchange name and the Routing Key.

A successful publication action in code would typically look like this:

```pseudocode
// 1. Define the targets
EXCHANGE_NAME = "gossip-monger.exchange"
ROUTING_KEY   = "notification.requested"

// 2. Prepare the payload (example JSON structure)
payload = { ... } // (Your full notification request JSON)

// 3. Publish the message
rabbitmq_channel.publish(
    exchange=EXCHANGE_NAME,
    routing_key=ROUTING_KEY,
    body=JSON.stringify(payload),
    // Optional: Set delivery mode to persistent for reliability
    properties={delivery_mode: 2} 
)
