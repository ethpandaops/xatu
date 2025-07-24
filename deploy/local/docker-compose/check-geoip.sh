#!/bin/bash

# Check if GeoLite2 databases exist in the root directory
CITY_DB="./GeoLite2-City.mmdb"
ASN_DB="./GeoLite2-ASN.mmdb"

# Default config without GeoIP
CONFIG_FILE="./deploy/local/docker-compose/xatu-server-generated.yaml"

# Copy the base config
cp ./deploy/local/docker-compose/xatu-server.yaml "$CONFIG_FILE"

# Check if both databases exist
if [ -f "$CITY_DB" ] && [ -f "$ASN_DB" ]; then
    echo "GeoIP databases found. Enabling GeoIP in config..."
    
    # Update the geoip section in the config
    sed -i '/^geoip:/,/^[^ ]/ {
        /^geoip:/!b
        n
        s/enabled: false/enabled: true/
        a\  type: maxmind\
  config:\
    database:\
      city: \/geoip\/GeoLite2-City.mmdb\
      asn: \/geoip\/GeoLite2-ASN.mmdb
    }' "$CONFIG_FILE"
    
    echo "GEOIP_ENABLED=true"
else
    echo "GeoIP databases not found. GeoIP will be disabled."
    echo "GEOIP_ENABLED=false"
fi