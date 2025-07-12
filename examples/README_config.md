# Test Scripts Configuration

The test scripts now automatically read the server configuration from `../config.yml`.

## Configuration File

The config file should be located at `/agent-server/config.yml`:

```yaml
server:
  host: "0.0.0.0"    # Server host (0.0.0.0 for all interfaces)
  port: 8081         # Server port

database:
  type: "sqlite"
  path: "./data/agents.db"

logging:
  level: "info"
  format: "json"

llm:
  providers:
    ollama:
      base_url: "http://localhost:11434"
```

## How Test Scripts Use Configuration

All test scripts now source `get_config.sh` which:

1. Reads the `config.yml` file from the parent directory
2. Extracts the server host and port
3. Sets environment variables:
   - `BASE_URL` - Full API base URL (e.g., `http://localhost:8081/api/v1`)
   - `SERVER_URL` - Server base URL (e.g., `http://localhost:8081`)

## Usage in Scripts

Instead of hardcoding URLs:

```bash
# Old way
BASE_URL="http://localhost:8081/api/v1"

# New way
source "$(dirname "$0")/get_config.sh"
# Now $BASE_URL and $SERVER_URL are available
```

## Environment Variable Override

You can also override the configuration using environment variables:

```bash
SERVER_HOST=localhost SERVER_PORT=8080 ./quick_test.sh
```

## Fallback Behavior

If no config file is found, the scripts use these defaults:
- Host: `localhost` (converted from `0.0.0.0` for client connections)
- Port: `8080` (matching the application defaults)