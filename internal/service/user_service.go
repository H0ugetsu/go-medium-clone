package service

import (
	"context"
	"errors"

	"github.com/h0ugetsu/realworld-api/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound          = errors.New("user not found")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrUsernameAlreadyExists = errors.New("username already exists")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrCannotFollowSelf      = errors.New("cannot follow yourself")
)

type UserService interface {
	CreateUser(ctx context.Context, params repository.CreateUserParams) (*repository.CreateUserRow, error)
	AuthenticateUser(ctx context.Context, email, password string) (*repository.User, error)
	GetCurrentUser(ctx context.Context, userID int64) (*repository.FindByIDRow, error)
	UpdateUser(ctx context.Context, userID int64, params repository.UpdateUserParams) (*repository.UpdateUserRow, error)
	GetProfile(ctx context.Context, username string, currentUserID *int64) (*repository.FindProfileByUsernameRow, error)
	FollowUser(ctx context.Context, followerID int64, username string) (*repository.FindProfileByUsernameRow, error)
	UnfollowUser(ctx context.Context, followerID int64, username string) (*repository.FindProfileByUsernameRow, error)
}

type userService struct {
	repo repository.Querier
}

func NewUserService(repo repository.Querier) UserService {
	return &userService{
		repo: repo,
	}
}

func (s *userService) CreateUser(ctx context.Context, params repository.CreateUserParams) (*repository.CreateUserRow, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	params.Password = string(hashedPassword)

	user, err := s.repo.CreateUser(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_username_key":
				return nil, ErrUsernameAlreadyExists
			default:
				return nil, ErrEmailAlreadyExists
			}
		}
		return nil, err
	}

	return user, nil
}

func (s *userService) AuthenticateUser(ctx context.Context, email, password string) (*repository.User, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (s *userService) UpdateUser(ctx context.Context, userID int64, params repository.UpdateUserParams) (*repository.UpdateUserRow, error) {
	var hashedPassword *string
	if params.Password != nil {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*params.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		hashedStr := string(hashed)
		hashedPassword = &hashedStr
	}

	user, err := s.repo.UpdateUser(ctx, repository.UpdateUserParams{
		ID:       userID,
		Username: params.Username,
		Email:    params.Email,
		Password: hashedPassword,
		Bio:      params.Bio,
		BioSet:   params.BioSet,
		Image:    params.Image,
		ImageSet: params.ImageSet,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_username_key":
				return nil, ErrUsernameAlreadyExists
			default:
				return nil, ErrEmailAlreadyExists
			}
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func (s *userService) GetCurrentUser(ctx context.Context, userID int64) (*repository.FindByIDRow, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return user, nil
}

func (s *userService) GetProfile(ctx context.Context, username string, currentUserID *int64) (*repository.FindProfileByUsernameRow, error) {
	profile, err := s.repo.FindProfileByUsername(ctx, repository.FindProfileByUsernameParams{
		Username:      username,
		CurrentUserID: currentUserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return profile, nil
}

func (s *userService) FollowUser(ctx context.Context, followerID int64, username string) (*repository.FindProfileByUsernameRow, error) {
	followeeID, err := s.repo.FindIDByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if followerID == followeeID {
		return nil, ErrCannotFollowSelf
	}

	if err := s.repo.Follow(ctx, repository.FollowParams{
		FollowerID: followerID,
		FolloweeID: followeeID,
	}); err != nil {
		return nil, err
	}

	return s.GetProfile(ctx, username, &followerID)
}

func (s *userService) UnfollowUser(ctx context.Context, followerID int64, username string) (*repository.FindProfileByUsernameRow, error) {
	followeeID, err := s.repo.FindIDByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if err := s.repo.Unfollow(ctx, repository.UnfollowParams{
		FollowerID: followerID,
		FolloweeID: followeeID,
	}); err != nil {
		return nil, err
	}

	return s.GetProfile(ctx, username, &followerID)
}
