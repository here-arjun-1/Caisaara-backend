FROM golang:alpine AS builder

WORKDIR /app

ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o caisaara-backend ./cmd/server

FROM debian:trixie-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates stockfish wget \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --no-create-home --uid 10001 app

ENV STOCKFISH_PATH=/usr/games/stockfish

WORKDIR /app

COPY --from=builder /app/caisaara-backend .

USER app

EXPOSE 8050

CMD ["./caisaara-backend"]
