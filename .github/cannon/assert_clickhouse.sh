#!/bin/bash

# Check if seeding.yaml location is provided as a parameter
if [ $# -eq 0 ]; then
    echo "Error: Please provide the location of seeding.yaml as an argument."
    exit 1
fi

SEEDING_YAML="$1"

# Set variables for ClickHouse connection
CLICKHOUSE_HOST=${CLICKHOUSE_HOST:-"localhost"}
CLICKHOUSE_PORT=${CLICKHOUSE_PORT:-"9000"}
CLICKHOUSE_USER=${CLICKHOUSE_USER:-"default"}
CLICKHOUSE_PASSWORD=${CLICKHOUSE_PASSWORD:-""}
CLICKHOUSE_DB=${CLICKHOUSE_DB:-"default"}

# Function to execute ClickHouse query
execute_query() {
    # Check if clickhouse-client is available
    if command -v clickhouse-client &> /dev/null; then
        clickhouse-client -h "$CLICKHOUSE_HOST" --port "$CLICKHOUSE_PORT" -u "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" -d "$CLICKHOUSE_DB" -q "$1"
    else        
        clickhouse client -h "$CLICKHOUSE_HOST" --port "$CLICKHOUSE_PORT" -u "$CLICKHOUSE_USER" --password "$CLICKHOUSE_PASSWORD" -d "$CLICKHOUSE_DB" -q "$1"
    fi
}

# Check if the seeding.yaml file exists
if [ ! -f "$SEEDING_YAML" ]; then
    echo "Error: seeding.yaml file not found at $SEEDING_YAML"
    exit 1
fi

# Parse the YAML file and execute queries
yq e '.networks[].types[] | [.name, .assert.query, .assert.expected] | @tsv' "$SEEDING_YAML" | while IFS=$'\t' read -r name query expected; do
    if [ -z "$query" ] || [ "$query" == "null" ]; then
        echo "Query is empty. Skipping..."
        continue
    fi

    while true; do
        echo "Asserting type: $name"
        echo "Executing query: $query"
        result=$(execute_query "$query")       

        if [ -n "$result" ]; then
            echo "Result: $result"
            
            # Evaluate the expected condition
            if [ -n "$expected" ] && [ "$expected" != "null" ]; then
                if [ "$result" -eq "$expected" ]; then
                    echo "Assertion passed for $name"
                    break
                else
                    echo "Assertion failed for $name. Expected: $expected, Got: $result"
                    sleep 2
                    continue
                fi
            else
                echo "No assertion condition provided for $name. Continuing..."
                break
            fi
        else
            echo "No data found for $name. Retrying in 5 seconds..."
            sleep 5
        fi
    done
done

echo "All assertions completed."