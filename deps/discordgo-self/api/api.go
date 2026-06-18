package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/hytams/discordgo-self/props"
	"github.com/hytams/discordgo-self/types"
)

const (
	// APIVersion is the Discord API version
	APIVersion = 9

	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second
)

// BaseURL is the Discord API base URL
var BaseURL = "https://discord.com/api/v9"

// Client represents the Discord REST API client
type Client struct {
	Token           string
	SuperProperties *props.SuperProperties
	HTTPClient      *http.Client
	RateLimiter     *RateLimiter

	userAgent string
	Locale    string
	Timezone  string
	Debug     bool
}

// ClientConfig holds configuration for the API client
type ClientConfig struct {
	Token           string
	SuperProperties *props.SuperProperties
	HTTPClient      *http.Client
	Locale          string
	Timezone        string
	Debug           bool
}

// NewClient creates a new API client
func NewClient(config ClientConfig) (*Client, error) {
	if config.Token == "" {
		return nil, fmt.Errorf("token is required")
	}

	superProps := config.SuperProperties
	if superProps == nil {
		superProps = props.NewSuperProperties()
	}

	if superProps.ClientBuildNumber == 0 {
		if err := superProps.UpdateBuildNumber(); err != nil {
			superProps.ClientBuildNumber = 345678
		}
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		transport := ConfigureTransport()
		httpClient = &http.Client{
			Transport: transport,
			Timeout:   DefaultTimeout,
		}
	}

	locale := config.Locale
	if locale == "" {
		locale = "en-US"
	}

	return &Client{
		Token:           config.Token,
		SuperProperties: superProps,
		HTTPClient:      httpClient,
		RateLimiter:     NewRateLimiter(),
		userAgent:       superProps.UserAgent(),
		Locale:          locale,
		Timezone:        config.Timezone,
		Debug:           config.Debug,
	}, nil
}

// Request represents an API request
type Request struct {
	Method      string
	Endpoint    string
	Body        interface{}
	ContentType string
	Reason      string // For audit log reason
}

// Do performs an API request
func (c *Client) Do(ctx context.Context, req Request) (*http.Response, error) {
	url := BaseURL + req.Endpoint

	var body io.Reader
	if req.Body != nil {
		data, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setHeaders(httpReq, req)

	bucket := c.RateLimiter.GetBucket(req.Endpoint)
	if err := bucket.Wait(ctx); err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Update rate limiter
	bucket.Update(resp.Header)

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		var rateLimitResp struct {
			RetryAfter float64 `json:"retry_after"`
			Global     bool    `json:"global"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&rateLimitResp); err == nil {
			resp.Body.Close()

			retryAfter := time.Duration(rateLimitResp.RetryAfter * float64(time.Second))

			if rateLimitResp.Global {
				c.RateLimiter.SetGlobal(retryAfter)
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryAfter):
			}

			// Retry the request
			return c.Do(ctx, req)
		}
	}

	return resp, nil
}

// DoJSON performs an API request and decodes JSON response
func (c *Client) DoJSON(ctx context.Context, req Request, v interface{}) error {
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check for error status
	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}

	if v != nil && resp.StatusCode != http.StatusNoContent {
		// Read body to detect compression
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		var reader io.Reader = bytes.NewReader(bodyBytes)

		// Check for gzip compression - either by header or by magic bytes (0x1f, 0x8b)
		isGzip := resp.Header.Get("Content-Encoding") == "gzip"
		if !isGzip && len(bodyBytes) >= 2 && bodyBytes[0] == 0x1f && bodyBytes[1] == 0x8b {
			isGzip = true
		}

		if isGzip {
			gzReader, err := gzip.NewReader(bytes.NewReader(bodyBytes))
			if err != nil {
				return fmt.Errorf("failed to create gzip reader: %w", err)
			}
			defer gzReader.Close()
			reader = gzReader
		}

		if err := json.NewDecoder(reader).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// DoMultipart performs a multipart API request
func (c *Client) DoMultipart(ctx context.Context, endpoint, method string, fields map[string]string, fileField string, fileData []byte, filename string) ([]byte, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add fields
	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}

	// Add file
	part, err := writer.CreateFormFile(fileField, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = part.Write(fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Set authentication and browser headers
	c.setHeaders(req, Request{ContentType: writer.FormDataContentType()})

	// Wait for rate limiter
	bucket := c.RateLimiter.GetBucket(endpoint)
	if err := bucket.Wait(ctx); err != nil {
		return nil, err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bucket.Update(resp.Header) // Update bucket

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return respBody, nil
}

// setHeaders sets the required headers for the request
func (c *Client) setHeaders(req *http.Request, apiReq Request) {
	// Authorization - user token (no Bot prefix)
	req.Header.Set("Authorization", c.Token)

	// Content-Type
	contentType := apiReq.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)

	// X-Super-Properties (base64 encoded)
	superProps, _ := c.SuperProperties.Encode()
	req.Header.Set("X-Super-Properties", superProps)

	// User-Agent
	req.Header.Set("User-Agent", c.userAgent)

	// Accept headers
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	// Discord-specific headers
	req.Header.Set("X-Discord-Locale", c.Locale)
	if c.Timezone != "" {
		req.Header.Set("X-Discord-Timezone", c.Timezone)
	}

	// Origin and Referer
	req.Header.Set("Origin", "https://discord.com")
	req.Header.Set("Referer", "https://discord.com/channels/@me")

	// Sec headers (Chrome-like)
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	// Audit log reason
	if apiReq.Reason != "" {
		req.Header.Set("X-Audit-Log-Reason", apiReq.Reason)
	}
}

// APIError represents a Discord API error
type APIError struct {
	StatusCode int
	Code       int                    `json:"code"`
	Message    string                 `json:"message"`
	Errors     map[string]interface{} `json:"errors,omitempty"`
}

func (e *APIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("Discord API error %d: %s (HTTP %d)", e.Code, e.Message, e.StatusCode)
	}
	return fmt.Sprintf("Discord API error: %s (HTTP %d)", e.Message, e.StatusCode)
}

func parseAPIError(resp *http.Response) error {
	var apiErr APIError
	apiErr.StatusCode = resp.StatusCode

	body, _ := io.ReadAll(resp.Body)
	if len(body) > 0 {
		json.Unmarshal(body, &apiErr)
	}

	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}

	return &apiErr
}

// ============================================
// Message API Methods
// ============================================

// SendMessage sends a message to a channel
func (c *Client) SendMessage(ctx context.Context, channelID types.Snowflake, data *types.MessageSendData) (*types.Message, error) {
	// Generate nonce if not set
	if data.Nonce == "" {
		data.Nonce = types.GenerateNonce()
	}

	var msg types.Message
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/channels/%s/messages", channelID),
		Body:     data,
	}, &msg)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}

// SendMessageSimple sends a simple text message
func (c *Client) SendMessageSimple(ctx context.Context, channelID types.Snowflake, content string) (*types.Message, error) {
	return c.SendMessage(ctx, channelID, &types.MessageSendData{
		Content: content,
	})
}

// SendMessageWithFiles sends a message with file attachments
func (c *Client) SendMessageWithFiles(ctx context.Context, channelID types.Snowflake, data *types.MessageSendData, files []*types.File) (*types.Message, error) {
	// Generate nonce if not set
	if data.Nonce == "" {
		data.Nonce = types.GenerateNonce()
	}

	// Build multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Build attachments array for payload_json
	attachments := make([]types.AttachmentPayload, len(files))
	for i, file := range files {
		attachments[i] = types.AttachmentPayload{
			ID:          i,
			Filename:    file.Name,
			Description: file.Description,
		}
	}

	// Create payload with attachments reference
	type payloadWithAttachments struct {
		*types.MessageSendData
		Attachments []types.AttachmentPayload `json:"attachments,omitempty"`
	}

	payload := payloadWithAttachments{
		MessageSendData: data,
		Attachments:     attachments,
	}

	// Add payload_json field
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	if err := writer.WriteField("payload_json", string(payloadJSON)); err != nil {
		return nil, fmt.Errorf("failed to write payload_json: %w", err)
	}

	// Add files
	for i, file := range files {
		fieldName := fmt.Sprintf("files[%d]", i)

		// Determine content type
		contentType := file.ContentType
		if contentType == "" {
			contentType = types.GuessContentType(file.Name)
		}

		// Create form file with proper headers
		h := make(map[string][]string)
		h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, file.Name)}
		h["Content-Type"] = []string{contentType}

		part, err := writer.CreatePart(h)
		if err != nil {
			return nil, fmt.Errorf("failed to create form part: %w", err)
		}

		if _, err := io.Copy(part, file.Reader); err != nil {
			return nil, fmt.Errorf("failed to write file data: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Make request
	url := BaseURL + fmt.Sprintf("/channels/%s/messages", channelID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	c.setMultipartHeaders(httpReq, writer.Boundary())

	// Wait for rate limiter
	endpoint := fmt.Sprintf("/channels/%s/messages", channelID)
	bucket := c.RateLimiter.GetBucket(endpoint)
	if err := bucket.Wait(ctx); err != nil {
		return nil, err
	}

	// Perform request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Update rate limiter
	bucket.Update(resp.Header)

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		var rateLimitResp struct {
			RetryAfter float64 `json:"retry_after"`
			Global     bool    `json:"global"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&rateLimitResp); err == nil {
			retryAfter := time.Duration(rateLimitResp.RetryAfter * float64(time.Second))

			if rateLimitResp.Global {
				c.RateLimiter.SetGlobal(retryAfter)
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryAfter):
			}

			// Retry - need to re-read files, so this is a simplified retry
			return nil, fmt.Errorf("rate limited, retry after %v", retryAfter)
		}
	}

	// Check for error status
	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp)
	}

	// Decode response
	var msg types.Message
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &msg, nil
}

// setMultipartHeaders sets headers for multipart requests
func (c *Client) setMultipartHeaders(req *http.Request, boundary string) {
	// Authorization
	req.Header.Set("Authorization", c.Token)

	// Content-Type with boundary
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)

	// X-Super-Properties
	superProps, _ := c.SuperProperties.Encode()
	req.Header.Set("X-Super-Properties", superProps)

	// User-Agent
	req.Header.Set("User-Agent", c.userAgent)

	// Accept headers
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Discord-specific headers
	req.Header.Set("X-Discord-Locale", c.Locale)
	if c.Timezone != "" {
		req.Header.Set("X-Discord-Timezone", c.Timezone)
	}

	// Origin and Referer
	req.Header.Set("Origin", "https://discord.com")
	req.Header.Set("Referer", "https://discord.com/channels/@me")

	// Sec headers
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
}

// SendFile sends a single file to a channel
func (c *Client) SendFile(ctx context.Context, channelID types.Snowflake, file *types.File, content string) (*types.Message, error) {
	return c.SendMessageWithFiles(ctx, channelID, &types.MessageSendData{
		Content: content,
	}, []*types.File{file})
}

// SendImage is a convenience method for sending an image
func (c *Client) SendImage(ctx context.Context, channelID types.Snowflake, imagePath string, content string) (*types.Message, error) {
	file, err := types.NewFileFromPath(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	return c.SendFile(ctx, channelID, file, content)
}

// EditMessage edits a message
func (c *Client) EditMessage(ctx context.Context, channelID, messageID types.Snowflake, data *types.MessageEditData) (*types.Message, error) {
	var msg types.Message
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID),
		Body:     data,
	}, &msg)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}

// DeleteMessage deletes a message
func (c *Client) DeleteMessage(ctx context.Context, channelID, messageID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID),
	})
	return err
}

// GetMessage gets a single message
func (c *Client) GetMessage(ctx context.Context, channelID, messageID types.Snowflake) (*types.Message, error) {
	var msg types.Message
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID),
	}, &msg)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}

// GetMessages gets messages from a channel
func (c *Client) GetMessages(ctx context.Context, channelID types.Snowflake, limit int, before, after, around types.Snowflake) ([]*types.Message, error) {
	endpoint := fmt.Sprintf("/channels/%s/messages?limit=%d", channelID, limit)

	if !before.IsZero() {
		endpoint += "&before=" + before.String()
	}
	if !after.IsZero() {
		endpoint += "&after=" + after.String()
	}
	if !around.IsZero() {
		endpoint += "&around=" + around.String()
	}

	var msgs []*types.Message
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &msgs)

	if err != nil {
		return nil, err
	}

	return msgs, nil
}

// AddReaction adds a reaction to a message
func (c *Client) AddReaction(ctx context.Context, channelID, messageID types.Snowflake, emoji string) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPut,
		Endpoint: fmt.Sprintf("/channels/%s/messages/%s/reactions/%s/@me", channelID, messageID, emoji),
	})
	return err
}

// RemoveReaction removes a reaction from a message
func (c *Client) RemoveReaction(ctx context.Context, channelID, messageID types.Snowflake, emoji string) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/channels/%s/messages/%s/reactions/%s/@me", channelID, messageID, emoji),
	})
	return err
}

// TriggerTyping triggers the typing indicator in a channel
func (c *Client) TriggerTyping(ctx context.Context, channelID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/channels/%s/typing", channelID),
	})
	return err
}

// ============================================
// Channel API Methods
// ============================================

// GetChannel gets a channel
func (c *Client) GetChannel(ctx context.Context, channelID types.Snowflake) (*types.Channel, error) {
	var ch types.Channel
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/channels/%s", channelID),
	}, &ch)

	if err != nil {
		return nil, err
	}

	return &ch, nil
}

// CreateDM creates a DM channel with a user
func (c *Client) CreateDM(ctx context.Context, recipientID types.Snowflake) (*types.Channel, error) {
	var ch types.Channel
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/users/@me/channels",
		Body: map[string]interface{}{
			"recipient_id": recipientID.String(),
		},
	}, &ch)

	if err != nil {
		return nil, err
	}

	return &ch, nil
}

// ============================================
// User API Methods
// ============================================

// GetCurrentUser gets the current authenticated user
func (c *Client) GetCurrentUser(ctx context.Context) (*types.CurrentUser, error) {
	var user types.CurrentUser
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: "/users/@me",
	}, &user)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUser gets a user by ID
func (c *Client) GetUser(ctx context.Context, userID types.Snowflake) (*types.User, error) {
	var user types.User
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/users/%s", userID),
	}, &user)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// ============================================
// Guild API Methods
// ============================================

// GetGuild gets a guild by ID
func (c *Client) GetGuild(ctx context.Context, guildID types.Snowflake) (*types.Guild, error) {
	var guild types.Guild
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s", guildID),
	}, &guild)

	if err != nil {
		return nil, err
	}

	return &guild, nil
}

// GetGuildChannels gets channels in a guild
func (c *Client) GetGuildChannels(ctx context.Context, guildID types.Snowflake) ([]*types.Channel, error) {
	var channels []*types.Channel
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/channels", guildID),
	}, &channels)

	if err != nil {
		return nil, err
	}

	return channels, nil
}

// GetGuildMember gets a guild member
func (c *Client) GetGuildMember(ctx context.Context, guildID, userID types.Snowflake) (*types.Member, error) {
	var member types.Member
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/members/%s", guildID, userID),
	}, &member)

	if err != nil {
		return nil, err
	}

	return &member, nil
}

// GetGuildRoles gets roles in a guild
func (c *Client) GetGuildRoles(ctx context.Context, guildID types.Snowflake) ([]*types.Role, error) {
	var roles []*types.Role
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/roles", guildID),
	}, &roles)

	if err != nil {
		return nil, err
	}

	return roles, nil
}

// LeaveGuild leaves a guild
func (c *Client) LeaveGuild(ctx context.Context, guildID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/users/@me/guilds/%s", guildID),
	})
	return err
}

// JoinGuild joins a guild using an invite code
func (c *Client) JoinGuild(ctx context.Context, inviteCode string) (*types.Invite, error) {
	var invite types.Invite
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/invites/%s", inviteCode),
	}, &invite)

	if err != nil {
		return nil, err
	}

	return &invite, nil
}

// GetInvite gets an invite by code
func (c *Client) GetInvite(ctx context.Context, inviteCode string) (*types.Invite, error) {
	var invite types.Invite
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/invites/%s?with_counts=true", inviteCode),
	}, &invite)

	if err != nil {
		return nil, err
	}

	return &invite, nil
}

// ============================================
// Slash Command / Interaction API Methods
// ============================================

// GetGuildCommands gets slash commands for a guild
func (c *Client) GetGuildCommands(ctx context.Context, applicationID, guildID types.Snowflake) ([]*types.ApplicationCommand, error) {
	var commands []*types.ApplicationCommand
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/applications/%s/guilds/%s/commands", applicationID, guildID),
	}, &commands)

	if err != nil {
		return nil, err
	}

	return commands, nil
}

// SearchGuildCommands searches for slash commands in a guild
func (c *Client) SearchGuildCommands(ctx context.Context, guildID, channelID types.Snowflake, query string, limit int) (interface{}, error) {
	endpoint := fmt.Sprintf("/channels/%s/application-commands/search?type=1&limit=%d", channelID, limit)
	if query != "" {
		endpoint += "&query=" + query
	}

	var result interface{}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &result)

	if err != nil {
		return nil, err
	}

	return result, nil
}

// InvokeSlashCommand invokes a slash command
func (c *Client) InvokeSlashCommand(ctx context.Context, data *types.SlashCommandData) error {
	if data.Nonce == "" {
		data.Nonce = types.GenerateNonce()
	}
	data.Type = 2 // Application command type

	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/interactions",
		Body:     data,
	})
	return err
}

// ClickButton clicks a button component
func (c *Client) ClickButton(ctx context.Context, channelID, messageID types.Snowflake, customID string, applicationID types.Snowflake) error {
	data := map[string]interface{}{
		"type":           3, // Message component
		"nonce":          types.GenerateNonce(),
		"guild_id":       nil,
		"channel_id":     channelID.String(),
		"message_flags":  0,
		"message_id":     messageID.String(),
		"application_id": applicationID.String(),
		"session_id":     "", // Empty string, gateway handles this
		"data": map[string]interface{}{
			"component_type": 2, // Button
			"custom_id":      customID,
		},
	}

	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/interactions",
		Body:     data,
	})
	return err
}

// SelectMenuSelect selects options in a select menu
func (c *Client) SelectMenuSelect(ctx context.Context, channelID, messageID types.Snowflake, customID string, values []string, applicationID types.Snowflake) error {
	data := map[string]interface{}{
		"type":           3, // Message component
		"nonce":          types.GenerateNonce(),
		"channel_id":     channelID.String(),
		"message_id":     messageID.String(),
		"application_id": applicationID.String(),
		"data": map[string]interface{}{
			"component_type": 3, // Select menu
			"custom_id":      customID,
			"values":         values,
		},
	}

	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/interactions",
		Body:     data,
	})
	return err
}

// SubmitModal submits a modal
func (c *Client) SubmitModal(ctx context.Context, channelID types.Snowflake, customID string, components []interface{}, applicationID types.Snowflake) error {
	data := map[string]interface{}{
		"type":           5, // Modal submit
		"nonce":          types.GenerateNonce(),
		"channel_id":     channelID.String(),
		"application_id": applicationID.String(),
		"data": map[string]interface{}{
			"custom_id":  customID,
			"components": components,
		},
	}

	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/interactions",
		Body:     data,
	})
	return err
}

// ============================================
// Bulk Message Operations
// ============================================

// BulkDeleteMessages deletes multiple messages (2-100 messages, max 14 days old)
func (c *Client) BulkDeleteMessages(ctx context.Context, channelID types.Snowflake, messageIDs []types.Snowflake) error {
	ids := make([]string, len(messageIDs))
	for i, id := range messageIDs {
		ids[i] = id.String()
	}

	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/channels/%s/messages/bulk-delete", channelID),
		Body: map[string]interface{}{
			"messages": ids,
		},
	})
	return err
}

// ============================================
// Pin Operations
// ============================================

// PinMessage pins a message in a channel
func (c *Client) PinMessage(ctx context.Context, channelID, messageID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPut,
		Endpoint: fmt.Sprintf("/channels/%s/pins/%s", channelID, messageID),
	})
	return err
}

// UnpinMessage unpins a message from a channel
func (c *Client) UnpinMessage(ctx context.Context, channelID, messageID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/channels/%s/pins/%s", channelID, messageID),
	})
	return err
}

// GetPinnedMessages gets all pinned messages in a channel
func (c *Client) GetPinnedMessages(ctx context.Context, channelID types.Snowflake) ([]*types.Message, error) {
	var messages []*types.Message
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/channels/%s/pins", channelID),
	}, &messages)
	return messages, err
}

// ============================================
// Thread Operations
// ============================================

// CreateThread creates a thread from a message
func (c *Client) CreateThread(ctx context.Context, channelID, messageID types.Snowflake, data *types.ThreadCreateData) (*types.Channel, error) {
	var thread types.Channel
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/channels/%s/messages/%s/threads", channelID, messageID),
		Body:     data,
	}, &thread)
	return &thread, err
}

// CreateThreadWithoutMessage creates a thread without a starting message
func (c *Client) CreateThreadWithoutMessage(ctx context.Context, channelID types.Snowflake, data *types.ThreadCreateData) (*types.Channel, error) {
	var thread types.Channel
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/channels/%s/threads", channelID),
		Body:     data,
	}, &thread)
	return &thread, err
}

// CreateForumPost creates a post in a forum channel
func (c *Client) CreateForumPost(ctx context.Context, channelID types.Snowflake, data *types.ForumThreadCreateData) (*types.Channel, error) {
	var thread types.Channel
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/channels/%s/threads", channelID),
		Body:     data,
	}, &thread)
	return &thread, err
}

// JoinThread joins a thread
func (c *Client) JoinThread(ctx context.Context, threadID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPut,
		Endpoint: fmt.Sprintf("/channels/%s/thread-members/@me", threadID),
	})
	return err
}

// LeaveThread leaves a thread
func (c *Client) LeaveThread(ctx context.Context, threadID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/channels/%s/thread-members/@me", threadID),
	})
	return err
}

// ArchiveThread archives a thread
func (c *Client) ArchiveThread(ctx context.Context, threadID types.Snowflake, locked bool) (*types.Channel, error) {
	var thread types.Channel
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/channels/%s", threadID),
		Body: map[string]interface{}{
			"archived": true,
			"locked":   locked,
		},
	}, &thread)
	return &thread, err
}

// UnarchiveThread unarchives a thread
func (c *Client) UnarchiveThread(ctx context.Context, threadID types.Snowflake) (*types.Channel, error) {
	var thread types.Channel
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/channels/%s", threadID),
		Body: map[string]interface{}{
			"archived": false,
		},
	}, &thread)
	return &thread, err
}

// GetActiveThreads gets all active threads in a guild
func (c *Client) GetActiveThreads(ctx context.Context, guildID types.Snowflake) (interface{}, error) {
	var result interface{}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/threads/active", guildID),
	}, &result)
	return result, err
}

// ============================================
// Webhook Operations
// ============================================

// GetChannelWebhooks gets all webhooks for a channel
func (c *Client) GetChannelWebhooks(ctx context.Context, channelID types.Snowflake) ([]*types.Webhook, error) {
	var webhooks []*types.Webhook
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/channels/%s/webhooks", channelID),
	}, &webhooks)
	return webhooks, err
}

// CreateWebhook creates a webhook
func (c *Client) CreateWebhook(ctx context.Context, channelID types.Snowflake, name string, avatar *string) (*types.Webhook, error) {
	var webhook types.Webhook
	body := map[string]interface{}{
		"name": name,
	}
	if avatar != nil {
		body["avatar"] = *avatar
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/channels/%s/webhooks", channelID),
		Body:     body,
	}, &webhook)
	return &webhook, err
}

// ExecuteWebhook sends a message via webhook
func (c *Client) ExecuteWebhook(ctx context.Context, webhookID types.Snowflake, webhookToken string, data *types.WebhookMessageData) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/webhooks/%s/%s", webhookID, webhookToken),
		Body:     data,
	})
	return err
}

// DeleteWebhook deletes a webhook
func (c *Client) DeleteWebhook(ctx context.Context, webhookID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/webhooks/%s", webhookID),
	})
	return err
}

// ============================================
// Poll Operations
// ============================================

// CreatePoll creates a poll in a channel
func (c *Client) CreatePoll(ctx context.Context, channelID types.Snowflake, content string, poll *types.PollCreateData) (*types.Message, error) {
	var msg types.Message
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/channels/%s/messages", channelID),
		Body: map[string]interface{}{
			"content": content,
			"poll":    poll,
			"nonce":   types.GenerateNonce(),
		},
	}, &msg)
	return &msg, err
}

// VotePoll votes on a poll
func (c *Client) VotePoll(ctx context.Context, channelID, messageID types.Snowflake, answerID int) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPut,
		Endpoint: fmt.Sprintf("/channels/%s/polls/%s/answers/%d/@me", channelID, messageID, answerID),
	})
	return err
}

// RemovePollVote removes a vote from a poll
func (c *Client) RemovePollVote(ctx context.Context, channelID, messageID types.Snowflake, answerID int) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/channels/%s/polls/%s/answers/%d/@me", channelID, messageID, answerID),
	})
	return err
}

// EndPoll ends a poll early
func (c *Client) EndPoll(ctx context.Context, channelID, messageID types.Snowflake) (*types.Message, error) {
	var msg types.Message
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/channels/%s/polls/%s/expire", channelID, messageID),
	}, &msg)
	return &msg, err
}

// ============================================
// Scheduled Event Operations
// ============================================

// GetGuildEvents gets scheduled events for a guild
func (c *Client) GetGuildEvents(ctx context.Context, guildID types.Snowflake) ([]*types.ScheduledEvent, error) {
	var events []*types.ScheduledEvent
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/scheduled-events?with_user_count=true", guildID),
	}, &events)
	return events, err
}

// RSVPEvent marks interest in an event
func (c *Client) RSVPEvent(ctx context.Context, guildID, eventID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPut,
		Endpoint: fmt.Sprintf("/guilds/%s/scheduled-events/%s/users/@me", guildID, eventID),
	})
	return err
}

// UnRSVPEvent removes interest from an event
func (c *Client) UnRSVPEvent(ctx context.Context, guildID, eventID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/guilds/%s/scheduled-events/%s/users/@me", guildID, eventID),
	})
	return err
}

// ============================================
// User Notes
// ============================================

// GetUserNote gets your note on a user
func (c *Client) GetUserNote(ctx context.Context, userID types.Snowflake) (string, error) {
	var result struct {
		Note string `json:"note"`
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/users/@me/notes/%s", userID),
	}, &result)
	return result.Note, err
}

// ============================================
// Read State / Mark as Read
// ============================================

// AckMessage marks a message as read
func (c *Client) AckMessage(ctx context.Context, channelID, messageID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/channels/%s/messages/%s/ack", channelID, messageID),
		Body: map[string]interface{}{
			"token": nil,
		},
	})
	return err
}

// AckGuild marks all messages in a guild as read
func (c *Client) AckGuild(ctx context.Context, guildID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/guilds/%s/ack", guildID),
	})
	return err
}

// ============================================
// Profile / Settings
// ============================================

// EditProfile edits the current user's profile
func (c *Client) EditProfile(ctx context.Context, data *types.ProfileEditData) (*types.CurrentUser, error) {
	var user types.CurrentUser
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: "/users/@me",
		Body:     data,
	}, &user)
	return &user, err
}

// SetCustomStatus sets a custom status
func (c *Client) SetCustomStatus(ctx context.Context, text string, emojiName *string, emojiID *string, expiresAt *string) error {
	customStatus := map[string]interface{}{
		"text": text,
	}
	if emojiName != nil {
		customStatus["emoji_name"] = *emojiName
	}
	if emojiID != nil {
		customStatus["emoji_id"] = *emojiID
	}
	if expiresAt != nil {
		customStatus["expires_at"] = *expiresAt
	}

	_, err := c.Do(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: "/users/@me/settings",
		Body: map[string]interface{}{
			"custom_status": customStatus,
		},
	})
	return err
}

// ClearCustomStatus clears the custom status
func (c *Client) ClearCustomStatus(ctx context.Context) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: "/users/@me/settings",
		Body: map[string]interface{}{
			"custom_status": nil,
		},
	})
	return err
}

// ============================================
// Group DM
// ============================================

// CreateGroupDM creates a group DM
func (c *Client) CreateGroupDM(ctx context.Context, recipientIDs []types.Snowflake) (*types.Channel, error) {
	ids := make([]string, len(recipientIDs))
	for i, id := range recipientIDs {
		ids[i] = id.String()
	}

	var channel types.Channel
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/users/@me/channels",
		Body: map[string]interface{}{
			"recipients": ids,
		},
	}, &channel)
	return &channel, err
}

// AddGroupDMRecipient adds a user to a group DM
func (c *Client) AddGroupDMRecipient(ctx context.Context, channelID, userID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPut,
		Endpoint: fmt.Sprintf("/channels/%s/recipients/%s", channelID, userID),
	})
	return err
}

// RemoveGroupDMRecipient removes a user from a group DM
func (c *Client) RemoveGroupDMRecipient(ctx context.Context, channelID, userID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/channels/%s/recipients/%s", channelID, userID),
	})
	return err
}

// ============================================
// Stage Channels
// ============================================

// RequestToSpeak requests to speak in a stage channel
func (c *Client) RequestToSpeak(ctx context.Context, guildID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/voice-states/@me", guildID),
		Body: map[string]interface{}{
			"request_to_speak_timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	})
	return err
}

// CancelRequestToSpeak cancels a request to speak
func (c *Client) CancelRequestToSpeak(ctx context.Context, guildID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/voice-states/@me", guildID),
		Body: map[string]interface{}{
			"request_to_speak_timestamp": nil,
		},
	})
	return err
}

// BecomeStageSpeaker becomes a speaker (if you have permission or were invited)
func (c *Client) BecomeStageSpeaker(ctx context.Context, guildID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/voice-states/@me", guildID),
		Body: map[string]interface{}{
			"suppress": false,
		},
	})
	return err
}

// BecomeStageAudience becomes an audience member
func (c *Client) BecomeStageAudience(ctx context.Context, guildID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/voice-states/@me", guildID),
		Body: map[string]interface{}{
			"suppress": true,
		},
	})
	return err
}

// ============================================
// Search Messages
// ============================================

// SearchMessages searches for messages in a channel/guild
func (c *Client) SearchMessages(ctx context.Context, channelID types.Snowflake, query string, limit int) (interface{}, error) {
	endpoint := fmt.Sprintf("/channels/%s/messages/search?content=%s&limit=%d", channelID, query, limit)

	var result interface{}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &result)
	return result, err
}

// SearchGuildMessages searches for messages in a guild
func (c *Client) SearchGuildMessages(ctx context.Context, guildID types.Snowflake, query string, limit int) (interface{}, error) {
	endpoint := fmt.Sprintf("/guilds/%s/messages/search?content=%s&limit=%d", guildID, query, limit)

	var result interface{}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &result)
	return result, err
}

// ============================================
// Misc
// ============================================

// GetGuildEmojis gets all emojis for a guild
func (c *Client) GetGuildEmojis(ctx context.Context, guildID types.Snowflake) ([]*types.Emoji, error) {
	var emojis []*types.Emoji
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/emojis", guildID),
	}, &emojis)
	return emojis, err
}

// GetGuildStickers gets all stickers for a guild
func (c *Client) GetGuildStickers(ctx context.Context, guildID types.Snowflake) ([]*types.Sticker, error) {
	var stickers []*types.Sticker
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/stickers", guildID),
	}, &stickers)
	return stickers, err
}

// GetNitroStickerPacks gets available Nitro sticker packs
func (c *Client) GetNitroStickerPacks(ctx context.Context) (interface{}, error) {
	var result interface{}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: "/sticker-packs",
	}, &result)
	return result, err
}

// ============================================
// Moderation API Methods
// ============================================

// BanUser bans a user from a guild
func (c *Client) BanUser(ctx context.Context, guildID, userID types.Snowflake, deleteMessageDays int, reason string) error {
	body := map[string]interface{}{}
	if deleteMessageDays > 0 {
		body["delete_message_days"] = deleteMessageDays
	}

	req := Request{
		Method:   http.MethodPut,
		Endpoint: fmt.Sprintf("/guilds/%s/bans/%s", guildID, userID),
		Body:     body,
	}
	if reason != "" {
		req.Reason = reason
	}

	_, err := c.Do(ctx, req)
	return err
}

// UnbanUser unbans a user from a guild
func (c *Client) UnbanUser(ctx context.Context, guildID, userID types.Snowflake, reason string) error {
	req := Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/guilds/%s/bans/%s", guildID, userID),
	}
	if reason != "" {
		req.Reason = reason
	}

	_, err := c.Do(ctx, req)
	return err
}

// GetGuildBans gets all bans for a guild
func (c *Client) GetGuildBans(ctx context.Context, guildID types.Snowflake) ([]*types.Ban, error) {
	var bans []*types.Ban
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/bans", guildID),
	}, &bans)
	return bans, err
}

// GetGuildBan gets a specific ban
func (c *Client) GetGuildBan(ctx context.Context, guildID, userID types.Snowflake) (*types.Ban, error) {
	var ban types.Ban
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/bans/%s", guildID, userID),
	}, &ban)
	return &ban, err
}

// KickUser kicks a user from a guild
func (c *Client) KickUser(ctx context.Context, guildID, userID types.Snowflake, reason string) error {
	req := Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/guilds/%s/members/%s", guildID, userID),
	}
	if reason != "" {
		req.Reason = reason
	}

	_, err := c.Do(ctx, req)
	return err
}

// TimeoutUser times out a user (communication disabled)
func (c *Client) TimeoutUser(ctx context.Context, guildID, userID types.Snowflake, until *time.Time, reason string) error {
	var timeoutStr *string
	if until != nil {
		t := until.UTC().Format(time.RFC3339)
		timeoutStr = &t
	}

	req := Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/members/%s", guildID, userID),
		Body: map[string]interface{}{
			"communication_disabled_until": timeoutStr,
		},
	}
	if reason != "" {
		req.Reason = reason
	}

	_, err := c.Do(ctx, req)
	return err
}

// ============================================
// Channel Management API Methods
// ============================================

// CreateChannel creates a channel in a guild
func (c *Client) CreateChannel(ctx context.Context, guildID types.Snowflake, data *types.ChannelCreateData) (*types.Channel, error) {
	var channel types.Channel
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/guilds/%s/channels", guildID),
		Body:     data,
	}, &channel)
	return &channel, err
}

// ModifyChannel modifies a channel
func (c *Client) ModifyChannel(ctx context.Context, channelID types.Snowflake, data *types.ChannelModifyData) (*types.Channel, error) {
	var channel types.Channel
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/channels/%s", channelID),
		Body:     data,
	}, &channel)
	return &channel, err
}

// DeleteChannel deletes a channel
func (c *Client) DeleteChannel(ctx context.Context, channelID types.Snowflake, reason string) error {
	req := Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/channels/%s", channelID),
	}
	if reason != "" {
		req.Reason = reason
	}

	_, err := c.Do(ctx, req)
	return err
}

// ModifyChannelPositions modifies channel positions
func (c *Client) ModifyChannelPositions(ctx context.Context, guildID types.Snowflake, positions []types.ChannelPosition) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/channels", guildID),
		Body:     positions,
	})
	return err
}

// ============================================
// Role Management API Methods
// ============================================

// CreateRole creates a role in a guild
func (c *Client) CreateRole(ctx context.Context, guildID types.Snowflake, data *types.RoleCreateData) (*types.Role, error) {
	var role types.Role
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/guilds/%s/roles", guildID),
		Body:     data,
	}, &role)
	return &role, err
}

// ModifyRole modifies a role
func (c *Client) ModifyRole(ctx context.Context, guildID, roleID types.Snowflake, data *types.RoleModifyData) (*types.Role, error) {
	var role types.Role
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/roles/%s", guildID, roleID),
		Body:     data,
	}, &role)
	return &role, err
}

// DeleteRole deletes a role
func (c *Client) DeleteRole(ctx context.Context, guildID, roleID types.Snowflake, reason string) error {
	req := Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/guilds/%s/roles/%s", guildID, roleID),
	}
	if reason != "" {
		req.Reason = reason
	}

	_, err := c.Do(ctx, req)
	return err
}

// AddMemberRole adds a role to a member
func (c *Client) AddMemberRole(ctx context.Context, guildID, userID, roleID types.Snowflake, reason string) error {
	req := Request{
		Method:   http.MethodPut,
		Endpoint: fmt.Sprintf("/guilds/%s/members/%s/roles/%s", guildID, userID, roleID),
	}
	if reason != "" {
		req.Reason = reason
	}

	_, err := c.Do(ctx, req)
	return err
}

// RemoveMemberRole removes a role from a member
func (c *Client) RemoveMemberRole(ctx context.Context, guildID, userID, roleID types.Snowflake, reason string) error {
	req := Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/guilds/%s/members/%s/roles/%s", guildID, userID, roleID),
	}
	if reason != "" {
		req.Reason = reason
	}

	_, err := c.Do(ctx, req)
	return err
}

// ModifyRolePositions modifies role positions
func (c *Client) ModifyRolePositions(ctx context.Context, guildID types.Snowflake, positions []types.RolePosition) ([]*types.Role, error) {
	var roles []*types.Role
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/roles", guildID),
		Body:     positions,
	}, &roles)
	return roles, err
}

// ============================================
// Voice State API Methods
// ============================================

// ModifyCurrentUserVoiceState modifies the current user's voice state
func (c *Client) ModifyCurrentUserVoiceState(ctx context.Context, guildID types.Snowflake, channelID *types.Snowflake, suppress *bool, requestToSpeak *time.Time) error {
	body := map[string]interface{}{}
	if channelID != nil {
		body["channel_id"] = channelID.String()
	}
	if suppress != nil {
		body["suppress"] = *suppress
	}
	if requestToSpeak != nil {
		body["request_to_speak_timestamp"] = requestToSpeak.UTC().Format(time.RFC3339)
	}

	_, err := c.Do(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/voice-states/@me", guildID),
		Body:     body,
	})
	return err
}

// ModifyUserVoiceState modifies another user's voice state
func (c *Client) ModifyUserVoiceState(ctx context.Context, guildID, userID types.Snowflake, channelID types.Snowflake, suppress *bool) error {
	body := map[string]interface{}{
		"channel_id": channelID.String(),
	}
	if suppress != nil {
		body["suppress"] = *suppress
	}

	_, err := c.Do(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/voice-states/%s", guildID, userID),
		Body:     body,
	})
	return err
}

// ============================================
// Stage Instance API Methods
// ============================================

// CreateStageInstance creates a stage instance
func (c *Client) CreateStageInstance(ctx context.Context, channelID types.Snowflake, topic string, privacyLevel int) (*types.StageInstance, error) {
	var stage types.StageInstance
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/stage-instances",
		Body: map[string]interface{}{
			"channel_id":    channelID.String(),
			"topic":         topic,
			"privacy_level": privacyLevel,
		},
	}, &stage)
	return &stage, err
}

// GetStageInstance gets a stage instance
func (c *Client) GetStageInstance(ctx context.Context, channelID types.Snowflake) (*types.StageInstance, error) {
	var stage types.StageInstance
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/stage-instances/%s", channelID),
	}, &stage)
	return &stage, err
}

// ModifyStageInstance modifies a stage instance
func (c *Client) ModifyStageInstance(ctx context.Context, channelID types.Snowflake, topic string, privacyLevel *int) (*types.StageInstance, error) {
	body := map[string]interface{}{
		"topic": topic,
	}
	if privacyLevel != nil {
		body["privacy_level"] = *privacyLevel
	}

	var stage types.StageInstance
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/stage-instances/%s", channelID),
		Body:     body,
	}, &stage)
	return &stage, err
}

// DeleteStageInstance deletes a stage instance
func (c *Client) DeleteStageInstance(ctx context.Context, channelID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/stage-instances/%s", channelID),
	})
	return err
}

// ============================================
// Guild Preview & Welcome Screen
// ============================================

// GetGuildPreview gets a guild preview
func (c *Client) GetGuildPreview(ctx context.Context, guildID types.Snowflake) (*types.GuildPreview, error) {
	var preview types.GuildPreview
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/preview", guildID),
	}, &preview)
	return &preview, err
}

// GetWelcomeScreen gets a guild's welcome screen
func (c *Client) GetWelcomeScreen(ctx context.Context, guildID types.Snowflake) (*types.WelcomeScreen, error) {
	var screen types.WelcomeScreen
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/welcome-screen", guildID),
	}, &screen)
	return &screen, err
}

// ModifyWelcomeScreen modifies a guild's welcome screen
func (c *Client) ModifyWelcomeScreen(ctx context.Context, guildID types.Snowflake, data *types.WelcomeScreenData) (*types.WelcomeScreen, error) {
	var screen types.WelcomeScreen
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/welcome-screen", guildID),
		Body:     data,
	}, &screen)
	return &screen, err
}

// ============================================
// Audit Log
// ============================================

// GetAuditLog gets the audit log for a guild
func (c *Client) GetAuditLog(ctx context.Context, guildID types.Snowflake, userID types.Snowflake, actionType int, before types.Snowflake, limit int) (*types.AuditLog, error) {
	endpoint := fmt.Sprintf("/guilds/%s/audit-logs?limit=%d", guildID, limit)
	if userID != 0 {
		endpoint += "&user_id=" + userID.String()
	}
	if actionType > 0 {
		endpoint += fmt.Sprintf("&action_type=%d", actionType)
	}
	if before != 0 {
		endpoint += "&before=" + before.String()
	}

	var auditLog types.AuditLog
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &auditLog)
	return &auditLog, err
}

// ============================================
// Permission Overwrites
// ============================================

// EditChannelPermissions edits channel permission overwrites
func (c *Client) EditChannelPermissions(ctx context.Context, channelID, overwriteID types.Snowflake, allow, deny string, permType int, reason string) error {
	req := Request{
		Method:   http.MethodPut,
		Endpoint: fmt.Sprintf("/channels/%s/permissions/%s", channelID, overwriteID),
		Body: map[string]interface{}{
			"allow": allow,
			"deny":  deny,
			"type":  permType,
		},
	}
	if reason != "" {
		req.Reason = reason
	}

	_, err := c.Do(ctx, req)
	return err
}

// DeleteChannelPermission deletes a channel permission overwrite
func (c *Client) DeleteChannelPermission(ctx context.Context, channelID, overwriteID types.Snowflake, reason string) error {
	req := Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/channels/%s/permissions/%s", channelID, overwriteID),
	}
	if reason != "" {
		req.Reason = reason
	}

	_, err := c.Do(ctx, req)
	return err
}

// ============================================
// Guild Member Operations
// ============================================

// ModifyGuildMember modifies a guild member
func (c *Client) ModifyGuildMember(ctx context.Context, guildID, userID types.Snowflake, data *types.MemberModifyData, reason string) (*types.Member, error) {
	req := Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/members/%s", guildID, userID),
		Body:     data,
	}
	if reason != "" {
		req.Reason = reason
	}

	var member types.Member
	err := c.DoJSON(ctx, req, &member)
	return &member, err
}

// GetGuildMembers gets members in a guild
func (c *Client) GetGuildMembers(ctx context.Context, guildID types.Snowflake, limit int, after types.Snowflake) ([]*types.Member, error) {
	endpoint := fmt.Sprintf("/guilds/%s/members?limit=%d", guildID, limit)
	if after != 0 {
		endpoint += "&after=" + after.String()
	}

	var members []*types.Member
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &members)
	return members, err
}

// SearchGuildMembers searches for members in a guild
func (c *Client) SearchGuildMembers(ctx context.Context, guildID types.Snowflake, query string, limit int) ([]*types.Member, error) {
	var members []*types.Member
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/members/search?query=%s&limit=%d", guildID, query, limit),
	}, &members)
	return members, err
}

// ============================================
// Prune
// ============================================

// GetGuildPruneCount gets the prune count
func (c *Client) GetGuildPruneCount(ctx context.Context, guildID types.Snowflake, days int, includeRoles []types.Snowflake) (int, error) {
	endpoint := fmt.Sprintf("/guilds/%s/prune?days=%d", guildID, days)
	for _, r := range includeRoles {
		endpoint += "&include_roles=" + r.String()
	}

	var result struct {
		Pruned int `json:"pruned"`
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &result)
	return result.Pruned, err
}

// BeginGuildPrune begins a guild prune
func (c *Client) BeginGuildPrune(ctx context.Context, guildID types.Snowflake, days int, computePruneCount bool, includeRoles []types.Snowflake, reason string) (*int, error) {
	roleStrings := make([]string, len(includeRoles))
	for i, r := range includeRoles {
		roleStrings[i] = r.String()
	}

	req := Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/guilds/%s/prune", guildID),
		Body: map[string]interface{}{
			"days":                days,
			"compute_prune_count": computePruneCount,
			"include_roles":       roleStrings,
		},
	}
	if reason != "" {
		req.Reason = reason
	}

	var result struct {
		Pruned *int `json:"pruned"`
	}
	err := c.DoJSON(ctx, req, &result)
	return result.Pruned, err
}

// ============================================
// Invites
// ============================================

// GetChannelInvites gets invites for a channel
func (c *Client) GetChannelInvites(ctx context.Context, channelID types.Snowflake) ([]*types.Invite, error) {
	var invites []*types.Invite
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/channels/%s/invites", channelID),
	}, &invites)
	return invites, err
}

// CreateChannelInvite creates an invite for a channel
func (c *Client) CreateChannelInvite(ctx context.Context, channelID types.Snowflake, maxAge, maxUses int, temporary, unique bool) (*types.Invite, error) {
	var invite types.Invite
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/channels/%s/invites", channelID),
		Body: map[string]interface{}{
			"max_age":   maxAge,
			"max_uses":  maxUses,
			"temporary": temporary,
			"unique":    unique,
		},
	}, &invite)
	return &invite, err
}

// DeleteInvite deletes an invite
func (c *Client) DeleteInvite(ctx context.Context, inviteCode string, reason string) (*types.Invite, error) {
	req := Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/invites/%s", inviteCode),
	}
	if reason != "" {
		req.Reason = reason
	}

	var invite types.Invite
	err := c.DoJSON(ctx, req, &invite)
	return &invite, err
}

// GetGuildInvites gets all invites for a guild
func (c *Client) GetGuildInvites(ctx context.Context, guildID types.Snowflake) ([]*types.Invite, error) {
	var invites []*types.Invite
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/invites", guildID),
	}, &invites)
	return invites, err
}

// ============================================
// Reaction API Methods
// ============================================

// GetReactions gets reactions for a message
func (c *Client) GetReactions(ctx context.Context, channelID, messageID types.Snowflake, emoji string, limit int, after types.Snowflake) ([]*types.User, error) {
	emoji = strings.ReplaceAll(emoji, "#", "%23")
	endpoint := fmt.Sprintf("/channels/%s/messages/%s/reactions/%s?limit=%d", channelID, messageID, emoji, limit)
	if after != 0 {
		endpoint += "&after=" + after.String()
	}

	var users []*types.User
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &users)
	return users, err
}

// ============================================
// Thread Management API Methods (Extended)
// ============================================

// AddThreadMember adds a member to a thread
func (c *Client) AddThreadMember(ctx context.Context, channelID, userID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPut,
		Endpoint: fmt.Sprintf("/channels/%s/thread-members/%s", channelID, userID),
	})
	return err
}

// RemoveThreadMember removes a member from a thread
func (c *Client) RemoveThreadMember(ctx context.Context, channelID, userID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/channels/%s/thread-members/%s", channelID, userID),
	})
	return err
}

// GetThreadMember gets a thread member
func (c *Client) GetThreadMember(ctx context.Context, channelID, userID types.Snowflake) (*types.ThreadMember, error) {
	var member types.ThreadMember
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/channels/%s/thread-members/%s", channelID, userID),
	}, &member)
	return &member, err
}

// GetThreadMembers gets all members of a thread
func (c *Client) GetThreadMembers(ctx context.Context, channelID types.Snowflake) ([]*types.ThreadMember, error) {
	var members []*types.ThreadMember
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/channels/%s/thread-members", channelID),
	}, &members)
	return members, err
}

// ============================================
// Webhook API Methods (Extended)
// ============================================

// GetGuildWebhooks gets all webhooks for a guild
func (c *Client) GetGuildWebhooks(ctx context.Context, guildID types.Snowflake) ([]*types.Webhook, error) {
	var webhooks []*types.Webhook
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/webhooks", guildID),
	}, &webhooks)
	return webhooks, err
}

// ModifyWebhook modifies a webhook
func (c *Client) ModifyWebhook(ctx context.Context, webhookID types.Snowflake, name string, avatar *string, channelID *types.Snowflake) (*types.Webhook, error) {
	body := map[string]interface{}{}
	if name != "" {
		body["name"] = name
	}
	if avatar != nil {
		body["avatar"] = *avatar
	}
	if channelID != nil {
		body["channel_id"] = channelID.String()
	}

	var webhook types.Webhook
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/webhooks/%s", webhookID),
		Body:     body,
	}, &webhook)
	return &webhook, err
}

// ============================================
// Auto Moderation API Methods
// ============================================

// GetAutoModerationRules gets auto moderation rules
func (c *Client) GetAutoModerationRules(ctx context.Context, guildID types.Snowflake) ([]*types.AutoModerationRule, error) {
	var rules []*types.AutoModerationRule
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/auto-moderation/rules", guildID),
	}, &rules)
	return rules, err
}

// GetAutoModerationRule gets a specific auto moderation rule
func (c *Client) GetAutoModerationRule(ctx context.Context, guildID, ruleID types.Snowflake) (*types.AutoModerationRule, error) {
	var rule types.AutoModerationRule
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/auto-moderation/rules/%s", guildID, ruleID),
	}, &rule)
	return &rule, err
}

// CreateAutoModerationRule creates an auto moderation rule
func (c *Client) CreateAutoModerationRule(ctx context.Context, guildID types.Snowflake, rule *types.AutoModerationRule) (*types.AutoModerationRule, error) {
	var newRule types.AutoModerationRule
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/guilds/%s/auto-moderation/rules", guildID),
		Body:     rule,
	}, &newRule)
	return &newRule, err
}

// ModifyAutoModerationRule modifies an auto moderation rule
func (c *Client) ModifyAutoModerationRule(ctx context.Context, guildID, ruleID types.Snowflake, rule *types.AutoModerationRuleModifyData) (*types.AutoModerationRule, error) {
	var updatedRule types.AutoModerationRule
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/auto-moderation/rules/%s", guildID, ruleID),
		Body:     rule,
	}, &updatedRule)
	return &updatedRule, err
}

// DeleteAutoModerationRule deletes an auto moderation rule
func (c *Client) DeleteAutoModerationRule(ctx context.Context, guildID, ruleID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/guilds/%s/auto-moderation/rules/%s", guildID, ruleID),
	})
	return err
}

// ============================================
// Scheduled Event API Methods
// ============================================

// ListScheduledEvents lists scheduled events in a guild
func (c *Client) ListScheduledEvents(ctx context.Context, guildID types.Snowflake, withUserCount bool) ([]*types.ScheduledEvent, error) {
	endpoint := fmt.Sprintf("/guilds/%s/scheduled-events", guildID)
	if withUserCount {
		endpoint += "?with_user_count=true"
	}

	var events []*types.ScheduledEvent
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &events)
	return events, err
}

// CreateScheduledEvent creates a scheduled event
func (c *Client) CreateScheduledEvent(ctx context.Context, guildID types.Snowflake, event *types.ScheduledEventParams) (*types.ScheduledEvent, error) {
	var newEvent types.ScheduledEvent
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/guilds/%s/scheduled-events", guildID),
		Body:     event,
	}, &newEvent)
	return &newEvent, err
}

// GetScheduledEvent gets a scheduled event
func (c *Client) GetScheduledEvent(ctx context.Context, guildID, eventID types.Snowflake, withUserCount bool) (*types.ScheduledEvent, error) {
	endpoint := fmt.Sprintf("/guilds/%s/scheduled-events/%s", guildID, eventID)
	if withUserCount {
		endpoint += "?with_user_count=true"
	}

	var event types.ScheduledEvent
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &event)
	return &event, err
}

// ModifyScheduledEvent modifies a scheduled event
func (c *Client) ModifyScheduledEvent(ctx context.Context, guildID, eventID types.Snowflake, event *types.ScheduledEventParams) (*types.ScheduledEvent, error) {
	var updatedEvent types.ScheduledEvent
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/scheduled-events/%s", guildID, eventID),
		Body:     event,
	}, &updatedEvent)
	return &updatedEvent, err
}

// DeleteScheduledEvent deletes a scheduled event
func (c *Client) DeleteScheduledEvent(ctx context.Context, guildID, eventID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/guilds/%s/scheduled-events/%s", guildID, eventID),
	})
	return err
}

// ListActiveThreads lists active threads in a guild
func (c *Client) ListActiveThreads(ctx context.Context, guildID types.Snowflake) ([]*types.Channel, error) {
	var result struct {
		Threads []*types.Channel      `json:"threads"`
		Members []*types.ThreadMember `json:"members"`
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/threads/active", guildID),
	}, &result)
	return result.Threads, err
}

// Interact sends an interaction (e.g. button click or slash command)
func (c *Client) Interact(ctx context.Context, interaction *types.InteractionPayload) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/interactions",
		Body:     interaction,
	})
	return err
}

// SearchApplicationCommands searches for slash commands
func (c *Client) SearchApplicationCommands(ctx context.Context, channelID types.Snowflake, query string) (interface{}, error) {
	endpoint := fmt.Sprintf("/channels/%s/application-commands/search?query=%s&type=1&limit=25", channelID, url.QueryEscape(query))
	var result interface{}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &result)
	return result, err
}

// GetUserSettings returns the current user's settings
func (c *Client) GetUserSettings(ctx context.Context) (*types.UserSettings, error) {
	var settings types.UserSettings
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: "/users/@me/settings",
	}, &settings)
	return &settings, err
}

// UpdateUserSettings updates the current user's settings
func (c *Client) UpdateUserSettings(ctx context.Context, settings *types.UserSettings) (*types.UserSettings, error) {
	var newSettings types.UserSettings
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: "/users/@me/settings",
		Body:     settings,
	}, &newSettings)
	return &newSettings, err
}

// ============================================
// Sticker Methods
// ============================================

// ListGuildStickers lists stickers in a guild
func (c *Client) ListGuildStickers(ctx context.Context, guildID types.Snowflake) ([]*types.Sticker, error) {
	var stickers []*types.Sticker
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/stickers", guildID),
	}, &stickers)
	return stickers, err
}

// GetGuildSticker gets a sticker in a guild
func (c *Client) GetGuildSticker(ctx context.Context, guildID, stickerID types.Snowflake) (*types.Sticker, error) {
	var sticker types.Sticker
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/stickers/%s", guildID, stickerID),
	}, &sticker)
	return &sticker, err
}

// GetDetectableApplications returns a list of detectable applications
func (c *Client) GetDetectableApplications(ctx context.Context) ([]*types.DetectableApplication, error) {
	var apps []*types.DetectableApplication
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: "/applications/detectable",
	}, &apps)
	return apps, err
}

// CreateGuildSticker creates a sticker in a guild
func (c *Client) CreateGuildSticker(ctx context.Context, guildID types.Snowflake, name, description, tags string, fileData []byte, filename string) (*types.Sticker, error) {
	endpoint := fmt.Sprintf("/guilds/%s/stickers", guildID)
	fields := map[string]string{
		"name":        name,
		"description": description,
		"tags":        tags,
	}

	resp, err := c.DoMultipart(ctx, endpoint, http.MethodPost, fields, "file", fileData, filename)
	if err != nil {
		return nil, err
	}

	var sticker types.Sticker
	if err := json.Unmarshal(resp, &sticker); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sticker: %w", err)
	}
	return &sticker, nil
}

// ModifyGuildSticker modifies a sticker
func (c *Client) ModifyGuildSticker(ctx context.Context, guildID, stickerID types.Snowflake, name, description, tags string) (*types.Sticker, error) {
	body := map[string]string{}
	if name != "" {
		body["name"] = name
	}
	if description != "" {
		body["description"] = description
	}
	if tags != "" {
		body["tags"] = tags
	}

	var sticker types.Sticker
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/stickers/%s", guildID, stickerID),
		Body:     body,
	}, &sticker)
	return &sticker, err
}

// DeleteGuildSticker deletes a sticker
func (c *Client) DeleteGuildSticker(ctx context.Context, guildID, stickerID types.Snowflake) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodDelete,
		Endpoint: fmt.Sprintf("/guilds/%s/stickers/%s", guildID, stickerID),
	})
	return err
}

// GetStickerPacks gets the list of sticker packs
func (c *Client) GetStickerPacks(ctx context.Context) ([]*types.StickerPack, error) {
	var result struct {
		StickerPacks []*types.StickerPack `json:"sticker_packs"`
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: "/sticker-packs",
	}, &result)
	return result.StickerPacks, err
}

// GetGuildVanityURL returns the vanity URL code of a guild
func (c *Client) GetGuildVanityURL(ctx context.Context, guildID types.Snowflake) (*string, error) {
	var result struct {
		Code *string `json:"code"`
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/guilds/%s/vanity-url", guildID),
	}, &result)
	return result.Code, err
}

// ChangeGuildVanityURL changes the vanity URL code of a guild
func (c *Client) ChangeGuildVanityURL(ctx context.Context, guildID types.Snowflake, code string) (*string, error) {
	body := map[string]string{
		"code": code,
	}
	var result struct {
		Code *string `json:"code"`
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/guilds/%s/vanity-url", guildID),
		Body:     body,
	}, &result)
	return result.Code, err
}

// ============================================
// Remote Auth Mobile API Methods
// ============================================

// RemoteAuthSession represents a remote auth session response
type RemoteAuthSession struct {
	HandshakeToken string `json:"handshake_token"`
}

// CreateRemoteAuthSession creates a remote auth session from the mobile client.
// This is used when a logged-in mobile device scans a QR code to authorize a desktop login.
// The fingerprint should be extracted from the QR code URL (https://discord.com/ra/<fingerprint>).
func (c *Client) CreateRemoteAuthSession(ctx context.Context, fingerprint string) (*RemoteAuthSession, error) {
	var session RemoteAuthSession
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/users/@me/remote-auth",
		Body: map[string]string{
			"fingerprint": fingerprint,
		},
	}, &session)
	return &session, err
}

// FinishRemoteAuth finishes a remote auth session by sending an authentication token to the desktop client.
// This approves the login request and allows the desktop client to authenticate.
func (c *Client) FinishRemoteAuth(ctx context.Context, handshakeToken string, temporaryToken bool) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/users/@me/remote-auth/finish",
		Body: map[string]interface{}{
			"handshake_token": handshakeToken,
			"temporary_token": temporaryToken,
		},
	})
	return err
}

// CancelRemoteAuth cancels a remote auth session without sending an authentication token.
// This denies the login request.
func (c *Client) CancelRemoteAuth(ctx context.Context, handshakeToken string) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/users/@me/remote-auth/cancel",
		Body: map[string]string{
			"handshake_token": handshakeToken,
		},
	})
	return err
}

// ExtractRemoteAuthFingerprint extracts the fingerprint from a Discord remote auth URL.
// The URL format is: https://discord.com/ra/<fingerprint>
func ExtractRemoteAuthFingerprint(qrURL string) (string, error) {
	parsed, err := url.Parse(qrURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	// Expected format: /ra/<fingerprint>
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "ra" {
		return "", fmt.Errorf("invalid remote auth URL format, expected /ra/<fingerprint>")
	}

	return parts[1], nil
}
