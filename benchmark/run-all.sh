#!/usr/bin/env bash
# Run HTTP and gRPC benchmarks sequentially with k6 and print a side-by-side summary.
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
export VUS="${VUS:-50}"
export DURATION="${DURATION:-30s}"

# Ensure duration has a unit suffix (accept both "30" and "30s")
[[ "$DURATION" =~ ^[0-9]+$ ]] && DURATION="${DURATION}s"

HTTP_RESULTS="$SCRIPT_DIR/k6/http-results.json"
GRPC_RESULTS="$SCRIPT_DIR/k6/grpc-results.json"

echo "========================================"
echo " Benchmark: HTTP vs gRPC  (k6)"
echo " email=$BENCH_EMAIL  vus=$VUS  duration=$DURATION"
echo "========================================"
echo ""

echo ">>> Running HTTP benchmark..."
k6 run --summary-export="$HTTP_RESULTS" "$SCRIPT_DIR/k6/http.js"
echo ""

echo ">>> Running gRPC benchmark..."
k6 run --summary-export="$GRPC_RESULTS" "$SCRIPT_DIR/k6/grpc.js"
echo ""

if ! command -v jq &>/dev/null; then
  echo "Install jq for a side-by-side summary: https://jqlang.github.io/jq/"
  echo "Results: $HTTP_RESULTS  $GRPC_RESULTS"
  exit 0
fi

parse() {
  local file="$1" rps_key="$2" dur_key="$3"
  RPS=$(jq ".metrics.iterations.values.rate | floor"              "$file" 2>/dev/null || echo "n/a")
  P50=$(jq ".metrics.${dur_key}.values.med | round | tostring"   "$file" 2>/dev/null || echo "n/a")
  P99=$(jq ".metrics.${dur_key}.values[\"p(99)\"] | round | tostring" "$file" 2>/dev/null || echo "n/a")
}

parse "$HTTP_RESULTS" "" "http_req_duration"
HTTP_RPS=$RPS; HTTP_P50="${P50}ms"; HTTP_P99="${P99}ms"

parse "$GRPC_RESULTS" "" "grpc_req_duration"
GRPC_RPS=$RPS; GRPC_P50="${P50}ms"; GRPC_P99="${P99}ms"

echo "========================================"
echo " Summary (same tool, same parameters)"
echo "========================================"
printf "%-20s %-15s %-15s\n" "Metric"       "HTTP"        "gRPC"
printf "%-20s %-15s %-15s\n" "------"       "----"        "----"
printf "%-20s %-15s %-15s\n" "RPS"          "$HTTP_RPS"   "$GRPC_RPS"
printf "%-20s %-15s %-15s\n" "Latency p50"  "$HTTP_P50"   "$GRPC_P50"
printf "%-20s %-15s %-15s\n" "Latency p99"  "$HTTP_P99"   "$GRPC_P99"
echo "========================================"
