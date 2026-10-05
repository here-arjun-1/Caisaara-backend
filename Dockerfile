FROM golang:alpine AS builder

WORKDIR /app

ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o caisaara-backend ./cmd/server

FROM alpine:3.24

RUN apk add --no-cache ca-certificates && adduser -D -H -u 10001 app

WORKDIR /app

COPY --from=builder /app/caisaara-backend .

USER app

EXPOSE 8050

CMD ["./caisaara-backend"]
