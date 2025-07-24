#!/bin/sh

# Check if GeoIP databases are valid non-empty files (not directories or empty files)
if [ -f /geoip/GeoLite2-City.mmdb ] && [ -s /geoip/GeoLite2-City.mmdb ] && [ -f /geoip/GeoLite2-ASN.mmdb ] && [ -s /geoip/GeoLite2-ASN.mmdb ]; then
    echo "GeoIP databases found. Enabling GeoIP..."
    
    # Replace the geoip section in the config to enable it
    awk '
    /^geoip:/ {
        print "geoip:"
        print "  enabled: true"
        print "  type: maxmind"
        print "  config:"
        print "    database:"
        print "      city: /geoip/GeoLite2-City.mmdb"
        print "      asn: /geoip/GeoLite2-ASN.mmdb"
        # Skip lines until we find the next top-level section
        while (getline && $0 !~ /^[a-zA-Z]/) {
            # Skip nested geoip config lines
        }
        # Print the line we just read (start of next section)
        if (NF > 0) print
        next
    }
    { print }
    ' /etc/xatu-server/config-template.yaml > /etc/xatu-server/config.yaml
else
    # Check what we actually have and log it for debugging
    if [ -d /geoip/GeoLite2-City.mmdb ] || [ -d /geoip/GeoLite2-ASN.mmdb ]; then
        echo "GeoIP paths are directories (Docker created them). GeoIP will be disabled."
    elif [ -f /geoip/GeoLite2-City.mmdb ] || [ -f /geoip/GeoLite2-ASN.mmdb ]; then
        echo "GeoIP files exist but are empty. GeoIP will be disabled."
    else
        echo "GeoIP files not found. GeoIP will be disabled."
    fi
    # Just copy the template as-is (geoip.enabled: false)
    cp /etc/xatu-server/config-template.yaml /etc/xatu-server/config.yaml
fi

# Start the server
exec /xatu server --config /etc/xatu-server/config.yaml