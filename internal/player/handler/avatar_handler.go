package handler

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type AvatarService interface {
	UploadPhoto(ctx context.Context, userID int64, data []byte) (string, error)
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
