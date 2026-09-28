# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o caisaara-backend ./cmd/server/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/caisaara-backend .

# Expose the port the app runs on (modify if your app uses a different port)
EXPOSE 8080

# Command to run the executable
CMD ["./caisaara-backend"]
