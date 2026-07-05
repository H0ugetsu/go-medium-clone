package middleware

import (
	"net/http"
	"strings"

	"github.com/h0ugetsu/realworld-api/internal/httputil/httperror"
	"github.com/h0ugetsu/realworld-api/internal/service"
	"github.com/labstack/echo/v5"
)

const UserIDContextKey = "userID"

func AuthMiddleware(authService service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			tokenString := c.Request().Header.Get("Authorization")
			if tokenString == "" {
				return httperror.New(http.StatusUnauthorized, map[string]any{
					"errors": map[string][]string{"token": {"is missing"}},
				})
			}

			if !strings.HasPrefix(strings.ToLower(tokenString), "token ") {
				return httperror.New(http.StatusUnauthorized, map[string]any{
					"errors": map[string][]string{"token": {"is missing"}},
				})
			}
			tokenString = tokenString[6:]

			userID, err := authService.VerifyToken(tokenString)
			if err != nil {
				return httperror.New(http.StatusUnauthorized, map[string]any{
					"errors": map[string][]string{"token": {"is invalid"}},
				})
			}

			c.Set(UserIDContextKey, userID)
			return next(c)
		}
	}
}

// OptionalAuthMiddleware は、トークンが無い/不正でもエラーにせず未認証として次に進める。
// トークンが有効な場合のみ UserIDContextKey にユーザーIDをセットする。
func OptionalAuthMiddleware(authService service.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			tokenString := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(strings.ToLower(tokenString), "token ") {
				return next(c)
			}
			tokenString = tokenString[6:]

			userID, err := authService.VerifyToken(tokenString)
			if err != nil {
				return next(c)
			}

			c.Set(UserIDContextKey, userID)
			return next(c)
		}
	}
}
