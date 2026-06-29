#!/usr/bin/env bash
# Run HTTP, gRPC, and inter-service gRPC benchmarks sequentially with k6 and print a summary.
# Uses the same tool and the same parameters for a fair comparison.
#
# Requirements: k6 >= 0.45.0  https://k6.io/docs/getting-started/installation/
#               jq             https://jqlang.github.io/jq/
#
# Usage (from project root):
#   bash benchmark/run-all.sh
#
# Reads API_KEY from deployments/.env automatically.
# Override any variable: VUS=100 DURATION=60s bash benchmark/run-all.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="$(dirname "$SCRIPT_DIR")/deployments/.env"

if [[ -z "$API_KEY" && -f "$ENV_FILE" ]]; then
  API_KEY=$(grep '^APP_API_KEY=' "$ENV_FILE" | cut -d'=' -f2 | tr -d '\r')
fi

export HOST="${HOST:-localhost}"
export HTTP_PORT="${HTTP_PORT:-8080}"
export GRPC_PORT="${GRPC_PORT:-9091}"
export API_KEY="${API_KEY:-secret}"
export BENCH_EMAIL="${BENCH_EMAIL:-bench@example.com}"
export BENCH_REPO="${BENCH_REPO:-golang/go}"
export BENCH_TAG="${BENCH_TAG:-v1.0.0}"
export VUS="${VUS:-50}"
export DURATION="${DURATION:-30s}"

# Ensure duration has a unit suffix (accept both "30" and "30s")
[[ "$DURATION" =~ ^[0-9]+$ ]] && DURATION="${DURATION}s"

HTTP_RESULTS="$SCRIPT_DIR/k6/http-results.json"
GRPC_RESULTS="$SCRIPT_DIR/k6/grpc-results.json"
UPDATE_TAG_RESULTS="$SCRIPT_DIR/k6/grpc-update-tag-results.json"

echo "========================================"
echo " Benchmark suite (k6)"
echo " vus=$VUS  duration=$DURATION"
echo "========================================"
echo ""

echo ">>> [1/3] Running HTTP benchmark (ListSubscriptions)..."
k6 run --summary-export="$HTTP_RESULTS" "$SCRIPT_DIR/k6/http.js"
echo ""

echo ">>> [2/3] Running gRPC benchmark (ListSubscriptions)..."
k6 run --summary-export="$GRPC_RESULTS" "$SCRIPT_DIR/k6/grpc.js"
echo ""

echo ">>> [3/3] Running gRPC benchmark (UpdateTag — tracking→subscription path)..."
k6 run --summary-export="$UPDATE_TAG_RESULTS" "$SCRIPT_DIR/k6/grpc-update-tag.js"
echo ""

if ! command -v jq &>/dev/null; then
  echo "Install jq for a side-by-side summary: https://jqlang.github.io/jq/"
  echo "Results: $HTTP_RESULTS  $GRPC_RESULTS  $UPDATE_TAG_RESULTS"
  exit 0
fi

parse_http() {
  local file="$1"
  RPS=$(jq ".metrics.iterations.rate | floor"                        "$file" 2>/dev/null || echo "n/a")
  P50=$(jq ".metrics.http_req_duration.med | round | tostring"      "$file" 2>/dev/null || echo "n/a")
  P95=$(jq ".metrics.http_req_duration[\"p(95)\"] | round | tostring" "$file" 2>/dev/null || echo "n/a")
}

parse_grpc() {
  local file="$1"
  RPS=$(jq ".metrics.iterations.rate | floor"                        "$file" 2>/dev/null || echo "n/a")
  P50=$(jq ".metrics.grpc_req_duration.med | round | tostring"      "$file" 2>/dev/null || echo "n/a")
  P95=$(jq ".metrics.grpc_req_duration[\"p(95)\"] | round | tostring" "$file" 2>/dev/null || echo "n/a")
}

parse_http "$HTTP_RESULTS"
HTTP_RPS=$RPS; HTTP_P50="${P50}ms"; HTTP_P95="${P95}ms"

parse_grpc "$GRPC_RESULTS"
GRPC_RPS=$RPS; GRPC_P50="${P50}ms"; GRPC_P95="${P95}ms"

parse_grpc "$UPDATE_TAG_RESULTS"
UPD_RPS=$RPS; UPD_P50="${P50}ms"; UPD_P95="${P95}ms"

echo "========================================"
echo " Summary"
echo " HTTP/gRPC ListSubscriptions | gRPC UpdateTag (tracking→subscription)"
echo "========================================"
printf "%-20s %-15s %-15s %-15s\n" "Metric"       "HTTP List"   "gRPC List"   "gRPC UpdateTag"
printf "%-20s %-15s %-15s %-15s\n" "------"       "---------"   "----------"  "---------------"
printf "%-20s %-15s %-15s %-15s\n" "RPS"          "$HTTP_RPS"   "$GRPC_RPS"   "$UPD_RPS"
printf "%-20s %-15s %-15s %-15s\n" "Latency p50"  "$HTTP_P50"   "$GRPC_P50"   "$UPD_P50"
printf "%-20s %-15s %-15s %-15s\n" "Latency p95"  "$HTTP_P95"   "$GRPC_P95"   "$UPD_P95"
echo "========================================"
