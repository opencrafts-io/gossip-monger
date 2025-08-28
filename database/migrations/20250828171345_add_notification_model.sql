-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd


CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  
  app_id VARCHAR(255) NOT NULL,
  included_segments TEXT[], -- default
  excluded_segments TEXT[],
  include_player_ids TEXT[],
  include_external_user_ids TEXT[],
  include_email_tokens TEXT[],
  include_phone_numbers TEXT[],
  include_ios_tokens TEXT[],
  include_wp_wns_uris TEXT[],
  include_amazon_reg_ids TEXT[],
  include_chrome_reg_ids TEXT[],
  include_chrome_web_reg_ids TEXT[],
  include_android_reg_ids TEXT[],
  
  -- Content Fields
  contents JSONB, -- {"en": "Message content", "es": "Contenido del mensaje"}
  headings JSONB, -- {"en": "Title", "es": "Título"}
  subtitle JSONB, -- {"en": "Subtitle", "es": "Subtítulo"}

  -- Buttons
  buttons JSONB, -- [{"id": "id1", "text": "Button1", "icon": "icon1"}, {"id": "id2", "text": "Button2", "icon": "icon2"}]
  web_buttons JSONB, -- Web-specific buttons
  
  big_picture TEXT, -- URL to large image
  large_icon TEXT, -- URL to large icon
  small_icon TEXT, -- URL to small icon (Android)
  ios_attachments JSONB, -- {"id1": "url1", "id2": "url2"}
  android_channel_id VARCHAR(255),
  android_accent_color VARCHAR(7), -- Hex color like "#FF0000"
  android_led_color VARCHAR(7),
  android_group VARCHAR(255),
  android_group_message JSONB, -- {"en": "You have $[notif_count] new messages"}
  android_sound TEXT,
  ios_sound TEXT,
  wp_wns_sound TEXT,
  adm_sound TEXT,
  chrome_web_image TEXT,
  chrome_web_icon TEXT,
  chrome_web_badge TEXT,
  chrome_web_color VARCHAR(7),
  chrome_web_sound TEXT,
  
  -- URLs and Actions
  url TEXT, -- URL to open when notification is clicked
  web_url TEXT, -- Web-specific URL
  app_url TEXT, -- App-specific URL
  data JSONB, -- Custom data payload
  filters JSONB, -- OneSignal filters
  tags JSONB, -- User tags
  
  -- Delivery Settings
  send_after TIMESTAMP,
  delayed_option VARCHAR(50), -- 'timezone', 'last-active'
  delivery_time_of_day TIME,
  ttl INTEGER, -- Time to live in seconds
  priority INTEGER DEFAULT 10, -- Priority (10 = normal, 5 = high)
  
  -- OneSignal Response Tracking
  onesignal_notification_id VARCHAR(255),
  onesignal_status VARCHAR(50), -- 'sent', 'delivered', 'failed'
  onesignal_response JSONB, -- Full OneSignal API response
  onesignal_error TEXT, -- Error message if failed
  
  -- Internal Tracking
  target_user_id UUID,
  source_service_id VARCHAR(100),
  source_user_id UUID,
  notification_type VARCHAR(50),
  status VARCHAR(20) DEFAULT 'pending', -- 'pending', 'sent', 'delivered', 'failed'
  
  -- Timestamps
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  sent_at TIMESTAMP,
  delivered_at TIMESTAMP,
  
  FOREIGN KEY (target_user_id) REFERENCES users(id) ON DELETE SET NULL,
  FOREIGN KEY (source_user_id) REFERENCES users(id) ON DELETE SET NULL
);


-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
DROP TABLE IF EXISTS notifications;
