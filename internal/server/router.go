package server

import (
	"errors"
	"net/http"

	"github.com/h0ugetsu/realworld-api/internal/config"
	"github.com/h0ugetsu/realworld-api/internal/handler"
	"github.com/h0ugetsu/realworld-api/internal/httputil/httperror"
	authmw "github.com/h0ugetsu/realworld-api/internal/middleware"
	"github.com/h0ugetsu/realworld-api/internal/service"
	"github.com/h0ugetsu/realworld-api/internal/validator"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func NewRouter(userHandler *handler.UserHandler, articleHandler *handler.ArticleHandler, authService service.AuthService, cfg *config.Config) *echo.Echo {
	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Validator = validator.NewValidator()
	e.HTTPErrorHandler = customHTTPErrorHandler

	api := e.Group("/api")

	users := api.Group("/users")
	{
		users.POST("", userHandler.Create)
		users.POST("/login", userHandler.Login)
	}

	user := api.Group("/user")
	{
		user.GET("", userHandler.CurrentUser, authmw.AuthMiddleware(authService))
		user.PUT("", userHandler.Update, authmw.AuthMiddleware(authService))
	}

	profiles := api.Group("/profiles")
	{
		profiles.GET("/:username", userHandler.Profile, authmw.OptionalAuthMiddleware(authService))
		profiles.POST("/:username/follow", userHandler.Follow, authmw.AuthMiddleware(authService))
		profiles.DELETE("/:username/follow", userHandler.Unfollow, authmw.AuthMiddleware(authService))
	}

	articles := api.Group("/articles")
	{
		articles.POST("", articleHandler.Create, authmw.AuthMiddleware(authService))
		articles.GET("", articleHandler.List, authmw.OptionalAuthMiddleware(authService))
		articles.GET("/feed", articleHandler.Feed, authmw.AuthMiddleware(authService))
		articles.GET("/:slug", articleHandler.Show, authmw.OptionalAuthMiddleware(authService))
		articles.PUT("/:slug", articleHandler.Update, authmw.AuthMiddleware(authService))
		articles.DELETE("/:slug", articleHandler.Delete, authmw.AuthMiddleware(authService))
		articles.POST("/:slug/favorite", articleHandler.Favorite, authmw.AuthMiddleware(authService))
		articles.DELETE("/:slug/favorite", articleHandler.Unfavorite, authmw.AuthMiddleware(authService))
		articles.POST("/:slug/comments", articleHandler.CreateComment, authmw.AuthMiddleware(authService))
		articles.GET("/:slug/comments", articleHandler.ListComments, authmw.OptionalAuthMiddleware(authService))
		articles.DELETE("/:slug/comments/:id", articleHandler.DeleteComment, authmw.AuthMiddleware(authService))
	}

	tags := api.Group("/tags")
	{
		tags.GET("", articleHandler.Tags)
	}

	return e
}

func customHTTPErrorHandler(c *echo.Context, err error) {
	if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil && resp.Committed {
		return
	}

	// 仕様が定めているドメインエラー: RealWorldの{"errors": {...}}形式
	var he *httperror.Error
	if errors.As(err, &he) {
		if writeErr := c.JSON(he.Status, he.Body); writeErr != nil {
			c.Logger().Error("failed to write error response", "error", errors.Join(err, writeErr))
		}
		return
	}

	// 仕様の範囲外(Echo自身のルーティングエラー・想定外のエラー):
	// 無理にRealWorld形式へ寄せず、素直な{"message": "..."}にする
	code := http.StatusInternalServerError
	message := http.StatusText(code)
	var sc echo.HTTPStatusCoder
	if errors.As(err, &sc) {
		if tmp := sc.StatusCode(); tmp != 0 {
			code = tmp
			message = err.Error()
		}
	}

	var writeErr error
	if c.Request().Method == http.MethodHead {
		writeErr = c.NoContent(code)
	} else {
		writeErr = c.JSON(code, map[string]string{"message": message})
	}
	if writeErr != nil {
		c.Logger().Error("failed to write error response", "error", errors.Join(err, writeErr))
	}
}
