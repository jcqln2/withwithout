// Package sse reads a Beacon API text/event-stream.
package sse

import (
	"bufio"
	"io"
	"strings"
)

// Event is one SSE message. Data lines are joined with "\n".
type Event struct {
	Name string
	Data string
}

// Read returns the next event. Comments (lines starting with ':') are skipped.
// io.EOF means the stream closed.
func Read(r *bufio.Reader) (Event, error) {
	var name string
	var data []string
	for {
		line, err := r.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				if name != "" || len(data) > 0 {
					return Event{Name: name, Data: strings.Join(data, "\n")}, nil
				}
				continue
			}
			if strings.HasPrefix(line, ":") {
				continue
			}
			field, value, _ := strings.Cut(line, ":")
			value = strings.TrimPrefix(value, " ")
			switch field {
			case "event":
				name = value
			case "data":
				data = append(data, value)
			}
			continue
		}
		if err == io.EOF {
			if name != "" || len(data) > 0 {
				return Event{Name: name, Data: strings.Join(data, "\n")}, nil
			}
			return Event{}, io.EOF
		}
		if err != nil {
			return Event{}, err
		}
	}
}
