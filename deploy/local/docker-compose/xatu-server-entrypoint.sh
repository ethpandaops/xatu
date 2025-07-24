#!/bin/sh

# Copy the template config
cp /etc/xatu-server/config-template.yaml /etc/xatu-server/config.yaml

# Check if GeoIP databases exist and are not empty
if [ -s /geoip/GeoLite2-City.mmdb ] && [ -s /geoip/GeoLite2-ASN.mmdb ]; then
    echo "GeoIP databases found. Enabling GeoIP..."
    
    # Create a temporary file with the geoip configuration
    cat > /tmp/geoip-config.yaml << 'EOF'
geoip:
  enabled: true
  type: maxmind
  config:
    database:
      city: /geoip/GeoLite2-City.mmdb
      asn: /geoip/GeoLite2-ASN.mmdb
EOF
    
    # Replace the geoip section in the config
    awk '
    /^geoip:/ {
        print "geoip:"
        print "  enabled: true"
        print "  type: maxmind"
        print "  config:"
        print "    database:"
        print "      city: /geoip/GeoLite2-City.mmdb"
        print "      asn: /geoip/GeoLite2-ASN.mmdb"
        # Skip the next line (enabled: false)
        getline
        next
    }
    { print }
    ' /etc/xatu-server/config-template.yaml > /etc/xatu-server/config.yaml
else
    echo "GeoIP databases not found. GeoIP will be disabled."
fi

# Start the server
exec /xatu server --config /etc/xatu-server/config.yaml