FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o launchpad ./cmd

FROM alpine:3.20

# ca-certificates for TLS; tzdata for consistent time zones in logs
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/launchpad .

EXPOSE 8080

CMD ["./launchpad"]
