package party

import (
	"encoding/json"
	"fmt"
)

// Events coming from the sockets

const (
	eventLoadVideo    = "load-video"
	eventPauseVideo   = "pause"
	eventPlayVideo    = "play"
	eventSyncResponse = "sync-response"
	eventPlaybackRate = "rate"
)

var (
	errUnhandledEvent = fmt.Errorf("unhandled event")
)

type loadVideoPayload struct {
	VideoID string `json:"id"`
}

type pauseVideoPayload struct {
	Timestamp float64 `json:"timestamp"`
}

type playVideoPayload struct {
	Timestamp float64 `json:"timestamp"`
}

type rateVideoPayload struct {
	rate float64 `json:"rate"`
}

type syncResponsePayload struct {
	Timestamp float64     `json:"timestamp"`
	VideoID   string      `json:"id"`
	Rate      float64     `json:"rate"`
	State     playerState `json:"state"`
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
	case eventPlayVideo:
		event := &playVideoPayload{}
		err := json.Unmarshal(base.Payload, event)

		if err != nil {
			return nil, err
		}

		return event, nil
	case eventPauseVideo:
		event := &pauseVideoPayload{}
		err := json.Unmarshal(base.Payload, event)

		if err != nil {
			return nil, err
		}

		return event, nil
	case eventSyncResponse:
		event := &syncResponsePayload{}
		err := json.Unmarshal(base.Payload, event)

		if err != nil {
			return nil, err
		}

		return event, nil
	case eventPlaybackRate:
		event := &rateVideoPayload{}
		err := json.Unmarshal(base.Payload, event)

		if err != nil {
			return nil, err
		}

		return event, nil
	}

	return nil, errUnhandledEvent
}
