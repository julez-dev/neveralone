package party

type playerState = int

const (
	NoVideo playerState = iota
	Playing
	Paused
)

type VideoStateSnapshot struct {
	PlayerState playerState
	VideoID     string
	Timestamp   float64
	Rate        float64
}

func (v *VideoStateSnapshot) updateFromEvent(event any) {
	switch e := event.(type) {
	case *loadVideoPayload:
		v.PlayerState = Playing
		v.VideoID = e.VideoID
		v.Timestamp = 0
		return
	case *playVideoPayload:
		v.PlayerState = Playing
		v.Timestamp = e.Timestamp
		return
	case *pauseVideoPayload:
		v.PlayerState = Paused
		v.Timestamp = e.Timestamp
		return
	case *rateVideoPayload:
		v.Rate = e.rate
		return
	case *syncResponsePayload:
		if e.State >= 0 && e.State < 3 {
			v.PlayerState = e.State
		}

		v.Rate = e.Rate
		v.VideoID = e.VideoID
		v.Timestamp = e.Timestamp
	}
}
