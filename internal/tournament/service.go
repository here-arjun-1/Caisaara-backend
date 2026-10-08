package tournament

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/here-arjun-1/Caisaara-backend/internal/game"
)

type Broadcaster interface {
	BroadcastTournamentEvent(tournamentID int64, event interface{})
}

type Service interface {
	CreateTournament(ctx context.Context, userID int64, req CreateTournamentRequest) (*Tournament, error)
	ListPublicTournaments(ctx context.Context) ([]*TournamentWithPlayerCount, error)
	GetTournamentDetails(ctx context.Context, id int64) (*TournamentWithPlayerCount, error)
	GetTournamentByInviteCode(ctx context.Context, code string) (*TournamentWithPlayerCount, error)
	JoinTournament(ctx context.Context, tournamentID int64, userID int64, inviteCode string) (*TournamentPlayer, error)
	JoinTournamentByInviteCode(ctx context.Context, code string, userID int64) (*TournamentPlayer, error)
	LeaveTournament(ctx context.Context, tournamentID int64, userID int64) error
	StartTournament(ctx context.Context, tournamentID int64, userID int64) (*TournamentWithPlayerCount, error)
	GenerateRoundPairings(ctx context.Context, tournamentID int64, userID int64) ([]*TournamentPairing, error)
	OnGameCompleted(ctx context.Context, gameID string, result string) error
	GetStandings(ctx context.Context, tournamentID int64) (*TournamentStandingsResponse, error)
	GetRounds(ctx context.Context, tournamentID int64, roundNumber ...int) (*TournamentRoundsResponse, error)
	GetGames(ctx context.Context, tournamentID int64, userID ...int64) (*TournamentGamesResponse, error)
	SetBroadcaster(b Broadcaster)
}

type TournamentService struct {
	Repo        TournamentRepository
	GameRepo    game.GameRepository
	Broadcaster Broadcaster
}

func NewService(repo TournamentRepository, gameRepo game.GameRepository, broadcaster ...Broadcaster) *TournamentService {
	s := &TournamentService{
		Repo:     repo,
		GameRepo: gameRepo,
	}
	if len(broadcaster) > 0 {
		s.Broadcaster = broadcaster[0]
	}
	return s
}

func (s *TournamentService) SetBroadcaster(b Broadcaster) {
	s.Broadcaster = b
}

const inviteCodeChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateInviteCode() string {
	b := make([]byte, 6)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeChars))))
		if err != nil {
			b[i] = inviteCodeChars[0]
			continue
		}
		b[i] = inviteCodeChars[num.Int64()]
	}
	return string(b)
}

func (s *TournamentService) CreateTournament(ctx context.Context, userID int64, req CreateTournamentRequest) (*Tournament, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrInvalidName
	}

	timeControl := strings.TrimSpace(req.TimeControl)
	if timeControl == "" {
		return nil, ErrInvalidTimeControl
	}

	if req.Format != FormatSwiss {
		return nil, ErrInvalidFormat
	}

	visibility := strings.TrimSpace(req.Visibility)
	if visibility == "" {
		visibility = VisibilityPublic
	}
	if visibility != VisibilityPublic && visibility != VisibilityPrivate {
		return nil, ErrInvalidVisibility
	}

	if req.MaxPlayers < 2 {
		return nil, ErrInvalidMaxPlayers
	}

	totalRounds := req.TotalRounds
	if totalRounds <= 0 {
		totalRounds = 5
	}

	inviteCode := generateInviteCode()

	now := time.Now()
	startAt := now
	if req.StartAt != nil && !req.StartAt.IsZero() {
		startAt = *req.StartAt
	}

	tournament := &Tournament{
		Name:         name,
		Description:  strings.TrimSpace(req.Description),
		Format:       req.Format,
		TimeControl:  timeControl,
		MaxPlayers:   req.MaxPlayers,
		Visibility:   visibility,
		InviteCode:   &inviteCode,
		Status:       StatusRegistration,
		TotalRounds:  totalRounds,
		CurrentRound: 0,
		CreatedBy:    userID,
		StartAt:      startAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.Repo.CreateTournament(ctx, tournament); err != nil {
		return nil, err
	}

	return tournament, nil
}

func (s *TournamentService) ListPublicTournaments(ctx context.Context) ([]*TournamentWithPlayerCount, error) {
	return s.Repo.ListPublicTournaments(ctx)
}

func (s *TournamentService) GetTournamentDetails(ctx context.Context, id int64) (*TournamentWithPlayerCount, error) {
	return s.Repo.FindByIDWithPlayerCount(ctx, id)
}

func (s *TournamentService) GetTournamentByInviteCode(ctx context.Context, code string) (*TournamentWithPlayerCount, error) {
	trimmedCode := strings.TrimSpace(code)
	if trimmedCode == "" {
		return nil, ErrTournamentNotFound
	}
	return s.Repo.FindByInviteCode(ctx, trimmedCode)
}

func (s *TournamentService) JoinTournament(ctx context.Context, tournamentID int64, userID int64, inviteCode string) (*TournamentPlayer, error) {
	t, err := s.Repo.FindByIDWithPlayerCount(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if t.Status != StatusRegistration {
		return nil, ErrNotRegistration
	}

	if t.Players >= t.MaxPlayers {
		return nil, ErrTournamentFull
	}

	if t.Visibility == VisibilityPrivate {
		trimmedCode := strings.TrimSpace(inviteCode)
		if trimmedCode == "" || t.InviteCode == nil || !strings.EqualFold(trimmedCode, *t.InviteCode) {
			return nil, ErrInvalidInviteCode
		}
	}

	joined, err := s.Repo.IsPlayerJoined(ctx, tournamentID, userID)
	if err != nil {
		return nil, err
	}
	if joined {
		return nil, ErrAlreadyJoined
	}

	now := time.Now()
	player := &TournamentPlayer{
		TournamentID: tournamentID,
		UserID:       userID,
		Score:        0.0,
		Wins:         0,
		Draws:        0,
		Losses:       0,
		GamesPlayed:  0,
		JoinedAt:     now,
	}

	if err := s.Repo.AddPlayer(ctx, player); err != nil {
		return nil, err
	}

	return player, nil
}

func (s *TournamentService) JoinTournamentByInviteCode(ctx context.Context, code string, userID int64) (*TournamentPlayer, error) {
	tw, err := s.GetTournamentByInviteCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.JoinTournament(ctx, tw.ID, userID, code)
}

func (s *TournamentService) LeaveTournament(ctx context.Context, tournamentID int64, userID int64) error {
	t, err := s.Repo.FindByIDWithPlayerCount(ctx, tournamentID)
	if err != nil {
		return err
	}

	if t.Status != StatusRegistration {
		return ErrCannotLeaveStarted
	}

	joined, err := s.Repo.IsPlayerJoined(ctx, tournamentID, userID)
	if err != nil {
		return err
	}
	if !joined {
		return ErrNotJoined
	}

	return s.Repo.RemovePlayer(ctx, tournamentID, userID)
}

func (s *TournamentService) StartTournament(ctx context.Context, tournamentID int64, userID int64) (*TournamentWithPlayerCount, error) {
	t, err := s.Repo.FindByIDWithPlayerCount(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if t.CreatedBy != userID {
		return nil, ErrNotCreator
	}

	if t.Status != StatusRegistration {
		return nil, ErrNotRegistration
	}

	if t.Players < 2 {
		return nil, ErrNotEnoughPlayers
	}

	if err := s.Repo.StartTournament(ctx, tournamentID); err != nil {
		return nil, err
	}

	swissPlayers, err := s.Repo.GetSwissPlayers(ctx, tournamentID)
	if err == nil && len(swissPlayers) >= 2 {
		pairingsResult, err := GenerateSwissPairings(swissPlayers)
		if err == nil {
			_, _ = s.Repo.SaveRoundPairings(ctx, tournamentID, 1, pairingsResult, s.GameRepo)
		}
	}

	if s.Broadcaster != nil {
		s.Broadcaster.BroadcastTournamentEvent(tournamentID, map[string]interface{}{
			"type":  "tournament_round_started",
			"round": 1,
		})
	}

	return s.Repo.FindByIDWithPlayerCount(ctx, tournamentID)
}

func (s *TournamentService) GenerateRoundPairings(ctx context.Context, tournamentID int64, userID int64) ([]*TournamentPairing, error) {
	t, err := s.Repo.FindByIDWithPlayerCount(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if t.CreatedBy != userID {
		return nil, ErrNotCreator
	}

	if t.Status != StatusOngoing {
		return nil, errors.New("tournament is not ongoing")
	}

	swissPlayers, err := s.Repo.GetSwissPlayers(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	pairingsResult, err := GenerateSwissPairings(swissPlayers)
	if err != nil {
		return nil, err
	}

	roundNumber := t.CurrentRound
	if roundNumber <= 0 {
		roundNumber = 1
	}

	return s.Repo.SaveRoundPairings(ctx, tournamentID, roundNumber, pairingsResult, s.GameRepo)
}

func (s *TournamentService) OnGameCompleted(ctx context.Context, gameID string, result string) error {
	tournamentID, roundCompleted, err := s.Repo.UpdatePairingOnGameCompleted(ctx, gameID, result)
	if err != nil || tournamentID == 0 {
		return err
	}

	if s.Broadcaster != nil {
		s.Broadcaster.BroadcastTournamentEvent(tournamentID, map[string]interface{}{
			"type": "tournament_standings_updated",
		})
	}

	if roundCompleted {
		t, err := s.Repo.FindByIDWithPlayerCount(ctx, tournamentID)
		if err != nil {
			return err
		}

		if t.Status == StatusCompleted {
			if s.Broadcaster != nil {
				standings, _ := s.Repo.GetStandings(ctx, tournamentID)
				var winnerID *int64
				if len(standings) > 0 {
					wID := standings[0].PlayerID
					winnerID = &wID
				}
				s.Broadcaster.BroadcastTournamentEvent(tournamentID, map[string]interface{}{
					"type":      "tournament_completed",
					"winner_id": winnerID,
				})
			}
		} else if t.Status == StatusOngoing && t.CurrentRound <= t.TotalRounds {
			swissPlayers, err := s.Repo.GetSwissPlayers(ctx, tournamentID)
			if err != nil {
				return err
			}

			pairingsResult, err := GenerateSwissPairings(swissPlayers)
			if err != nil {
				return err
			}

			_, err = s.Repo.SaveRoundPairings(ctx, tournamentID, t.CurrentRound, pairingsResult, s.GameRepo)
			if err != nil {
				return err
			}

			if s.Broadcaster != nil {
				s.Broadcaster.BroadcastTournamentEvent(tournamentID, map[string]interface{}{
					"type":  "tournament_round_started",
					"round": t.CurrentRound,
				})
			}
		}
	}

	return nil
}

func (s *TournamentService) GetStandings(ctx context.Context, tournamentID int64) (*TournamentStandingsResponse, error) {
	_, err := s.Repo.FindByID(ctx, tournamentID)
	if err != nil {
		return nil, ErrTournamentNotFound
	}

	standings, err := s.Repo.GetStandings(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	return &TournamentStandingsResponse{
		TournamentID: strconv.FormatInt(tournamentID, 10),
		Standings:    standings,
	}, nil
}

func (s *TournamentService) GetRounds(ctx context.Context, tournamentID int64, roundNumber ...int) (*TournamentRoundsResponse, error) {
	_, err := s.Repo.FindByID(ctx, tournamentID)
	if err != nil {
		return nil, ErrTournamentNotFound
	}

	rounds, err := s.Repo.GetRounds(ctx, tournamentID, roundNumber...)
	if err != nil {
		return nil, err
	}

	return &TournamentRoundsResponse{
		TournamentID: strconv.FormatInt(tournamentID, 10),
		Rounds:       rounds,
	}, nil
}

func (s *TournamentService) GetGames(ctx context.Context, tournamentID int64, userID ...int64) (*TournamentGamesResponse, error) {
	_, err := s.Repo.FindByID(ctx, tournamentID)
	if err != nil {
		return nil, ErrTournamentNotFound
	}

	games, err := s.Repo.GetGames(ctx, tournamentID, userID...)
	if err != nil {
		return nil, err
	}

	return &TournamentGamesResponse{
		TournamentID: strconv.FormatInt(tournamentID, 10),
		Games:        games,
	}, nil
}
