package handler

import (
	"errors"
	"net/http"

	"github.com/h0ugetsu/realworld-api/internal/httputil"
	"github.com/h0ugetsu/realworld-api/internal/httputil/httperror"
	"github.com/h0ugetsu/realworld-api/internal/middleware"
	"github.com/h0ugetsu/realworld-api/internal/repository"
	"github.com/h0ugetsu/realworld-api/internal/service"
	"github.com/labstack/echo/v5"
)

type UserHandler struct {
	userService service.UserService
	authService service.AuthService
}

func NewUserHandler(userService service.UserService, authService service.AuthService) *UserHandler {
	return &UserHandler{
		userService: userService,
		authService: authService,
	}
}

type registerUserReq struct {
	User struct {
		Username string `json:"username" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8,max=66"`
	} `json:"user"`
}

type registerUserRes struct {
	User struct {
		Username string  `json:"username"`
		Email    string  `json:"email"`
		Token    string  `json:"token"`
		Bio      *string `json:"bio"`
		Image    *string `json:"image"`
	} `json:"user"`
}

func (h *UserHandler) Create(c *echo.Context) error {
	var req registerUserReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	user, err := h.userService.CreateUser(c.Request().Context(), repository.CreateUserParams{
		Username: req.User.Username,
		Email:    req.User.Email,
		Password: req.User.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailAlreadyExists):
			return httperror.New(http.StatusConflict, map[string]any{
				"errors": map[string][]string{"email": {"has already been taken"}},
			})
		case errors.Is(err, service.ErrUsernameAlreadyExists):
			return httperror.New(http.StatusConflict, map[string]any{
				"errors": map[string][]string{"username": {"has already been taken"}},
			})
		default:
			return err
		}
	}

	token, err := h.authService.GenerateToken(user.ID)
	if err != nil {
		return err
	}

	var res registerUserRes
	res.User.Username = user.Username
	res.User.Email = user.Email
	res.User.Token = token
	res.User.Bio = user.Bio
	res.User.Image = user.Image

	return c.JSON(http.StatusCreated, res)
}

type loginUserReq struct {
	User struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8,max=66"`
	} `json:"user"`
}

type loginUserRes struct {
	User struct {
		Username string  `json:"username"`
		Email    string  `json:"email"`
		Token    string  `json:"token"`
		Bio      *string `json:"bio"`
		Image    *string `json:"image"`
	} `json:"user"`
}

func (h *UserHandler) Login(c *echo.Context) error {
	var req loginUserReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	user, err := h.userService.AuthenticateUser(c.Request().Context(), req.User.Email, req.User.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return httperror.New(http.StatusUnauthorized, map[string]any{
				"errors": map[string][]string{"credentials": {"invalid"}},
			})
		}
		return err
	}

	token, err := h.authService.GenerateToken(user.ID)
	if err != nil {
		return err
	}

	var res loginUserRes
	res.User.Username = user.Username
	res.User.Email = user.Email
	res.User.Token = token
	res.User.Bio = user.Bio
	res.User.Image = user.Image

	return c.JSON(http.StatusOK, res)
}

type currentUserRes struct {
	User struct {
		Username string  `json:"username"`
		Email    string  `json:"email"`
		Token    string  `json:"token"`
		Bio      *string `json:"bio"`
		Image    *string `json:"image"`
	} `json:"user"`
}

func (h *UserHandler) CurrentUser(c *echo.Context) error {
	userID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	user, err := h.userService.GetCurrentUser(c.Request().Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"user": {"not found"}},
			})
		}
		return err
	}

	token, err := h.authService.GenerateToken(user.ID)
	if err != nil {
		return err
	}

	var res currentUserRes
	res.User.Username = user.Username
	res.User.Email = user.Email
	res.User.Token = token
	res.User.Bio = user.Bio
	res.User.Image = user.Image

	return c.JSON(http.StatusOK, res)
}

type updateUserReq struct {
	User struct {
		Username httputil.Field[string] `json:"username"`
		Email    httputil.Field[string] `json:"email"`
		Password httputil.Field[string] `json:"password"`
		Bio      httputil.Field[string] `json:"bio"`
		Image    httputil.Field[string] `json:"image"`
	} `json:"user"`
}

type updateUserRes struct {
	User struct {
		Username string  `json:"username"`
		Email    string  `json:"email"`
		Token    string  `json:"token"`
		Bio      *string `json:"bio"`
		Image    *string `json:"image"`
	} `json:"user"`
}

func (h *UserHandler) Update(c *echo.Context) error {
	userID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	var req updateUserReq
	if err := c.Bind(&req); err != nil {
		return err
	}

	errs := map[string][]string{}
	params := repository.UpdateUserParams{}

	if f := req.User.Username; f.Present {
		switch {
		case f.Null || f.Value == "":
			errs["username"] = append(errs["username"], "can't be blank")
		default:
			v := f.Value
			params.Username = &v
		}
	}

	if f := req.User.Email; f.Present {
		switch {
		case f.Null || f.Value == "":
			errs["email"] = append(errs["email"], "can't be blank")
		case c.Validate(&struct {
			Email string `validate:"email"`
		}{f.Value}) != nil:
			errs["email"] = append(errs["email"], "is invalid")
		default:
			v := f.Value
			params.Email = &v
		}
	}

	if f := req.User.Password; f.Present {
		switch {
		case f.Null || f.Value == "":
			errs["password"] = append(errs["password"], "can't be blank")
		case len(f.Value) < 8 || len(f.Value) > 66:
			errs["password"] = append(errs["password"], "is invalid")
		default:
			v := f.Value
			params.Password = &v
		}
	}

	if f := req.User.Bio; f.Present {
		params.BioSet = true
		if !f.Null && f.Value != "" {
			v := f.Value
			params.Bio = &v
		}
	}

	if f := req.User.Image; f.Present {
		params.ImageSet = true
		if !f.Null && f.Value != "" {
			v := f.Value
			params.Image = &v
		}
	}

	if len(errs) > 0 {
		return httperror.New(http.StatusUnprocessableEntity, map[string]any{"errors": errs})
	}

	user, err := h.userService.UpdateUser(c.Request().Context(), userID, params)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailAlreadyExists):
			return httperror.New(http.StatusUnprocessableEntity, map[string]any{
				"errors": map[string][]string{"email": {"has already been taken"}},
			})
		case errors.Is(err, service.ErrUsernameAlreadyExists):
			return httperror.New(http.StatusUnprocessableEntity, map[string]any{
				"errors": map[string][]string{"username": {"has already been taken"}},
			})
		case errors.Is(err, service.ErrUserNotFound):
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"user": {"not found"}},
			})
		default:
			return err
		}
	}

	token, err := h.authService.GenerateToken(user.ID)
	if err != nil {
		return err
	}

	var res updateUserRes
	res.User.Username = user.Username
	res.User.Email = user.Email
	res.User.Token = token
	res.User.Bio = user.Bio
	res.User.Image = user.Image

	return c.JSON(http.StatusOK, res)
}

type profileRes struct {
	Profile struct {
		Username  string  `json:"username"`
		Bio       *string `json:"bio"`
		Image     *string `json:"image"`
		Following bool    `json:"following"`
	} `json:"profile"`
}

func (h *UserHandler) Profile(c *echo.Context) error {
	username := c.Param("username")

	var currentUserID *int64
	if userID, ok := c.Get(middleware.UserIDContextKey).(int64); ok {
		currentUserID = &userID
	}

	profile, err := h.userService.GetProfile(c.Request().Context(), username, currentUserID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"profile": {"not found"}},
			})
		}
		return err
	}

	var res profileRes
	res.Profile.Username = profile.Username
	res.Profile.Bio = profile.Bio
	res.Profile.Image = profile.Image
	res.Profile.Following = profile.Following

	return c.JSON(http.StatusOK, res)
}

func (h *UserHandler) Follow(c *echo.Context) error {
	userID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	username := c.Param("username")

	profile, err := h.userService.FollowUser(c.Request().Context(), userID, username)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"profile": {"not found"}},
			})
		case errors.Is(err, service.ErrCannotFollowSelf):
			return httperror.New(http.StatusUnprocessableEntity, map[string]any{
				"errors": map[string][]string{"username": {"cannot follow yourself"}},
			})
		default:
			return err
		}
	}

	var res profileRes
	res.Profile.Username = profile.Username
	res.Profile.Bio = profile.Bio
	res.Profile.Image = profile.Image
	res.Profile.Following = profile.Following

	return c.JSON(http.StatusOK, res)
}

func (h *UserHandler) Unfollow(c *echo.Context) error {
	userID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	username := c.Param("username")

	profile, err := h.userService.UnfollowUser(c.Request().Context(), userID, username)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"profile": {"not found"}},
			})
		}
		return err
	}

	var res profileRes
	res.Profile.Username = profile.Username
	res.Profile.Bio = profile.Bio
	res.Profile.Image = profile.Image
	res.Profile.Following = profile.Following

	return c.JSON(http.StatusOK, res)
}
