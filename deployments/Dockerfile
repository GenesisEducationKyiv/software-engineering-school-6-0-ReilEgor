FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o bin/repo_notifier ./cmd/app/


FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/static ./static
COPY --from=builder /app/bin/repo_notifier .

RUN mkdir -p /app/data

CMD ["./repo_notifier"]