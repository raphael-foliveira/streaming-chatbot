package httpx

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

func WriteEventStream(w http.ResponseWriter, event, data string) error {
	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}

	lines := strings.SplitSeq(data, "\n")
	for line := range lines {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintf(w, "\n")
	return err
}

func WriteEventStreamTemplate(ctx context.Context, w http.ResponseWriter, event string, template templ.Component) error {
	var buf bytes.Buffer
	if err := template.Render(ctx, &buf); err != nil {
		return err
	}

	return WriteEventStream(w, event, buf.String())
}

func SetupSSE(c echo.Context) *echo.Response {
	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	return w
}
