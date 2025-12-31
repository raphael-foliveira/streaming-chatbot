package httpx

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}
