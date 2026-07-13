package websocket

import (
	"context"

	"github.com/kirban/social-media/internal/logger"
	"github.com/kirban/social-media/internal/model"
)

// MessageHandler processes a single inbound frame from a client. The transport
// treats the payload as opaque bytes and leaves all decoding and behavior to
// the application. A nil handler means inbound frames are read and discarded
// (fine for a push-only channel such as feed notifications).
type MessageHandler func(ctx context.Context, userID model.UserID, payload []byte)

// Hub is a transport-level registry of authenticated WebSocket connections. It
// tracks every connection per user and fans opaque payloads out to them. It has
// no knowledge of what those payloads mean — features (feed notifications, chat,
// presence, …) build on top via Send and an optional MessageHandler.
//
// A single Hub runs for the lifetime of the server. All mutation of the clients
// map happens on the Run goroutine, so it needs no external locking.
type Hub struct {
	log     *logger.AppLogger
	handler MessageHandler

	// clients indexes connections by user, so a payload can be delivered to a
	// specific recipient and to every device that user currently has open.
	clients map[model.UserID]map[*Client]struct{}

	register   chan *Client
	unregister chan *Client
	deliver    chan outbound

	// done is closed when Run returns, so client goroutines blocked on the
	// channels above can unblock during shutdown instead of leaking.
	done chan struct{}

	// originPatterns are the Origin header host patterns accepted on upgrade.
	// Empty means same-origin only (the coder/websocket default).
	originPatterns []string
}

// outbound is an opaque payload addressed to a single user.
type outbound struct {
	to      model.UserID
	payload []byte
}

func NewHub(l *logger.AppLogger, allowedOrigins []string) *Hub {
	return &Hub{
		log:            l,
		clients:        make(map[model.UserID]map[*Client]struct{}),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		deliver:        make(chan outbound),
		done:           make(chan struct{}),
		originPatterns: allowedOrigins,
	}
}

// OnMessage sets the handler invoked for every inbound frame. Call it before
// Run. Leave it unset for a push-only channel.
func (h *Hub) OnMessage(handler MessageHandler) {
	h.handler = handler
}

// Send queues payload for delivery to every connection the given user has open.
// It is the entry point features use to push a message (e.g. a new-post
// notification to a follower). Payload is sent verbatim; marshal it beforehand.
// Sending to a user with no open connections is a no-op. Safe from any goroutine.
func (h *Hub) Send(userID model.UserID, payload []byte) {
	h.route(outbound{to: userID, payload: payload})
}

// Run is the Hub's event loop. It owns every read and write of the clients map
// and returns when ctx is cancelled (server shutdown), closing all sockets.
func (h *Hub) Run(ctx context.Context) {
	defer close(h.done)
	for {
		select {
		case <-ctx.Done():
			h.closeAll()
			return

		case c := <-h.register:
			conns := h.clients[c.userID]
			if conns == nil {
				conns = make(map[*Client]struct{})
				h.clients[c.userID] = conns
			}
			conns[c] = struct{}{}
			h.log.Info().Str("user", c.userID).Msg("ws client registered")

		case c := <-h.unregister:
			h.remove(c)

		case msg := <-h.deliver:
			for c := range h.clients[msg.to] {
				select {
				case c.send <- msg.payload:
				default:
					// Slow consumer: drop it rather than block the whole hub.
					h.log.Warn().Str("user", c.userID).Msg("ws send buffer full, dropping client")
					h.remove(c)
				}
			}
		}
	}
}

// remove drops a single client and closes its send channel. It is idempotent:
// a client can be scheduled for removal both by its own read pump and by the
// slow-consumer path above, and the membership check keeps the close safe.
func (h *Hub) remove(c *Client) {
	conns, ok := h.clients[c.userID]
	if !ok {
		return
	}
	if _, ok := conns[c]; !ok {
		return
	}
	delete(conns, c)
	close(c.send)
	if len(conns) == 0 {
		delete(h.clients, c.userID)
	}
	h.log.Info().Str("user", c.userID).Msg("ws client unregistered")
}

func (h *Hub) closeAll() {
	for _, conns := range h.clients {
		for c := range conns {
			close(c.send)
		}
	}
	h.clients = make(map[model.UserID]map[*Client]struct{})
}

// add registers a client. It reports false if the hub is already shutting down,
// in which case the caller must not proceed to run its pumps.
func (h *Hub) add(c *Client) bool {
	select {
	case h.register <- c:
		return true
	case <-h.done:
		return false
	}
}

// drop unregisters a client, giving up if the hub has already stopped (which
// has already closed every send channel via closeAll).
func (h *Hub) drop(c *Client) {
	select {
	case h.unregister <- c:
	case <-h.done:
	}
}

// route queues a payload for delivery, giving up if the hub has stopped.
func (h *Hub) route(msg outbound) {
	select {
	case h.deliver <- msg:
	case <-h.done:
	}
}
