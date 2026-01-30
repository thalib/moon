
#!/bin/bash
# Health check script for Moon API
# Supports PREFIX environment variable for custom URL prefixes
# Usage: PREFIX=/api/v1 ./scripts/health.sh

# Default to empty prefix if not set
PREFIX=${PREFIX:-}

echo "Checking Moon health endpoint..."
echo "Using prefix: ${PREFIX:-<empty>}"
curl -X "GET" "http://localhost:6006${PREFIX}/health"
echo ""
echo ""
echo "Expected response format:"
echo '{"status":"live","name":"moon","version":"1.99"}'
echo ""
echo "Note: status can be 'live' or 'down'. HTTP status is always 200."

