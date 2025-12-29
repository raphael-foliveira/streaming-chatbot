package httpx

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func HxRedirect(c echo.Context, url string) error {
	c.Response().Header().Set("HX-Redirect", url)
	return c.NoContent(http.StatusNoContent)
}
