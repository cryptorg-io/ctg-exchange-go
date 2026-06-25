package ctgexchange

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// DefaultWSBaseURL is the production WebSocket base URL.
const DefaultWSBaseURL = "wss://api.ctg.exchange"

// ErrStreamClosed is returned by Stream.Next once the stream has
// stopped and will deliver no more messages.
var ErrStreamClosed = errors.New("ctgexchange: stream closed")

// StreamConfig configures a WebSocket stream.
type StreamConfig struct {
	BaseURL          string        // default DefaultWSBaseURL
	Channels         []string      // (re)subscribed on every connect
	DisableReconnect bool          // turn off auto-reconnect
	ReconnectDelay   time.Duration // default 2s
}

// Stream is a WebSocket stream. Call Next in a loop to consume
// messages; it connects on the first call and, unless reconnect is
// disabled, transparently reconnects and re-subscribes on a drop.
// A Stream is not safe for concurrent Next calls.
type Stream struct {
	url            string
	reconnect      bool
	reconnectDelay time.Duration
	authFn         func(context.Context, *websocket.Conn) error

	mu       sync.Mutex
	channels []string
	conn     *websocket.Conn

	msgs      chan StreamMessage
	ctx       context.Context
	cancel    context.CancelFunc
	startOnce sync.Once
	closeOnce sync.Once
	runErr    error
}

func newStream(
	path string,
	cfg StreamConfig,
	authFn func(context.Context, *websocket.Conn) error,
) *Stream {
	base := cfg.BaseURL
	if base == "" {
		base = DefaultWSBaseURL
	}
	delay := cfg.ReconnectDelay
	if delay <= 0 {
		delay = 2 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Stream{
		url:            strings.TrimRight(base, "/") + path,
		reconnect:      !cfg.DisableReconnect,
		reconnectDelay: delay,
		authFn:         authFn,
		channels:       dedup(cfg.Channels),
		msgs:           make(chan StreamMessage, 64),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// NewMarketDataStream opens the public market-data stream — channels
// orderbook / ticker / candles / trades. No authentication.
func NewMarketDataStream(cfg StreamConfig) *Stream {
	return newStream("/api/v1/stream", cfg, nil)
}

// NewUserStream opens the private stream — the caller's orders / trades
// / balances — authenticated in-band with a signed first frame.
func NewUserStream(apiKey, apiSecret string, cfg StreamConfig) (*Stream, error) {
	if apiKey == "" || apiSecret == "" {
		return nil, ErrMissingCredentials
	}
	return newStream("/api/v1/me/stream", cfg, userAuth(apiKey, apiSecret)), nil
}

// Next returns the next message. It blocks until one arrives, ctx is
// cancelled, or the stream closes (then it returns ErrStreamClosed or
// the failure that ended the stream).
func (s *Stream) Next(ctx context.Context) (StreamMessage, error) {
	s.startOnce.Do(func() { go s.run() })
	select {
	case msg, ok := <-s.msgs:
		if !ok {
			if s.runErr != nil {
				return StreamMessage{}, s.runErr
			}
			return StreamMessage{}, ErrStreamClosed
		}
		return msg, nil
	case <-ctx.Done():
		return StreamMessage{}, ctx.Err()
	}
}

// Subscribe adds channels (e.g. "orderbook@CTGUSDT"). They are kept
// across reconnects.
func (s *Stream) Subscribe(channels ...string) {
	s.mu.Lock()
	s.channels = dedup(append(s.channels, channels...))
	conn := s.conn
	s.mu.Unlock()
	if conn != nil {
		_ = wsSend(s.ctx, conn,
			map[string]any{"method": "subscribe", "channels": channels})
	}
}

// Unsubscribe drops channels.
func (s *Stream) Unsubscribe(channels ...string) {
	drop := make(map[string]struct{}, len(channels))
	for _, c := range channels {
		drop[c] = struct{}{}
	}
	s.mu.Lock()
	kept := s.channels[:0]
	for _, c := range s.channels {
		if _, ok := drop[c]; !ok {
			kept = append(kept, c)
		}
	}
	s.channels = kept
	conn := s.conn
	s.mu.Unlock()
	if conn != nil {
		_ = wsSend(s.ctx, conn,
			map[string]any{"method": "unsubscribe", "channels": channels})
	}
}

// Close stops the stream and any reconnection. After Close, Next
// returns ErrStreamClosed.
func (s *Stream) Close() {
	s.closeOnce.Do(s.cancel)
}

func (s *Stream) run() {
	defer close(s.msgs)
	for {
		if s.ctx.Err() != nil {
			return
		}
		conn, err := s.connect()
		if err != nil {
			var apiErr *APIError
			authRejected := errors.As(err, &apiErr) && apiErr.StatusCode == 401
			if !s.reconnect || authRejected {
				s.runErr = err
				return
			}
			if !sleepCtx(s.ctx, s.reconnectDelay) {
				return
			}
			continue
		}
		s.readLoop(conn)
		if s.ctx.Err() != nil || !s.reconnect {
			return
		}
		if !sleepCtx(s.ctx, s.reconnectDelay) {
			return
		}
	}
}

func (s *Stream) connect() (*websocket.Conn, error) {
	conn, _, err := websocket.Dial(s.ctx, s.url, nil)
	if err != nil {
		return nil, err
	}
	conn.SetReadLimit(1 << 20)
	if s.authFn != nil {
		if err := s.authFn(s.ctx, conn); err != nil {
			conn.CloseNow()
			return nil, err
		}
	}
	s.mu.Lock()
	s.conn = conn
	chans := append([]string(nil), s.channels...)
	s.mu.Unlock()
	if len(chans) > 0 {
		if err := wsSend(s.ctx, conn,
			map[string]any{"method": "subscribe", "channels": chans}); err != nil {
			conn.CloseNow()
			return nil, err
		}
	}
	return conn, nil
}

func (s *Stream) readLoop(conn *websocket.Conn) {
	defer func() {
		s.mu.Lock()
		if s.conn == conn {
			s.conn = nil
		}
		s.mu.Unlock()
		conn.CloseNow()
	}()
	for {
		_, data, err := conn.Read(s.ctx)
		if err != nil {
			return
		}
		msg, ok := parseFrame(data)
		if !ok {
			continue
		}
		select {
		case s.msgs <- msg:
		case <-s.ctx.Done():
			return
		}
	}
}

func userAuth(apiKey, apiSecret string) func(context.Context, *websocket.Conn) error {
	return func(ctx context.Context, conn *websocket.Conn) error {
		frame := WSAuthMessage(apiKey, apiSecret, time.Now().Unix())
		if err := wsSend(ctx, conn, frame); err != nil {
			return err
		}
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		var reply struct {
			Op      string `json:"op"`
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		if json.Unmarshal(data, &reply) != nil ||
			reply.Op != "auth" || !reply.Success {
			msg := reply.Error
			if msg == "" {
				msg = "WebSocket auth rejected"
			}
			return &APIError{StatusCode: 401, Message: msg}
		}
		return nil
	}
}

// parseFrame decodes a raw frame into a StreamMessage. It returns
// ok=false for control frames (auth replies, subscribe acks) — only
// data frames (snapshot/update) are surfaced.
func parseFrame(data []byte) (StreamMessage, bool) {
	var raw struct {
		Type    string          `json:"type"`
		Channel string          `json:"channel"`
		Symbol  string          `json:"symbol"`
		Data    json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return StreamMessage{}, false
	}
	if raw.Type != "snapshot" && raw.Type != "update" {
		return StreamMessage{}, false
	}
	return StreamMessage{
		Type:    raw.Type,
		Channel: raw.Channel,
		Symbol:  raw.Symbol,
		Data:    raw.Data,
	}, true
}

func wsSend(ctx context.Context, conn *websocket.Conn, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, data)
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func dedup(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
