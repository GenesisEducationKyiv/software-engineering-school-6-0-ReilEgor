FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY shared/go.mod shared/go.sum* ./shared/
COPY services/notification/go.mod services/notification/go.sum* ./services/notification/

RUN go work init ./shared ./services/notification && go mod download

COPY shared/ ./shared/
COPY services/notification/ ./services/notification/

RUN go build -o bin/notification_sender ./services/notification/cmd/notification-sender/


FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/bin/notification_sender .

CMD ["./notification_sender"]
