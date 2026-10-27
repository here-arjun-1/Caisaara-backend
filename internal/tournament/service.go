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
	FindByID(ctx context.Context, id int64) (*Tournament, error)
	GetTournamentDetails(ctx context.Context, id int64) (*TournamentWithPlayerCount, error)
	GetTournamentByInviteCode(ctx context.Context, code string) (*TournamentWithPlayerCount, error)
	JoinTournament(ctx context.Context, tournamentID int64, userID int64, inviteCode string) (*TournamentPlayer, error)
	JoinTournamentByInviteCode(ctx context.Context, code string, userID int64) (*TournamentPlayer, error)
	LeaveTournament(ctx context.Context, tournamentID int64, userID int64) error
	StartTournament(ctx context.Context, tournamentID int64, userID int64) (*TournamentWithPlayerCount, error)
	EndTournament(ctx context.Context, tournamentID int64, userID int64) (*int64, error)
	GenerateRoundPairings(ctx context.Context, tournamentID int64, userID int64) ([]*TournamentPairing, error)
	OnGameCompleted(ctx context.Context, gameID string, result string) error
	GetStandings(ctx context.Context, tournamentID int64) (*TournamentStandingsResponse, error)
	GetRounds(ctx context.Context, tournamentID int64, roundNumber ...int) (*TournamentRoundsResponse, error)
	GetGames(ctx context.Context, tournamentID int64, userID ...int64) (*TournamentGamesResponse, error)
	DeleteOldTournaments(ctx context.Context, olderThan time.Duration) (int64, error)
	RemoveUserFromTournaments(ctx context.Context, userID int64) error
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
	if _, _, ok := parseTimeControl(timeControl); !ok {
		return nil, ErrInvalidTimeControl
	}

	if req.Format != FormatSwiss && req.Format != FormatRoundRobin && req.Format != FormatKnockout {
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

	minPlayers := req.MinPlayers
	if minPlayers < 2 {
		minPlayers = 2
	}
	if minPlayers > req.MaxPlayers {
		return nil, ErrInvalidMinPlayers
	}

	if req.MinRating < 0 {
		return nil, ErrInvalidMinRating
	}

	if req.TotalRounds < 0 || req.TotalRounds > 50 {
		return nil, ErrInvalidTotalRounds
	}

	totalRounds := req.TotalRounds
	if totalRounds == 0 {
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
		MinPlayers:   minPlayers,
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
		MinRating:    req.MinRating,
	}

	if err := s.Repo.CreateTournament(ctx, tournament); err != nil {
		return nil, err
	}

	return tournament, nil
}

func (s *TournamentService) ListPublicTournaments(ctx context.Context) ([]*TournamentWithPlayerCount, error) {
	return s.Repo.ListPublicTournaments(ctx)
}

func (s *TournamentService) FindByID(ctx context.Context, id int64) (*Tournament, error) {
	return s.Repo.FindByID(ctx, id)
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

	if t.MinRating > 0 {
		tcMinutes, tcIncrement, _ := parseTimeControl(t.TimeControl)
		mode, _, _, _ := game.ResolveTimeControlMode(tcMinutes, tcIncrement)

		rating, err := s.Repo.GetPlayerRating(ctx, userID, mode)
		if err != nil {
			return nil, err
		}
		if rating < t.MinRating {
			return nil, ErrRatingTooLow
		}
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

	if t.Players < t.MinPlayers {
		return nil, ErrNotEnoughPlayers
	}

	if err := s.Repo.StartTournament(ctx, tournamentID); err != nil {
		return nil, err
	}

	pairings, err := s.createFirstRound(ctx, t)
	if err != nil {
		_ = s.Repo.UndoStartTournament(ctx, tournamentID)
		return nil, err
	}

	if s.Broadcaster != nil {
		s.Broadcaster.BroadcastTournamentEvent(tournamentID, map[string]interface{}{
			"type":     "tournament_round_started",
			"round":    1,
			"pairings": pairings,
		})
	}

	return s.Repo.FindByIDWithPlayerCount(ctx, tournamentID)
}

func (s *TournamentService) createFirstRound(ctx context.Context, t *TournamentWithPlayerCount) ([]*TournamentPairing, error) {
	swissPlayers, err := s.Repo.GetSwissPlayers(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	if len(swissPlayers) < 2 || len(swissPlayers) < t.MinPlayers {
		return nil, ErrNotEnoughPlayers
	}

	var pairings []SwissPairingResult
	totalRounds := t.TotalRounds

	switch t.Format {
	case FormatRoundRobin:
		playerIDs := make([]int64, len(swissPlayers))
		for i, p := range swissPlayers {
			playerIDs[i] = p.UserID
		}

		schedule, rounds, err := GenerateRoundRobinSchedule(playerIDs)
		if err != nil {
			return nil, err
		}
		pairings = schedule[1]
		totalRounds = rounds
	case FormatKnockout:
		knockoutPairings, rounds, err := GenerateKnockoutRound1(swissPlayers)
		if err != nil {
			return nil, err
		}
		pairings = knockoutPairings
		totalRounds = rounds
	default:
		maxRounds := len(swissPlayers) - 1
		if len(swissPlayers)%2 != 0 {
			maxRounds = len(swissPlayers)
		}
		if totalRounds > maxRounds {
			totalRounds = maxRounds
		}

		swissPairings, err := GenerateSwissPairings(swissPlayers)
		if err != nil {
			return nil, err
		}
		pairings = swissPairings
	}

	if totalRounds != t.TotalRounds {
		if err := s.Repo.UpdateTotalRounds(ctx, t.ID, totalRounds); err != nil {
			return nil, err
		}
	}

	return s.Repo.SaveRoundPairings(ctx, t.ID, 1, pairings, s.GameRepo)
}

func (s *TournamentService) EndTournament(ctx context.Context, tournamentID int64, userID int64) (*int64, error) {
	t, err := s.Repo.FindByIDWithPlayerCount(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	if t.CreatedBy != userID {
		return nil, ErrNotCreator
	}

	if t.Status != StatusOngoing {
		return nil, ErrNotOngoing
	}

	if err := s.Repo.EndTournament(ctx, tournamentID); err != nil {
		return nil, err
	}

	standings, err := s.Repo.GetStandings(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	withdrawn, err := s.Repo.GetWithdrawnPlayers(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	winnerID := pickWinner(standings, withdrawn)

	if s.Broadcaster != nil {
		s.Broadcaster.BroadcastTournamentEvent(tournamentID, map[string]interface{}{
			"type":      "tournament_completed",
			"winner_id": winnerID,
		})
	}

	return winnerID, nil
}

func (s *TournamentService) generateNextPairings(ctx context.Context, tournamentID int64, t *TournamentWithPlayerCount, swissPlayers []*SwissPlayer, withdrawn map[int64]bool) ([]SwissPairingResult, error) {
	switch t.Format {
	case FormatRoundRobin:
		playerIDs := make([]int64, len(swissPlayers))
		for i, p := range swissPlayers {
			playerIDs[i] = p.UserID
		}
		schedule, _, err := GenerateRoundRobinSchedule(playerIDs)
		if err != nil {
			return nil, err
		}
		return removeWithdrawnFromPairings(schedule[t.CurrentRound], withdrawn), nil
	case FormatKnockout:
		prevRoundNumber := t.CurrentRound - 1
		if prevRoundNumber <= 0 {
			prevRoundNumber = 1
		}
		rounds, err := s.Repo.GetRounds(ctx, tournamentID, prevRoundNumber)
		if err != nil || len(rounds) == 0 {
			return nil, errors.New("previous round not found")
		}
		rID, err := strconv.ParseInt(rounds[0].ID, 10, 64)
		if err != nil {
			return nil, err
		}
		winners, err := s.Repo.GetRoundWinners(ctx, tournamentID, rID)
		if err != nil || len(winners) == 0 {
			return nil, errors.New("no round winners found")
		}

		activeWinners := make([]int64, 0, len(winners))
		for _, id := range winners {
			if !withdrawn[id] {
				activeWinners = append(activeWinners, id)
			}
		}
		if len(activeWinners) < 2 {
			return byePairings(activeWinners), nil
		}
		return GenerateKnockoutNextRound(activeWinners)
	default:
		activePlayers := make([]*SwissPlayer, 0, len(swissPlayers))
		activeIDs := make([]int64, 0, len(swissPlayers))
		for _, p := range swissPlayers {
			if !withdrawn[p.UserID] {
				activePlayers = append(activePlayers, p)
				activeIDs = append(activeIDs, p.UserID)
			}
		}
		if len(activePlayers) < 2 {
			return byePairings(activeIDs), nil
		}
		return GenerateSwissPairings(activePlayers)
	}
}

func removeWithdrawnFromPairings(pairings []SwissPairingResult, withdrawn map[int64]bool) []SwissPairingResult {
	result := make([]SwissPairingResult, 0, len(pairings))
	for _, p := range pairings {
		whiteOut := p.WhitePlayerID != nil && withdrawn[*p.WhitePlayerID]
		blackOut := p.BlackPlayerID != nil && withdrawn[*p.BlackPlayerID]

		if p.IsBye {
			if !whiteOut {
				result = append(result, p)
			}
			continue
		}

		if whiteOut && blackOut {
			continue
		}
		if whiteOut {
			result = append(result, SwissPairingResult{WhitePlayerID: p.BlackPlayerID, IsBye: true})
			continue
		}
		if blackOut {
			result = append(result, SwissPairingResult{WhitePlayerID: p.WhitePlayerID, IsBye: true})
			continue
		}
		result = append(result, p)
	}
	return result
}

func byePairings(playerIDs []int64) []SwissPairingResult {
	pairings := make([]SwissPairingResult, 0, len(playerIDs))
	for _, id := range playerIDs {
		playerID := id
		pairings = append(pairings, SwissPairingResult{WhitePlayerID: &playerID, IsBye: true})
	}
	return pairings
}

func pickWinner(standings []*StandingsPlayerResponse, withdrawn map[int64]bool) *int64 {
	for _, p := range standings {
		if !withdrawn[p.PlayerID] {
			winnerID := p.PlayerID
			return &winnerID
		}
	}
	return nil
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

	withdrawn, err := s.Repo.GetWithdrawnPlayers(ctx, tournamentID)
	if err != nil {
		return nil, err
	}

	pairingsResult, err := s.generateNextPairings(ctx, tournamentID, t, swissPlayers, withdrawn)
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
		return s.startNextRound(ctx, tournamentID)
	}

	return nil
}

func (s *TournamentService) startNextRound(ctx context.Context, tournamentID int64) error {
	for {
		t, err := s.Repo.FindByIDWithPlayerCount(ctx, tournamentID)
		if err != nil {
			return err
		}

		withdrawn, err := s.Repo.GetWithdrawnPlayers(ctx, tournamentID)
		if err != nil {
			return err
		}

		if t.Status == StatusCompleted {
			if s.Broadcaster != nil {
				standings, _ := s.Repo.GetStandings(ctx, tournamentID)
				s.Broadcaster.BroadcastTournamentEvent(tournamentID, map[string]interface{}{
					"type":      "tournament_completed",
					"winner_id": pickWinner(standings, withdrawn),
				})
			}
			return nil
		}

		if t.Status != StatusOngoing || t.CurrentRound > t.TotalRounds {
			return nil
		}

		swissPlayers, err := s.Repo.GetSwissPlayers(ctx, tournamentID)
		if err != nil {
			return err
		}

		pairingsResult, err := s.generateNextPairings(ctx, tournamentID, t, swissPlayers, withdrawn)
		if err != nil {
			return err
		}

		savedPairings, err := s.Repo.SaveRoundPairings(ctx, tournamentID, t.CurrentRound, pairingsResult, s.GameRepo)
		if err != nil {
			return err
		}

		if s.Broadcaster != nil {
			s.Broadcaster.BroadcastTournamentEvent(tournamentID, map[string]interface{}{
				"type":     "tournament_round_started",
				"round":    t.CurrentRound,
				"pairings": savedPairings,
			})
		}

		finished, err := s.Repo.FinishRoundIfNoGames(ctx, tournamentID, t.CurrentRound)
		if err != nil {
			return err
		}
		if !finished {
			return nil
		}
	}
}

func (s *TournamentService) RemoveUserFromTournaments(ctx context.Context, userID int64) error {
	cancelledIDs, err := s.Repo.RemoveUserFromTournaments(ctx, userID)
	if err != nil {
		return err
	}

	if s.Broadcaster != nil {
		for _, id := range cancelledIDs {
			s.Broadcaster.BroadcastTournamentEvent(id, map[string]interface{}{
				"type": "tournament_cancelled",
			})
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

func (s *TournamentService) DeleteOldTournaments(ctx context.Context, olderThan time.Duration) (int64, error) {
	return s.Repo.DeleteOldTournaments(ctx, olderThan)
}

func StartTournamentCleanup(ctx context.Context, repo TournamentRepository, interval time.Duration) {
	if repo == nil {
		return
	}
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				_, _ = repo.DeleteOldTournaments(ctx, 30*24*time.Hour)
			}
		}
	}()
}
