#!/usr/bin/env bash
# Creates a confirmed subscription for benchmarking.
# Run once before http/run.sh and grpc/run.sh.

HOST="${HOST:-localhost}"
HTTP_PORT="${HTTP_PORT:-8080}"
API_KEY="${API_KEY:-secret}"
EMAIL="${BENCH_EMAIL:-bench@example.com}"
REPO="${BENCH_REPO:-golang/go}"

echo "Seeding subscription: email=$EMAIL repo=$REPO"

curl -sf -X POST "http://${HOST}:${HTTP_PORT}/api/v1/subscribe" \
  -H "X-Api-Key: ${API_KEY}" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"${EMAIL}\", \"repository\": \"${REPO}\"}" \
  && echo "" \
  && echo "Done. Confirm the subscription via the link sent to $EMAIL, then run the benchmark scripts."
