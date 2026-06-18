package voice

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// VoiceConnection represents a connection to a Discord voice server.
type VoiceConnection struct {
	sync.RWMutex

	GuildID   string
	ChannelID string
	UserID    string
	SessionID string
	Token     string
	Endpoint  string

	wsConn *websocket.Conn

	udpConn *net.UDPConn
	ssrc    uint32
	address string
	port    int

	ctx    context.Context
	cancel context.CancelFunc

	heartbeatInterval time.Duration
	lastHeartbeatAck  time.Time
	ready             chan bool
	secretKey         [32]byte
	sequence          uint16
	timestamp         uint32

	OnDisconnect func(err error)
	Debug        bool
}

// NewVoiceConnection creates a new voice connection instance.
func NewVoiceConnection(guildID, channelID, userID, sessionID, token, endpoint string) *VoiceConnection {
	ctx, cancel := context.WithCancel(context.Background())
	return &VoiceConnection{
		GuildID:   guildID,
		ChannelID: channelID,
		UserID:    userID,
		SessionID: sessionID,
		Token:     token,
		Endpoint:  endpoint,
		ctx:       ctx,
		cancel:    cancel,
		ready:     make(chan bool, 1),
	}
}

// Connect establishes the voice gateway connection.
func (v *VoiceConnection) Connect() error {
	url := "wss://" + v.Endpoint + "/?v=4"

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}

	v.Lock()
	v.wsConn = conn
	v.Unlock()

	go v.readLoop()

	select {
	case <-v.ready:
		return nil
	case <-time.After(10 * time.Second):
		v.Close()
		return context.DeadlineExceeded
	case <-v.ctx.Done():
		return v.ctx.Err()
	}
}

func (v *VoiceConnection) readLoop() {
	defer v.Close()

	for {
		select {
		case <-v.ctx.Done():
			return
		default:
			_, message, err := v.wsConn.ReadMessage()
			if err != nil {
				if v.OnDisconnect != nil {
					v.OnDisconnect(err)
				}
				return
			}

			var p struct {
				Op int             `json:"op"`
				D  json.RawMessage `json:"d"`
			}

			if err := json.Unmarshal(message, &p); err != nil {
				continue
			}

			switch p.Op {
			case OpReady:
				v.handleReady(p.D)
			case OpHello:
				v.handleHello(p.D)
			case OpSessionDescription:
				v.handleSessionDescription(p.D)
			case OpHeartbeatACK:
				v.lastHeartbeatAck = time.Now()
			}
		}
	}
}

func (v *VoiceConnection) handleHello(data json.RawMessage) {
	var d struct {
		HeartbeatInterval float64 `json:"heartbeat_interval"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return
	}

	v.heartbeatInterval = time.Duration(d.HeartbeatInterval) * time.Millisecond
	go v.heartbeat()

	v.identify()
}

func (v *VoiceConnection) identify() {
	payload := map[string]interface{}{
		"op": OpIdentify,
		"d": map[string]interface{}{
			"server_id":  v.GuildID,
			"user_id":    v.UserID,
			"session_id": v.SessionID,
			"token":      v.Token,
		},
	}
	v.sendJSON(payload)
}

func (v *VoiceConnection) handleReady(data json.RawMessage) {
	var d struct {
		SSRC int    `json:"ssrc"`
		IP   string `json:"ip"`
		Port int    `json:"port"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return
	}

	v.Lock()
	v.ssrc = uint32(d.SSRC)
	v.address = d.IP
	v.port = d.Port
	v.Unlock()

	go func() {
		localIP, localPort, err := v.discoverIP()
		if err != nil {
			v.Close()
			return
		}

		v.selectProtocol(localIP, localPort)
	}()
}

func (v *VoiceConnection) discoverIP() (string, int, error) {
	v.RLock()
	addr := fmt.Sprintf("%s:%d", v.address, v.port)
	ssrc := v.ssrc
	v.RUnlock()

	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return "", 0, err
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return "", 0, err
	}
	defer conn.Close()

	v.Lock()
	v.udpConn = conn
	v.Unlock()

	packet := make([]byte, 74)
	packet[0] = 0
	packet[1] = 1
	packet[2] = 0
	packet[3] = 70
	packet[4] = byte(ssrc >> 24)
	packet[5] = byte(ssrc >> 16)
	packet[6] = byte(ssrc >> 8)
	packet[7] = byte(ssrc)

	if _, err := conn.Write(packet); err != nil {
		return "", 0, err
	}

	resp := make([]byte, 74)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, _, err := conn.ReadFromUDP(resp)
	if err != nil {
		return "", 0, err
	}

	if n != 74 {
		return "", 0, fmt.Errorf("invalid discovery packet length: %d", n)
	}

	ipEnd := 8
	for i := 8; i < 72; i++ {
		if resp[i] == 0 {
			ipEnd = i
			break
		}
	}
	ip := string(resp[8:ipEnd])

	port := int(resp[72]) | int(resp[73])<<8

	return ip, port, nil
}

func (v *VoiceConnection) selectProtocol(ip string, port int) {
	payload := map[string]interface{}{
		"op": OpSelectProtocol,
		"d": map[string]interface{}{
			"protocol": "udp",
			"data": map[string]interface{}{
				"address": ip,
				"port":    port,
				"mode":    "xsalsa20_poly1305",
			},
		},
	}
	v.sendJSON(payload)
}

func (v *VoiceConnection) handleSessionDescription(data json.RawMessage) {
	var d struct {
		Mode      string   `json:"mode"`
		SecretKey [32]byte `json:"secret_key"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return
	}

	v.Lock()
	v.secretKey = d.SecretKey
	v.Unlock()
}

// Speaking sends a speaking status update.
func (v *VoiceConnection) Speaking(speaking bool) error {
	v.Lock()
	defer v.Unlock()

	var flags int
	if speaking {
		flags = 1 << 0
	}

	payload := map[string]interface{}{
		"op": OpSpeaking,
		"d": map[string]interface{}{
			"speaking": flags,
			"delay":    0,
			"ssrc":     v.ssrc,
		},
	}
	return v.sendJSON(payload)
}

// SendOpus sends an Opus packet over UDP.
func (v *VoiceConnection) SendOpus(opus []byte) error {
	v.Lock()
	v.sequence++
	v.timestamp += 960

	seq := v.sequence
	ts := v.timestamp
	ssrc := v.ssrc
	key := v.secretKey
	udpConn := v.udpConn
	v.Unlock()

	if udpConn == nil {
		return fmt.Errorf("UDP connection not established")
	}

	header := CreateHeader(seq, ts, ssrc)
	packet := EncryptXSalsa20Poly1305(opus, header, &key)

	_, err := udpConn.Write(packet)
	return err
}

func (v *VoiceConnection) heartbeat() {
	ticker := time.NewTicker(v.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-v.ctx.Done():
			return
		case <-ticker.C:
			v.sendJSON(map[string]interface{}{
				"op": OpHeartbeat,
				"d":  time.Now().Unix(),
			})
		}
	}
}

func (v *VoiceConnection) sendJSON(v2 interface{}) error {
	v.Lock()
	defer v.Unlock()
	if v.wsConn == nil {
		return websocket.ErrCloseSent
	}
	return v.wsConn.WriteJSON(v2)
}

// Close closes the voice connection.
func (v *VoiceConnection) Close() {
	v.cancel()
	v.Lock()
	defer v.Unlock()
	if v.wsConn != nil {
		v.wsConn.Close()
		v.wsConn = nil
	}
	if v.udpConn != nil {
		v.udpConn.Close()
		v.udpConn = nil
	}
}
