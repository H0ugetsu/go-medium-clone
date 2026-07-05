package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/h0ugetsu/realworld-api/internal/httputil"
	"github.com/h0ugetsu/realworld-api/internal/httputil/httperror"
	"github.com/h0ugetsu/realworld-api/internal/middleware"
	"github.com/h0ugetsu/realworld-api/internal/repository"
	"github.com/h0ugetsu/realworld-api/internal/service"
	"github.com/labstack/echo/v5"
)

type ArticleHandler struct {
	articleService service.ArticleService
}

func NewArticleHandler(articleService service.ArticleService) *ArticleHandler {
	return &ArticleHandler{
		articleService: articleService,
	}
}

type articleRes struct {
	Article struct {
		Slug           string   `json:"slug"`
		Title          string   `json:"title"`
		Description    string   `json:"description"`
		Body           string   `json:"body"`
		TagList        []string `json:"tagList"`
		CreatedAt      string   `json:"createdAt"`
		UpdatedAt      string   `json:"updatedAt"`
		Favorited      bool     `json:"favorited"`
		FavoritesCount int64    `json:"favoritesCount"`
		Author         struct {
			Username  string  `json:"username"`
			Bio       *string `json:"bio"`
			Image     *string `json:"image"`
			Following bool    `json:"following"`
		} `json:"author"`
	} `json:"article"`
}

func newArticleRes(article *repository.GetArticleBySlugRow) articleRes {
	var res articleRes
	res.Article.Slug = article.Slug
	res.Article.Title = article.Title
	res.Article.Description = article.Description
	res.Article.Body = article.Body
	res.Article.TagList = article.TagList
	res.Article.CreatedAt = article.CreatedAt.Format(timeFormat)
	res.Article.UpdatedAt = article.UpdatedAt.Format(timeFormat)
	res.Article.Favorited = article.Favorited
	res.Article.FavoritesCount = article.FavoritesCount
	res.Article.Author.Username = article.AuthorUsername
	res.Article.Author.Bio = article.AuthorBio
	res.Article.Author.Image = article.AuthorImage
	res.Article.Author.Following = article.AuthorFollowing
	return res
}

const timeFormat = "2006-01-02T15:04:05.000Z"

const (
	defaultLimit = 20
	maxLimit     = 100
)

func parsePagination(c *echo.Context) (limit, offset int32) {
	limit = defaultLimit
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			limit = int32(n)
		}
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	offset = 0
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 32); err == nil {
			offset = int32(n)
		}
	}
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

type createArticleReq struct {
	Article struct {
		Title       string   `json:"title" validate:"required"`
		Description string   `json:"description" validate:"required"`
		Body        string   `json:"body" validate:"required"`
		TagList     []string `json:"tagList"`
	} `json:"article"`
}

func (h *ArticleHandler) Create(c *echo.Context) error {
	authorID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	var req createArticleReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	article, err := h.articleService.CreateArticle(c.Request().Context(), authorID, repository.CreateArticleParams{
		Title:       req.Article.Title,
		Description: req.Article.Description,
		Body:        req.Article.Body,
	}, req.Article.TagList)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, newArticleRes(article))
}

func (h *ArticleHandler) Show(c *echo.Context) error {
	slug := c.Param("slug")

	var currentUserID *int64
	if userID, ok := c.Get(middleware.UserIDContextKey).(int64); ok {
		currentUserID = &userID
	}

	article, err := h.articleService.GetArticle(c.Request().Context(), slug, currentUserID)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"article": {"not found"}},
			})
		}
		return err
	}

	return c.JSON(http.StatusOK, newArticleRes(article))
}

type articleListItemRes struct {
	Slug           string   `json:"slug"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	TagList        []string `json:"tagList"`
	CreatedAt      string   `json:"createdAt"`
	UpdatedAt      string   `json:"updatedAt"`
	Favorited      bool     `json:"favorited"`
	FavoritesCount int64    `json:"favoritesCount"`
	Author         struct {
		Username  string  `json:"username"`
		Bio       *string `json:"bio"`
		Image     *string `json:"image"`
		Following bool    `json:"following"`
	} `json:"author"`
}

type articleListRes struct {
	Articles      []articleListItemRes `json:"articles"`
	ArticlesCount int64                `json:"articlesCount"`
}

func (h *ArticleHandler) List(c *echo.Context) error {
	var currentUserID *int64
	if userID, ok := c.Get(middleware.UserIDContextKey).(int64); ok {
		currentUserID = &userID
	}

	limit, offset := parsePagination(c)

	params := repository.ListArticlesParams{
		Limit:  limit,
		Offset: offset,
	}
	if v := c.QueryParam("tag"); v != "" {
		params.Tag = &v
	}
	if v := c.QueryParam("author"); v != "" {
		params.Author = &v
	}
	if v := c.QueryParam("favorited"); v != "" {
		params.FavoritedBy = &v
	}

	articles, total, err := h.articleService.ListArticles(c.Request().Context(), params, currentUserID)
	if err != nil {
		return err
	}

	res := articleListRes{
		Articles:      make([]articleListItemRes, 0, len(articles)),
		ArticlesCount: total,
	}
	for _, article := range articles {
		var item articleListItemRes
		item.Slug = article.Slug
		item.Title = article.Title
		item.Description = article.Description
		item.TagList = article.TagList
		item.CreatedAt = article.CreatedAt.Format(timeFormat)
		item.UpdatedAt = article.UpdatedAt.Format(timeFormat)
		item.Favorited = article.Favorited
		item.FavoritesCount = article.FavoritesCount
		item.Author.Username = article.AuthorUsername
		item.Author.Bio = article.AuthorBio
		item.Author.Image = article.AuthorImage
		item.Author.Following = article.AuthorFollowing
		res.Articles = append(res.Articles, item)
	}

	return c.JSON(http.StatusOK, res)
}

type updateArticleReq struct {
	Article struct {
		Title       *string                  `json:"title"`
		Description *string                  `json:"description"`
		Body        *string                  `json:"body"`
		TagList     httputil.Field[[]string] `json:"tagList"`
	} `json:"article"`
}

func (h *ArticleHandler) Update(c *echo.Context) error {
	userID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	slug := c.Param("slug")

	var req updateArticleReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	var tagList *[]string
	if f := req.Article.TagList; f.Present {
		if f.Null {
			return httperror.New(http.StatusUnprocessableEntity, map[string]any{
				"errors": map[string][]string{"tagList": {"can't be null"}},
			})
		}
		v := f.Value
		tagList = &v
	}

	article, err := h.articleService.UpdateArticle(c.Request().Context(), userID, slug, repository.UpdateArticleParams{
		Title:       req.Article.Title,
		Description: req.Article.Description,
		Body:        req.Article.Body,
	}, tagList)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"article": {"not found"}},
			})
		case errors.Is(err, service.ErrForbidden):
			return httperror.New(http.StatusForbidden, map[string]any{
				"errors": map[string][]string{"article": {"forbidden"}},
			})
		default:
			return err
		}
	}

	return c.JSON(http.StatusOK, newArticleRes(article))
}

func (h *ArticleHandler) Delete(c *echo.Context) error {
	userID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	slug := c.Param("slug")

	if err := h.articleService.DeleteArticle(c.Request().Context(), userID, slug); err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"article": {"not found"}},
			})
		case errors.Is(err, service.ErrForbidden):
			return httperror.New(http.StatusForbidden, map[string]any{
				"errors": map[string][]string{"article": {"forbidden"}},
			})
		default:
			return err
		}
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *ArticleHandler) Favorite(c *echo.Context) error {
	userID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	slug := c.Param("slug")

	article, err := h.articleService.FavoriteArticle(c.Request().Context(), userID, slug)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"article": {"not found"}},
			})
		}
		return err
	}

	return c.JSON(http.StatusOK, newArticleRes(article))
}

type commentItemRes struct {
	ID        int64  `json:"id"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	Author    struct {
		Username  string  `json:"username"`
		Bio       *string `json:"bio"`
		Image     *string `json:"image"`
		Following bool    `json:"following"`
	} `json:"author"`
}

type commentRes struct {
	Comment commentItemRes `json:"comment"`
}

func newCommentItemRes(id int64, body string, createdAt, updatedAt time.Time, authorUsername string, authorBio, authorImage *string, authorFollowing bool) commentItemRes {
	var item commentItemRes
	item.ID = id
	item.Body = body
	item.CreatedAt = createdAt.Format(timeFormat)
	item.UpdatedAt = updatedAt.Format(timeFormat)
	item.Author.Username = authorUsername
	item.Author.Bio = authorBio
	item.Author.Image = authorImage
	item.Author.Following = authorFollowing
	return item
}

type createCommentReq struct {
	Comment struct {
		Body string `json:"body" validate:"required"`
	} `json:"comment"`
}

func (h *ArticleHandler) CreateComment(c *echo.Context) error {
	authorID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	slug := c.Param("slug")

	var req createCommentReq
	if err := c.Bind(&req); err != nil {
		return err
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	comment, err := h.articleService.CreateComment(c.Request().Context(), slug, authorID, req.Comment.Body)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"article": {"not found"}},
			})
		}
		return err
	}

	return c.JSON(http.StatusCreated, commentRes{
		Comment: newCommentItemRes(
			comment.ID, comment.Body, comment.CreatedAt, comment.UpdatedAt,
			comment.AuthorUsername, comment.AuthorBio, comment.AuthorImage, comment.AuthorFollowing,
		),
	})
}

type commentListRes struct {
	Comments []commentItemRes `json:"comments"`
}

func (h *ArticleHandler) ListComments(c *echo.Context) error {
	slug := c.Param("slug")

	var currentUserID *int64
	if userID, ok := c.Get(middleware.UserIDContextKey).(int64); ok {
		currentUserID = &userID
	}

	comments, err := h.articleService.ListComments(c.Request().Context(), slug, currentUserID)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"article": {"not found"}},
			})
		}
		return err
	}

	res := commentListRes{
		Comments: make([]commentItemRes, 0, len(comments)),
	}
	for _, comment := range comments {
		res.Comments = append(res.Comments, newCommentItemRes(
			comment.ID, comment.Body, comment.CreatedAt, comment.UpdatedAt,
			comment.AuthorUsername, comment.AuthorBio, comment.AuthorImage, comment.AuthorFollowing,
		))
	}

	return c.JSON(http.StatusOK, res)
}

func (h *ArticleHandler) DeleteComment(c *echo.Context) error {
	userID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	slug := c.Param("slug")
	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return httperror.New(http.StatusNotFound, map[string]any{
			"errors": map[string][]string{"comment": {"not found"}},
		})
	}

	if err := h.articleService.DeleteComment(c.Request().Context(), userID, slug, commentID); err != nil {
		switch {
		case errors.Is(err, service.ErrArticleNotFound):
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"article": {"not found"}},
			})
		case errors.Is(err, service.ErrCommentNotFound):
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"comment": {"not found"}},
			})
		case errors.Is(err, service.ErrForbidden):
			return httperror.New(http.StatusForbidden, map[string]any{
				"errors": map[string][]string{"comment": {"forbidden"}},
			})
		default:
			return err
		}
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *ArticleHandler) Unfavorite(c *echo.Context) error {
	userID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	slug := c.Param("slug")

	article, err := h.articleService.UnfavoriteArticle(c.Request().Context(), userID, slug)
	if err != nil {
		if errors.Is(err, service.ErrArticleNotFound) {
			return httperror.New(http.StatusNotFound, map[string]any{
				"errors": map[string][]string{"article": {"not found"}},
			})
		}
		return err
	}

	return c.JSON(http.StatusOK, newArticleRes(article))
}

func (h *ArticleHandler) Feed(c *echo.Context) error {
	userID, ok := c.Get(middleware.UserIDContextKey).(int64)
	if !ok {
		return httperror.New(http.StatusUnauthorized, map[string]any{
			"errors": map[string][]string{"token": {"is invalid"}},
		})
	}

	limit, offset := parsePagination(c)

	articles, total, err := h.articleService.ListFeed(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return err
	}

	res := articleListRes{
		Articles:      make([]articleListItemRes, 0, len(articles)),
		ArticlesCount: total,
	}
	for _, article := range articles {
		var item articleListItemRes
		item.Slug = article.Slug
		item.Title = article.Title
		item.Description = article.Description
		item.TagList = article.TagList
		item.CreatedAt = article.CreatedAt.Format(timeFormat)
		item.UpdatedAt = article.UpdatedAt.Format(timeFormat)
		item.Favorited = article.Favorited
		item.FavoritesCount = article.FavoritesCount
		item.Author.Username = article.AuthorUsername
		item.Author.Bio = article.AuthorBio
		item.Author.Image = article.AuthorImage
		item.Author.Following = article.AuthorFollowing
		res.Articles = append(res.Articles, item)
	}

	return c.JSON(http.StatusOK, res)
}

type tagListRes struct {
	Tags []string `json:"tags"`
}

func (h *ArticleHandler) Tags(c *echo.Context) error {
	tags, err := h.articleService.ListTags(c.Request().Context())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, tagListRes{Tags: tags})
}
