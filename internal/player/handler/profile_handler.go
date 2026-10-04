package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/service"
	"time"
)

type ProfileHandler struct {
	ProfileService *service.ProfileService
}

func NewProfileHandler(profileService *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		ProfileService: profileService,
	}
}

func (h *ProfileHandler) GetMyProfile(c *gin.Context) {
	isGuest := c.GetBool("is_guest")
	if isGuest {
		guestID := c.GetString("guest_id")
		c.JSON(http.StatusOK, dto.MyProfileResponse{
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	profile, err := h.ProfileService.GetMyProfile(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) UpdateMyProfile(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "user not authenticated",
		})
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)

	var req dto.UpdateProfileData
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	profile, err := h.ProfileService.UpdateMyProfile(c.Request.Context(), userID, req)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) GetPublicProfile(c *gin.Context) {
	profile, err := h.ProfileService.GetPublicProfile(c.Request.Context(), c.Param("username"))
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, profile)
}

func writeError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrInternal) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	if errors.Is(err, service.ErrUserNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error": err.Error(),
	})
}
