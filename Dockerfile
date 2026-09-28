# Build stage
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o caisaara-backend ./cmd/server

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy only the compiled binary
COPY --from=builder /app/caisaara-backend .

EXPOSE 8050

CMD ["./caisaara-backend"]