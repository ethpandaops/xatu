ALTER TABLE node_record
ADD COLUMN geo_city VARCHAR(128),
ADD COLUMN geo_country VARCHAR(128),
ADD COLUMN geo_country_code VARCHAR(128),
ADD COLUMN geo_continent_code VARCHAR(128),
ADD COLUMN geo_longitude FLOAT,
ADD COLUMN geo_latitude FLOAT,
ADD COLUMN geo_autonomous_system_number INT,
ADD COLUMN geo_autonomous_system_organization VARCHAR(128);
