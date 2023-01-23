package party

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"log"
	"sync"
	"time"
)

const (
	usageCheckWait   = time.Second * 45
	requestStateWait = time.Second * 2
)

type Session struct {
	logger zerolog.Logger
	ID     uuid.UUID

	state *VideoStateSnapshot
	vLock *sync.Mutex

	pLock       *sync.Mutex
	players     map[string]*Player
	connections map[string]*connection

	messageQueue chan *message
	Register     chan *User
	Unregister   chan *User

	AttachWS chan *AttachSocket
	DetachWS chan string

	o *sync.Once
}

type AttachSocket struct {
	PlayerID   string
	Connection *websocket.Conn
}

type message struct {
	playerID string
	event    any
	raw      []byte
}

func NewSession(logger zerolog.Logger, host *User) *Session {
	id := uuid.New()

	s := &Session{
		logger:       logger.With().Str("room-id", id.String()).Logger(),
		ID:           id,
		players:      map[string]*Player{},
		connections:  map[string]*connection{},
		messageQueue: make(chan *message),
		Register:     make(chan *User),
		Unregister:   make(chan *User),
		AttachWS:     make(chan *AttachSocket),
		DetachWS:     make(chan string),
		state: &VideoStateSnapshot{
			PlayerState: NoVideo,
		},
		o:     &sync.Once{},
		pLock: &sync.Mutex{},
		vLock: &sync.Mutex{},
	}

	s.players[host.ID.String()] = NewPlayer(host, true)

	return s
}

func (s *Session) GetPlayersCopy() []*Player {
	s.pLock.Lock()
	defer s.pLock.Unlock()

	copied := make([]*Player, 0, len(s.players))

	for _, p := range s.players {
		copied = append(copied, &Player{
			User: &User{
				ID:   p.User.ID,
				Name: p.User.Name,
			},
			IsHost: p.IsHost,
		})
	}

	return copied
}

func (s *Session) GetCurrentState() *VideoStateSnapshot {
	s.vLock.Lock()
	defer s.vLock.Unlock()
	return &VideoStateSnapshot{
		PlayerState: s.state.PlayerState,
		VideoID:     s.state.VideoID,
		Timestamp:   s.state.Timestamp,
	}
}

func (s *Session) Run(ctx context.Context) {
	ticker := time.NewTicker(usageCheckWait)
	stateTicker := time.NewTicker(requestStateWait)

	defer func() {
		ticker.Stop()
		stateTicker.Stop()
		s.logger.Debug().Msg("session loop exited")
	}()

	for {
		select {
		case <-stateTicker.C:
			if len(s.connections) < 1 {
				break
			}

			// get first open connection
			var conn *connection
			for _, c := range s.connections {
				conn = c
				break
			}

			msg := &actionMessage{Action: actionRequestState}
			data, err := json.Marshal(msg)

			if err != nil {
				break
			}

			conn.send <- data
		case <-ticker.C:
			// Close room when no more connections
			if len(s.connections) < 1 {
				return
			}
		case user := <-s.Register:
			s.pLock.Lock()
			id := user.ID.String()
			_, exists := s.players[id]

			if !exists {
				s.players[id] = NewPlayer(user, false)
			}
			s.pLock.Unlock()

			ticker.Reset(usageCheckWait)
		case user := <-s.Unregister:
			s.pLock.Lock()
			id := user.ID.String()
			_, ok := s.players[id]

			if ok {
				delete(s.players, id)
			}
			s.pLock.Unlock()

			ticker.Reset(usageCheckWait)
		case ws := <-s.AttachWS:
			_, exists := s.connections[ws.PlayerID]

			if exists {
				s.logger.Info().Str("user-id", ws.PlayerID).Msg("user with active ws connection tried to join")

				ws.Connection.SetWriteDeadline(time.Now().Add(writeWait))
				err := ws.Connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, "user already has active connection for this room"))

				if err != nil {
					s.logger.Err(err).Str("user-id", ws.PlayerID).Msg("error closing blocked ws")
				}

				break
			}

			newConn := newConnection(s.logger, ws.PlayerID, s, ws.Connection)
			s.connections[ws.PlayerID] = newConn

			// start reader
			go func(conn *connection) {
				conn.readWS()
			}(newConn)

			// start writer
			go func(conn *connection) {
				conn.writeWS()
			}(newConn)

			ticker.Reset(usageCheckWait)
			stateTicker.Reset(requestStateWait)
		case id := <-s.DetachWS:
			if conn, ok := s.connections[id]; ok {
				delete(s.connections, id)
				conn.closeSendOnce()
			}

			ticker.Reset(usageCheckWait)
		case message := <-s.messageQueue:
			// handle messages coming from the socket
			s.vLock.Lock()
			s.state.updateFromEvent(message.event)
			s.vLock.Unlock()

			var allowSelfSend bool

			if _, ok := message.event.(*syncResponsePayload); ok {
				break
			}

			if _, ok := message.event.(*loadVideoPayload); ok {
				allowSelfSend = true
				log.Println("self send allowed")
			}

			for _, conn := range s.connections {
				if conn.userID != message.playerID || allowSelfSend {
					s.logger.Debug().
						Str("sender-user-id", message.playerID).
						Str("receiver-user-id", conn.userID).
						Str("content", string(message.raw)).
						Msg("sending data to user")

					conn.send <- message.raw
				}
			}
		case <-ctx.Done():
			for id, conn := range s.connections {
				s.logger.Info().Str("user-id", id).Msg("closing connection because root context was canceled")
				conn.socket.Close()
			}
			return
		}
	}
}

func (s *Session) CloseChannels() {
	s.o.Do(func() {
		close(s.messageQueue)
		close(s.Register)
		close(s.Unregister)
		close(s.AttachWS)
		close(s.DetachWS)
	})
}
