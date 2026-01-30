
#!/bin/bash
# Health check script for Moon API

echo "Checking Moon health endpoint..."
curl -X "GET" "http://localhost:6006/health"
echo ""
echo ""
echo "Expected response format:"
echo '{"status":"live","name":"moon","version":"1.99"}'
echo ""
echo "Note: status can be 'live' or 'down'. HTTP status is always 200."
