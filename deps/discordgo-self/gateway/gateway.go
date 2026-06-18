package gateway

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hytams/discordgo-self/props"
	"github.com/hytams/discordgo-self/types"
)

const (
	GatewayVersion       = 9
	DefaultGatewayURL    = "wss://gateway.discord.gg/"
	MaxReconnectAttempts = 5
	ReconnectDelay       = 1 * time.Second
)

// Gateway represents a connection to the Discord Gateway.
type Gateway struct {
	Token           string
	SuperProperties *props.SuperProperties
	Capabilities    int

	SessionID string
	ResumeURL string
	Sequence  atomic.Int64

	conn   *websocket.Conn
	connMu sync.Mutex

	heartbeat *Heartbeat

	handlers   map[string][]EventHandler
	handlersMu sync.RWMutex

	ReadyData *ReadyEvent

	connected  atomic.Bool
	resuming   atomic.Bool
	reconnects int

	ctx       context.Context
	cancel    context.CancelFunc
	closeCh   chan struct{}
	closeOnce sync.Once

	OnError func(error)

	Debug bool
}

// EventHandler parses gateway events.
type EventHandler func(eventType string, data json.RawMessage)

// ReadyEvent represents the READY event data.
type ReadyEvent struct {
	Version           int                   `json:"v"`
	User              *types.CurrentUser    `json:"user"`
	Guilds            []json.RawMessage     `json:"guilds"`
	SessionID         string                `json:"session_id"`
	ResumeGatewayURL  string                `json:"resume_gateway_url"`
	Shard             []int                 `json:"shard,omitempty"`
	Application       json.RawMessage       `json:"application,omitempty"`
	PrivateChannels   []*types.Channel      `json:"private_channels"`
	Relationships     []*types.Relationship `json:"relationships"`
	Presences         []json.RawMessage     `json:"presences"`
	Sessions          []*types.Session      `json:"sessions"`
	UserSettings      *types.UserSettings   `json:"user_settings,omitempty"`
	UserGuildSettings json.RawMessage       `json:"user_guild_settings,omitempty"`
	ReadState         json.RawMessage       `json:"read_state,omitempty"`
	Experiments       []interface{}         `json:"experiments,omitempty"`
	Trace             []string              `json:"_trace,omitempty"`
}

// GatewayConfig holds configuration for the gateway.
type GatewayConfig struct {
	Token           string
	SuperProperties *props.SuperProperties
	Capabilities    int
	Debug           bool
}

// NewGateway creates a new gateway connection.
func NewGateway(config GatewayConfig) (*Gateway, error) {
	if config.Token == "" {
		return nil, errors.New("token is required")
	}

	superProps := config.SuperProperties
	if superProps == nil {
		superProps = props.NewSuperProperties()
	}

	capabilities := config.Capabilities
	if capabilities == 0 {
		capabilities = props.DefaultCapabilities().Value()
	}

	if superProps.ClientBuildNumber == 0 {
		superProps.UpdateBuildNumber()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Gateway{
		Token:           config.Token,
		SuperProperties: superProps,
		Capabilities:    capabilities,
		handlers:        make(map[string][]EventHandler),
		ctx:             ctx,
		cancel:          cancel,
		closeCh:         make(chan struct{}),
		Debug:           config.Debug,
	}, nil
}

// Connect connects to the gateway.
func (g *Gateway) Connect() error {
	return g.connect(false)
}

func (g *Gateway) connect(resume bool) error {
	g.connMu.Lock()
	defer g.connMu.Unlock()

	gatewayURL := DefaultGatewayURL
	if resume && g.ResumeURL != "" {
		gatewayURL = g.ResumeURL
	}

	url := fmt.Sprintf("%s?v=%d&encoding=json&compress=zlib", gatewayURL, GatewayVersion)

	if g.Debug {
		fmt.Printf("[Gateway] Connecting to %s\n", url)
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		ReadBufferSize:   16384,
		WriteBufferSize:  16384,
	}

	headers := http.Header{
		"Accept-Encoding": []string{"gzip, deflate, br"},
		"Accept-Language": []string{"en-US,en;q=0.9"},
		"Cache-Control":   []string{"no-cache"},
		"Origin":          []string{"https://discord.com"},
		"Pragma":          []string{"no-cache"},
		"User-Agent":      []string{g.SuperProperties.UserAgent()},
	}

	if g.conn != nil {
		_ = g.conn.Close()
		g.conn = nil
	}

	conn, _, err := dialer.Dial(url, headers)
	if err != nil {
		return fmt.Errorf("failed to connect to gateway: %w", err)
	}

	g.conn = conn
	g.resuming.Store(resume)

	go g.readLoop()

	return nil
}

func (g *Gateway) readLoop() {
	if g.Debug {
		fmt.Println("[Gateway] Read loop started")
	}

	defer func() {
		if g.Debug {
			fmt.Println("[Gateway] Read loop stopped")
		}
		g.connected.Store(false)
		if g.heartbeat != nil {
			g.heartbeat.Stop()
		}
	}()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-g.closeCh:
			return
		default:
		}

		mt, message, err := g.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}

			if g.OnError != nil {
				g.OnError(err)
			}

			g.handleReconnect()
			return
		}

		if g.Debug {
			fmt.Printf("[Gateway] Received message: type=%d len=%d\n", mt, len(message))
		}

		var data []byte
		if mt == websocket.BinaryMessage {
			data, err = g.decompress(message)
			if err != nil {
				if g.OnError != nil {
					g.OnError(fmt.Errorf("failed to decompress: %w", err))
				}
				continue
			}
		} else {
			data = message
		}

		if data == nil {
			continue
		}

		var payload PayloadWrapper
		if err := json.Unmarshal(data, &payload); err != nil {
			if g.OnError != nil {
				g.OnError(fmt.Errorf("failed to parse payload: %w", err))
			}
			continue
		}

		if g.handlePayload(&payload) {
			return
		}
	}
}

func (g *Gateway) decompress(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var out bytes.Buffer
	_, err = io.Copy(&out, reader)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// handlePayload processes a gateway payload. Returns true when the read loop should exit.
func (g *Gateway) handlePayload(payload *PayloadWrapper) bool {
	if g.Debug {
		fmt.Printf("[Gateway] Received op=%d t=%s\n", payload.Op, payload.T)
	}

	switch payload.Op {
	case OpcodeDispatch:
		g.handleDispatch(payload)

	case OpcodeHeartbeat:
		g.sendHeartbeat()

	case OpcodeReconnect:
		g.handleReconnect()
		return true

	case OpcodeInvalidSession:
		g.handleInvalidSession(payload)
		return true

	case OpcodeHello:
		g.handleHello(payload)

	case OpcodeHeartbeatAck:
		if g.heartbeat != nil {
			g.heartbeat.Ack()
		}
	}

	return false
}

func (g *Gateway) handleDispatch(payload *PayloadWrapper) {
	if payload.S != nil {
		g.Sequence.Store(*payload.S)
	}

	data := payload.D

	if g.Debug {
		fmt.Printf("[Gateway] Dispatching event: %s (len=%d)\n", payload.T, len(data))
	}

	switch payload.T {
	case EventReady:
		if g.Debug {
			fmt.Println("[Gateway] Processing READY event...")
		}
		var ready ReadyEvent
		if err := json.Unmarshal(data, &ready); err == nil {
			g.SessionID = ready.SessionID
			g.ResumeURL = ready.ResumeGatewayURL
			g.ReadyData = &ready
			g.connected.Store(true)
			g.reconnects = 0
			if g.Debug {
				fmt.Printf("[Gateway] READY success: User=%s, Sessions=%s, Guilds=%d\n", ready.User.Username, ready.SessionID, len(ready.Guilds))
			}
		} else if g.Debug {
			fmt.Printf("[Gateway] Failed to unmarshal READY event: %v\n", err)
		}

	case EventResumed:
		g.connected.Store(true)
		g.resuming.Store(false)
		g.reconnects = 0
	}

	g.dispatchEvent(payload.T, data)
}

func (g *Gateway) handleHello(payload *PayloadWrapper) {
	data := payload.D
	var hello HelloPayload
	if err := json.Unmarshal(data, &hello); err != nil {
		if g.OnError != nil {
			g.OnError(fmt.Errorf("failed to parse hello: %w", err))
		}
		return
	}

	interval := time.Duration(hello.HeartbeatInterval) * time.Millisecond
	seq := g.Sequence.Load()
	g.heartbeat = NewHeartbeat(interval, &seq, func(p *PayloadWrapper) error {
		return g.Send(p)
	})
	g.heartbeat.Start()

	if g.resuming.Load() && g.SessionID != "" {
		g.sendResume()
	} else {
		g.sendIdentify()
	}
}

func (g *Gateway) handleInvalidSession(payload *PayloadWrapper) {
	var resumable bool
	_ = json.Unmarshal(payload.D, &resumable)

	if g.heartbeat != nil {
		g.heartbeat.Stop()
	}

	time.Sleep(1*time.Second + time.Duration(g.reconnects)*time.Second)

	if resumable && g.SessionID != "" {
		g.connect(true)
	} else {
		g.SessionID = ""
		g.Sequence.Store(0)
		g.connect(false)
	}
}

func (g *Gateway) handleReconnect() {
	if g.heartbeat != nil {
		g.heartbeat.Stop()
	}

	g.connected.Store(false)
	g.reconnects++

	if g.reconnects > MaxReconnectAttempts {
		if g.OnError != nil {
			g.OnError(errors.New("max reconnect attempts reached"))
		}
		return
	}

	delay := ReconnectDelay * time.Duration(1<<uint(g.reconnects-1))
	time.Sleep(delay)

	g.connect(true)
}

func (g *Gateway) sendIdentify() error {
	payload := BuildIdentifyPayload(g.Token, g.SuperProperties, g.Capabilities)
	return g.Send(payload)
}

func (g *Gateway) sendResume() error {
	payload := BuildResumePayload(g.Token, g.SessionID, g.Sequence.Load())
	return g.Send(payload)
}

func (g *Gateway) sendHeartbeat() error {
	seq := g.Sequence.Load()
	payload := BuildHeartbeatPayload(&seq)
	return g.Send(payload)
}

// Send sends a payload to the gateway.
func (g *Gateway) Send(payload *PayloadWrapper) error {
	g.connMu.Lock()
	defer g.connMu.Unlock()

	if g.conn == nil {
		return errors.New("not connected")
	}

	if g.Debug {
		data, _ := json.Marshal(payload)
		fmt.Printf("[Gateway] Sending: %s\n", string(data))
	}

	return g.conn.WriteJSON(payload)
}

// On registers an event handler.
func (g *Gateway) On(event string, handler EventHandler) {
	g.handlersMu.Lock()
	defer g.handlersMu.Unlock()

	g.handlers[event] = append(g.handlers[event], handler)
}

func (g *Gateway) dispatchEvent(event string, data json.RawMessage) {
	g.handlersMu.RLock()
	defer g.handlersMu.RUnlock()

	if handlers, ok := g.handlers[event]; ok {
		for _, handler := range handlers {
			go handler(event, data)
		}
	}

	if handlers, ok := g.handlers["*"]; ok {
		for _, handler := range handlers {
			go handler(event, data)
		}
	}
}

// Close closes the gateway connection.
func (g *Gateway) Close() error {
	g.closeOnce.Do(func() {
		close(g.closeCh)
		g.cancel()

		if g.heartbeat != nil {
			g.heartbeat.Stop()
		}

		g.connMu.Lock()
		if g.conn != nil {
			g.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			g.conn.Close()
		}
		g.connMu.Unlock()
	})

	return nil
}

// IsConnected returns whether the gateway is connected.
func (g *Gateway) IsConnected() bool {
	return g.connected.Load()
}

// Latency returns the current heartbeat latency.
func (g *Gateway) Latency() time.Duration {
	if g.heartbeat == nil {
		return 0
	}
	return g.heartbeat.Latency()
}

// UpdatePresence updates the user's presence.
func (g *Gateway) UpdatePresence(status string, activities []interface{}, afk bool) error {
	data := map[string]interface{}{
		"status":     status,
		"activities": activities,
		"afk":        afk,
		"since":      0,
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	payload := PayloadWrapper{
		Op: 3,
		D:  raw,
	}
	return g.Send(&payload)
}

// UpdateVoiceState updates the user's voice state.
func (g *Gateway) UpdateVoiceState(guildID, channelID string, mute, deaf bool) error {
	data := map[string]interface{}{
		"guild_id":   guildID,
		"channel_id": channelID,
		"self_mute":  mute,
		"self_deaf":  deaf,
	}
	if channelID == "" {
		data["channel_id"] = nil
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}

	payload := PayloadWrapper{
		Op: 4,
		D:  raw,
	}
	return g.Send(&payload)
}

// SetStatus sets the user's status.
func (g *Gateway) SetStatus(status string) error {
	return g.UpdatePresence(status, nil, false)
}
