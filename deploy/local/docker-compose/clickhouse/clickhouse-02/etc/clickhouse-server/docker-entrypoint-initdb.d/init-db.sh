#!/bin/bash
set -e

# Only create config if files don't already exist (avoid permission issues on restart)
if [ ! -f /var/lib/clickhouse/init_completed ]; then
    echo "Running first-time initialization..."
    
    # Create a marker file to indicate initialization has been completed
    touch /var/lib/clickhouse/init_completed || true
    
    echo "Initialization completed."
else
    echo "Initialization already completed, skipping..."
fi