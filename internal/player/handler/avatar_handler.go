package handler

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/player/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type AvatarService interface {
	UploadPhoto(ctx context.Context, userID int64, data []byte) (string, error)
	ListDefaultAvatars() []dto.DefaultAvatarResponse
	ChooseDefault(ctx context.Context, userID int64, avatarID string) (string, error)
	RemoveAvatar(ctx context.Context, userID int64) error
}

type AvatarHandler struct {
	AvatarService AvatarService
}

func NewAvatarHandler(avatarService AvatarService) *AvatarHandler {
	return &AvatarHandler{
		AvatarService: avatarService,
	}
}

func (h *AvatarHandler) UploadPhoto(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	fileHeader, err := c.FormFile("photo")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "photo file is required")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, "could not read photo")
		return
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "could not read photo")
		return
	}

	avatarURL, err := h.AvatarService.UploadPhoto(c.Request.Context(), userID, data)
	if err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "photo uploaded successfully", gin.H{"avatar_url": avatarURL})
}

func (h *AvatarHandler) ListDefaultAvatars(c *gin.Context) {
	response.Success(c, http.StatusOK, "avatars fetched successfully", gin.H{"avatars": h.AvatarService.ListDefaultAvatars()})
}

func (h *AvatarHandler) ChooseDefault(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)

	var req dto.ChooseAvatarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	avatarURL, err := h.AvatarService.ChooseDefault(c.Request.Context(), userID, req.AvatarID)
	if err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "avatar updated successfully", gin.H{"avatar_url": avatarURL})
}

func (h *AvatarHandler) RemoveAvatar(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	if err := h.AvatarService.RemoveAvatar(c.Request.Context(), userID); err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "avatar removed successfully", nil)
}
