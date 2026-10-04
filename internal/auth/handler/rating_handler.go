package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type RatingHandler struct {
	RatingService RatingService
}

func NewRatingHandler(ratingService RatingService) *RatingHandler {
	return &RatingHandler{
		RatingService: ratingService,
	}
}

func (h *RatingHandler) SetRating(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req dto.SetRatingData
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	rating, err := h.RatingService.SetInitialRating(c.Request.Context(), userID, req.Level)

	if errors.Is(err, service.ErrInternal) {
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	if errors.Is(err, service.ErrRatingAlreadySet) {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}

	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "rating set successfully", dto.SetRatingResponse{
		Level:  req.Level,
		Rating: rating,
	})
}
