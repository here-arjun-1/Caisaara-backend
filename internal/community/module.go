package community

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/here-arjun-1/Caisaara-backend/internal/auth/middleware"
	"github.com/here-arjun-1/Caisaara-backend/internal/community/handler"
	"github.com/here-arjun-1/Caisaara-backend/internal/community/repository"
	"github.com/here-arjun-1/Caisaara-backend/internal/community/service"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Module struct {
	postHandler    *handler.PostHandler
	commentHandler *handler.CommentHandler
}

func NewModule(db *pgxpool.Pool) *Module {
	postRepository := repository.NewPostRepository(db)
	commentRepository := repository.NewCommentRepository(db)

	postService := service.NewPostService(postRepository)
	commentService := service.NewCommentService(commentRepository, postRepository)

	return &Module{
		postHandler:    handler.NewPostHandler(postService),
		commentHandler: handler.NewCommentHandler(commentService),
	}
}

func (m *Module) RegisterRoutes(public *gin.RouterGroup, protected *gin.RouterGroup) {
	readLimiter := middleware.NewTokenBucketLimiter(60, time.Minute)
	createPostLimiter := middleware.NewTokenBucketLimiter(5, time.Minute)
	commentLimiter := middleware.NewTokenBucketLimiter(20, time.Minute)
	reactionLimiter := middleware.NewTokenBucketLimiter(30, time.Minute)
	deleteLimiter := middleware.NewTokenBucketLimiter(20, time.Minute)

	public.GET("/posts", readLimiter.Limit, m.postHandler.GetFeed)
	public.GET("/posts/:id", readLimiter.Limit, m.postHandler.GetPost)
	public.GET("/posts/:id/comments", readLimiter.Limit, m.commentHandler.ListComments)

	registered := protected.Group("")
	registered.Use(middleware.RequireRegisteredUser())
	{
		registered.POST("/posts", createPostLimiter.Limit, m.postHandler.CreatePost)
		registered.DELETE("/posts/:id", deleteLimiter.Limit, m.postHandler.DeletePost)

		registered.PUT("/posts/:id/reaction", reactionLimiter.Limit, m.postHandler.SetReaction)
		registered.DELETE("/posts/:id/reaction", reactionLimiter.Limit, m.postHandler.DeleteReaction)

		registered.POST("/posts/:id/comments", commentLimiter.Limit, m.commentHandler.CreateComment)
		registered.DELETE("/comments/:id", deleteLimiter.Limit, m.commentHandler.DeleteComment)
	}
}
