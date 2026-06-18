package state

import (
	"sync"

	"github.com/hytams/discordgo-self/types"
)

// State tracks the current world state.
type State struct {
	sync.RWMutex

	Guilds map[types.Snowflake]*types.Guild

	Channels map[types.Snowflake]*types.Channel

	Users map[types.Snowflake]*types.User

	Members map[types.Snowflake]map[types.Snowflake]*types.Member

	VoiceStates map[types.Snowflake]map[types.Snowflake]*types.VoiceState

	Messages map[types.Snowflake][]*types.Message

	Presences map[types.Snowflake]*types.Presence

	Relationships map[types.Snowflake]*types.Relationship

	UserSettings *types.UserSettings

	ReadStates map[types.Snowflake]*types.ReadState

	Sessions map[string]*types.Session

	Experiments interface{}
}

// NewState creates a new empty state.
func NewState() *State {
	return &State{
		Guilds:        make(map[types.Snowflake]*types.Guild),
		Channels:      make(map[types.Snowflake]*types.Channel),
		Users:         make(map[types.Snowflake]*types.User),
		Members:       make(map[types.Snowflake]map[types.Snowflake]*types.Member),
		VoiceStates:   make(map[types.Snowflake]map[types.Snowflake]*types.VoiceState),
		Messages:      make(map[types.Snowflake][]*types.Message),
		Presences:     make(map[types.Snowflake]*types.Presence),
		Relationships: make(map[types.Snowflake]*types.Relationship),
		ReadStates:    make(map[types.Snowflake]*types.ReadState),
		Sessions:      make(map[string]*types.Session),
	}
}

// GetGuild gets a guild by ID.
func (s *State) GetGuild(id types.Snowflake) (*types.Guild, bool) {
	s.RLock()
	defer s.RUnlock()
	g, ok := s.Guilds[id]
	return g, ok
}

// SetGuild sets a guild.
func (s *State) SetGuild(guild *types.Guild) {
	s.Lock()
	defer s.Unlock()
	s.Guilds[guild.ID] = guild
}

// RemoveGuild removes a guild.
func (s *State) RemoveGuild(id types.Snowflake) {
	s.Lock()
	defer s.Unlock()
	delete(s.Guilds, id)
}

// GetChannel gets a channel by ID.
func (s *State) GetChannel(id types.Snowflake) (*types.Channel, bool) {
	s.RLock()
	defer s.RUnlock()
	c, ok := s.Channels[id]
	return c, ok
}

// SetChannel sets a channel.
func (s *State) SetChannel(channel *types.Channel) {
	s.Lock()
	defer s.Unlock()
	s.Channels[channel.ID] = channel
}

// RemoveChannel removes a channel.
func (s *State) RemoveChannel(id types.Snowflake) {
	s.Lock()
	defer s.Unlock()
	delete(s.Channels, id)
}

// GetUser gets a user by ID.
func (s *State) GetUser(id types.Snowflake) (*types.User, bool) {
	s.RLock()
	defer s.RUnlock()
	u, ok := s.Users[id]
	return u, ok
}

// SetUser sets a user.
func (s *State) SetUser(user *types.User) {
	s.Lock()
	defer s.Unlock()
	s.Users[user.ID] = user
}

// RemoveUser removes a user.
func (s *State) RemoveUser(id types.Snowflake) {
	s.Lock()
	defer s.Unlock()
	delete(s.Users, id)
}

// GetMember gets a member by guild ID and user ID.
func (s *State) GetMember(guildID, userID types.Snowflake) (*types.Member, bool) {
	s.RLock()
	defer s.RUnlock()
	if guildMembers, ok := s.Members[guildID]; ok {
		m, ok := guildMembers[userID]
		return m, ok
	}
	return nil, false
}

// SetMember sets a member.
func (s *State) SetMember(guildID types.Snowflake, member *types.Member) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.Members[guildID]; !ok {
		s.Members[guildID] = make(map[types.Snowflake]*types.Member)
	}
	s.Members[guildID][member.User.ID] = member
}

// RemoveMember removes a member.
func (s *State) RemoveMember(guildID, userID types.Snowflake) {
	s.Lock()
	defer s.Unlock()

	if guildMembers, ok := s.Members[guildID]; ok {
		delete(guildMembers, userID)
	}
}

// GetPresence gets a presence by user ID.
func (s *State) GetPresence(id types.Snowflake) (*types.Presence, bool) {
	s.RLock()
	defer s.RUnlock()
	p, ok := s.Presences[id]
	return p, ok
}

// SetPresence sets a presence.
func (s *State) SetPresence(userID types.Snowflake, presence *types.Presence) {
	s.Lock()
	defer s.Unlock()
	s.Presences[userID] = presence
}

// GetRelationship gets a relationship by user ID.
func (s *State) GetRelationship(id types.Snowflake) (*types.Relationship, bool) {
	s.RLock()
	defer s.RUnlock()
	r, ok := s.Relationships[id]
	return r, ok
}

// SetRelationship sets a relationship.
func (s *State) SetRelationship(rel *types.Relationship) {
	s.Lock()
	defer s.Unlock()
	s.Relationships[rel.ID] = rel
}

// RemoveRelationship removes a relationship.
func (s *State) RemoveRelationship(id types.Snowflake) {
	s.Lock()
	defer s.Unlock()
	delete(s.Relationships, id)
}

// GetVoiceState gets a voice state.
func (s *State) GetVoiceState(guildID, userID types.Snowflake) (*types.VoiceState, bool) {
	s.RLock()
	defer s.RUnlock()
	if guildVoiceStates, ok := s.VoiceStates[guildID]; ok {
		vs, ok := guildVoiceStates[userID]
		return vs, ok
	}
	return nil, false
}

// SetVoiceState sets a voice state.
func (s *State) SetVoiceState(guildID types.Snowflake, vs *types.VoiceState) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.VoiceStates[guildID]; !ok {
		s.VoiceStates[guildID] = make(map[types.Snowflake]*types.VoiceState)
	}
	s.VoiceStates[guildID][vs.UserID] = vs
}

// RemoveVoiceState removes a voice state.
func (s *State) RemoveVoiceState(guildID, userID types.Snowflake) {
	s.Lock()
	defer s.Unlock()

	if guildVoiceStates, ok := s.VoiceStates[guildID]; ok {
		delete(guildVoiceStates, userID)
	}
}

// GetMessage gets a message by channel ID and message ID.
func (s *State) GetMessage(channelID, messageID types.Snowflake) (*types.Message, bool) {
	s.RLock()
	defer s.RUnlock()
	if msgs, ok := s.Messages[channelID]; ok {
		for _, m := range msgs {
			if m.ID == messageID {
				return m, true
			}
		}
	}
	return nil, false
}

// GetChannelMessages gets all messages for a channel.
func (s *State) GetChannelMessages(channelID types.Snowflake) []*types.Message {
	s.RLock()
	defer s.RUnlock()
	if msgs, ok := s.Messages[channelID]; ok {
		result := make([]*types.Message, len(msgs))
		copy(result, msgs)
		return result
	}
	return nil
}

// MessageAdd adds a message to the cache, respecting the max count.
func (s *State) MessageAdd(message *types.Message, maxCount int) {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.Messages[message.ChannelID]; !ok {
		s.Messages[message.ChannelID] = make([]*types.Message, 0)
	}

	msgs := s.Messages[message.ChannelID]

	for i, m := range msgs {
		if m.ID == message.ID {
			msgs[i] = message
			return
		}
	}

	msgs = append(msgs, message)

	if len(msgs) > maxCount {
		msgs = msgs[len(msgs)-maxCount:]
	}

	s.Messages[message.ChannelID] = msgs
}

// MessageRemove removes a message.
func (s *State) MessageRemove(channelID, messageID types.Snowflake) {
	s.Lock()
	defer s.Unlock()

	if msgs, ok := s.Messages[channelID]; ok {
		for i, m := range msgs {
			if m.ID == messageID {
				s.Messages[channelID] = append(msgs[:i], msgs[i+1:]...)
				return
			}
		}
	}
}

// SetUserSettings sets the user settings.
func (s *State) SetUserSettings(settings *types.UserSettings) {
	s.Lock()
	defer s.Unlock()
	s.UserSettings = settings
}

// SetReadState sets a read state.
func (s *State) SetReadState(rs *types.ReadState) {
	s.Lock()
	defer s.Unlock()
	s.ReadStates[rs.ID] = rs
}

// GetReadState gets a read state by channel ID.
func (s *State) GetReadState(id types.Snowflake) (*types.ReadState, bool) {
	s.RLock()
	defer s.RUnlock()
	rs, ok := s.ReadStates[id]
	return rs, ok
}

// SetSession sets a session.
func (s *State) SetSession(session *types.Session) {
	s.Lock()
	defer s.Unlock()
	s.Sessions[session.SessionID] = session
}

// RemoveSession removes a session.
func (s *State) RemoveSession(id string) {
	s.Lock()
	defer s.Unlock()
	delete(s.Sessions, id)
}
