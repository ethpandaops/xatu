CREATE TABLE relay_monitor_location (
  location_id SERIAL PRIMARY KEY,
  create_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  update_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  meta_network_name VARCHAR(256),
  meta_client_name VARCHAR(256),
  type VARCHAR(256),
  relay_name VARCHAR(256),
  value TEXT,
  CONSTRAINT relay_monitor_location_unique UNIQUE (meta_network_name, meta_client_name, type, relay_name)
);