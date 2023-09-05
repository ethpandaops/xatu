CREATE TABLE cannon_location (
  location_id SERIAL PRIMARY KEY,
  create_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  update_time TIMESTAMPTZ NOT NULL DEFAULT now(),
  network_id VARCHAR(256),
  type VARCHAR(256),
  value TEXT,
  CONSTRAINT cannon_location_unique UNIQUE (network_id, type)
);
