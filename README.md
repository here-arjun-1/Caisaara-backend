# Caisaara Backend

Caisaara is a chess app. This repo is its backend, written in Go (Gin), with Postgres + PostGIS for data, Redis for live game state and matchmaking, and Stockfish for bots and game analysis.

What it does:

- Login/signup with email verification, guest accounts, JWT access + refresh tokens
- Player profiles, avatars and Glicko-2 ratings per time control
- Live games over websockets, matchmaking, invite links, challenges and spectating
- Bots and post-game analysis using Stockfish
- Daily puzzles, puzzle training and puzzle rush (lichess puzzle db)
- Tournaments (swiss, round robin, knockout)
- Friends, DMs, clubs with chat, community posts, notifications (in-app + push), nearby players
- Leaderboard, tilt detection, privacy settings and account deletion

Each feature has its own folder in `internal/` with its own `routes.go`, `handler.go`, `service.go` and `repository.go`. Everything gets wired up in `internal/router/router.go`.

## Running it

You need Go 1.26+ and Docker. To run the server outside Docker you also need Stockfish (`brew install stockfish` on mac).

```
cp .env.example .env
docker compose up -d --build
```

The server won't start unless `dbURL`, `JWT_SECRET`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `AWS_REGION`, `S3_BUCKET` and `S3_PUBLIC_URL` are set. For compose also fill `POSTGRES_*` and `GRAFANA_ADMIN_*`. The rest are optional:

- `PORT` (8050), `REDIS_URL` (localhost:6379), `STOCKFISH_PATH` (stockfish)
- `FIREBASE_CREDENTIALS_BASE64` for push notifications (base64 of the firebase service account json). If empty, push is off but everything else works
- `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` only if the machine has no AWS credentials already

API runs on http://localhost:8050, check `/health`. Migrations run on their own at startup (`migrations/sql`).

If you use `go run ./cmd/server` instead, compose doesn't expose postgres/redis ports, so expose them locally or point `.env` at your own.

### Puzzles

Download the puzzle db from https://database.lichess.org/#puzzles and seed it:

```
zstd -d lichess_db_puzzle.csv.zst
go run ./cmd/seed-puzzles -file lichess_db_puzzle.csv
```

Check `-h` for filters (rating, popularity etc).

## API

Protected routes need `Authorization: Bearer <access_token>`. Websockets take the token as `?token=<access_token>`. Some routes also need a registered (non-guest) account.

Every response looks like this:

```json
{ "success": true, "message": "...", "data": {} }
{ "success": false, "error": "..." }
```

### Auth

| Method | Route | Use |
|---|---|---|
| POST | `/register` | Sign up, sends a code to email |
| POST | `/verify-registration` | Verify email code and create account |
| POST | `/guest-login` | Play as guest |
| POST | `/login` | Login, returns access + refresh token |
| POST | `/refresh` | Get new access token |
| POST | `/logout` | Logout this device |
| POST | `/logout-all` | Logout all devices |
| POST | `/forgot-password` | Send reset code to email |
| POST | `/verify-reset-code` | Check reset code |
| POST | `/reset-password` | Set new password |
| POST | `/api/rating` | Set starting rating |
| DELETE | `/api/account` | Delete account |

### Players and profile

| Method | Route | Use |
|---|---|---|
| GET | `/players/search` | Search players by username |
| GET | `/players/:username` | Public profile |
| GET | `/players/:username/stats/:mode` | Stats (all modes or one) |
| GET | `/players/:username/games` | Player's games |
| GET | `/avatars` | List default avatars |
| GET | `/api/profile` | My profile |
| PATCH | `/api/profile` | Update my profile |
| POST | `/api/profile/avatar` | Upload photo |
| PUT | `/api/profile/avatar/default` | Pick a default avatar |
| DELETE | `/api/profile/avatar` | Remove avatar |
| GET / PUT | `/api/privacy` | Get / update privacy settings |

The `/players` routes also work under `/api`.

### Games

| Method | Route | Use |
|---|---|---|
| GET | `/api/games/history` | My game history |
| GET | `/api/games/:gameID` | Game details |
| GET | `/api/games/:gameID/moves` | Move list |
| POST | `/api/games/:gameID/resign` | Resign |
| POST | `/api/games/:gameID/timeout` | Claim win on time |
| POST | `/api/games/:gameID/draw/offer` | Offer draw |
| POST | `/api/games/:gameID/draw/accept` | Accept draw |
| POST | `/api/games/:gameID/draw/decline` | Decline draw |
| GET / POST | `/api/games/:gameID/messages` | In-game chat |
| POST | `/api/matchmaking/join` | Join queue |
| POST | `/api/matchmaking/cancel` | Leave queue |
| GET | `/api/matchmaking/status` | Queue status |

### Invites and challenges

| Method | Route | Use |
|---|---|---|
| POST | `/api/invite` | Create invite link |
| GET | `/api/invite/:code` | Preview invite |
| POST | `/api/invite/:code/join` | Join invite game |
| GET | `/play/:code` | Invite link opened in browser/app |
| GET | `/api/challenges` | My incoming challenges |
| POST | `/api/challenges/:code/decline` | Decline challenge |
| POST | `/api/games/:gameID/watch-invite` | Link for others to watch |

### Bots and analysis

| Method | Route | Use |
|---|---|---|
| GET | `/api/bot-games/bots` | List bots |
| POST | `/api/bot-games` | Start bot game |
| GET | `/api/bot-games` | My bot games |
| GET | `/api/bot-games/:gameID` | Bot game details |
| POST | `/api/games/:gameID/analysis` | Start analysis |
| GET | `/api/games/:gameID/analysis` | Analysis result |
| GET | `/api/games/:gameID/analysis/moves/:moveNumber` | Per-move analysis (all or one) |
| POST | `/api/games/:gameID/analysis/retry` | Retry failed analysis |
| GET | `/api/tilt/status` | Tilt status from recent games |

### Puzzles

All need login.

| Method | Route | Use |
|---|---|---|
| GET | `/puzzles/daily` | Daily puzzle |
| POST | `/puzzles/daily/attempt` | Submit daily move |
| POST | `/puzzles/daily/hint` | Hint |
| POST | `/puzzles/daily/solution` | Show solution |
| GET | `/puzzles/next` | Next training puzzle |
| GET | `/puzzles/themes` | List themes |
| GET | `/puzzles/custom` | Puzzle by theme/filters |
| POST | `/puzzles/custom/attempt` | Submit custom puzzle move |
| GET | `/puzzles/stats` | My puzzle stats |
| GET | `/puzzles/history` | My puzzle history |
| POST | `/puzzles/rush/start` | Start puzzle rush |
| POST | `/puzzles/rush/move` | Rush move |
| POST | `/puzzles/rush/end` | End rush |
| GET | `/puzzles/rush/leaderboard` | Rush leaderboard |
| GET | `/puzzles/rush/stats` | My rush stats |
| GET | `/puzzles/rush/history` | My rush history |

### Tournaments

| Method | Route | Use |
|---|---|---|
| GET | `/api/tournaments` | Public tournaments |
| GET | `/api/tournaments/:id` | Details |
| GET | `/api/tournaments/:id/standings` | Standings |
| GET | `/api/tournaments/:id/rounds/:roundNumber` | Rounds (all or one) |
| GET | `/api/tournaments/:id/games` | All games |
| GET | `/api/tournaments/invite/:code` | Find by invite code |
| POST | `/api/tournaments` | Create |
| POST | `/api/tournaments/:id/start` | Start (creator) |
| POST | `/api/tournaments/:id/end` | End (creator) |
| POST | `/api/tournaments/:id/join` | Join |
| POST | `/api/tournaments/:id/leave` | Leave |
| POST | `/api/tournaments/invite/:code/join` | Join with invite code |
| POST | `/api/tournaments/:id/regenerate-invite` | New invite code |
| GET | `/api/tournaments/:id/my-games` | My games in it |

### Friends and messages

| Method | Route | Use |
|---|---|---|
| GET | `/api/friends` | Friend list |
| DELETE | `/api/friends/:userID` | Remove friend |
| POST | `/api/friends/requests` | Send request |
| GET | `/api/friends/requests` | Incoming requests |
| GET | `/api/friends/requests/sent` | Sent requests |
| POST | `/api/friends/requests/:id/accept` | Accept |
| POST | `/api/friends/requests/:id/reject` | Reject |
| DELETE | `/api/friends/requests/:id` | Cancel sent request |
| GET | `/api/friends/status/:username` | Friendship status |
| GET | `/api/friends/blocks` | Blocked users |
| POST / DELETE | `/api/friends/blocks/:userID` | Block / unblock |
| GET | `/api/messages` | Conversations |
| GET / POST | `/api/messages/:userID` | Read / send DM |

### Clubs

| Method | Route | Use |
|---|---|---|
| GET / POST | `/api/clubs` | List / create clubs |
| GET | `/api/clubs/mine` | My clubs |
| GET | `/api/clubs/code/:code` | Club by share code (also `/club/:code`) |
| POST | `/api/clubs/code/:code/join` | Join by code |
| GET / DELETE | `/api/clubs/:clubID` | Details / delete |
| POST | `/api/clubs/:clubID/join` | Join |
| POST | `/api/clubs/:clubID/leave` | Leave |
| POST | `/api/clubs/:clubID/regenerate-invite` | New share code |
| GET | `/api/clubs/:clubID/members` | Members |
| DELETE | `/api/clubs/:clubID/members/:userID` | Kick member |
| GET | `/api/clubs/:clubID/leaderboard` | Club leaderboard |
| POST / DELETE | `/api/clubs/:clubID/admins/:userID` | Make / remove admin |
| POST | `/api/clubs/:clubID/transfer/:userID` | Transfer ownership |
| GET | `/api/clubs/:clubID/bans` | Banned members |
| POST / DELETE | `/api/clubs/:clubID/bans/:userID` | Ban / unban |
| GET / POST | `/api/clubs/:clubID/messages` | Club chat |

### Community

| Method | Route | Use |
|---|---|---|
| GET | `/posts` | Feed |
| GET | `/posts/:id` | Single post |
| GET | `/posts/:id/comments` | Comments |
| POST | `/api/posts` | Create post |
| DELETE | `/api/posts/:id` | Delete post |
| PUT / DELETE | `/api/posts/:id/reaction` | Like/dislike / remove |
| POST | `/api/posts/:id/comments` | Comment |
| DELETE | `/api/comments/:id` | Delete comment |
| GET | `/api/bookmarks` | My bookmarks |
| PUT / DELETE | `/api/posts/:id/bookmark` | Bookmark / remove |

### Notifications, nearby, leaderboard

| Method | Route | Use |
|---|---|---|
| GET | `/api/notifications` | My notifications |
| GET | `/api/notifications/unread-count` | Unread count |
| POST | `/api/notifications/:id/read` | Mark one read |
| POST | `/api/notifications/read-all` | Mark all read |
| POST / DELETE | `/api/notifications/devices` | Register / remove push device token |
| PUT | `/api/nearby/location` | Update my location |
| PATCH | `/api/nearby/visibility` | Show/hide me in nearby |
| GET | `/api/nearby/players` | Players near me |
| GET | `/api/leaderboard` | Global leaderboard |

### Websockets

| Route | Use |
|---|---|
| `/ws/:gameID` | Play a live game |
| `/ws/bot-games/:gameID` | Play a bot game |
| `/ws/watch/:code` | Spectate a game |
| `/ws/tournament/:tournamentID` | Tournament updates |
| `/ws/clubs/:clubID` | Club chat |
| `/ws/messages` | DMs |
| `/ws/notifications` | Live notifications |

### Other

- `GET /health` to check the server is up
- `GET /.well-known/assetlinks.json` for Android app links

## Monitoring

Compose also starts Prometheus (9090), Grafana (3000), Loki (3100) and Alloy for logs, only on localhost. Config is in `monitoring/`.

## Before pushing

CI runs gofmt and golangci-lint, so run these first:

```
gofmt -l .
golangci-lint run
```

## Deploy

Anything merged into `main` is deployed to EC2 by GitHub Actions (`.github/workflows/deploy.yml`). It builds the image, copies it over and runs compose. If the new container isn't healthy it rolls back to the previous image.
