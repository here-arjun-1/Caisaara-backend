package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/corentings/chess/v2"
	"github.com/google/uuid"
	"github.com/here-arjun-1/Caisaara-backend/internal/bot"
	"github.com/here-arjun-1/Caisaara-backend/internal/community/model"
	"github.com/jackc/pgx/v5"
)

func (s *PostService) attachGame(ctx context.Context, post *model.Post, gameID string) error {
	if _, err := uuid.Parse(gameID); err != nil {
		return ErrGameNotFound
	}

	g, err := s.GameRepository.FindGame(ctx, gameID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrGameNotFound
	}
	if err != nil {
		slog.Error("find game for post failed", "game_id", gameID, "error", err)
		return ErrInternal
	}

	if post.UserID != g.WhitePlayerID && post.UserID != g.BlackPlayerID {
		return ErrNotAllowed
	}
	if g.Status != "finished" || len(g.Moves) == 0 {
		return ErrGameNotFinished
	}

	pgn, err := buildPGN(g.Moves, g.Result, g.WhiteName, g.BlackName, g.EndedAt)
	if err != nil {
		slog.Error("build pgn failed", "game_id", gameID, "error", err)
		return ErrInternal
	}

	post.FEN = &g.Position
	post.Game = &model.PostGame{
		ID:          gameID,
		PGN:         pgn,
		WhiteName:   g.WhiteName,
		BlackName:   g.BlackName,
		WhiteRating: g.WhiteRating,
		BlackRating: g.BlackRating,
		Result:      g.Result,
		EndReason:   g.EndReason,
		Mode:        g.Mode,
		PlayedAt:    g.EndedAt,
	}

	return nil
}

func (s *PostService) attachBotGame(ctx context.Context, post *model.Post, gameID string) error {
	if _, err := uuid.Parse(gameID); err != nil {
		return ErrGameNotFound
	}

	g, err := s.GameRepository.FindBotGame(ctx, gameID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrGameNotFound
	}
	if err != nil {
		slog.Error("find bot game for post failed", "game_id", gameID, "error", err)
		return ErrInternal
	}

	if post.UserID != g.PlayerID {
		return ErrNotAllowed
	}

	moves := strings.Fields(g.Moves)
	if g.Status != "finished" || len(moves) == 0 {
		return ErrGameNotFinished
	}

	botName := "Bot"
	if character, ok := bot.FindCharacter(g.BotID); ok {
		botName = character.Name
	}
	botRating := g.BotRating

	whiteName, blackName := g.Username, botName
	var whiteRating, blackRating *int
	if g.PlayerColor == "white" {
		blackRating = &botRating
	} else {
		whiteName, blackName = botName, g.Username
		whiteRating = &botRating
	}

	pgn, err := buildPGN(moves, g.Result, whiteName, blackName, g.EndedAt)
	if err != nil {
		slog.Error("build bot game pgn failed", "game_id", gameID, "error", err)
		return ErrInternal
	}

	post.FEN = &g.Position
	post.Game = &model.PostGame{
		ID:          gameID,
		PGN:         pgn,
		WhiteName:   whiteName,
		BlackName:   blackName,
		WhiteRating: whiteRating,
		BlackRating: blackRating,
		Result:      g.Result,
		EndReason:   g.EndReason,
		Mode:        "bot",
		PlayedAt:    g.EndedAt,
	}

	return nil
}

func buildPGN(moves []string, result, whiteName, blackName string, playedAt *time.Time) (string, error) {
	board := chess.NewGame()

	for _, move := range moves {
		err := board.PushNotationMove(move, chess.UCINotation{}, nil)
		if err != nil {
			err = board.PushNotationMove(move, chess.AlgebraicNotation{}, nil)
		}
		if err != nil {
			return "", err
		}
	}

	switch result {
	case "white_win":
		board.Resign(chess.Black)
	case "black_win":
		board.Resign(chess.White)
	case "draw":
		_ = board.Draw(chess.DrawOffer)
	}

	board.AddTagPair("Event", "Caisaara")
	board.AddTagPair("White", whiteName)
	board.AddTagPair("Black", blackName)
	board.AddTagPair("Result", board.Outcome().String())
	if playedAt != nil {
		board.AddTagPair("Date", playedAt.Format("2006.01.02"))
	}

	return board.String(), nil
}
