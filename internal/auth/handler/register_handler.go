package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type RegisterHandler struct {
	RegisterService RegisterService
}

func NewRegisterHandler(registerService RegisterService) *RegisterHandler {
	return &RegisterHandler{
		RegisterService: registerService,
	}
}

func (h *RegisterHandler) Register(c *gin.Context) {

	var req dto.RegisterData

	err := c.ShouldBindJSON(&req)

	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.RegisterService.Register(c.Request.Context(), req)

	if errors.Is(err, service.ErrInternal) {
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	if errors.Is(err, service.ErrUsernameTaken) ||
		errors.Is(err, service.ErrEmailTaken) {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}

	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "verification email sent", nil)
}

func (h *RegisterHandler) VerifyRegistration(c *gin.Context) {
	var req dto.VerifyRegistrationData

	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	accessToken, refreshToken, err := h.RegisterService.VerifyRegistration(c.Request.Context(), req)

	if errors.Is(err, service.ErrInternal) {
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	if errors.Is(err, service.ErrUsernameTaken) ||
		errors.Is(err, service.ErrEmailTaken) {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}

	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	SetAccessTokenCookie(c, accessToken)
	SetRefreshTokenCookie(c, refreshToken)

	response.Success(c, http.StatusCreated, "user registered successfully", dto.VerifyRegistrationResponse{
		NeedsRating: true,
	})
}

func (h *RegisterHandler) GuestLogin(c *gin.Context) {
	guestID, guestToken, err := h.RegisterService.GuestLogin(c.Request.Context())

	if errors.Is(err, service.ErrInternal) {
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	SetGuestTokenCookie(c, guestToken)

	response.Success(c, http.StatusOK, "guest login successful", dto.GuestLoginResponse{
		GuestID: guestID,
	})
}
