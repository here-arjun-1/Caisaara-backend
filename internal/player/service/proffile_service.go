package service

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
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
)

var ratedModes = []string{
	game.ModeBullet,
	game.ModeBlitz,
	game.ModeRapid,
	game.ModeDaily,
}

type ProfileRepository interface {
	FindByUserID(ctx context.Context, userID int64) (*model.Profile, error)
	FindByUsername(ctx context.Context, username string) (*model.Profile, error)
	SaveProfile(ctx context.Context, p *model.Profile) error
	FindModeRatings(ctx context.Context, userID int64) ([]model.ModeRating, error)
	FindModeGameStats(ctx context.Context, userID int64, mode string) (*model.ModeGameStats, error)
	FindRecentGamesByMode(ctx context.Context, userID int64, mode string, limit int) ([]model.RecentGame, error)
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

	if req.AvatarURL != nil {
		v := strings.TrimSpace(*req.AvatarURL)
		if v != "" && !isHTTPSURL(v) {
			return nil, ErrInvalidAvatarURL
		}
		p.AvatarURL = emptyToNil(v)
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
		Username:    p.Username,
		Email:       p.Email,
		DisplayName: p.DisplayName,
		Country:     p.Country,
		Bio:         p.Bio,
		AvatarURL:   p.AvatarURL,
		Rating:      p.Rating,
		Provisional: glicko2.IsProvisional(p.RatingDeviation),
		Ratings:     toModeRatings(p),
		SkillLevel:  p.SkillLevel,
		NeedsRating: p.SkillLevel == nil,
		JoinedAt:    p.CreatedAt,
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

func isHTTPSURL(v string) bool {
	if len(v) > 500 {
		return false
	}
	u, err := url.Parse(v)
	return err == nil && u.Scheme == "https" && u.Host != ""
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
	if gamesPlayed > 0 {
		winPercent = stats.Wins * 100 / gamesPlayed
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
		Mode:         mode,
		Rating:       r.Rating,
		Provisional:  glicko2.IsProvisional(r.RatingDeviation),
		BestRating:   r.BestRating,
		BestRatingAt: r.BestRatingAt,
		GamesPlayed:  gamesPlayed,
		Wins:         stats.Wins,
		Losses:       stats.Losses,
		Draws:        stats.Draws,
		WinPercent:   winPercent,
		RecentGames:  recentGames,
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
		ratings[mode] = dto.ModeRatingResponse{
			Rating:      r.Rating,
			Provisional: glicko2.IsProvisional(r.RatingDeviation),
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

	rating := defaultRating
	if p.Rating != nil {
		rating = *p.Rating
	}

	return model.ModeRating{
		Mode:            mode,
		Rating:          rating,
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
