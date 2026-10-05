package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/community/dto"
	"github.com/here-arjun-1/Caisaara-backend/internal/response"
)

type CommentService interface {
	CreateComment(ctx context.Context, userID, postID int64, req dto.CreateCommentData) (*dto.CommentResponse, error)
	ListComments(ctx context.Context, postID, afterID int64, limit int) (*dto.CommentListResponse, error)
	DeleteComment(ctx context.Context, userID, commentID int64) error
}

type CommentHandler struct {
	CommentService CommentService
}

func NewCommentHandler(commentService CommentService) *CommentHandler {
	return &CommentHandler{
		CommentService: commentService,
	}
}

func (h *CommentHandler) CreateComment(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	postID, ok := idParam(c, "id")
	if !ok {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8*1024)

	var req dto.CreateCommentData
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	comment, err := h.CommentService.CreateComment(c.Request.Context(), userID, postID, req)
	if err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "comment added successfully", comment)
}

func (h *CommentHandler) ListComments(c *gin.Context) {
	postID, ok := idParam(c, "id")
	if !ok {
		return
	}
	afterID, ok := queryInt(c, "after_id")
	if !ok {
		return
	}
	limit, ok := queryInt(c, "limit")
	if !ok {
		return
	}

	comments, err := h.CommentService.ListComments(c.Request.Context(), postID, afterID, int(limit))
	if err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "comments fetched successfully", comments)
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
	userID := c.GetInt64("user_id")
	if userID <= 0 {
		response.Error(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	commentID, ok := idParam(c, "id")
	if !ok {
		return
	}

	if err := h.CommentService.DeleteComment(c.Request.Context(), userID, commentID); err != nil {
		writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "comment deleted successfully", nil)
}
