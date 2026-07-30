FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/edgegate ./cmd/edgegate

FROM alpine:3.22
RUN addgroup -S edgegate && adduser -S edgegate -G edgegate
WORKDIR /app
COPY --from=builder /out/edgegate ./edgegate
COPY config.example.json ./config.json
USER edgegate
EXPOSE 8080
ENTRYPOINT ["./edgegate", "-config", "/app/config.json"]
