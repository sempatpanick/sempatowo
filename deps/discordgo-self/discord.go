package discordgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hytams/discordgo-self/api"
	"github.com/hytams/discordgo-self/gateway"
	"github.com/hytams/discordgo-self/props"
	"github.com/hytams/discordgo-self/state"
	"github.com/hytams/discordgo-self/types"
	"github.com/hytams/discordgo-self/voice"
)

// Re-export types for convenience
type (
	Snowflake         = types.Snowflake
	User              = types.User
	CurrentUser       = types.CurrentUser
	Channel           = types.Channel
	Message           = types.Message
	MessageSendData   = types.MessageSendData
	MessageEditData   = types.MessageEditData
	Embed             = types.Embed
	EmbedField        = types.EmbedField
	Relationship      = types.Relationship
	File              = types.File
	Guild             = types.Guild
	Role              = types.Role
	Member            = types.Member
	Invite            = types.Invite
	ReadState         = types.ReadState
	Session           = types.Session
	UserSettings      = types.UserSettings
	VoiceServerUpdate = types.VoiceServerUpdate
)

// Re-export gateway events
const (
	EventReady                 = gateway.EventReady
	EventResumed               = gateway.EventResumed
	EventMessageCreate         = gateway.EventMessageCreate
	EventMessageUpdate         = gateway.EventMessageUpdate
	EventMessageDelete         = gateway.EventMessageDelete
	EventMessageReactionAdd    = gateway.EventMessageReactionAdd
	EventMessageReactionRemove = gateway.EventMessageReactionRemove
	EventTypingStart           = gateway.EventTypingStart
	EventPresenceUpdate        = gateway.EventPresenceUpdate
	EventChannelCreate         = gateway.EventChannelCreate
	EventChannelUpdate         = gateway.EventChannelUpdate
	EventChannelDelete         = gateway.EventChannelDelete
	EventRelationshipAdd       = gateway.EventRelationshipAdd
	EventRelationshipRemove    = gateway.EventRelationshipRemove
	EventGuildCreate           = gateway.EventGuildCreate
	EventGuildUpdate           = gateway.EventGuildUpdate
	EventGuildDelete           = gateway.EventGuildDelete
	EventGuildMemberAdd        = gateway.EventGuildMemberAdd
	EventGuildMemberUpdate     = gateway.EventGuildMemberUpdate
	EventGuildMemberRemove     = gateway.EventGuildMemberRemove
	EventGuildRoleCreate       = gateway.EventGuildRoleCreate
	EventGuildRoleUpdate       = gateway.EventGuildRoleUpdate
	EventGuildRoleDelete       = gateway.EventGuildRoleDelete
	EventGuildBanAdd           = gateway.EventGuildBanAdd
	EventGuildBanRemove        = gateway.EventGuildBanRemove
	EventVoiceStateUpdate      = gateway.EventVoiceStateUpdate
	EventVoiceServerUpdate     = gateway.EventVoiceServerUpdate
	EventInteractionCreate     = gateway.EventInteractionCreate
	EventReadStateUpdate       = gateway.EventReadStateUpdate
	EventSessionsReplace       = gateway.EventSessionsReplace
	EventUserSettingsUpdate    = gateway.EventUserSettingsUpdate
)

// Status constants
const (
	StatusOnline    = types.StatusOnline
	StatusIdle      = types.StatusIdle
	StatusDND       = types.StatusDND
	StatusInvisible = types.StatusInvisible
	StatusOffline   = types.StatusOffline
)

// Client represents the main Discord client.
type Client struct {
	Token           string
	API             *api.Client
	Gateway         *gateway.Gateway
	User            *CurrentUser
	State           *state.State
	SuperProperties *props.SuperProperties

	messageCreateHandlers []func(*Message)
	messageUpdateHandlers []func(*Message)
	messageDeleteHandlers []func(channelID, messageID Snowflake)
	readyHandlers         []func()
	rawHandlers           []func(string, json.RawMessage)

	guildCreateHandlers       []func(*Guild)
	guildUpdateHandlers       []func(*Guild)
	guildDeleteHandlers       []func(guildID Snowflake)
	guildMemberAddHandlers    []func(*Member, Snowflake)
	guildMemberRemoveHandlers []func(userID, guildID Snowflake)
	guildMemberUpdateHandlers []func(*Member, Snowflake)
	guildBanAddHandlers       []func(userID, guildID Snowflake)
	guildBanRemoveHandlers    []func(userID, guildID Snowflake)

	channelCreateHandlers []func(*Channel)
	channelUpdateHandlers []func(*Channel)
	channelDeleteHandlers []func(*Channel)

	roleCreateHandlers []func(*Role, Snowflake)
	roleUpdateHandlers []func(*Role, Snowflake)
	roleDeleteHandlers []func(roleID, guildID Snowflake)

	readStateUpdateHandlers    []func(*ReadState)
	sessionsReplaceHandlers    []func([]*Session)
	userSettingsUpdateHandlers []func(*UserSettings)
	voiceServerUpdateHandlers  []func(*VoiceServerUpdate)

	voiceStateUpdateHandlers  []func(*types.VoiceState)
	interactionCreateHandlers []func(*types.Interaction)
	typingStartHandlers       []func(channelID, userID Snowflake)
	presenceUpdateHandlers    []func(*types.Presence)

	voiceConnections map[Snowflake]*voice.VoiceConnection
	detectableApps   []*types.DetectableApplication
	detectableAppsMu sync.RWMutex

	Debug  bool
	Config *Config

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// Config represents the client configuration
type Config struct {
	// StateEnabled controls whether the library manages state (caching)
	StateEnabled bool

	// CacheMembers controls whether to cache guild members (high memory usage)
	CacheMembers bool

	// CachePresences controls whether to cache presences (high memory usage)
	CachePresences bool

	// CacheMessages controls whether to cache messages
	CacheMessages bool

	// MaxMessageCount is the maximum number of messages to cache per channel
	MaxMessageCount int
}

// DefaultConfig returns a default configuration (Full State)
func DefaultConfig() *Config {
	return &Config{
		StateEnabled:    true,
		CacheMembers:    true,
		CachePresences:  true,
		CacheMessages:   true,
		MaxMessageCount: 100,
	}
}

// Option is a function that configures a Client
type Option func(*Client)

// WithConfig sets the client configuration
func WithConfig(config *Config) Option {
	return func(c *Client) {
		c.Config = config
	}
}

// WithDebug enables debug mode
func WithDebug(debug bool) Option {
	return func(c *Client) {
		c.Debug = debug
		if c.Gateway != nil {
			c.Gateway.Debug = debug
		}
		if c.API != nil {
			c.API.Debug = debug
		}
	}
}

// WithSuperProperties sets custom super properties
func WithSuperProperties(sp *props.SuperProperties) Option {
	return func(c *Client) {
		c.SuperProperties = sp
		if c.Gateway != nil {
			c.Gateway.SuperProperties = sp
		}
		if c.API != nil {
			c.API.SuperProperties = sp
		}
	}
}

// WithLocale sets the locale
func WithLocale(locale string) Option {
	return func(c *Client) {
		if c.SuperProperties != nil {
			c.SuperProperties.SystemLocale = locale
		}
	}
}

// New creates a new Discord client instance with the provided token and options.
func New(token string, opts ...Option) (*Client, error) {
	if token == "" {
		return nil, errors.New("token is required")
	}

	sp := props.NewSuperProperties()

	apiConfig := api.ClientConfig{
		Token:           token,
		SuperProperties: sp,
	}
	apiClient, err := api.NewClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create api client: %w", err)
	}

	gatewayConfig := gateway.GatewayConfig{
		Token:           token,
		SuperProperties: sp,
	}
	gatewayClient, err := gateway.NewGateway(gatewayConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		Token:            token,
		API:              apiClient,
		Gateway:          gatewayClient,
		SuperProperties:  sp,
		Config:           DefaultConfig(),
		State:            state.NewState(),
		ctx:              ctx,
		cancel:           cancel,
		voiceConnections: make(map[Snowflake]*voice.VoiceConnection),
	}

	for _, opt := range opts {
		opt(client)
	}

	client.setupGatewayHandlers()

	return client, nil
}

// setupGatewayHandlers sets up internal gateway event handlers.
func (c *Client) setupGatewayHandlers() {
	c.Gateway.On("*", func(event string, data json.RawMessage) {
		c.mu.RLock()
		handlers := c.rawHandlers
		c.mu.RUnlock()

		for _, handler := range handlers {
			handler(event, data)
		}

		switch event {
		case EventReady:
			c.handleReady(data)
		case EventMessageCreate:
			c.handleMessageCreate(data)
		case EventMessageUpdate:
			c.handleMessageUpdate(data)
		case EventMessageDelete:
			c.handleMessageDelete(data)
		case EventGuildRoleCreate:
			c.handleRoleCreate(data)
		case EventGuildRoleUpdate:
			c.handleRoleUpdate(data)
		case EventGuildRoleDelete:
			c.handleRoleDelete(data)
		case EventGuildCreate:
			c.handleGuildCreate(data)
		case EventGuildUpdate:
			c.handleGuildUpdate(data)
		case EventGuildDelete:
			c.handleGuildDelete(data)
		case EventGuildMemberAdd:
			c.handleGuildMemberAdd(data)
		case EventGuildMemberUpdate:
			c.handleGuildMemberUpdate(data)
		case EventGuildMemberRemove:
			c.handleGuildMemberRemove(data)
		case EventGuildBanAdd:
			c.handleGuildBanAdd(data)
		case EventGuildBanRemove:
			c.handleGuildBanRemove(data)
		case EventChannelCreate:
			c.handleChannelCreate(data)
		case EventChannelUpdate:
			c.handleChannelUpdate(data)
		case EventChannelDelete:
			c.handleChannelDelete(data)
		case EventVoiceStateUpdate:
			c.handleVoiceStateUpdate(data)
		case EventInteractionCreate:
			c.handleInteractionCreate(data)
		case EventTypingStart:
			c.handleTypingStart(data)
		case EventPresenceUpdate:
			c.handlePresenceUpdate(data)
		case EventReadStateUpdate:
			c.handleReadStateUpdate(data)
		case EventSessionsReplace:
			c.handleSessionsReplace(data)
		case EventUserSettingsUpdate:
			c.handleUserSettingsUpdate(data)
		case EventVoiceServerUpdate:
			c.handleVoiceServerUpdate(data)
		}
	})
}

// handleReady handles the READY event.
func (c *Client) handleReady(_ json.RawMessage) {
	if c.Gateway.ReadyData != nil {
		c.mu.Lock()
		c.User = c.Gateway.ReadyData.User

		if c.Config.StateEnabled {
			for _, ch := range c.Gateway.ReadyData.PrivateChannels {
				c.State.SetChannel(ch)
			}

			for _, rel := range c.Gateway.ReadyData.Relationships {
				c.State.SetRelationship(rel)
			}

			if c.Gateway.ReadyData.UserSettings != nil {
				c.State.SetUserSettings(c.Gateway.ReadyData.UserSettings)
			}

			if len(c.Gateway.ReadyData.ReadState) > 0 {
				var readStates []*types.ReadState

				if err := json.Unmarshal(c.Gateway.ReadyData.ReadState, &readStates); err != nil {
					var wrapper struct {
						Entries []*types.ReadState `json:"entries"`
					}
					if err2 := json.Unmarshal(c.Gateway.ReadyData.ReadState, &wrapper); err2 != nil {
						if c.Debug {
							fmt.Printf("[ReadState] Warning: could not parse read_state (tried array and object formats)\n")
						}
					} else {
						readStates = wrapper.Entries
					}
				}

				for _, rs := range readStates {
					if rs != nil {
						c.State.SetReadState(rs)
					}
				}
			}

			for _, s := range c.Gateway.ReadyData.Sessions {
				c.State.SetSession(s)
			}

			c.State.Experiments = c.Gateway.ReadyData.Experiments
		}
		c.mu.Unlock()
	}

	// Call ready handlers
	c.mu.RLock()
	handlers := c.readyHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler()
	}
}

// handleMessageCreate handles MESSAGE_CREATE events
func (c *Client) handleMessageCreate(data json.RawMessage) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	if c.Config.StateEnabled && c.Config.CacheMessages {
		c.State.MessageAdd(&msg, c.Config.MaxMessageCount)
	}

	c.mu.RLock()
	handlers := c.messageCreateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&msg)
	}
}

// handleMessageUpdate handles MESSAGE_UPDATE events
func (c *Client) handleMessageUpdate(data json.RawMessage) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return
	}

	if c.Config.StateEnabled && c.Config.CacheMessages {
		c.State.MessageAdd(&msg, c.Config.MaxMessageCount)
	}

	c.mu.RLock()
	handlers := c.messageUpdateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&msg)
	}
}

// handleMessageDelete handles MESSAGE_DELETE events
func (c *Client) handleMessageDelete(data json.RawMessage) {
	var deleteData struct {
		ID        Snowflake `json:"id"`
		ChannelID Snowflake `json:"channel_id"`
		GuildID   Snowflake `json:"guild_id,omitempty"`
	}
	if err := json.Unmarshal(data, &deleteData); err != nil {
		return
	}

	if c.Config.StateEnabled && c.Config.CacheMessages {
		c.State.MessageRemove(deleteData.ChannelID, deleteData.ID)
	}

	c.mu.RLock()
	handlers := c.messageDeleteHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(deleteData.ChannelID, deleteData.ID)
	}
}

// handleRoleCreate handles GUILD_ROLE_CREATE events
func (c *Client) handleRoleCreate(data json.RawMessage) {
	var roleData struct {
		GuildID Snowflake `json:"guild_id"`
		Role    *Role     `json:"role"`
	}
	if err := json.Unmarshal(data, &roleData); err != nil {
		return
	}

	c.mu.RLock()
	handlers := c.roleCreateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(roleData.Role, roleData.GuildID)
	}
}

// handleRoleUpdate handles GUILD_ROLE_UPDATE events
func (c *Client) handleRoleUpdate(data json.RawMessage) {
	var roleData struct {
		GuildID Snowflake `json:"guild_id"`
		Role    *Role     `json:"role"`
	}
	if err := json.Unmarshal(data, &roleData); err != nil {
		return
	}

	c.mu.RLock()
	handlers := c.roleUpdateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(roleData.Role, roleData.GuildID)
	}
}

// handleRoleDelete handles GUILD_ROLE_DELETE events
func (c *Client) handleRoleDelete(data json.RawMessage) {
	var roleData struct {
		GuildID Snowflake `json:"guild_id"`
		RoleID  Snowflake `json:"role_id"`
	}
	if err := json.Unmarshal(data, &roleData); err != nil {
		return
	}

	c.mu.RLock()
	handlers := c.roleDeleteHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(roleData.RoleID, roleData.GuildID)
	}
}

// handleGuildCreate handles GUILD_CREATE events
func (c *Client) handleGuildCreate(data json.RawMessage) {
	var guild Guild
	if err := json.Unmarshal(data, &guild); err != nil {
		return
	}

	if c.Config.StateEnabled {
		c.State.SetGuild(&guild)
	}

	c.mu.RLock()
	handlers := c.guildCreateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&guild)
	}
}

// handleGuildUpdate handles GUILD_UPDATE events
func (c *Client) handleGuildUpdate(data json.RawMessage) {
	var guild Guild
	if err := json.Unmarshal(data, &guild); err != nil {
		return
	}

	if c.Config.StateEnabled {
		c.State.SetGuild(&guild)

		if c.Config.CacheMembers {
			for _, m := range guild.Members {
				// Cache member with guild ID
				c.State.SetMember(guild.ID, m)
			}
		}

		if c.Config.CachePresences {
			for _, p := range guild.Presences {
				c.State.SetPresence(p.User.ID, p)
			}
		}

		for _, vs := range guild.VoiceStates {
			c.State.SetVoiceState(guild.ID, vs)
		}
	}

	c.mu.RLock()
	handlers := c.guildUpdateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&guild)
	}
}

// handleGuildDelete handles GUILD_DELETE events
func (c *Client) handleGuildDelete(data json.RawMessage) {
	var guildData struct {
		ID Snowflake `json:"id"`
	}
	if err := json.Unmarshal(data, &guildData); err != nil {
		return
	}

	if c.Config.StateEnabled {
		c.State.RemoveGuild(guildData.ID)
	}

	c.mu.RLock()
	handlers := c.guildDeleteHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(guildData.ID)
	}
}

// handleGuildMemberAdd handles GUILD_MEMBER_ADD events
func (c *Client) handleGuildMemberAdd(data json.RawMessage) {
	var memberData struct {
		GuildID Snowflake `json:"guild_id"`
		Member
	}
	if err := json.Unmarshal(data, &memberData); err != nil {
		return
	}

	if c.Config.StateEnabled && c.Config.CacheMembers {
		c.State.SetMember(memberData.GuildID, &memberData.Member)
	}

	c.mu.RLock()
	handlers := c.guildMemberAddHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&memberData.Member, memberData.GuildID)
	}
}

// handleGuildMemberUpdate handles GUILD_MEMBER_UPDATE events
func (c *Client) handleGuildMemberUpdate(data json.RawMessage) {
	var memberData struct {
		GuildID Snowflake `json:"guild_id"`
		Member
	}
	if err := json.Unmarshal(data, &memberData); err != nil {
		return
	}

	if c.Config.StateEnabled && c.Config.CacheMembers {
		c.State.SetMember(memberData.GuildID, &memberData.Member)
	}

	c.mu.RLock()
	handlers := c.guildMemberUpdateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&memberData.Member, memberData.GuildID)
	}
}

// handleGuildMemberRemove handles GUILD_MEMBER_REMOVE events
func (c *Client) handleGuildMemberRemove(data json.RawMessage) {
	var memberData struct {
		GuildID Snowflake `json:"guild_id"`
		User    struct {
			ID Snowflake `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(data, &memberData); err != nil {
		return
	}

	if c.Config.StateEnabled && c.Config.CacheMembers {
		c.State.RemoveMember(memberData.GuildID, memberData.User.ID)
	}

	c.mu.RLock()
	handlers := c.guildMemberRemoveHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(memberData.User.ID, memberData.GuildID)
	}
}

// handleGuildBanAdd handles GUILD_BAN_ADD events
func (c *Client) handleGuildBanAdd(data json.RawMessage) {
	var banData struct {
		GuildID Snowflake `json:"guild_id"`
		User    struct {
			ID Snowflake `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(data, &banData); err != nil {
		return
	}

	c.mu.RLock()
	handlers := c.guildBanAddHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(banData.User.ID, banData.GuildID)
	}
}

// handleGuildBanRemove handles GUILD_BAN_REMOVE events
func (c *Client) handleGuildBanRemove(data json.RawMessage) {
	var banData struct {
		GuildID Snowflake `json:"guild_id"`
		User    struct {
			ID Snowflake `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(data, &banData); err != nil {
		return
	}

	c.mu.RLock()
	handlers := c.guildBanRemoveHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(banData.User.ID, banData.GuildID)
	}
}

// handleChannelCreate handles CHANNEL_CREATE events
func (c *Client) handleChannelCreate(data json.RawMessage) {
	var channel Channel
	if err := json.Unmarshal(data, &channel); err != nil {
		return
	}

	if c.Config.StateEnabled {
		c.State.SetChannel(&channel)
	}

	c.mu.RLock()
	handlers := c.channelCreateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&channel)
	}
}

// handleChannelUpdate handles CHANNEL_UPDATE events
func (c *Client) handleChannelUpdate(data json.RawMessage) {
	var channel Channel
	if err := json.Unmarshal(data, &channel); err != nil {
		return
	}

	if c.Config.StateEnabled {
		c.State.SetChannel(&channel)
	}

	c.mu.RLock()
	handlers := c.channelUpdateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&channel)
	}
}

// handleChannelDelete handles CHANNEL_DELETE events
func (c *Client) handleChannelDelete(data json.RawMessage) {
	var channel Channel
	if err := json.Unmarshal(data, &channel); err != nil {
		return
	}

	if c.Config.StateEnabled {
		c.State.RemoveChannel(channel.ID)
	}

	c.mu.RLock()
	handlers := c.channelDeleteHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&channel)
	}
}

// handleVoiceStateUpdate handles VOICE_STATE_UPDATE events
func (c *Client) handleVoiceStateUpdate(data json.RawMessage) {
	var vs types.VoiceState
	if err := json.Unmarshal(data, &vs); err != nil {
		return
	}

	if c.Config.StateEnabled {
		if vs.ChannelID.IsZero() {
			c.State.RemoveVoiceState(vs.GuildID, vs.UserID)
		} else {
			c.State.SetVoiceState(vs.GuildID, &vs)
		}
	}

	c.mu.RLock()
	handlers := c.voiceStateUpdateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&vs)
	}
}

// handleInteractionCreate handles INTERACTION_CREATE events
func (c *Client) handleInteractionCreate(data json.RawMessage) {
	var i types.Interaction
	if err := json.Unmarshal(data, &i); err != nil {
		return
	}

	c.mu.RLock()
	handlers := c.interactionCreateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&i)
	}
}

// handleTypingStart handles TYPING_START events
func (c *Client) handleTypingStart(data json.RawMessage) {
	var typingData struct {
		ChannelID Snowflake `json:"channel_id"`
		UserID    Snowflake `json:"user_id"`
	}
	if err := json.Unmarshal(data, &typingData); err != nil {
		return
	}

	c.mu.RLock()
	handlers := c.typingStartHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(typingData.ChannelID, typingData.UserID)
	}
}

// handlePresenceUpdate handles PRESENCE_UPDATE events
func (c *Client) handlePresenceUpdate(data json.RawMessage) {
	var p types.Presence
	if err := json.Unmarshal(data, &p); err != nil {
		fmt.Printf("Error unmarshalling presence: %v\n", err)
		return
	}

	if c.Config.StateEnabled && c.Config.CachePresences {
		c.State.SetPresence(p.User.ID, &p)
	}

	c.mu.RLock()
	handlers := c.presenceUpdateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&p)
	}
}

// handleReadStateUpdate handles READ_STATE_UPDATE events
func (c *Client) handleReadStateUpdate(data json.RawMessage) {
	var rs ReadState
	if err := json.Unmarshal(data, &rs); err != nil {
		return
	}

	if c.Config.StateEnabled {
		c.State.SetReadState(&rs)
	}

	c.mu.RLock()
	handlers := c.readStateUpdateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&rs)
	}
}

// handleSessionsReplace handles SESSIONS_REPLACE events
func (c *Client) handleSessionsReplace(data json.RawMessage) {
	var sessions []*Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return
	}

	if c.Config.StateEnabled {
		c.mu.Lock()
		c.State.Sessions = make(map[string]*Session)
		for _, s := range sessions {
			c.State.Sessions[s.SessionID] = s
		}
		c.mu.Unlock()
	}

	c.mu.RLock()
	handlers := c.sessionsReplaceHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(sessions)
	}
}

// handleUserSettingsUpdate handles USER_SETTINGS_UPDATE events
func (c *Client) handleUserSettingsUpdate(data json.RawMessage) {
	var settings UserSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return
	}

	if c.Config.StateEnabled {
		c.State.SetUserSettings(&settings)
	}

	c.mu.RLock()
	handlers := c.userSettingsUpdateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&settings)
	}
}

// handleVoiceServerUpdate handles VOICE_SERVER_UPDATE events
func (c *Client) handleVoiceServerUpdate(data json.RawMessage) {
	var vsu VoiceServerUpdate
	if err := json.Unmarshal(data, &vsu); err != nil {
		return
	}

	c.mu.RLock()
	handlers := c.voiceServerUpdateHandlers
	c.mu.RUnlock()

	for _, handler := range handlers {
		go handler(&vsu)
	}
}

// Open connects to the Discord gateway
func (c *Client) Open() error {
	return c.Gateway.Connect()
}

// Close disconnects from Discord
func (c *Client) Close() error {
	c.cancel()
	return c.Gateway.Close()
}

// WaitUntilReady blocks until the client is ready
func (c *Client) WaitUntilReady(timeout time.Duration) error {
	start := time.Now()
	for {
		if c.Gateway.IsConnected() && c.User != nil {
			return nil
		}

		if time.Since(start) > timeout {
			return errors.New("timeout waiting for ready")
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// ============================================
// Event Handler Registration
// ============================================

// OnReady registers a handler for the READY event
func (c *Client) OnReady(handler func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readyHandlers = append(c.readyHandlers, handler)
}

// OnMessageCreate registers a handler for MESSAGE_CREATE events
func (c *Client) OnMessageCreate(handler func(*Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messageCreateHandlers = append(c.messageCreateHandlers, handler)
}

// OnMessageUpdate registers a handler for MESSAGE_UPDATE events
func (c *Client) OnMessageUpdate(handler func(*Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messageUpdateHandlers = append(c.messageUpdateHandlers, handler)
}

// OnMessageDelete registers a handler for MESSAGE_DELETE events
func (c *Client) OnMessageDelete(handler func(channelID, messageID Snowflake)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messageDeleteHandlers = append(c.messageDeleteHandlers, handler)
}

// OnReadStateUpdate registers a handler for READ_STATE_UPDATE events
func (c *Client) OnReadStateUpdate(handler func(*ReadState)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readStateUpdateHandlers = append(c.readStateUpdateHandlers, handler)
}

// OnSessionsReplace registers a handler for SESSIONS_REPLACE events
func (c *Client) OnSessionsReplace(handler func([]*Session)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionsReplaceHandlers = append(c.sessionsReplaceHandlers, handler)
}

// OnUserSettingsUpdate registers a handler for USER_SETTINGS_UPDATE events
func (c *Client) OnUserSettingsUpdate(handler func(*UserSettings)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userSettingsUpdateHandlers = append(c.userSettingsUpdateHandlers, handler)
}

// OnVoiceServerUpdate registers a handler for VOICE_SERVER_UPDATE events
func (c *Client) OnVoiceServerUpdate(handler func(*VoiceServerUpdate)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.voiceServerUpdateHandlers = append(c.voiceServerUpdateHandlers, handler)
}

// OnRawEvent registers a handler for all raw gateway events
func (c *Client) OnRawEvent(handler func(string, json.RawMessage)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rawHandlers = append(c.rawHandlers, handler)
}

// OnGuildCreate registers a handler for GUILD_CREATE events
func (c *Client) OnGuildCreate(handler func(*Guild)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.guildCreateHandlers = append(c.guildCreateHandlers, handler)
}

// OnGuildUpdate registers a handler for GUILD_UPDATE events
func (c *Client) OnGuildUpdate(handler func(*Guild)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.guildUpdateHandlers = append(c.guildUpdateHandlers, handler)
}

// OnGuildDelete registers a handler for GUILD_DELETE events
func (c *Client) OnGuildDelete(handler func(guildID Snowflake)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.guildDeleteHandlers = append(c.guildDeleteHandlers, handler)
}

// OnGuildMemberAdd registers a handler for GUILD_MEMBER_ADD events
func (c *Client) OnGuildMemberAdd(handler func(*Member, Snowflake)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.guildMemberAddHandlers = append(c.guildMemberAddHandlers, handler)
}

// OnGuildMemberRemove registers a handler for GUILD_MEMBER_REMOVE events
func (c *Client) OnGuildMemberRemove(handler func(userID, guildID Snowflake)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.guildMemberRemoveHandlers = append(c.guildMemberRemoveHandlers, handler)
}

// OnGuildMemberUpdate registers a handler for GUILD_MEMBER_UPDATE events
func (c *Client) OnGuildMemberUpdate(handler func(*Member, Snowflake)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.guildMemberUpdateHandlers = append(c.guildMemberUpdateHandlers, handler)
}

// OnGuildBanAdd registers a handler for GUILD_BAN_ADD events
func (c *Client) OnGuildBanAdd(handler func(userID, guildID Snowflake)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.guildBanAddHandlers = append(c.guildBanAddHandlers, handler)
}

// OnGuildBanRemove registers a handler for GUILD_BAN_REMOVE events
func (c *Client) OnGuildBanRemove(handler func(userID, guildID Snowflake)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.guildBanRemoveHandlers = append(c.guildBanRemoveHandlers, handler)
}

// OnChannelCreate registers a handler for CHANNEL_CREATE events
func (c *Client) OnChannelCreate(handler func(*Channel)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channelCreateHandlers = append(c.channelCreateHandlers, handler)
}

// OnChannelUpdate registers a handler for CHANNEL_UPDATE events
func (c *Client) OnChannelUpdate(handler func(*Channel)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channelUpdateHandlers = append(c.channelUpdateHandlers, handler)
}

// OnChannelDelete registers a handler for CHANNEL_DELETE events
func (c *Client) OnChannelDelete(handler func(*Channel)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channelDeleteHandlers = append(c.channelDeleteHandlers, handler)
}

// OnVoiceStateUpdate registers a handler for VOICE_STATE_UPDATE events
func (c *Client) OnVoiceStateUpdate(handler func(*types.VoiceState)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.voiceStateUpdateHandlers = append(c.voiceStateUpdateHandlers, handler)
}

// OnInteractionCreate registers a handler for INTERACTION_CREATE events
func (c *Client) OnInteractionCreate(handler func(*types.Interaction)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interactionCreateHandlers = append(c.interactionCreateHandlers, handler)
}

// OnTypingStart registers a handler for TYPING_START events
func (c *Client) OnTypingStart(handler func(channelID, userID Snowflake)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.typingStartHandlers = append(c.typingStartHandlers, handler)
}

// OnPresenceUpdate registers a handler for PRESENCE_UPDATE events
func (c *Client) OnPresenceUpdate(handler func(*types.Presence)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.presenceUpdateHandlers = append(c.presenceUpdateHandlers, handler)
}

// OnRoleCreate registers a handler for GUILD_ROLE_CREATE events
func (c *Client) OnRoleCreate(handler func(*Role, Snowflake)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.roleCreateHandlers = append(c.roleCreateHandlers, handler)
}

// OnRoleUpdate registers a handler for GUILD_ROLE_UPDATE events
func (c *Client) OnRoleUpdate(handler func(*Role, Snowflake)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.roleUpdateHandlers = append(c.roleUpdateHandlers, handler)
}

// OnRoleDelete registers a handler for GUILD_ROLE_DELETE events
func (c *Client) OnRoleDelete(handler func(roleID, guildID Snowflake)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.roleDeleteHandlers = append(c.roleDeleteHandlers, handler)
}

// ============================================
// Message Methods
// ============================================

// SendMessage sends a message to a channel
func (c *Client) SendMessage(channelID Snowflake, content string) (*Message, error) {
	return c.API.SendMessageSimple(c.ctx, channelID, content)
}

// SendMessageComplex sends a message with full options
func (c *Client) SendMessageComplex(channelID Snowflake, data *MessageSendData) (*Message, error) {
	return c.API.SendMessage(c.ctx, channelID, data)
}

// SendEmbed sends a message with an embed
func (c *Client) SendEmbed(channelID Snowflake, embed *Embed) (*Message, error) {
	return c.API.SendMessage(c.ctx, channelID, &MessageSendData{
		Embeds: []*Embed{embed},
	})
}

// SendReply sends a reply to a message
func (c *Client) SendReply(channelID, messageID Snowflake, content string) (*Message, error) {
	return c.API.SendMessage(c.ctx, channelID, &MessageSendData{
		Content: content,
		MessageReference: &types.MessageReference{
			MessageID: messageID,
			ChannelID: channelID,
		},
	})
}

// EditMessage edits a message
func (c *Client) EditMessage(channelID, messageID Snowflake, content string) (*Message, error) {
	return c.API.EditMessage(c.ctx, channelID, messageID, &MessageEditData{
		Content: &content,
	})
}

// DeleteMessage deletes a message
func (c *Client) DeleteMessage(channelID, messageID Snowflake) error {
	return c.API.DeleteMessage(c.ctx, channelID, messageID)
}

// GetMessage gets a message by ID
func (c *Client) GetMessage(channelID, messageID Snowflake) (*Message, error) {
	return c.API.GetMessage(c.ctx, channelID, messageID)
}

// GetMessages gets messages from a channel
func (c *Client) GetMessages(channelID Snowflake, limit int) ([]*Message, error) {
	return c.API.GetMessages(c.ctx, channelID, limit, 0, 0, 0)
}

// AddReaction adds a reaction to a message
func (c *Client) AddReaction(channelID, messageID Snowflake, emoji string) error {
	return c.API.AddReaction(c.ctx, channelID, messageID, emoji)
}

// RemoveReaction removes a reaction from a message
func (c *Client) RemoveReaction(channelID, messageID Snowflake, emoji string) error {
	return c.API.RemoveReaction(c.ctx, channelID, messageID, emoji)
}

// TriggerTyping triggers the typing indicator in a channel
func (c *Client) TriggerTyping(channelID Snowflake) error {
	return c.API.TriggerTyping(c.ctx, channelID)
}

// ============================================
// File Upload Methods
// ============================================

// SendFile sends a file to a channel
func (c *Client) SendFile(channelID Snowflake, file *File, content string) (*Message, error) {
	return c.API.SendFile(c.ctx, channelID, file, content)
}

// SendImage sends an image file to a channel
func (c *Client) SendImage(channelID Snowflake, imagePath string, content string) (*Message, error) {
	return c.API.SendImage(c.ctx, channelID, imagePath, content)
}

// SendMessageWithFiles sends a message with multiple file attachments
func (c *Client) SendMessageWithFiles(channelID Snowflake, data *MessageSendData, files []*File) (*Message, error) {
	return c.API.SendMessageWithFiles(c.ctx, channelID, data, files)
}

// SendFileFromBytes sends a file from bytes
func (c *Client) SendFileFromBytes(channelID Snowflake, filename string, data []byte, content string) (*Message, error) {
	file := types.NewFileFromBytes(filename, data)
	return c.SendFile(channelID, file, content)
}

// NewFile creates a new file for upload
func NewFile(name string, reader interface{}) *File {
	switch r := reader.(type) {
	case []byte:
		return types.NewFileFromBytes(name, r)
	default:
		return nil
	}
}

// NewFileFromPath creates a new file from a path
func NewFileFromPath(path string) (*File, error) {
	return types.NewFileFromPath(path)
}

// Channel Methods
// ============================================

// GetChannel gets a channel by ID
func (c *Client) GetChannel(channelID Snowflake) (*Channel, error) {
	return c.API.GetChannel(c.ctx, channelID)
}

// CreateDM creates a DM channel with a user
func (c *Client) CreateDM(userID Snowflake) (*Channel, error) {
	return c.API.CreateDM(c.ctx, userID)
}

// GetUser gets a user by ID
func (c *Client) GetUser(userID Snowflake) (*User, error) {
	return c.API.GetUser(c.ctx, userID)
}

// GetCurrentUser gets the current authenticated user
func (c *Client) GetCurrentUser() (*CurrentUser, error) {
	return c.API.GetCurrentUser(c.ctx)
}

// ============================================
// Presence Methods
// ============================================

// SetStatus sets the user's status
func (c *Client) SetStatus(status string) error {
	return c.Gateway.SetStatus(status)
}

// SetOnline sets the user's status to online
func (c *Client) SetOnline() error {
	return c.SetStatus(string(StatusOnline))
}

// SetIdle sets the user's status to idle
func (c *Client) SetIdle() error {
	return c.SetStatus(string(StatusIdle))
}

// SetDND sets the user's status to do not disturb
func (c *Client) SetDND() error {
	return c.SetStatus(string(StatusDND))
}

// SetInvisible sets the user's status to invisible
func (c *Client) SetInvisible() error {
	return c.SetStatus(string(StatusInvisible))
}

// Guild Methods

// GetGuild gets a guild by ID
func (c *Client) GetGuild(guildID Snowflake) (*Guild, error) {
	return c.API.GetGuild(c.ctx, guildID)
}

// GetGuildChannels gets channels in a guild
func (c *Client) GetGuildChannels(guildID Snowflake) ([]*Channel, error) {
	return c.API.GetGuildChannels(c.ctx, guildID)
}

// GetGuildMember gets a guild member
func (c *Client) GetGuildMember(guildID, userID Snowflake) (*types.Member, error) {
	return c.API.GetGuildMember(c.ctx, guildID, userID)
}

// GetGuildRoles gets roles in a guild
func (c *Client) GetGuildRoles(guildID Snowflake) ([]*types.Role, error) {
	return c.API.GetGuildRoles(c.ctx, guildID)
}

// LeaveGuild leaves a guild
func (c *Client) LeaveGuild(guildID Snowflake) error {
	return c.API.LeaveGuild(c.ctx, guildID)
}

// JoinGuild joins a guild using an invite code
func (c *Client) JoinGuild(inviteCode string) (*types.Invite, error) {
	return c.API.JoinGuild(c.ctx, inviteCode)
}

// GetInvite gets information about an invite
func (c *Client) GetInvite(inviteCode string) (*types.Invite, error) {
	return c.API.GetInvite(c.ctx, inviteCode)
}

// ============================================
// Interaction / Slash Command Methods
// ============================================

// ClickButton clicks a button on a message
func (c *Client) ClickButton(channelID, messageID Snowflake, customID string, applicationID Snowflake) error {
	return c.API.ClickButton(c.ctx, channelID, messageID, customID, applicationID)
}

// SelectMenuSelect selects options in a select menu
func (c *Client) SelectMenuSelect(channelID, messageID Snowflake, customID string, values []string, applicationID Snowflake) error {
	return c.API.SelectMenuSelect(c.ctx, channelID, messageID, customID, values, applicationID)
}

// InvokeSlashCommand invokes a slash command
func (c *Client) InvokeSlashCommand(data *types.SlashCommandData) error {
	return c.API.InvokeSlashCommand(c.ctx, data)
}

// ============================================
// Relationship / Friend Methods
// ============================================

// GetRelationships gets all relationships (friends, blocked, etc.)
func (c *Client) GetRelationships() ([]*Relationship, error) {
	return c.API.GetRelationships(c.ctx)
}

// SendFriendRequest sends a friend request by username
func (c *Client) SendFriendRequest(username string) error {
	return c.API.SendFriendRequest(c.ctx, username, "")
}

// RemoveFriend removes a friend
func (c *Client) RemoveFriend(userID Snowflake) error {
	return c.API.RemoveFriend(c.ctx, userID)
}

// BlockUser blocks a user
func (c *Client) BlockUser(userID Snowflake) error {
	return c.API.BlockUser(c.ctx, userID)
}

// UnblockUser unblocks a user
func (c *Client) UnblockUser(userID Snowflake) error {
	return c.API.UnblockUser(c.ctx, userID)
}

// AcceptFriendRequest accepts a friend request
func (c *Client) AcceptFriendRequest(userID Snowflake) error {
	return c.API.AcceptFriendRequest(c.ctx, userID)
}

// ============================================
// Utility Methods
// ============================================

// Latency returns the gateway heartbeat latency
func (c *Client) Latency() time.Duration {
	return c.Gateway.Latency()
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	return c.Gateway.IsConnected()
}

// ParseSnowflake parses a string into a Snowflake
func ParseSnowflake(s string) (Snowflake, error) {
	return types.ParseSnowflake(s)
}

// ============================================
// Bulk Operations
// ============================================

// BulkDeleteMessages deletes multiple messages (2-100 messages, <14 days old)
func (c *Client) BulkDeleteMessages(channelID Snowflake, messageIDs []Snowflake) error {
	return c.API.BulkDeleteMessages(c.ctx, channelID, messageIDs)
}

// ============================================
// Pin Operations
// ============================================

// PinMessage pins a message
func (c *Client) PinMessage(channelID, messageID Snowflake) error {
	return c.API.PinMessage(c.ctx, channelID, messageID)
}

// UnpinMessage unpins a message
func (c *Client) UnpinMessage(channelID, messageID Snowflake) error {
	return c.API.UnpinMessage(c.ctx, channelID, messageID)
}

// GetPinnedMessages gets all pinned messages in a channel
func (c *Client) GetPinnedMessages(channelID Snowflake) ([]*Message, error) {
	return c.API.GetPinnedMessages(c.ctx, channelID)
}

// ============================================
// Thread Operations
// ============================================

// CreateThread creates a thread from a message
func (c *Client) CreateThread(channelID, messageID Snowflake, name string) (*Channel, error) {
	return c.API.CreateThread(c.ctx, channelID, messageID, &types.ThreadCreateData{
		Name: name,
	})
}

// CreateForumPost creates a post in a forum channel
func (c *Client) CreateForumPost(channelID Snowflake, title, content string) (*Channel, error) {
	return c.API.CreateForumPost(c.ctx, channelID, &types.ForumThreadCreateData{
		Name: title,
		Message: &types.ForumThreadMessage{
			Content: content,
		},
	})
}

// JoinThread joins a thread
func (c *Client) JoinThread(threadID Snowflake) error {
	return c.API.JoinThread(c.ctx, threadID)
}

// LeaveThread leaves a thread
func (c *Client) LeaveThread(threadID Snowflake) error {
	return c.API.LeaveThread(c.ctx, threadID)
}

// ArchiveThread archives a thread
func (c *Client) ArchiveThread(threadID Snowflake) (*Channel, error) {
	return c.API.ArchiveThread(c.ctx, threadID, false)
}

// ============================================
// Webhook Operations
// ============================================

// ExecuteWebhook sends a message via webhook
func (c *Client) ExecuteWebhook(webhookID Snowflake, token string, content string) error {
	return c.API.ExecuteWebhook(c.ctx, webhookID, token, &types.WebhookMessageData{
		Content: content,
	})
}

// CreateWebhook creates a webhook in a channel
func (c *Client) CreateWebhook(channelID Snowflake, name string) (*types.Webhook, error) {
	return c.API.CreateWebhook(c.ctx, channelID, name, nil)
}

// ============================================
// Poll Operations
// ============================================

// CreatePoll creates a poll
func (c *Client) CreatePoll(channelID Snowflake, question string, options []string, durationHours int) (*Message, error) {
	answers := make([]*types.PollCreateAnswer, len(options))
	for i, opt := range options {
		answers[i] = &types.PollCreateAnswer{
			PollMedia: types.PollMedia{Text: opt},
		}
	}
	return c.API.CreatePoll(c.ctx, channelID, "", &types.PollCreateData{
		Question:   types.PollMedia{Text: question},
		Answers:    answers,
		Duration:   durationHours,
		LayoutType: 1,
	})
}

// VotePoll votes on a poll
func (c *Client) VotePoll(channelID, messageID Snowflake, answerID int) error {
	return c.API.VotePoll(c.ctx, channelID, messageID, answerID)
}

// ============================================
// User Notes
// ============================================

// GetUserNote gets your note on a user
func (c *Client) GetUserNote(userID Snowflake) (string, error) {
	return c.API.GetUserNote(c.ctx, userID)
}

// SetUserNote sets your note on a user
func (c *Client) SetUserNote(userID Snowflake, note string) error {
	return c.API.SetUserNote(c.ctx, userID, note)
}

// ============================================
// Mark as Read
// ============================================

// MarkAsRead marks a message/channel as read
func (c *Client) MarkAsRead(channelID, messageID Snowflake) error {
	return c.API.AckMessage(c.ctx, channelID, messageID)
}

// MarkGuildAsRead marks all messages in a guild as read
func (c *Client) MarkGuildAsRead(guildID Snowflake) error {
	return c.API.AckGuild(c.ctx, guildID)
}

// ============================================
// Custom Status / Profile
// ============================================

// SetCustomStatus sets a custom status message
func (c *Client) SetCustomStatus(text string) error {
	return c.API.SetCustomStatus(c.ctx, text, nil, nil, nil)
}

// SetCustomStatusWithEmoji sets a custom status with an emoji
func (c *Client) SetCustomStatusWithEmoji(text, emojiName string) error {
	return c.API.SetCustomStatus(c.ctx, text, &emojiName, nil, nil)
}

// ClearCustomStatus clears your custom status
func (c *Client) ClearCustomStatus() error {
	return c.API.ClearCustomStatus(c.ctx)
}

// EditProfile edits your profile
func (c *Client) EditProfile(data *types.ProfileEditData) (*CurrentUser, error) {
	return c.API.EditProfile(c.ctx, data)
}

// ============================================
// Scheduled Events
// ============================================

// GetGuildEvents gets scheduled events for a guild
func (c *Client) GetGuildEvents(guildID Snowflake) ([]*types.ScheduledEvent, error) {
	return c.API.GetGuildEvents(c.ctx, guildID)
}

// RSVPEvent marks interest in an event
func (c *Client) RSVPEvent(guildID, eventID Snowflake) error {
	return c.API.RSVPEvent(c.ctx, guildID, eventID)
}

// ============================================
// Group DM
// ============================================

// CreateGroupDM creates a group DM with multiple users
func (c *Client) CreateGroupDM(userIDs []Snowflake) (*Channel, error) {
	return c.API.CreateGroupDM(c.ctx, userIDs)
}

// AddToGroupDM adds a user to a group DM
func (c *Client) AddToGroupDM(channelID, userID Snowflake) error {
	return c.API.AddGroupDMRecipient(c.ctx, channelID, userID)
}

// RemoveFromGroupDM removes a user from a group DM
func (c *Client) RemoveFromGroupDM(channelID, userID Snowflake) error {
	return c.API.RemoveGroupDMRecipient(c.ctx, channelID, userID)
}

// ============================================
// Search
// ============================================

// SearchMessages searches for messages in a channel
func (c *Client) SearchMessages(channelID Snowflake, query string, limit int) (interface{}, error) {
	return c.API.SearchMessages(c.ctx, channelID, query, limit)
}

// SearchGuildMessages searches for messages in a guild
func (c *Client) SearchGuildMessages(guildID Snowflake, query string, limit int) (interface{}, error) {
	return c.API.SearchGuildMessages(c.ctx, guildID, query, limit)
}

// ============================================
// Emojis & Stickers
// ============================================

// GetGuildEmojis gets all emojis for a guild
func (c *Client) GetGuildEmojis(guildID Snowflake) ([]*types.Emoji, error) {
	return c.API.GetGuildEmojis(c.ctx, guildID)
}

// GetGuildStickers gets all stickers for a guild
func (c *Client) GetGuildStickers(guildID Snowflake) ([]*types.Sticker, error) {
	return c.API.GetGuildStickers(c.ctx, guildID)
}

// ============================================
// Moderation
// ============================================

// BanUser bans a user from a guild
func (c *Client) BanUser(guildID, userID Snowflake, deleteMessageDays int, reason string) error {
	return c.API.BanUser(c.ctx, guildID, userID, deleteMessageDays, reason)
}

// UnbanUser unbans a user from a guild
func (c *Client) UnbanUser(guildID, userID Snowflake, reason string) error {
	return c.API.UnbanUser(c.ctx, guildID, userID, reason)
}

// GetGuildBans gets all bans for a guild
func (c *Client) GetGuildBans(guildID Snowflake) ([]*types.Ban, error) {
	return c.API.GetGuildBans(c.ctx, guildID)
}

// KickUser kicks a user from a guild
func (c *Client) KickUser(guildID, userID Snowflake, reason string) error {
	return c.API.KickUser(c.ctx, guildID, userID, reason)
}

// TimeoutUser times out a user (mutes them temporarily)
func (c *Client) TimeoutUser(guildID, userID Snowflake, until *time.Time, reason string) error {
	return c.API.TimeoutUser(c.ctx, guildID, userID, until, reason)
}

// ============================================
// Channel Management
// ============================================

// CreateChannel creates a channel in a guild
func (c *Client) CreateChannel(guildID Snowflake, data *types.ChannelCreateData) (*Channel, error) {
	return c.API.CreateChannel(c.ctx, guildID, data)
}

// ModifyChannel modifies a channel
func (c *Client) ModifyChannel(channelID Snowflake, data *types.ChannelModifyData) (*Channel, error) {
	return c.API.ModifyChannel(c.ctx, channelID, data)
}

// DeleteChannel deletes a channel
func (c *Client) DeleteChannel(channelID Snowflake, reason string) error {
	return c.API.DeleteChannel(c.ctx, channelID, reason)
}

// ============================================
// Role Management
// ============================================

// CreateRole creates a role in a guild
func (c *Client) CreateRole(guildID Snowflake, data *types.RoleCreateData) (*Role, error) {
	return c.API.CreateRole(c.ctx, guildID, data)
}

// ModifyRole modifies a role
func (c *Client) ModifyRole(guildID, roleID Snowflake, data *types.RoleModifyData) (*Role, error) {
	return c.API.ModifyRole(c.ctx, guildID, roleID, data)
}

// DeleteRole deletes a role
func (c *Client) DeleteRole(guildID, roleID Snowflake, reason string) error {
	return c.API.DeleteRole(c.ctx, guildID, roleID, reason)
}

// AddMemberRole adds a role to a member
func (c *Client) AddMemberRole(guildID, userID, roleID Snowflake, reason string) error {
	return c.API.AddMemberRole(c.ctx, guildID, userID, roleID, reason)
}

// RemoveMemberRole removes a role from a member
func (c *Client) RemoveMemberRole(guildID, userID, roleID Snowflake, reason string) error {
	return c.API.RemoveMemberRole(c.ctx, guildID, userID, roleID, reason)
}

// ============================================
// Member Management
// ============================================

// GetGuildMembers gets members in a guild
func (c *Client) GetGuildMembers(guildID Snowflake, limit int, after Snowflake) ([]*Member, error) {
	return c.API.GetGuildMembers(c.ctx, guildID, limit, after)
}

// SearchGuildMembers searches for members in a guild
func (c *Client) SearchGuildMembers(guildID Snowflake, query string, limit int) ([]*Member, error) {
	return c.API.SearchGuildMembers(c.ctx, guildID, query, limit)
}

// ModifyGuildMember modifies a guild member
func (c *Client) ModifyGuildMember(guildID, userID Snowflake, data *types.MemberModifyData, reason string) (*Member, error) {
	return c.API.ModifyGuildMember(c.ctx, guildID, userID, data, reason)
}

// ============================================
// Invites
// ============================================

// CreateChannelInvite creates an invite for a channel
func (c *Client) CreateChannelInvite(channelID Snowflake, maxAge, maxUses int, temporary, unique bool) (*Invite, error) {
	return c.API.CreateChannelInvite(c.ctx, channelID, maxAge, maxUses, temporary, unique)
}

// DeleteInvite deletes an invite
func (c *Client) DeleteInvite(inviteCode string, reason string) (*Invite, error) {
	return c.API.DeleteInvite(c.ctx, inviteCode, reason)
}

// GetGuildInvites gets all invites for a guild
func (c *Client) GetGuildInvites(guildID Snowflake) ([]*Invite, error) {
	return c.API.GetGuildInvites(c.ctx, guildID)
}

// ============================================
// Stage Instance
// ============================================

// CreateStageInstance creates a stage instance
func (c *Client) CreateStageInstance(channelID Snowflake, topic string) (*types.StageInstance, error) {
	return c.API.CreateStageInstance(c.ctx, channelID, topic, 2) // 2 = GUILD_ONLY
}

// GetStageInstance gets a stage instance
func (c *Client) GetStageInstance(channelID Snowflake) (*types.StageInstance, error) {
	return c.API.GetStageInstance(c.ctx, channelID)
}

// DeleteStageInstance deletes a stage instance
func (c *Client) DeleteStageInstance(channelID Snowflake) error {
	return c.API.DeleteStageInstance(c.ctx, channelID)
}

// ============================================
// Audit Log
// ============================================

// GetAuditLog gets the audit log for a guild
func (c *Client) GetAuditLog(guildID Snowflake, limit int) (*types.AuditLog, error) {
	return c.API.GetAuditLog(c.ctx, guildID, 0, 0, 0, limit)
}

// ============================================
// Permissions
// ============================================

// EditChannelPermissions edits channel permission overwrites
func (c *Client) EditChannelPermissions(channelID, targetID Snowflake, allow, deny string, targetType int) error {
	return c.API.EditChannelPermissions(c.ctx, channelID, targetID, allow, deny, targetType, "")
}

// DeleteChannelPermission deletes a channel permission overwrite
func (c *Client) DeleteChannelPermission(channelID, targetID Snowflake) error {
	return c.API.DeleteChannelPermission(c.ctx, channelID, targetID, "")
}

// ============================================
// Guild Preview & Welcome Screen
// ============================================

// GetGuildPreview gets a guild preview
func (c *Client) GetGuildPreview(guildID Snowflake) (*types.GuildPreview, error) {
	return c.API.GetGuildPreview(c.ctx, guildID)
}

// GetWelcomeScreen gets a guild's welcome screen
func (c *Client) GetWelcomeScreen(guildID Snowflake) (*types.WelcomeScreen, error) {
	return c.API.GetWelcomeScreen(c.ctx, guildID)
}

// ============================================
// Reaction Methods
// ============================================

// GetReactions gets reactions for a message
func (c *Client) GetReactions(channelID, messageID Snowflake, emoji string, limit int, after Snowflake) ([]*User, error) {
	return c.API.GetReactions(c.ctx, channelID, messageID, emoji, limit, after)
}

// ============================================
// Thread Member Methods
// ============================================

// AddThreadMember adds a member to a thread
func (c *Client) AddThreadMember(channelID, userID Snowflake) error {
	return c.API.AddThreadMember(c.ctx, channelID, userID)
}

// RemoveThreadMember removes a member from a thread
func (c *Client) RemoveThreadMember(channelID, userID Snowflake) error {
	return c.API.RemoveThreadMember(c.ctx, channelID, userID)
}

// GetThreadMember gets a thread member
func (c *Client) GetThreadMember(channelID, userID Snowflake) (*types.ThreadMember, error) {
	return c.API.GetThreadMember(c.ctx, channelID, userID)
}

// GetThreadMembers gets all members of a thread
func (c *Client) GetThreadMembers(channelID Snowflake) ([]*types.ThreadMember, error) {
	return c.API.GetThreadMembers(c.ctx, channelID)
}

// ============================================
// Webhook Methods (Extended)
// ============================================

// GetGuildWebhooks gets all webhooks for a guild
func (c *Client) GetGuildWebhooks(guildID Snowflake) ([]*types.Webhook, error) {
	return c.API.GetGuildWebhooks(c.ctx, guildID)
}

// ModifyWebhook modifies a webhook
func (c *Client) ModifyWebhook(webhookID Snowflake, name string, avatar *string, channelID *Snowflake) (*types.Webhook, error) {
	return c.API.ModifyWebhook(c.ctx, webhookID, name, avatar, channelID)
}

// ============================================
// Auto Moderation Methods
// ============================================

// GetAutoModerationRules gets auto moderation rules
func (c *Client) GetAutoModerationRules(guildID Snowflake) ([]*types.AutoModerationRule, error) {
	return c.API.GetAutoModerationRules(c.ctx, guildID)
}

// GetAutoModerationRule gets a specific auto moderation rule
func (c *Client) GetAutoModerationRule(guildID, ruleID Snowflake) (*types.AutoModerationRule, error) {
	return c.API.GetAutoModerationRule(c.ctx, guildID, ruleID)
}

// CreateAutoModerationRule creates an auto moderation rule
func (c *Client) CreateAutoModerationRule(guildID Snowflake, rule *types.AutoModerationRule) (*types.AutoModerationRule, error) {
	return c.API.CreateAutoModerationRule(c.ctx, guildID, rule)
}

// ModifyAutoModerationRule modifies an auto moderation rule
func (c *Client) ModifyAutoModerationRule(guildID, ruleID Snowflake, rule *types.AutoModerationRuleModifyData) (*types.AutoModerationRule, error) {
	return c.API.ModifyAutoModerationRule(c.ctx, guildID, ruleID, rule)
}

// DeleteAutoModerationRule deletes an auto moderation rule
func (c *Client) DeleteAutoModerationRule(guildID, ruleID Snowflake) error {
	return c.API.DeleteAutoModerationRule(c.ctx, guildID, ruleID)
}

// ListActiveThreads lists active threads in a guild
func (c *Client) ListActiveThreads(guildID Snowflake) ([]*Channel, error) {
	return c.API.ListActiveThreads(c.ctx, guildID)
}

// ============================================
// Scheduled Event Methods
// ============================================

// ListScheduledEvents lists scheduled events in a guild
func (c *Client) ListScheduledEvents(guildID Snowflake, withUserCount bool) ([]*types.ScheduledEvent, error) {
	return c.API.ListScheduledEvents(c.ctx, guildID, withUserCount)
}

// CreateScheduledEvent creates a scheduled event
func (c *Client) CreateScheduledEvent(guildID Snowflake, event *types.ScheduledEventParams) (*types.ScheduledEvent, error) {
	return c.API.CreateScheduledEvent(c.ctx, guildID, event)
}

// GetScheduledEvent gets a scheduled event
func (c *Client) GetScheduledEvent(guildID, eventID Snowflake, withUserCount bool) (*types.ScheduledEvent, error) {
	return c.API.GetScheduledEvent(c.ctx, guildID, eventID, withUserCount)
}

// ModifyScheduledEvent modifies a scheduled event
func (c *Client) ModifyScheduledEvent(guildID, eventID Snowflake, event *types.ScheduledEventParams) (*types.ScheduledEvent, error) {
	return c.API.ModifyScheduledEvent(c.ctx, guildID, eventID, event)
}

// DeleteScheduledEvent deletes a scheduled event
func (c *Client) DeleteScheduledEvent(guildID, eventID Snowflake) error {
	return c.API.DeleteScheduledEvent(c.ctx, guildID, eventID)
}

// Interact sends an interaction (e.g. button click or slash command)
func (c *Client) Interact(interaction *types.InteractionPayload) error {
	return c.API.Interact(c.ctx, interaction)
}

// ============================================
// Presence Methods
// ============================================

// UpdateStatusComplex updates the user's status with activities
func (c *Client) UpdateStatusComplex(status string, activities []*types.Activity, afk bool) error {
	if c.Gateway == nil {
		return fmt.Errorf("gateway not initialized")
	}
	var rawActivities []interface{}
	for _, a := range activities {
		rawActivities = append(rawActivities, a)
	}
	return c.Gateway.UpdatePresence(status, rawActivities, afk)
}

// SetPlaying sets the "Playing" activity. It automatically looks up the ApplicationID if the name matches a detectable game.
func (c *Client) SetPlaying(name, details, state, largeImage, smallImage string, start time.Time) error {
	appID := c.FindApplicationID(name)

	activity := &types.Activity{
		Name:          name,
		Type:          types.ActivityTypePlaying,
		ApplicationID: appID,
	}

	if details != "" {
		activity.Details = details
	}
	if state != "" {
		activity.State = state
	}
	if !start.IsZero() {
		activity.Timestamps = &types.Timestamps{
			Start: start.UnixMilli(),
		}
	}
	if largeImage != "" || smallImage != "" {
		activity.Assets = &types.Assets{
			LargeImage: largeImage,
			SmallImage: smallImage,
		}
	}

	return c.UpdateStatusComplex("online", []*types.Activity{activity}, false)
}

// FindApplicationID attempts to find the ApplicationID for a given game name
func (c *Client) FindApplicationID(name string) Snowflake {
	c.detectableAppsMu.RLock()
	defer c.detectableAppsMu.RUnlock()

	name = strings.ToLower(name)
	for _, app := range c.detectableApps {
		if strings.ToLower(app.Name) == name {
			sf, _ := types.ParseSnowflake(app.ID)
			return sf
		}
		for _, alias := range app.Aliases {
			if strings.ToLower(alias) == name {
				sf, _ := types.ParseSnowflake(app.ID)
				return sf
			}
		}
	}
	return 0
}

// RefreshDetectableApps fetches the latest list of detectable applications from Discord
func (c *Client) RefreshDetectableApps() error {
	apps, err := c.API.GetDetectableApplications(c.ctx)
	if err != nil {
		return err
	}

	c.detectableAppsMu.Lock()
	c.detectableApps = apps
	c.detectableAppsMu.Unlock()
	return nil
}

// SetListening sets the "Listening" activity (e.g. Spotify)
func (c *Client) SetListening(name, artist, album, largeImage string, start, end time.Time) error {
	details := artist
	state := album

	activity := &types.Activity{
		Name:    name,
		Type:    types.ActivityTypeListening,
		Details: details,
		State:   state,
	}

	if !start.IsZero() || !end.IsZero() {
		activity.Timestamps = &types.Timestamps{}
		if !start.IsZero() {
			activity.Timestamps.Start = start.UnixMilli()
		}
		if !end.IsZero() {
			activity.Timestamps.End = end.UnixMilli()
		}
	}

	if largeImage != "" {
		activity.Assets = &types.Assets{
			LargeImage: largeImage,
		}
	}

	return c.UpdateStatusComplex("online", []*types.Activity{activity}, false)
}

// SetWatching sets the "Watching" activity
func (c *Client) SetWatching(name, details, state string) error {
	activity := &types.Activity{
		Name: name,
		Type: types.ActivityTypeWatching,
	}

	if details != "" {
		activity.Details = details
	}
	if state != "" {
		activity.State = state
	}

	return c.UpdateStatusComplex("online", []*types.Activity{activity}, false)
}

// ClearActivities clears all current activities
func (c *Client) ClearActivities() error {
	return c.UpdateStatusComplex("online", nil, false)
}

// ============================================
// DM & Channel Methods
// ============================================

// CloseDM closes a DM (deletes channel)
func (c *Client) CloseDM(channelID Snowflake) error {
	return c.API.DeleteChannel(c.ctx, channelID, "")
}

// AckMessage marks a message as read
func (c *Client) AckMessage(channelID, messageID Snowflake) error {
	return c.API.AckMessage(c.ctx, channelID, messageID)
}

// ============================================
// Message Methods
// ============================================

// SearchApplicationCommands searches for slash commands
func (c *Client) SearchApplicationCommands(channelID Snowflake, query string) (interface{}, error) {
	return c.API.SearchApplicationCommands(c.ctx, channelID, query)
}

// ============================================
// User Settings Methods
// ============================================

// GetUserSettings returns the current user's settings
func (c *Client) GetUserSettings() (*types.UserSettings, error) {
	return c.API.GetUserSettings(c.ctx)
}

// UpdateUserSettings updates the current user's settings
func (c *Client) UpdateUserSettings(settings *types.UserSettings) (*types.UserSettings, error) {
	return c.API.UpdateUserSettings(c.ctx, settings)
}

// ============================================
// Sticker Methods
// ============================================

// ListGuildStickers lists stickers in a guild
func (c *Client) ListGuildStickers(guildID Snowflake) ([]*types.Sticker, error) {
	return c.API.ListGuildStickers(c.ctx, guildID)
}

// GetGuildSticker gets a sticker in a guild
func (c *Client) GetGuildSticker(guildID, stickerID Snowflake) (*types.Sticker, error) {
	return c.API.GetGuildSticker(c.ctx, guildID, stickerID)
}

// CreateGuildSticker creates a sticker in a guild
func (c *Client) CreateGuildSticker(guildID Snowflake, name, description, tags string, fileData []byte, filename string) (*types.Sticker, error) {
	return c.API.CreateGuildSticker(c.ctx, guildID, name, description, tags, fileData, filename)
}

// ModifyGuildSticker modifies a sticker
func (c *Client) ModifyGuildSticker(guildID, stickerID Snowflake, name, description, tags string) (*types.Sticker, error) {
	return c.API.ModifyGuildSticker(c.ctx, guildID, stickerID, name, description, tags)
}

// DeleteGuildSticker deletes a sticker
func (c *Client) DeleteGuildSticker(guildID, stickerID Snowflake) error {
	return c.API.DeleteGuildSticker(c.ctx, guildID, stickerID)
}

// GetStickerPacks gets the list of sticker packs
func (c *Client) GetStickerPacks() ([]*types.StickerPack, error) {
	return c.API.GetStickerPacks(c.ctx)
}

// GetGuildVanityURL returns the vanity URL code of a guild
func (c *Client) GetGuildVanityURL(guildID Snowflake) (*string, error) {
	return c.API.GetGuildVanityURL(c.ctx, guildID)
}

// ChangeGuildVanityURL changes the vanity URL code of a guild
func (c *Client) ChangeGuildVanityURL(guildID Snowflake, code string) (*string, error) {
	return c.API.ChangeGuildVanityURL(c.ctx, guildID, code)
}

// ============================================
// Voice Methods
// ============================================

// JoinVoiceChannel joins a voice channel and returns a VoiceConnection
func (c *Client) JoinVoiceChannel(guildID, channelID Snowflake, mute, deaf bool) (*voice.VoiceConnection, error) {
	if c.Gateway == nil {
		return nil, fmt.Errorf("gateway not initialized")
	}

	// Create channels to wait for events
	vsuChan := make(chan *VoiceServerUpdate, 1)
	vStateChan := make(chan *types.VoiceState, 1)

	// Register temporary handlers
	c.OnVoiceServerUpdate(func(vsu *VoiceServerUpdate) {
		if vsu.GuildID == guildID {
			vsuChan <- vsu
		}
	})
	c.OnVoiceStateUpdate(func(vs *types.VoiceState) {
		if vs.GuildID == guildID && vs.UserID == c.User.ID {
			vStateChan <- vs
		}
	})

	// Send Voice State Update to Gateway
	err := c.Gateway.UpdateVoiceState(guildID.String(), channelID.String(), mute, deaf)
	if err != nil {
		return nil, err
	}

	// Wait for events with timeout
	var vsu *VoiceServerUpdate
	var vState *types.VoiceState

	timeout := time.After(10 * time.Second)
	for vsu == nil || vState == nil {
		select {
		case v := <-vsuChan:
			vsu = v
		case v := <-vStateChan:
			vState = v
		case <-timeout:
			return nil, fmt.Errorf("timeout waiting for voice gateway events")
		}
	}

	// Create Voice Connection
	vc := voice.NewVoiceConnection(
		guildID.String(),
		channelID.String(),
		c.User.ID.String(),
		vState.SessionID,
		vsu.Token,
		vsu.Endpoint,
	)

	if err := vc.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to voice gateway: %w", err)
	}

	c.mu.Lock()
	c.voiceConnections[guildID] = vc
	c.mu.Unlock()

	return vc, nil
}

// LeaveVoiceChannel leaves a voice channel in a guild
func (c *Client) LeaveVoiceChannel(guildID Snowflake) error {
	c.mu.Lock()
	vc, ok := c.voiceConnections[guildID]
	if ok {
		vc.Close()
		delete(c.voiceConnections, guildID)
	}
	c.mu.Unlock()

	if c.Gateway == nil {
		return fmt.Errorf("gateway not initialized")
	}

	return c.Gateway.UpdateVoiceState(guildID.String(), "", false, false)
}
