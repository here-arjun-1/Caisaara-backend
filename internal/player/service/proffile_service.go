package service

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/here-arjun-1/Caisaara-backend/internal/game"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/rating/glicko2"
	"github.com/jackc/pgx/v5"
)

const (
	defaultRating    = 1200
	recentGamesLimit = 10
	searchLimit      = 10
	minSearchLength  = 2
	maxSearchLength  = 50
)

var ratedModes = []string{
	game.ModeBullet,
	game.ModeBlitz,
	game.ModeRapid,
}

var likeEscaper = strings.NewReplacer(
	"\\", "\\\\",
	"%", "\\%",
	"_", "\\_",
)

type ProfileRepository interface {
	FindByUserID(ctx context.Context, userID int64) (*model.Profile, error)
	FindByUsername(ctx context.Context, username string) (*model.Profile, error)
	SaveProfile(ctx context.Context, p *model.Profile) error
	FindModeRatings(ctx context.Context, userID int64) ([]model.ModeRating, error)
	FindModeGameStats(ctx context.Context, userID int64, mode string) (*model.ModeGameStats, error)
	FindRecentGamesByMode(ctx context.Context, userID int64, mode string, limit int) ([]model.RecentGame, error)
	FindRatingHistory(ctx context.Context, userID int64, mode string, limit int) ([]model.RatingHistoryPoint, error)
	SearchByUsername(ctx context.Context, prefix string, limit int) ([]model.Profile, error)
}

type ProfileService struct {
	ProfileRepository ProfileRepository
}

func NewProfileService(profileRepository ProfileRepository) *ProfileService {
	return &ProfileService{
		ProfileRepository: profileRepository,
	}
}

func (h *ProfileService) GetMyProfile(ctx context.Context, userID int64) (*dto.MyProfileResponse, error) {
	p, err := h.ProfileRepository.FindByUserID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		slog.Error("find profile by id failed", "error", err)
		return nil, ErrInternal
	}

	if err := h.loadRatings(ctx, p); err != nil {
		return nil, err
	}

	return toMyProfileResponse(p), nil
}

func (h *ProfileService) GetPublicProfile(ctx context.Context, username string) (*dto.PublicProfileResponse, error) {
	username = strings.ToLower(strings.TrimSpace(username))

	p, err := h.ProfileRepository.FindByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		slog.Error("find profile by username failed", "error", err)
		return nil, ErrInternal
	}

	if err := h.loadRatings(ctx, p); err != nil {
		return nil, err
	}

	return &dto.PublicProfileResponse{
		UserID:      p.UserID,
		Username:    p.Username,
		DisplayName: p.DisplayName,
		Country:     p.Country,
		Bio:         p.Bio,
		AvatarURL:   p.AvatarURL,
		Rating:      p.Rating,
		Provisional: glicko2.IsProvisional(p.RatingDeviation),
		Ratings:     toModeRatings(p),
		JoinedAt:    p.CreatedAt,
	}, nil
}

func (h *ProfileService) SearchPlayers(ctx context.Context, query string) ([]dto.PlayerSearchResult, error) {
	query = strings.ToLower(strings.TrimSpace(query))

	length := utf8.RuneCountInString(query)
	if length < minSearchLength || length > maxSearchLength {
		return nil, ErrInvalidSearchQuery
	}

	profiles, err := h.ProfileRepository.SearchByUsername(ctx, likeEscaper.Replace(query), searchLimit)
	if err != nil {
		slog.Error("search players failed", "error", err)
		return nil, ErrInternal
	}

	results := make([]dto.PlayerSearchResult, 0, len(profiles))
	for _, p := range profiles {
		results = append(results, dto.PlayerSearchResult{
			Username:    p.Username,
			DisplayName: p.DisplayName,
			AvatarURL:   p.AvatarURL,
			Rating:      p.Rating,
		})
	}

	return results, nil
}

func (h *ProfileService) UpdateMyProfile(ctx context.Context, userID int64, req dto.UpdateProfileData) (*dto.MyProfileResponse, error) {
	p, err := h.ProfileRepository.FindByUserID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		slog.Error("find profile by id failed", "error", err)
		return nil, ErrInternal
	}

	if req.DisplayName != nil {
		v := strings.TrimSpace(*req.DisplayName)
		if utf8.RuneCountInString(v) > 30 || strings.ContainsAny(v, "\n\r\t") {
			return nil, ErrInvalidDisplayName
		}
		p.DisplayName = emptyToNil(v)
	}

	if req.Country != nil {
		v := strings.ToUpper(strings.TrimSpace(*req.Country))
		if v != "" && !isCountryCode(v) {
			return nil, ErrInvalidCountry
		}
		p.Country = emptyToNil(v)
	}

	if req.Bio != nil {
		v := strings.TrimSpace(*req.Bio)
		if utf8.RuneCountInString(v) > 160 {
			return nil, ErrInvalidBio
		}
		p.Bio = emptyToNil(v)
	}

	if err := h.ProfileRepository.SaveProfile(ctx, p); err != nil {
		slog.Error("save profile failed", "error", err)
		return nil, ErrInternal
	}

	if err := h.loadRatings(ctx, p); err != nil {
		return nil, err
	}

	return toMyProfileResponse(p), nil
}

func toMyProfileResponse(p *model.Profile) *dto.MyProfileResponse {
	return &dto.MyProfileResponse{
		Username:     p.Username,
		Email:        p.Email,
		DisplayName:  p.DisplayName,
		Country:      p.Country,
		Bio:          p.Bio,
		AvatarURL:    p.AvatarURL,
		Rating:       p.Rating,
		Provisional:  glicko2.IsProvisional(p.RatingDeviation),
		Ratings:      toModeRatings(p),
		SkillLevel:   p.SkillLevel,
		NeedsRating:  p.SkillLevel == nil,
		FriendsCount: p.FriendsCount,
		JoinedAt:     p.CreatedAt,
	}
}

func emptyToNil(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func isCountryCode(v string) bool {
	if len(v) != 2 {
		return false
	}
	for _, c := range v {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

func (h *ProfileService) GetModeStats(ctx context.Context, username string, mode string) (*dto.ModeStatsResponse, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if !isRatedMode(mode) {
		return nil, ErrInvalidMode
	}

	username = strings.ToLower(strings.TrimSpace(username))

	p, err := h.ProfileRepository.FindByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		slog.Error("find profile by username failed", "error", err)
		return nil, ErrInternal
	}

	if err := h.loadRatings(ctx, p); err != nil {
		return nil, err
	}

	stats, err := h.ProfileRepository.FindModeGameStats(ctx, p.UserID, mode)
	if err != nil {
		slog.Error("find mode game stats failed", "error", err)
		return nil, ErrInternal
	}

	games, err := h.ProfileRepository.FindRecentGamesByMode(ctx, p.UserID, mode, recentGamesLimit)
	if err != nil {
		slog.Error("find recent games failed", "error", err)
		return nil, ErrInternal
	}

	gamesPlayed := stats.Wins + stats.Losses + stats.Draws
	winPercent := 0
	winRate := 0.0
	if gamesPlayed > 0 {
		winPercent = stats.Wins * 100 / gamesPlayed
		winRate = math.Round((float64(stats.Wins)/float64(gamesPlayed))*10000) / 100
	}

	historyPoints, _ := h.ProfileRepository.FindRatingHistory(ctx, p.UserID, mode, 100)
	ratingHistory := make([]dto.RatingPointResponse, 0, len(historyPoints))
	for _, pt := range historyPoints {
		ratingHistory = append(ratingHistory, dto.RatingPointResponse{
			GameID:           pt.GameID,
			EndedAt:          pt.EndedAt,
			Rating:           pt.Rating,
			RatingChange:     pt.RatingChange,
			OpponentUsername: pt.OpponentUsername,
			Outcome:          pt.Outcome,
		})
	}

	recentGames := make([]dto.RecentGameResponse, 0, len(games))
	for _, g := range games {
		recentGames = append(recentGames, dto.RecentGameResponse{
			GameID:           g.GameID,
			OpponentUsername: g.OpponentUsername,
			Color:            g.Color,
			Outcome:          gameOutcome(g.Color, g.Result),
			EndReason:        g.EndReason,
			EndedAt:          g.EndedAt,
		})
	}

	r := findModeRating(p, mode)

	return &dto.ModeStatsResponse{
		Mode:            mode,
		Rating:          r.Rating,
		Provisional:     glicko2.IsProvisional(r.RatingDeviation),
		BestRating:      r.BestRating,
		BestRatingAt:    r.BestRatingAt,
		GamesPlayed:     gamesPlayed,
		Wins:            stats.Wins,
		Losses:          stats.Losses,
		Draws:           stats.Draws,
		WinPercent:      winPercent,
		WinRate:         winRate,
		AverageGameTime: math.Round(stats.AverageGameTimeSeconds*100) / 100,
		RatingHistory:   ratingHistory,
		RecentGames:     recentGames,
	}, nil
}

func (h *ProfileService) GetPlayerStats(ctx context.Context, username string, mode string) (*dto.PlayerStatsResponse, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != "" && mode != "all" && !isRatedMode(mode) {
		return nil, ErrInvalidMode
	}
	if mode == "all" {
		mode = ""
	}

	username = strings.ToLower(strings.TrimSpace(username))

	p, err := h.ProfileRepository.FindByUsername(ctx, username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		slog.Error("find profile by username failed", "error", err)
		return nil, ErrInternal
	}

	if err := h.loadRatings(ctx, p); err != nil {
		return nil, err
	}

	stats, err := h.ProfileRepository.FindModeGameStats(ctx, p.UserID, mode)
	if err != nil {
		slog.Error("find mode game stats failed", "error", err)
		return nil, ErrInternal
	}

	historyPoints, err := h.ProfileRepository.FindRatingHistory(ctx, p.UserID, mode, 100)
	if err != nil {
		slog.Error("find rating history failed", "error", err)
		return nil, ErrInternal
	}

	ratingHistory := make([]dto.RatingPointResponse, 0, len(historyPoints))
	for _, pt := range historyPoints {
		ratingHistory = append(ratingHistory, dto.RatingPointResponse{
			GameID:           pt.GameID,
			EndedAt:          pt.EndedAt,
			Rating:           pt.Rating,
			RatingChange:     pt.RatingChange,
			OpponentUsername: pt.OpponentUsername,
			Outcome:          pt.Outcome,
		})
	}

	gamesPlayed := stats.Wins + stats.Losses + stats.Draws
	winRate := 0.0
	if gamesPlayed > 0 {
		winRate = math.Round((float64(stats.Wins)/float64(gamesPlayed))*10000) / 100
	}

	currentRating := defaultRating
	if mode != "" {
		currentRating = findModeRating(p, mode).Rating
	} else if p.Rating != nil {
		currentRating = *p.Rating
	}

	displayMode := mode
	if displayMode == "" {
		displayMode = "all"
	}

	return &dto.PlayerStatsResponse{
		Username:        p.Username,
		Mode:            displayMode,
		Rating:          currentRating,
		GamesPlayed:     gamesPlayed,
		Wins:            stats.Wins,
		Losses:          stats.Losses,
		Draws:           stats.Draws,
		WinRate:         winRate,
		AverageGameTime: math.Round(stats.AverageGameTimeSeconds*100) / 100,
		RatingHistory:   ratingHistory,
	}, nil
}

func (h *ProfileService) loadRatings(ctx context.Context, p *model.Profile) error {
	ratings, err := h.ProfileRepository.FindModeRatings(ctx, p.UserID)
	if err != nil {
		slog.Error("find mode ratings failed", "error", err)
		return ErrInternal
	}

	p.Ratings = ratings
	return nil
}

func toModeRatings(p *model.Profile) map[string]dto.ModeRatingResponse {
	ratings := make(map[string]dto.ModeRatingResponse)

	for _, mode := range ratedModes {
		r := findModeRating(p, mode)
		provisional := false
		if r.GamesPlayed > 0 {
			provisional = glicko2.IsProvisional(r.RatingDeviation)
		}
		ratings[mode] = dto.ModeRatingResponse{
			Rating:      r.Rating,
			Provisional: provisional,
		}
	}

	return ratings
}

func findModeRating(p *model.Profile, mode string) model.ModeRating {
	for _, r := range p.Ratings {
		if r.Mode == mode {
			return r
		}
	}

	return model.ModeRating{
		Mode:            mode,
		Rating:          defaultRating,
		RatingDeviation: glicko2.DefaultRD,
	}
}

func gameOutcome(color string, result string) string {
	switch result {
	case game.ResultDraw:
		return "draw"
	case game.ResultWhiteWin:
		if color == "white" {
			return "win"
		}
		return "loss"
	case game.ResultBlackWin:
		if color == "black" {
			return "win"
		}
		return "loss"
	}
	return ""
}

func isRatedMode(mode string) bool {
	for _, m := range ratedModes {
		if m == mode {
			return true
		}
	}
	return false
}
