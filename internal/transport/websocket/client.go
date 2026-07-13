package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/kirban/social-media/internal/middleware"
	"github.com/kirban/social-media/internal/model"
)

const (
	// writeTimeout bounds a single socket write or ping.
	writeTimeout = 10 * time.Second
	// pingInterval keeps idle connections alive and detects dead peers.
	pingInterval = 30 * time.Second
	// sendBuffer is how many outbound messages we queue before treating the
	// client as too slow and dropping it.
	sendBuffer = 16
	// readLimit caps a single inbound frame (bytes), bounding per-client memory.
	readLimit = 32 * 1024
)

// Client is a single WebSocket connection owned by one authenticated user.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	userID model.UserID
	send   chan []byte
}

// ServeWS upgrades an HTTP request to a WebSocket and starts the client pumps.
// It relies on the auth middleware having placed the user ID in the context.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(model.UserID)
	if !ok || userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.originPatterns,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("ws accept failed")
		return
	}
	conn.SetReadLimit(readLimit)

	c := &Client{
		hub:    h,
		conn:   conn,
		userID: userID,
		send:   make(chan []byte, sendBuffer),
	}
	if !h.add(c) {
		// Server is shutting down; refuse the connection cleanly.
		conn.Close(websocket.StatusGoingAway, "server shutting down")
		return
	}

	// writePump runs in its own goroutine; readPump keeps this one until the
	// connection closes.
	go c.writePump()
	c.readPump()
}

// readPump reads inbound frames and forwards their raw payload to the hub's
// handler, if one is set. It owns the connection's single reader and returns
// when the socket closes.
func (c *Client) readPump() {
	defer func() {
		c.hub.drop(c)
		c.conn.CloseNow()
	}()

	ctx := context.Background()
	for {
		_, payload, err := c.conn.Read(ctx)
		if err != nil {
			return // normal closure or broken connection
		}
		if c.hub.handler != nil {
			c.hub.handler(ctx, c.userID, payload)
		}
	}
}

// writePump owns the connection's single writer: it drains the send channel and
// emits periodic pings. It returns when the hub closes send or a write fails.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.conn.CloseNow()
	}()

	for {
		select {
		case payload, ok := <-c.send:
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
			err := c.conn.Write(ctx, websocket.MessageText, payload)
			cancel()
			if err != nil {
				return
			}

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
			err := c.conn.Ping(ctx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
