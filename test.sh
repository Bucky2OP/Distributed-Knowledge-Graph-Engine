#!/bin/bash

echo "🧪 Testing Knowledge Graph Engine..."
echo ""

BASE_URL="http://localhost:8080"

echo "1. Health Check..."
curl -s $BASE_URL/health | jq
echo ""

echo "2. Adding nodes..."
curl -s -X POST $BASE_URL/node -H "Content-Type: application/json" \
  -d '{"id":"alice","props":{"name":"Alice","role":"developer"}}' | jq

curl -s -X POST $BASE_URL/node -H "Content-Type: application/json" \
  -d '{"id":"bob","props":{"name":"Bob","role":"manager"}}' | jq

curl -s -X POST $BASE_URL/node -H "Content-Type: application/json" \
  -d '{"id":"charlie","props":{"name":"Charlie","role":"designer"}}' | jq
echo ""

echo "3. Adding edges..."
curl -s -X POST $BASE_URL/edge -H "Content-Type: application/json" \
  -d '{"From":"alice","To":"bob","Label":"reports_to"}' | jq

curl -s -X POST $BASE_URL/edge -H "Content-Type: application/json" \
  -d '{"From":"charlie","To":"bob","Label":"reports_to"}' | jq

curl -s -X POST $BASE_URL/edge -H "Content-Type: application/json" \
  -d '{"From":"alice","To":"charlie","Label":"collaborates_with"}' | jq
echo ""

echo "4. Exporting graph..."
curl -s $BASE_URL/export | jq
echo ""

echo "✅ Test complete!"