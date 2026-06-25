FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY shared/go.mod shared/go.sum* ./shared/
COPY services/subscription/go.mod services/subscription/go.sum* ./services/subscription/

RUN go work init ./shared ./services/subscription && go mod download

COPY shared/ ./shared/
COPY services/subscription/ ./services/subscription/
COPY static/ ./static/

RUN go build -o bin/repo_notifier ./services/subscription/cmd/api/


FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/static ./static
COPY --from=builder /app/bin/repo_notifier .

RUN mkdir -p /app/data

CMD ["./repo_notifier"]
