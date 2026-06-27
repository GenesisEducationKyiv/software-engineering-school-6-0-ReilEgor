FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY shared/go.mod shared/go.sum* ./shared/
COPY services/tracking/go.mod services/tracking/go.sum* ./services/tracking/
RUN go work init ./shared ./services/tracking && go mod download

COPY shared/ ./shared/
COPY services/tracking/ ./services/tracking/

RUN go build -o bin/notification_worker ./services/tracking/cmd/notification-worker/


FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/bin/notification_worker .

CMD ["./notification_worker"]
