-- name: CreateNotification :one
INSERT INTO notifications (
    app_id,
    included_segments,
    excluded_segments,
    include_player_ids,
    include_external_user_ids,
    include_email_tokens,
    include_phone_numbers,
    include_ios_tokens,
    include_wp_wns_uris,
    include_amazon_reg_ids,
    include_chrome_reg_ids,
    include_chrome_web_reg_ids,
    include_android_reg_ids,
    contents,
    headings,
    subtitle,
    big_picture,
    large_icon,
    small_icon,
    ios_attachments,
    android_channel_id,
    android_accent_color,
    android_led_color,
    android_group,
    android_group_message,
    android_sound,
    ios_sound,
    wp_wns_sound,
    adm_sound,
    chrome_web_image,
    chrome_web_icon,
    chrome_web_badge,
    chrome_web_color,
    chrome_web_sound,
    url,
    web_url,
    app_url,
    data,
    filters,
    tags,
    send_after,
    delayed_option,
    delivery_time_of_day,
    ttl,
    priority,
    target_user_id,
    source_service_id,
    source_user_id,
    notification_type,
    onesignal_notification_id,
    onesignal_status,
    onesignal_response,
    onesignal_error
 
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
    $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33, $34, $35, $36, $37, $38, $39,
    $40, $41, $42, $43, $44, $45, $46, $47, $48, $49, $50, $51, $52, $53 
)
RETURNING *;

-- name: GetNotificationByID :one
SELECT * FROM notifications 
WHERE id = $1;

-- name: GetNotificationByOneSignalID :one
SELECT * FROM notifications 
WHERE onesignal_notification_id = $1;

-- name: GetNotificationsByTargetUser :many
SELECT * FROM notifications 
WHERE target_user_id = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: GetNotificationsByType :many
SELECT * FROM notifications 
WHERE notification_type = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: GetNotificationsByStatus :many
SELECT * FROM notifications 
WHERE status = $1
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: GetPendingNotifications :many
SELECT * FROM notifications 
WHERE status = 'pending'
  AND (send_after IS NULL OR send_after <= NOW())
ORDER BY created_at ASC
LIMIT $1;

-- name: GetNotificationsByExternalUserID :many
SELECT * FROM notifications 
WHERE $1 = ANY(include_external_user_ids)
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: UpdateNotificationStatus :exec
UPDATE notifications
SET
    status = $2,
    onesignal_notification_id = COALESCE($3, onesignal_notification_id),
    onesignal_status = COALESCE($4, onesignal_status),
    onesignal_response = COALESCE($5, onesignal_response),
    onesignal_error = COALESCE($6, onesignal_error),
    sent_at = CASE WHEN $2 = 'sent' THEN NOW() ELSE sent_at END,
    delivered_at = CASE WHEN $2 = 'delivered' THEN NOW() ELSE delivered_at END,
    failed_at = CASE WHEN $2 = 'failed' THEN NOW() ELSE failed_at END,
    updated_at = NOW()
WHERE id = $1;

-- name: MarkNotificationAsRead :exec
UPDATE notifications
SET
    read_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: UpdateNotificationOneSignalData :exec
UPDATE notifications
SET
    onesignal_notification_id = $2,
    onesignal_status = $3,
    onesignal_response = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteNotification :exec
DELETE FROM notifications
WHERE id = $1;

-- name: GetNotificationStats :one
SELECT 
    COUNT(*) as total_notifications,
    COUNT(*) FILTER (WHERE status = 'pending') as pending_notifications,
    COUNT(*) FILTER (WHERE status = 'sent') as sent_notifications,
    COUNT(*) FILTER (WHERE status = 'delivered') as delivered_notifications,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_notifications,
    COUNT(*) FILTER (WHERE read_at IS NOT NULL) as read_notifications,
    COUNT(*) FILTER (WHERE created_at > NOW() - INTERVAL '24 hours') as notifications_last_24h
FROM notifications
WHERE target_user_id = $1;

-- name: GetNotificationStatsByType :one
SELECT 
    COUNT(*) as total_notifications,
    COUNT(*) FILTER (WHERE status = 'pending') as pending_notifications,
    COUNT(*) FILTER (WHERE status = 'sent') as sent_notifications,
    COUNT(*) FILTER (WHERE status = 'delivered') as delivered_notifications,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_notifications
FROM notifications
WHERE notification_type = $1;

-- name: CleanupOldNotifications :exec
DELETE FROM notifications
WHERE created_at < NOW() - INTERVAL '90 days'
  AND status IN ('delivered', 'failed');

