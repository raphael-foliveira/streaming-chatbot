package search

import (
	"math/rand"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	searchviews "github.com/raphael-foliveira/htmbot/modules/search/views"
	"github.com/raphael-foliveira/htmbot/platform/httpx"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Register(e *echo.Echo) {
	g := e.Group("/search")

	g.GET("", h.Index)
	g.GET("/results", h.SearchResults)
}

func (h *Handler) Index(c echo.Context) error {
	return httpx.Render(c, searchviews.Index())
}

func (h *Handler) SearchResults(c echo.Context) error {
	query := c.QueryParam("query")

	if query == "" {
		return httpx.Render(c, searchviews.SearchResults([]string{}))
	}

	resultsLen := rand.Intn(20) + 1
	results := make([]string, resultsLen)
	for i := range results {
		results[i] = uuid.New().String()
	}
	return httpx.Render(c, searchviews.SearchResults(results))
}
