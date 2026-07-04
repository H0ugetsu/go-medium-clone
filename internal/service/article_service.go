package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"

	"github.com/h0ugetsu/realworld-api/internal/repository"
	"github.com/jackc/pgx/v5"
)

var (
	ErrArticleNotFound = errors.New("article not found")
	ErrCommentNotFound = errors.New("comment not found")
	ErrForbidden       = errors.New("forbidden")
)

type ArticleService interface {
	CreateArticle(ctx context.Context, authorID int64, params repository.CreateArticleParams, tagList []string) (*repository.GetArticleBySlugRow, error)
	GetArticle(ctx context.Context, slug string, currentUserID *int64) (*repository.GetArticleBySlugRow, error)
	ListArticles(ctx context.Context, params repository.ListArticlesParams, currentUserID *int64) ([]*repository.ListArticlesRow, int64, error)
	UpdateArticle(ctx context.Context, userID int64, slug string, params repository.UpdateArticleParams, tagList *[]string) (*repository.GetArticleBySlugRow, error)
	DeleteArticle(ctx context.Context, userID int64, slug string) error
	FavoriteArticle(ctx context.Context, userID int64, slug string) (*repository.GetArticleBySlugRow, error)
	UnfavoriteArticle(ctx context.Context, userID int64, slug string) (*repository.GetArticleBySlugRow, error)
	CreateComment(ctx context.Context, articleSlug string, authorID int64, body string) (*repository.GetCommentByIDRow, error)
	ListComments(ctx context.Context, articleSlug string, currentUserID *int64) ([]*repository.ListCommentsByArticleIDRow, error)
	DeleteComment(ctx context.Context, userID int64, articleSlug string, commentID int64) error
	ListFeed(ctx context.Context, currentUserID int64, limit, offset int32) ([]*repository.ListFeedArticlesRow, int64, error)
	ListTags(ctx context.Context) ([]string, error)
}

type articleService struct {
	repo repository.Querier
}

func NewArticleService(repo repository.Querier) ArticleService {
	return &articleService{
		repo: repo,
	}
}

var slugInvalidCharsRe = regexp.MustCompile(`[^\p{L}\p{N}]+`)

func generateSlug(title string) string {
	slug := slugInvalidCharsRe.ReplaceAllString(title, "-")
	slug = strings.ToLower(strings.Trim(slug, "-"))

	suffix := randomSlugSuffix()
	if slug == "" {
		return suffix
	}
	return slug + "-" + suffix
}

func randomSlugSuffix() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *articleService) CreateArticle(ctx context.Context, authorID int64, params repository.CreateArticleParams, tagList []string) (*repository.GetArticleBySlugRow, error) {
	params.Slug = generateSlug(params.Title)
	params.AuthorID = authorID

	article, err := s.repo.CreateArticle(ctx, params)
	if err != nil {
		return nil, err
	}

	for _, tagName := range tagList {
		tagID, err := s.repo.UpsertTag(ctx, tagName)
		if err != nil {
			return nil, err
		}
		if err := s.repo.AddArticleTag(ctx, repository.AddArticleTagParams{
			ArticleID: article.ID,
			TagID:     tagID,
		}); err != nil {
			return nil, err
		}
	}

	return s.GetArticle(ctx, article.Slug, &authorID)
}

func (s *articleService) GetArticle(ctx context.Context, slug string, currentUserID *int64) (*repository.GetArticleBySlugRow, error) {
	article, err := s.repo.GetArticleBySlug(ctx, repository.GetArticleBySlugParams{
		Slug:          slug,
		CurrentUserID: currentUserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	return article, nil
}

func (s *articleService) ListArticles(ctx context.Context, params repository.ListArticlesParams, currentUserID *int64) ([]*repository.ListArticlesRow, int64, error) {
	params.CurrentUserID = currentUserID

	articles, err := s.repo.ListArticles(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	if len(articles) == 0 {
		return articles, 0, nil
	}

	return articles, articles[0].TotalCount, nil
}

func (s *articleService) authorizeArticleOwner(ctx context.Context, userID int64, slug string) (*repository.FindArticleRefBySlugRow, error) {
	ref, err := s.repo.FindArticleRefBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	if ref.AuthorID != userID {
		return nil, ErrForbidden
	}

	return ref, nil
}

func (s *articleService) UpdateArticle(ctx context.Context, userID int64, slug string, params repository.UpdateArticleParams, tagList *[]string) (*repository.GetArticleBySlugRow, error) {
	ref, err := s.authorizeArticleOwner(ctx, userID, slug)
	if err != nil {
		return nil, err
	}

	params.Slug = slug

	article, err := s.repo.UpdateArticle(ctx, params)
	if err != nil {
		return nil, err
	}

	if tagList != nil {
		if err := s.repo.ClearArticleTags(ctx, ref.ID); err != nil {
			return nil, err
		}
		for _, tagName := range *tagList {
			tagID, err := s.repo.UpsertTag(ctx, tagName)
			if err != nil {
				return nil, err
			}
			if err := s.repo.AddArticleTag(ctx, repository.AddArticleTagParams{
				ArticleID: ref.ID,
				TagID:     tagID,
			}); err != nil {
				return nil, err
			}
		}
	}

	return s.GetArticle(ctx, article.Slug, &userID)
}

func (s *articleService) DeleteArticle(ctx context.Context, userID int64, slug string) error {
	if _, err := s.authorizeArticleOwner(ctx, userID, slug); err != nil {
		return err
	}

	return s.repo.DeleteArticle(ctx, slug)
}

func (s *articleService) FavoriteArticle(ctx context.Context, userID int64, slug string) (*repository.GetArticleBySlugRow, error) {
	ref, err := s.repo.FindArticleRefBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	if err := s.repo.FavoriteArticle(ctx, repository.FavoriteArticleParams{
		UserID:    userID,
		ArticleID: ref.ID,
	}); err != nil {
		return nil, err
	}

	return s.GetArticle(ctx, slug, &userID)
}

func (s *articleService) UnfavoriteArticle(ctx context.Context, userID int64, slug string) (*repository.GetArticleBySlugRow, error) {
	ref, err := s.repo.FindArticleRefBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	if err := s.repo.UnfavoriteArticle(ctx, repository.UnfavoriteArticleParams{
		UserID:    userID,
		ArticleID: ref.ID,
	}); err != nil {
		return nil, err
	}

	return s.GetArticle(ctx, slug, &userID)
}

func (s *articleService) CreateComment(ctx context.Context, articleSlug string, authorID int64, body string) (*repository.GetCommentByIDRow, error) {
	articleRef, err := s.repo.FindArticleRefBySlug(ctx, articleSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	comment, err := s.repo.CreateComment(ctx, repository.CreateCommentParams{
		ArticleID: articleRef.ID,
		AuthorID:  authorID,
		Body:      body,
	})
	if err != nil {
		return nil, err
	}

	return s.repo.GetCommentByID(ctx, repository.GetCommentByIDParams{
		ID:            comment.ID,
		CurrentUserID: &authorID,
	})
}

func (s *articleService) ListComments(ctx context.Context, articleSlug string, currentUserID *int64) ([]*repository.ListCommentsByArticleIDRow, error) {
	articleRef, err := s.repo.FindArticleRefBySlug(ctx, articleSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}

	return s.repo.ListCommentsByArticleID(ctx, repository.ListCommentsByArticleIDParams{
		ArticleID:     articleRef.ID,
		CurrentUserID: currentUserID,
	})
}

func (s *articleService) DeleteComment(ctx context.Context, userID int64, articleSlug string, commentID int64) error {
	articleRef, err := s.repo.FindArticleRefBySlug(ctx, articleSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrArticleNotFound
		}
		return err
	}

	commentRef, err := s.repo.FindCommentRefByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCommentNotFound
		}
		return err
	}

	if commentRef.ArticleID != articleRef.ID {
		return ErrCommentNotFound
	}

	if commentRef.AuthorID != userID {
		return ErrForbidden
	}

	return s.repo.DeleteComment(ctx, commentID)
}

func (s *articleService) ListFeed(ctx context.Context, currentUserID int64, limit, offset int32) ([]*repository.ListFeedArticlesRow, int64, error) {
	articles, err := s.repo.ListFeedArticles(ctx, repository.ListFeedArticlesParams{
		CurrentUserID: currentUserID,
		Limit:         limit,
		Offset:        offset,
	})
	if err != nil {
		return nil, 0, err
	}

	if len(articles) == 0 {
		return articles, 0, nil
	}

	return articles, articles[0].TotalCount, nil
}

func (s *articleService) ListTags(ctx context.Context) ([]string, error) {
	return s.repo.ListTags(ctx)
}
