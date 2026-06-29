# HTTP vs gRPC Benchmark

Tested operation: `ListSubscriptions` — same business logic, different transport.

## Requirements

- k6 >= 0.45.0
- jq

## How to run

```bash
# seed test data once, confirm subscription via email link
bash benchmark/seed.sh

# run both
bash benchmark/run-all.sh

# override defaults
VUS=100 DURATION=60s bash benchmark/run-all.sh
```

| Env var | Default |
|---------|---------|
| `HOST` | `localhost` |
| `HTTP_PORT` | `8080` |
| `GRPC_PORT` | `9091` |
| `API_KEY` | auto-loaded from `deployments/.env` |
| `BENCH_EMAIL` | `bench@example.com` |
| `VUS` | `50` |
| `DURATION` | `30s` |

## Results

**WSL2 (Ubuntu), localhost, 50 VUs, 30 s, k6 v0.54**

| Metric | HTTP | gRPC |
|--------|------|------|
| RPS | 470 req/s | 289 req/s |
| Latency p50 | 9.86 ms | **9.78 ms** |
| Latency p90 | 22.66 ms | **20.11 ms** |
| Latency p95 | 31.84 ms | **25.62 ms** |
| Latency avg | 106 ms | 172 ms |
| Latency max | 8 715 ms | 9 330 ms |
| Errors | 0 / 16 951 | 0 / 10 618 |

gRPC wins at p50–p95. HTTP has higher RPS due to rare tail spikes (~9 s for both) consuming
VU time — avg and max are skewed by those outliers, p50–p95 is the honest comparison.
