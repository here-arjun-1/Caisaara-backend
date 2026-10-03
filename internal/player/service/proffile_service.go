package service

import (
	"errors"
	"log"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/here-arjun-1/Caisaara-backend/internal/player/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/model"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/repository"
	"github.com/jackc/pgx/v5"
)

type ProfileService struct {
	ProfileRepository *repository.ProfileRepository
}

func NewProfileService(profileRepository *repository.ProfileRepository) *ProfileService {
	return &ProfileService{
		ProfileRepository: profileRepository,
	}
}

func (h *ProfileService) GetMyProfile(userID int64) (*dto.MyProfileResponse, error) {
	p, err := h.ProfileRepository.FindByUserID(userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		log.Printf("find profile by id failed: %v", err)
		return nil, ErrInternal
	}

	return toMyProfileResponse(p), nil
}

func (h *ProfileService) GetPublicProfile(username string) (*dto.PublicProfileResponse, error) {
	username = strings.ToLower(strings.TrimSpace(username))

	p, err := h.ProfileRepository.FindByUsername(username)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		log.Printf("find profile by username failed: %v", err)
		return nil, ErrInternal
	}

	return &dto.PublicProfileResponse{
		Username:    p.Username,
		DisplayName: p.DisplayName,
		Country:     p.Country,
		Bio:         p.Bio,
		AvatarURL:   p.AvatarURL,
		Rating:      p.Rating,
		JoinedAt:    p.CreatedAt,
	}, nil
}

func (h *ProfileService) UpdateMyProfile(userID int64, req dto.UpdateProfileData) (*dto.MyProfileResponse, error) {
	p, err := h.ProfileRepository.FindByUserID(userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		log.Printf("find profile by id failed: %v", err)
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

	if err := h.ProfileRepository.SaveProfile(p); err != nil {
		log.Printf("save profile failed: %v", err)
		return nil, ErrInternal
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
