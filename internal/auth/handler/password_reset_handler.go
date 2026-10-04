package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/service"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type PasswordResetHandler struct {
	PasswordResetService PasswordResetService
}

func NewPasswordResetHandler(passwordResetService PasswordResetService) *PasswordResetHandler {
	return &PasswordResetHandler{
		PasswordResetService: passwordResetService,
	}
}

func (h *PasswordResetHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.PasswordResetService.ForgotPassword(c.Request.Context(), req)
	if errors.Is(err, service.ErrInternal) {
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}

	response.Success(c, http.StatusOK, "if an account with that email exists, a verification code has been sent", nil)
}

func (h *PasswordResetHandler) VerifyCode(c *gin.Context) {
	var req dto.VerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	resetToken, err := h.PasswordResetService.VerifyCode(c.Request.Context(), req)
	if errors.Is(err, service.ErrInternal) {
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "code verified successfully", dto.VerifyCodeResponse{
		ResetToken: resetToken,
	})
}

func (h *PasswordResetHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	err := h.PasswordResetService.ResetPassword(c.Request.Context(), req)
	if errors.Is(err, service.ErrInternal) {
		response.Error(c, http.StatusInternalServerError, "internal server error")
		return
	}
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "password updated successfully", nil)
}
