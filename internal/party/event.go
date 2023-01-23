package party

import (
	"encoding/json"
	"fmt"
)

const (
	eventLoadVideo = "load-video"
)

var (
	errUnhandledEvent = fmt.Errorf("unhandled event")
)

type loadVideoPayload struct {
	VideoID string `json:"id"`
}

func parseEvent(b []byte) (any, error) {
	base := &struct {
		Name    string          `json:"action"`
		Payload json.RawMessage `json:"payload"`
	}{}

	if err := json.Unmarshal(b, base); err != nil {
		return nil, err
	}

	switch base.Name {
	case eventLoadVideo:
		event := &loadVideoPayload{}
		err := json.Unmarshal(base.Payload, event)

		if err != nil {
			return nil, err
		}

		return event, nil
	}

	return nil, errUnhandledEvent
}
