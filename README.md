# Caisaara Backend

This is the backend for Caisaara, our chess app. It's written in Go (Gin) and uses Postgres with PostGIS, Redis, and Stockfish for the bots and game analysis.

Right now it handles login/signup, player profiles and ratings (Glicko-2), live games over websockets, matchmaking, bots, puzzles, tournaments (swiss, round robin, knockout), friends, DMs, clubs, community posts, a leaderboard and a few other things. Each feature has its own folder inside `internal/`.

## Running it

You need Go 1.26+ and Docker. If you want to run the server outside Docker you also need Stockfish installed (`brew install stockfish` on mac).

First copy the env file and fill it in:

```
cp .env.example .env
```

The server won't start unless `dbURL`, `JWT_SECRET`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `AWS_REGION`, `S3_BUCKET` and `S3_PUBLIC_URL` are set. The rest have defaults (port is 8050, redis is localhost:6379).

The easiest way is to run everything with compose:

```
docker compose up -d --build
```

That brings up the backend, postgres, redis and the monitoring stuff. API will be on http://localhost:8050, you can hit `/health` to check it's up.

If you'd rather run the server with `go run ./cmd/server`, keep in mind compose doesn't expose the postgres and redis ports, so either add them in `compose.yml` locally or point `.env` at your own postgres/redis.

Migrations run on their own when the server starts, you don't need to do anything for them. They live in `migrations/sql`.

## Puzzles

Puzzles come from the lichess puzzle db. Download it from https://database.lichess.org/#puzzles, unzip it and run the seeder:

```
zstd -d lichess_db_puzzle.csv.zst
go run ./cmd/seed-puzzles -file lichess_db_puzzle.csv
```

It has some flags for filtering by rating, popularity etc, check `-h`.

## Routes

Most routes are under `/api` and need a JWT. Websockets are under `/ws` (games, tournaments, spectating, DMs). Easiest way to see what's there is the `routes.go` in each package, all of them get wired up in `internal/router/router.go`.

## Monitoring

Compose also starts Prometheus (9090), Grafana (3000), Loki (3100) and Alloy for shipping logs. These only listen on localhost. Config is in the `monitoring` folder.

## Before pushing

CI checks formatting and runs golangci-lint, so run these first or the build will fail:

```
gofmt -l .
golangci-lint run
```

## Deploy

Anything merged into `main` gets deployed to our EC2 server through GitHub Actions (`.github/workflows/deploy.yml`). It builds the image, copies it over, and runs compose. If the new container doesn't come up healthy it goes back to the previous image.
