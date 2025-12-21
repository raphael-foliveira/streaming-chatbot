package httpx

import (
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

func Render(c echo.Context, cc templ.Component) error {
	return cc.Render(c.Request().Context(), c.Response().Writer)
}
