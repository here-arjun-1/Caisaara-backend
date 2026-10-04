package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/service"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type ProfileService interface {
	GetMyProfile(ctx context.Context, userID int64) (*dto.MyProfileResponse, error)
	GetPublicProfile(ctx context.Context, username string) (*dto.PublicProfileResponse, error)
	UpdateMyProfile(ctx context.Context, userID int64, req dto.UpdateProfileData) (*dto.MyProfileResponse, error)
}

type ProfileHandler struct {
	ProfileService ProfileService
}

func NewProfileHandler(profileService ProfileService) *ProfileHandler {
	return &ProfileHandler{
		ProfileService: profileService,
	}
}

func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	isGuest := c.GetBool("is_guest")
	if isGuest {
		guestID := c.GetString("guest_id")
		response.Success(c, http.StatusOK, "profile fetched successfully", dto.MyProfileResponse{
			Username:    "Guest_" + guestID[:8],
			DisplayName: nil,
			Country:     nil,
			Bio:         nil,
			AvatarURL:   nil,
			Rating:      nil,
			SkillLevel:  nil,
			NeedsRating: true,
			JoinedAt:    time.Now(),
		})
		return
	}

	userID := c.GetInt64("user_id")
	if userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	profile, err := h.ProfileService.GetMyProfile(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "profile fetched successfully", profile)
}

func (h *ProfileHandler) UpdateMyProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)

	var req dto.UpdateProfileData
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	profile, err := h.ProfileService.UpdateMyProfile(c.Request.Context(), userID, req)
	if err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "profile updated successfully", profile)
}

func (h *ProfileHandler) GetPublicProfile(c *gin.Context) {
	profile, err := h.ProfileService.GetPublicProfile(c.Request.Context(), c.Param("username"))
	if err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "profile fetched successfully", profile)
}

func writeError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrInternal) {
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	if errors.Is(err, service.ErrUserNotFound) {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}

	response.Error(c, http.StatusBadRequest, err.Error())
}
