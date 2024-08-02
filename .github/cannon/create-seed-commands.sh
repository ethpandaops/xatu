#!/bin/bash

# Read the seeding.yaml file
yaml_content=$(cat ./seeding.yaml)

while IFS= read -r line; do
    if [[ $line =~ ^[[:space:]]*-[[:space:]]*id:[[:space:]]*([0-9]+) ]]; then
        network_id="${BASH_REMATCH[1]}"
    elif [[ $line =~ ^[[:space:]]*-[[:space:]]*name:[[:space:]]*(.+) ]]; then
        type="${BASH_REMATCH[1]}"
    elif [[ $line =~ ^[[:space:]]*finalizedEpoch:[[:space:]]*([0-9]+) ]]; then
        epoch="${BASH_REMATCH[1]}"
        value="{\"backfillingCheckpointMarker\":{\"finalizedEpoch\":\"$epoch\"}}"
        echo "INSERT INTO cannon_location (network_id, type, value)"
        echo "VALUES ('$network_id', '$type', '$value')"
        echo "ON CONFLICT (network_id, type) DO UPDATE"
        echo "SET value = EXCLUDED.value, update_time = now();"
        echo
    fi
done <<< "$yaml_content"
