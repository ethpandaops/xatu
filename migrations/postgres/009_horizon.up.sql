CREATE TABLE horizon_location (
  location_id SERIAL PRIMARY KEY,
  create_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  update_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  network_id VARCHAR(256),
  type VARCHAR(256),
  head_slot BIGINT NOT NULL DEFAULT 0,
  fill_slot BIGINT NOT NULL DEFAULT 0,
  CONSTRAINT horizon_location_unique UNIQUE (network_id, type)
);
