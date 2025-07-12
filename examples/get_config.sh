#!/bin/bash

# Simple script to read configuration values
# Usage: 
#   source ./get_config.sh
#   echo $BASE_URL

# Default config file location
CONFIG_FILE="../config.yml"

# Function to read config value using yq (if available) or grep
get_config_value() {
    local key="$1"
    local default="$2"
    
    if [ ! -f "$CONFIG_FILE" ]; then
        echo "$default"
        return
    fi
    
    # Try yq first (YAML processor)
    if command -v yq > /dev/null 2>&1; then
        local value=$(yq eval "$key" "$CONFIG_FILE" 2>/dev/null)
        if [ "$value" != "null" ] && [ -n "$value" ]; then
            echo "$value"
            return
        fi
    fi
    
    # Fallback to grep/awk for simple values
    case "$key" in
        ".server.host")
            grep -A1 "^server:" "$CONFIG_FILE" | grep "host:" | awk '{print $2}' | tr -d '"' | head -1 || echo "$default"
            ;;
        ".server.port")
            grep -A2 "^server:" "$CONFIG_FILE" | grep "port:" | awk '{print $2}' | head -1 || echo "$default"
            ;;
        *)
            echo "$default"
            ;;
    esac
}

# Read server configuration
HOST=$(get_config_value ".server.host" "localhost")
PORT=$(get_config_value ".server.port" "8080")

# Convert 0.0.0.0 to localhost for client connections
if [ "$HOST" = "0.0.0.0" ]; then
    HOST="localhost"
fi

# Export the base URL
export BASE_URL="http://${HOST}:${PORT}/api/v1"
export SERVER_URL="http://${HOST}:${PORT}"

# If run directly, just output the base URL
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    echo "$BASE_URL"
fi