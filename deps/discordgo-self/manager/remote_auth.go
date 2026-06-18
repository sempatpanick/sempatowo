package manager

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	RemoteAuthURL = "wss://remote-auth-gateway.discord.gg/?v=2"
	OriginURL     = "https://discord.com"
)

// RemoteAuthClient handles the authentication flow via QR code scanning.
type RemoteAuthClient struct {
	conn       *websocket.Conn
	privateKey *rsa.PrivateKey
	publicKey  []byte

	SuperProps *SuperProperties

	pendingTicket string

	closeCh chan struct{}
	once    sync.Once

	OnFingerprint func(fingerprint string)
	OnUserData    func(user *RemoteUser)
	OnToken       func(token string)
	OnCaptcha     func(captcha *CaptchaInfo) (solution string)
}

// CaptchaInfo contains captcha challenge data.
type CaptchaInfo struct {
	Service   string `json:"captcha_service"`
	Sitekey   string `json:"captcha_sitekey"`
	SessionID string `json:"captcha_session_id"`
	RqData    string `json:"captcha_rqdata"`
	RqToken   string `json:"captcha_rqtoken"`
}

// RemoteUser represents a user via remote auth.
type RemoteUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	GlobalName    string `json:"global_name"`
	Avatar        string `json:"avatar"`
}

// SuperProperties for client spoofing.
type SuperProperties struct {
	OS                string  `json:"os"`
	Browser           string  `json:"browser"`
	Device            string  `json:"device"`
	SystemLocale      string  `json:"system_locale"`
	BrowserUserAgent  string  `json:"browser_user_agent,omitempty"`
	BrowserVersion    string  `json:"browser_version,omitempty"`
	OSVersion         string  `json:"os_version,omitempty"`
	ReleaseChannel    string  `json:"release_channel"`
	ClientBuildNumber int     `json:"client_build_number"`
	ClientEventSource *string `json:"client_event_source"`
}

// Encode returns the base64-encoded JSON.
func (sp *SuperProperties) Encode() string {
	data, _ := json.Marshal(sp)
	return base64.StdEncoding.EncodeToString(data)
}

// IOSProps returns iOS properties.
func IOSProps() *SuperProperties {
	return &SuperProperties{
		OS:                "iOS",
		Browser:           "Discord iOS",
		Device:            "iPhone",
		SystemLocale:      "en-US",
		BrowserUserAgent:  "",
		BrowserVersion:    "251.0",
		OSVersion:         "17.4.1",
		ReleaseChannel:    "stable",
		ClientBuildNumber: 63529,
		ClientEventSource: nil,
	}
}

// AndroidProps returns Android properties.
func AndroidProps() *SuperProperties {
	return &SuperProperties{
		OS:                "Android",
		Browser:           "Discord Android",
		Device:            "samsung SM-S918B",
		SystemLocale:      "en-US",
		BrowserUserAgent:  "",
		BrowserVersion:    "251.7",
		OSVersion:         "14",
		ReleaseChannel:    "stable",
		ClientBuildNumber: 251007,
		ClientEventSource: nil,
	}
}

// WindowsChromeProps returns Windows Chrome properties.
func WindowsChromeProps() *SuperProperties {
	return &SuperProperties{
		OS:                "Windows",
		Browser:           "Chrome",
		Device:            "",
		SystemLocale:      "en-US",
		BrowserUserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		BrowserVersion:    "131.0.0.0",
		OSVersion:         "10",
		ReleaseChannel:    "stable",
		ClientBuildNumber: 345678,
		ClientEventSource: nil,
	}
}

// LinuxChromeProps returns Linux Chrome properties.
func LinuxChromeProps() *SuperProperties {
	return &SuperProperties{
		OS:                "Linux",
		Browser:           "Chrome",
		Device:            "",
		SystemLocale:      "en-US",
		BrowserUserAgent:  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		BrowserVersion:    "131.0.0.0",
		OSVersion:         "x86_64",
		ReleaseChannel:    "stable",
		ClientBuildNumber: 345678,
		ClientEventSource: nil,
	}
}

// MacOSChromeProps returns macOS Chrome properties.
func MacOSChromeProps() *SuperProperties {
	return &SuperProperties{
		OS:                "Mac OS X",
		Browser:           "Chrome",
		Device:            "",
		SystemLocale:      "en-US",
		BrowserUserAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		BrowserVersion:    "131.0.0.0",
		OSVersion:         "10.15.7",
		ReleaseChannel:    "stable",
		ClientBuildNumber: 345678,
		ClientEventSource: nil,
	}
}

// DiscordDesktopProps returns Discord Desktop properties.
func DiscordDesktopProps() *SuperProperties {
	return &SuperProperties{
		OS:                "Windows",
		Browser:           "Discord Client",
		Device:            "",
		SystemLocale:      "en-US",
		BrowserUserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) discord/1.0.9168 Chrome/128.0.6613.186 Electron/32.2.2 Safari/537.36",
		BrowserVersion:    "32.2.2",
		OSVersion:         "10.0.22631",
		ReleaseChannel:    "stable",
		ClientBuildNumber: 345678,
		ClientEventSource: nil,
	}
}

// NewRemoteAuthClient creates a new remote auth client.
func NewRemoteAuthClient() (*RemoteAuthClient, error) {
	return NewRemoteAuthClientWithPlatform(nil)
}

// NewRemoteAuthClientWithPlatform creates a client with custom platform spoofing.
func NewRemoteAuthClientWithPlatform(superProps *SuperProperties) (*RemoteAuthClient, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	pubASN1, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	if superProps == nil {
		superProps = &SuperProperties{
			OS:                "Windows",
			Browser:           "Chrome",
			Device:            "",
			SystemLocale:      "en-US",
			BrowserUserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
			BrowserVersion:    "131.0.0.0",
			OSVersion:         "10",
			ReleaseChannel:    "stable",
			ClientBuildNumber: 345678,
			ClientEventSource: nil,
		}
	}

	return &RemoteAuthClient{
		privateKey: privateKey,
		publicKey:  pubASN1,
		SuperProps: superProps,
		closeCh:    make(chan struct{}),
	}, nil
}

// Start connects to the remote auth gateway.
func (c *RemoteAuthClient) Start() error {
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	headers := http.Header{}
	headers.Set("Origin", OriginURL)

	if c.SuperProps != nil {
		headers.Set("X-Super-Properties", c.SuperProps.Encode())
		if c.SuperProps.BrowserUserAgent != "" {
			headers.Set("User-Agent", c.SuperProps.BrowserUserAgent)
		}
	}

	conn, _, err := dialer.Dial(RemoteAuthURL, headers)
	if err != nil {
		return fmt.Errorf("failed to connect to remote auth: %w", err)
	}
	c.conn = conn
	fmt.Printf("[DEBUG] Connected to Remote Auth Gateway (Spoofing: %s/%s)\n", c.SuperProps.OS, c.SuperProps.Browser)

	go c.readLoop()

	return nil
}

func (c *RemoteAuthClient) readLoop() {
	defer c.conn.Close()
	fmt.Println("[DEBUG] ReadLoop started, waiting for messages...")

	for {
		select {
		case <-c.closeCh:
			return
		default:
		}

		_, message, err := c.conn.ReadMessage()
		if err != nil {
			fmt.Printf("[DEBUG] ReadMessage error: %v\n", err)
			return
		}

		fmt.Printf("[DEBUG] Received message: %s\n", string(message))

		var payload struct {
			Op   string          `json:"op"`
			Data json.RawMessage `json:"data,omitempty"`
		}

		if err := json.Unmarshal(message, &payload); err != nil {
			fmt.Printf("[DEBUG] JSON parse error: %v\n", err)
			continue
		}

		fmt.Printf("[DEBUG] Op: %s\n", payload.Op)

		switch payload.Op {
		case "hello":
			var helloData struct {
				HeartbeatInterval int `json:"heartbeat_interval"`
			}
			if err := json.Unmarshal(message, &helloData); err != nil {
				fmt.Printf("Failed to parse hello: %v\n", err)
				continue
			}
			if helloData.HeartbeatInterval > 0 {
				go c.heartbeatLoop(time.Duration(helloData.HeartbeatInterval) * time.Millisecond)
			} else {
				fmt.Println("Warning: Interval 0, using default 40s")
				go c.heartbeatLoop(40 * time.Second)
			}

			c.sendInit()

		case "nonce_proof":
			var nonceData struct {
				Nonce string `json:"encrypted_nonce"`
			}
			if err := json.Unmarshal(message, &nonceData); err != nil || nonceData.Nonce == "" {
				fmt.Println("Failed to parse nonce_proof or empty nonce")
				continue
			}
			c.handleNonceProof(nonceData.Nonce)

		case "pending_remote_init":
			var initData struct {
				Fingerprint string `json:"fingerprint"`
			}
			if err := json.Unmarshal(message, &initData); err != nil || initData.Fingerprint == "" {
				fmt.Println("Failed to parse fingerprint")
				continue
			}

			qrURL := fmt.Sprintf("https://discord.com/ra/%s", initData.Fingerprint)
			if c.OnFingerprint != nil {
				c.OnFingerprint(qrURL)
			}

		case "pending_ticket":
			var ticketData struct {
				EncryptedUserPayload string `json:"encrypted_user_payload"`
			}
			if err := json.Unmarshal(message, &ticketData); err == nil && ticketData.EncryptedUserPayload != "" {
				fmt.Println("[DEBUG] User scanned QR code, waiting for confirmation...")
			}

		case "pending_login":
			var loginData struct {
				Ticket string `json:"ticket"`
			}
			if err := json.Unmarshal(message, &loginData); err == nil && loginData.Ticket != "" {
				fmt.Println("[DEBUG] Login confirmed, exchanging ticket for token...")
				c.exchangeTicket(loginData.Ticket)
			}

		case "pending_finish":
			var finishData struct {
				EncryptedToken string `json:"encrypted_token"`
			}
			if err := json.Unmarshal(message, &finishData); err == nil && finishData.EncryptedToken != "" {
				c.handleFinish(finishData.EncryptedToken)
			}
		}
	}
}

func (c *RemoteAuthClient) sendInit() {
	encodedKey := base64.StdEncoding.EncodeToString(c.publicKey)
	payload := map[string]interface{}{
		"op":                 "init",
		"encoded_public_key": encodedKey,
	}
	c.conn.WriteJSON(payload)
}

func (c *RemoteAuthClient) handleNonceProof(encryptedNonce string) {
	fmt.Println("[DEBUG] handleNonceProof called")

	nonceBytes, err := base64.StdEncoding.DecodeString(encryptedNonce)
	if err != nil {
		nonceBytes, err = base64.RawStdEncoding.DecodeString(encryptedNonce)
		if err != nil {
			fmt.Printf("[DEBUG] Base64 decode failed: %v\n", err)
			return
		}
	}
	fmt.Printf("[DEBUG] Nonce bytes length: %d\n", len(nonceBytes))

	hash := sha256.New()
	decryptedNonce, err := rsa.DecryptOAEP(hash, rand.Reader, c.privateKey, nonceBytes, nil)
	if err != nil {
		fmt.Printf("[DEBUG] SHA256 decrypt failed, trying SHA1: %v\n", err)
		hashSHA1 := sha1.New()
		decryptedNonce, err = rsa.DecryptOAEP(hashSHA1, rand.Reader, c.privateKey, nonceBytes, nil)
		if err != nil {
			fmt.Printf("[DEBUG] Decryption failed (Nonce) SHA256/SHA1: %v. Bytes: %d\n", err, len(nonceBytes))
			return
		}
	}
	fmt.Printf("[DEBUG] Decrypted nonce length: %d\n", len(decryptedNonce))

	proofHash := sha256.Sum256(decryptedNonce)
	proof := base64.RawURLEncoding.EncodeToString(proofHash[:])
	fmt.Printf("[DEBUG] Sending proof: %s\n", proof)

	payload := map[string]interface{}{
		"op":    "nonce_proof",
		"proof": proof,
	}
	if err := c.conn.WriteJSON(payload); err != nil {
		fmt.Printf("[DEBUG] Failed to send proof: %v\n", err)
	}
}

func (c *RemoteAuthClient) handleFinish(encryptedToken string) {
	tokenBytes, err := base64.StdEncoding.DecodeString(encryptedToken)
	if err != nil {
		tokenBytes, err = base64.RawStdEncoding.DecodeString(encryptedToken)
		if err != nil {
			fmt.Printf("Base64 decode failed (Token): %v\n", err)
			return
		}
	}

	hash := sha256.New()
	decryptedToken, err := rsa.DecryptOAEP(hash, rand.Reader, c.privateKey, tokenBytes, nil)
	if err != nil {
		hashSHA1 := sha1.New()
		decryptedToken, err = rsa.DecryptOAEP(hashSHA1, rand.Reader, c.privateKey, tokenBytes, nil)
		if err != nil {
			fmt.Printf("Token decryption failed SHA256/SHA1: %v. Bytes: %d\n", err, len(tokenBytes))
			return
		}
	}

	token := string(decryptedToken)

	if len(token) > 20 {
		fmt.Printf("[DEBUG] Token decrypted successfully. Length: %d, Preview: %s...%s\n",
			len(token), token[:10], token[len(token)-5:])
	} else {
		fmt.Printf("[DEBUG] Token decrypted but seems short/invalid. Length: %d, Content: %s\n", len(token), token)
	}

	if c.OnToken != nil {
		c.OnToken(token)
	}
	c.Close()
}

func (c *RemoteAuthClient) heartbeatLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.conn.WriteJSON(map[string]string{"op": "heartbeat"})
		case <-c.closeCh:
			return
		}
	}
}

// exchangeTicket exchanges the ticket for an actual token.
func (c *RemoteAuthClient) exchangeTicket(ticket string) {
	body := fmt.Sprintf(`{"ticket":"%s"}`, ticket)

	req, err := http.NewRequest("POST", "https://discord.com/api/v9/users/@me/remote-auth/login", bytes.NewBufferString(body))
	if err != nil {
		fmt.Printf("[DEBUG] Failed to create request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://discord.com")

	if c.SuperProps != nil {
		req.Header.Set("X-Super-Properties", c.SuperProps.Encode())
		if c.SuperProps.BrowserUserAgent != "" {
			req.Header.Set("User-Agent", c.SuperProps.BrowserUserAgent)
		} else {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[DEBUG] HTTP request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("[DEBUG] Token exchange response: %s\n", string(respBody))

	var captchaCheck struct {
		CaptchaKey     []string `json:"captcha_key"`
		CaptchaService string   `json:"captcha_service"`
		CaptchaSitekey string   `json:"captcha_sitekey"`
		CaptchaSession string   `json:"captcha_session_id"`
		CaptchaRqData  string   `json:"captcha_rqdata"`
		CaptchaRqToken string   `json:"captcha_rqtoken"`
	}
	if err := json.Unmarshal(respBody, &captchaCheck); err == nil && len(captchaCheck.CaptchaKey) > 0 {
		fmt.Println("\n[!] CAPTCHA REQUIRED!")
		fmt.Printf("[!] Service: %s\n", captchaCheck.CaptchaService)
		fmt.Printf("[!] Sitekey: %s\n", captchaCheck.CaptchaSitekey)

		if c.OnCaptcha != nil {
			captchaInfo := &CaptchaInfo{
				Service:   captchaCheck.CaptchaService,
				Sitekey:   captchaCheck.CaptchaSitekey,
				SessionID: captchaCheck.CaptchaSession,
				RqData:    captchaCheck.CaptchaRqData,
				RqToken:   captchaCheck.CaptchaRqToken,
			}
			solution := c.OnCaptcha(captchaInfo)
			if solution != "" {
				fmt.Println("[+] Captcha solved! Retrying...")
				c.pendingTicket = ticket
				c.exchangeTicketWithCaptcha(ticket, solution, captchaInfo.RqToken)
				return
			}
		}

		fmt.Println("[!] No captcha solver configured. Login failed.")
		return
	}

	var result struct {
		EncryptedToken string `json:"encrypted_token"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil || result.EncryptedToken == "" {
		fmt.Printf("[DEBUG] Failed to parse token response: %v\n", err)
		return
	}

	c.handleFinish(result.EncryptedToken)
}

// Close closes the connection.
func (c *RemoteAuthClient) Close() {
	c.once.Do(func() {
		close(c.closeCh)
		if c.conn != nil {
			c.conn.Close()
		}
	})
}

func (c *RemoteAuthClient) exchangeTicketWithCaptcha(ticket, captchaSolution, rqToken string) {
	body := fmt.Sprintf(`{"ticket":"%s","captcha_key":"%s","captcha_rqtoken":"%s"}`, ticket, captchaSolution, rqToken)

	req, err := http.NewRequest("POST", "https://discord.com/api/v9/users/@me/remote-auth/login", bytes.NewBufferString(body))
	if err != nil {
		fmt.Printf("[DEBUG] Failed to create captcha retry request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://discord.com")

	if c.SuperProps != nil {
		req.Header.Set("X-Super-Properties", c.SuperProps.Encode())
		if c.SuperProps.BrowserUserAgent != "" {
			req.Header.Set("User-Agent", c.SuperProps.BrowserUserAgent)
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[DEBUG] Captcha retry HTTP request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("[DEBUG] Captcha retry status: %d\n", resp.StatusCode)
	fmt.Printf("[DEBUG] Captcha retry response: %s\n", string(respBody))

	if resp.StatusCode != 200 {
		var errResp struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
			Errors  any    `json:"errors"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Message != "" {
			fmt.Printf("[!] Discord Error: %s (code: %d)\n", errResp.Message, errResp.Code)
		}
		return
	}

	var result struct {
		EncryptedToken string `json:"encrypted_token"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil || result.EncryptedToken == "" {
		fmt.Printf("[DEBUG] Failed to parse captcha retry response: %v\n", err)
		fmt.Printf("[DEBUG] Response body was: %s\n", string(respBody))
		return
	}

	c.handleFinish(result.EncryptedToken)
}
