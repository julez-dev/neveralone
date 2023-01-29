package party

import (
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"log"
	"sync"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type connection struct {
	logger zerolog.Logger
	userID string

	socket *websocket.Conn

	// outbound messages
	send chan []byte
	once *sync.Once

	session *Session
}

func newConnection(logger zerolog.Logger, userID string, session *Session, socket *websocket.Conn) *connection {
	return &connection{
		logger:  logger.With().Str("user-id", userID).Logger(),
		userID:  userID,
		socket:  socket,
		send:    make(chan []byte),
		session: session,
		once:    &sync.Once{},
	}
}

func (c *connection) readWS() {
	defer func() {
		c.session.DetachWS <- c.userID
		c.socket.Close()
	}()

	c.socket.SetReadLimit(maxMessageSize)
	c.socket.SetReadDeadline(time.Now().Add(pongWait))
	c.socket.SetPongHandler(func(data string) error {
		c.socket.SetReadDeadline(time.Now().Add(pongWait))
		c.logger.Debug().Str("data", data).Msg("pong handler triggered")
		return nil
	})

	for {
		_, data, err := c.socket.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Err(err).Msg("got unexpected error while reading websocket")
			}

			return
		}

		parsed, err := parseEvent(data)

		if err != nil {
			c.logger.Err(err).Str("data", string(data)).Send()
			continue
		}

		c.session.messageQueue <- &message{
			playerID: c.userID,
			event:    parsed,
			raw:      data,
			sender:   c,
		}
	}
}

func (c *connection) writeWS() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.socket.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.socket.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				// the session was closed
				c.socket.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.socket.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			_, err = w.Write(message)

			if err != nil {
				log.Println(err)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.logger.Debug().Msg("writing ping to websocket")
			c.socket.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Err(err).Msg("got unexpected error while writing ping message")
				return
			}
		}
	}
}

func (c *connection) closeSendOnce() {
	c.once.Do(func() {
		close(c.send)
	})
}
