package httpx

import (
	"fmt"
	"net/http"
	"strings"
)

func WriteEventStream(w http.ResponseWriter, event, data string) error {
	// Write the event type
	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}

	// Split data by newlines and prefix each line with "data: "
	// This ensures line breaks in the data don't break the SSE format
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
			return err
		}
	}

	// Write the final newline to complete the event
	_, err := fmt.Fprintf(w, "\n")
	return err
}
