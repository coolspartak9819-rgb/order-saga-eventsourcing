FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG SERVICE=order-service
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/app ./cmd/${SERVICE}

FROM alpine:latest AS runner

WORKDIR /app

COPY --from=builder /app/app ./app
COPY scripts/init.sql ./scripts/init.sql

EXPOSE 8080 50051 50052

CMD ["./app"]
