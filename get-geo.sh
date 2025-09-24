#!/bin/bash

set -e

echo "==============================================="
echo "Xatu GeoIP Data Setup Script"
echo "==============================================="
echo ""
echo "This script will download MaxMind GeoLite2 and GeoNames data files."
echo ""

# Check for required tools
command -v wget >/dev/null 2>&1 || { echo "Error: wget is required but not installed. Please install wget." >&2; exit 1; }
command -v unzip >/dev/null 2>&1 || { echo "Error: unzip is required but not installed. Please install unzip." >&2; exit 1; }
command -v tar >/dev/null 2>&1 || { echo "Error: tar is required but not installed. Please install tar." >&2; exit 1; }

# MaxMind License Key
echo "MaxMind GeoLite2 License Key Required"
echo "--------------------------------------"
echo "To download MaxMind GeoLite2 databases, you need a free license key."
echo ""
echo "To get a license key:"
echo "1. Sign up for a free account at: https://www.maxmind.com/en/geolite2/signup"
echo "2. After logging in, go to: https://www.maxmind.com/en/account/licenses"
echo "3. Generate a new license key"
echo ""
read -p "Enter your MaxMind License Key: " LICENSE_KEY

if [ -z "$LICENSE_KEY" ]; then
    echo "Error: License key is required"
    exit 1
fi

echo ""
echo "Starting downloads..."
echo ""

# Create temporary directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Download MaxMind databases
echo "Downloading MaxMind GeoLite2 City database..."
wget -q --show-progress -O "$TEMP_DIR/city.tar.gz" "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=${LICENSE_KEY}&suffix=tar.gz"

echo "Downloading MaxMind GeoLite2 ASN database..."
wget -q --show-progress -O "$TEMP_DIR/asn.tar.gz" "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-ASN&license_key=${LICENSE_KEY}&suffix=tar.gz"

# Extract MaxMind databases
echo "Extracting MaxMind databases..."
cd "$TEMP_DIR"
tar -xzf city.tar.gz
tar -xzf asn.tar.gz

# Find and move MaxMind databases to current directory
CITY_DB=$(find . -name "GeoLite2-City.mmdb" -type f | head -1)
ASN_DB=$(find . -name "GeoLite2-ASN.mmdb" -type f | head -1)

if [ -z "$CITY_DB" ] || [ -z "$ASN_DB" ]; then
    echo "Error: Failed to extract MaxMind databases. Check your license key."
    exit 1
fi

# Go back to original directory
cd - > /dev/null

echo "Moving MaxMind databases to current directory..."
cp "$TEMP_DIR/$CITY_DB" ./GeoLite2-City.mmdb
cp "$TEMP_DIR/$ASN_DB" ./GeoLite2-ASN.mmdb

# Download GeoNames data
echo ""
echo "Downloading GeoNames cities data (cities with population > 1000)..."
wget -q --show-progress -O "$TEMP_DIR/cities1000.zip" "http://download.geonames.org/export/dump/cities1000.zip"

echo "Extracting cities data..."
unzip -q -o "$TEMP_DIR/cities1000.zip" -d "$TEMP_DIR"
cp "$TEMP_DIR/cities1000.txt" ./cities1000.txt

echo ""
echo "Downloading GeoNames all countries data (this is large, ~400MB)..."
wget -q --show-progress -O "$TEMP_DIR/allCountries.zip" "http://download.geonames.org/export/dump/allCountries.zip"

echo "Extracting and processing country centroids..."
unzip -q -o "$TEMP_DIR/allCountries.zip" -d "$TEMP_DIR"

# Extract country centroids (feature code PCLI = country)
# Format: geoname_id<tab>latitude<tab>longitude
echo "Creating countries.txt with country centroids..."
grep "PCLI" "$TEMP_DIR/allCountries.txt" | awk -F'\t' '{print $1 "\t" $5 "\t" $6}' > ./countries.txt

# Display summary
echo ""
echo "==============================================="
echo "Setup Complete!"
echo "==============================================="
echo ""
echo "Downloaded files:"
echo "  ✓ GeoLite2-City.mmdb    - MaxMind city database"
echo "  ✓ GeoLite2-ASN.mmdb     - MaxMind ASN database"
echo "  ✓ cities1000.txt        - GeoNames city centroids"
echo "  ✓ countries.txt         - GeoNames country centroids"
echo ""
echo "These files have been added to .gitignore"
echo ""
echo "You can now use these files in your Xatu server configuration:"
echo ""
echo "geoip:"
echo "  enabled: true"
echo "  type: maxmind"
echo "  config:"
echo "    database:"
echo "      city: ./GeoLite2-City.mmdb"
echo "      asn: ./GeoLite2-ASN.mmdb"
echo "    geonames:"
echo "      cities: ./cities1000.txt"
echo "      countries: ./countries.txt"
echo ""

# Check file sizes to verify downloads
echo "File sizes:"
ls -lh GeoLite2-City.mmdb GeoLite2-ASN.mmdb cities1000.txt countries.txt 2>/dev/null | awk '{print "  " $9 ": " $5}'