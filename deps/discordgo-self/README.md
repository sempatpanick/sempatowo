# discordgo-self

A Discord selfbot library for Go, providing comprehensive access to Discord's user API. This library is inspired by and references implementations from [discord.js-selfbot-v13](https://github.com/aiko-chan-ai/discord.js-selfbot-v13) and [discord.py-self](https://github.com/dolfies/discord.py-self).

> **Disclaimer**: Selfbots violate Discord's Terms of Service. Using this library may result in account termination. This project is intended for educational and research purposes only.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [API Reference](#api-reference)
  - [Client Methods](#client-methods)
  - [Message Operations](#message-operations)
  - [Channel Operations](#channel-operations)
  - [Guild Operations](#guild-operations)
  - [Relationship Management](#relationship-management)
  - [Voice Operations](#voice-operations)
  - [Rich Presence](#rich-presence)
  - [Interactions](#interactions)
- [Event Handlers](#event-handlers)
- [State Management](#state-management)
- [Examples](#examples)
- [Credits](#credits)
- [License](#license)

## Features

### Core Features
- Full Discord Gateway (WebSocket) support with automatic reconnection
- Complete REST API client with rate limiting
- Per-message zlib compression for optimal performance
- TLS fingerprint spoofing using uTLS (mimics Chrome browser)

### Messaging
- Send, edit, and delete messages
- File and image attachments with multipart upload
- Embeds and rich content
- Reactions management
- Message search functionality
- Bulk message deletion

### User Features
- Rich Presence (Playing, Listening, Watching, Streaming)
- Custom status with emoji support
- Relationship management (friends, blocks, friend requests)
- User notes
- Profile editing

### Guild Features
- Member management (kick, ban, timeout, roles)
- Channel management (create, modify, delete, permissions)
- Role management
- Scheduled events
- Auto-moderation rules
- Webhooks
- Audit logs

### Advanced Features
- Voice channel connection (WebSocket + UDP)
- QR code login (Remote Authentication)
- Slash command discovery and invocation
- Button and select menu interactions
- Modal submissions
- Thread and forum post management
- Stage channel operations

### State Management
- In-memory caching for guilds, channels, users, and members
- Configurable cache policies
- Thread-safe access methods
- FIFO message cache

## Installation

```bash
go get github.com/hytams/discordgo-self@latest
```

### Requirements
- Go 1.21 or higher

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"

    discord "github.com/hytams/discordgo-self"
)

func main() {
    client, err := discord.New(discord.Config{
        Token: os.Getenv("DISCORD_TOKEN"),
    })
    if err != nil {
        log.Fatal(err)
    }

    client.OnMessageCreate(func(msg *discord.Message) {
        if msg.Content == "!ping" {
            client.SendMessage(msg.ChannelID, "Pong!")
        }
    })

    if err := client.Connect(); err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    fmt.Println("Client is running. Press Ctrl+C to exit.")
    
    sc := make(chan os.Signal, 1)
    signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
    <-sc
}
```

## Configuration

```go
config := discord.Config{
    Token:           "your_user_token",  // Required
    Debug:           false,              // Enable debug logging
    StateEnabled:    true,               // Enable state caching
    CacheMembers:    true,               // Cache guild members
    CachePresences:  false,              // Cache user presences
    CacheMessages:   true,               // Cache messages
    MaxMessages:     100,                // Max messages per channel
    Locale:          "en-US",            // API locale
    Timezone:        "UTC",              // Timezone header
}
```

## API Reference

### Client Methods

| Method | Description |
|--------|-------------|
| `New(config)` | Create a new client instance |
| `Connect()` | Connect to the Discord Gateway |
| `Close()` | Disconnect and cleanup resources |

### Message Operations

```go
// Send a simple message
msg, err := client.SendMessage(channelID, "Hello, World!")

// Send message with embed
embed := &types.Embed{
    Title:       "Embed Title",
    Description: "Embed description text",
    Color:       0x00FF00,
}
msg, err := client.SendMessageEmbed(channelID, "Check this out:", embed)

// Send file
file, _ := types.NewFileFromPath("image.png")
msg, err := client.SendFile(channelID, file, "Here is an image")

// Edit message
msg, err := client.EditMessage(channelID, messageID, "Updated content")

// Delete message
err := client.DeleteMessage(channelID, messageID)

// Get message history
messages, err := client.GetMessages(channelID, 50, beforeID, afterID, aroundID)

// Add reaction
err := client.AddReaction(channelID, messageID, "thumbsup")

// Remove reaction
err := client.RemoveReaction(channelID, messageID, "thumbsup")

// Bulk delete messages (2-100 messages, max 14 days old)
err := client.BulkDeleteMessages(channelID, messageIDs)
```

### Channel Operations

```go
// Get channel
channel, err := client.GetChannel(channelID)

// Create DM channel
dm, err := client.CreateDM(userID)

// Create group DM
groupDM, err := client.CreateGroupDM([]types.Snowflake{userID1, userID2})

// Create channel in guild
channel, err := client.CreateChannel(guildID, &types.ChannelCreateData{
    Name: "new-channel",
    Type: types.ChannelTypeGuildText,
})

// Modify channel
channel, err := client.ModifyChannel(channelID, &types.ChannelModifyData{
    Name:  "renamed-channel",
    Topic: "New channel topic",
})

// Delete channel
err := client.DeleteChannel(channelID, "Cleanup")

// Pin message
err := client.PinMessage(channelID, messageID)

// Unpin message
err := client.UnpinMessage(channelID, messageID)
```

### Guild Operations

```go
// Get guild
guild, err := client.GetGuild(guildID)

// Get guild channels
channels, err := client.GetGuildChannels(guildID)

// Get guild members
members, err := client.GetGuildMembers(guildID, 100, afterUserID)

// Search members
members, err := client.SearchGuildMembers(guildID, "username", 10)

// Get member
member, err := client.GetGuildMember(guildID, userID)

// Kick member
err := client.KickUser(guildID, userID, "Reason for kick")

// Ban member
err := client.BanUser(guildID, userID, 7, "Reason for ban") // 7 = delete 7 days of messages

// Unban member
err := client.UnbanUser(guildID, userID, "Reason for unban")

// Timeout member
until := time.Now().Add(10 * time.Minute)
err := client.TimeoutUser(guildID, userID, &until, "Reason for timeout")

// Add role to member
err := client.AddMemberRole(guildID, userID, roleID, "Added role")

// Remove role from member
err := client.RemoveMemberRole(guildID, userID, roleID, "Removed role")

// Create role
role, err := client.CreateRole(guildID, &types.RoleCreateData{
    Name:        "New Role",
    Color:       0xFF0000,
    Permissions: "0",
})

// Leave guild
err := client.LeaveGuild(guildID)

// Join guild via invite
invite, err := client.JoinGuild("inviteCode")
```

### Relationship Management

```go
// Get all relationships (friends + blocked + pending)
relationships, err := client.GetRelationships()

// Get friends only
friends, err := client.GetFriends()

// Get blocked users
blocked, err := client.GetBlockedUsers()

// Get pending friend requests
incoming, outgoing, err := client.GetPendingFriendRequests()

// Send friend request
err := client.SendFriendRequest("username", "1234") // discriminator optional for new usernames

// Accept friend request
err := client.AcceptFriendRequest(userID)

// Remove friend
err := client.RemoveFriend(userID)

// Block user
err := client.BlockUser(userID)

// Unblock user
err := client.UnblockUser(userID)

// Set user note
err := client.SetUserNote(userID, "This is my note about this user")
```

### Voice Operations

```go
// Join voice channel
vc, err := client.JoinVoiceChannel(guildID, channelID, false, false) // mute, deaf

// Check if connected
if vc.Ready {
    // Send speaking status
    vc.Speaking(true)
    
    // Send Opus audio packet
    vc.SendOpus(opusData)
}

// Leave voice channel
err := vc.Disconnect()
```

### Rich Presence

```go
// Set online status
err := client.SetStatus("online") // online, idle, dnd, invisible

// Set "Playing" activity
err := client.SetPlaying("Game Name", "Details", "State", "", "", time.Now())

// Set "Listening" activity (Spotify-style)
err := client.SetListening("Song Name", "Artist", "Album", "", time.Now(), time.Now().Add(3*time.Minute))

// Set "Watching" activity
err := client.SetWatching("Stream Title", "Details", "State")

// Set custom status
err := client.SetCustomStatus("Working on code", nil, nil, nil)

// Clear all activities
err := client.ClearActivities()
```

### Interactions

```go
// Invoke slash command
err := client.InvokeSlashCommand(&types.SlashCommandData{
    ApplicationID: appID,
    GuildID:       guildID,
    ChannelID:     channelID,
    Data: &types.ApplicationCommandData{
        ID:      commandID,
        Name:    "commandname",
        Type:    1,
        Version: version,
    },
})

// Click button
err := client.ClickButton(channelID, messageID, "custom_id", applicationID)

// Select menu option
err := client.SelectMenuSelect(channelID, messageID, "custom_id", []string{"option1"}, applicationID)

// Submit modal
err := client.SubmitModal(channelID, "modal_custom_id", components, applicationID)
```

## Event Handlers

```go
// Ready event (fired when connected)
client.OnReady(func(r *discord.Ready) {
    fmt.Printf("Logged in as %s\n", r.User.Username)
})

// Message events
client.OnMessageCreate(func(msg *discord.Message) {})
client.OnMessageUpdate(func(msg *discord.Message) {})
client.OnMessageDelete(func(del *discord.MessageDelete) {})

// Guild events
client.OnGuildCreate(func(guild *discord.Guild) {})
client.OnGuildUpdate(func(guild *discord.Guild) {})
client.OnGuildDelete(func(del *discord.GuildDelete) {})

// Member events
client.OnGuildMemberAdd(func(member *discord.Member) {})
client.OnGuildMemberRemove(func(member *discord.GuildMemberRemove) {})
client.OnGuildMemberUpdate(func(member *discord.Member) {})

// Channel events
client.OnChannelCreate(func(channel *discord.Channel) {})
client.OnChannelUpdate(func(channel *discord.Channel) {})
client.OnChannelDelete(func(channel *discord.Channel) {})

// Voice events
client.OnVoiceStateUpdate(func(vs *discord.VoiceState) {})
client.OnVoiceServerUpdate(func(vs *discord.VoiceServerUpdate) {})

// Presence events
client.OnPresenceUpdate(func(p *discord.PresenceUpdate) {})

// Relationship events
client.OnRelationshipAdd(func(r *discord.Relationship) {})
client.OnRelationshipRemove(func(r *discord.Relationship) {})

// Typing event
client.OnTypingStart(func(t *discord.TypingStart) {})
```

## State Management

The client maintains an in-memory state cache for quick access to guilds, channels, users, and members.

```go
// Enable state in config
config := discord.Config{
    StateEnabled:   true,
    CacheMembers:   true,
    CachePresences: true,
    CacheMessages:  true,
    MaxMessages:    100,
}

client, _ := discord.New(config)

// Access cached data
guild := client.State.GetGuild(guildID)
channel := client.State.GetChannel(channelID)
user := client.State.GetUser(userID)
member := client.State.GetMember(guildID, userID)
presence := client.State.GetPresence(userID)
```

## Examples

See the [examples](./examples) directory for complete working examples:

| Example | Description |
|---------|-------------|
| [basic](./examples/basic) | Simple message echo bot |
| [messaging](./examples/messaging) | Advanced messaging features |
| [qr_login](./examples/qr_login) | QR code authentication flow |
| [full](./examples/full) | Comprehensive feature demonstration |

## Credits

This library was developed with reference to the following projects:

### discord.js-selfbot-v13
- **Repository**: [github.com/aiko-chan-ai/discord.js-selfbot-v13](https://github.com/aiko-chan-ai/discord.js-selfbot-v13)
- **Author**: aiko-chan-ai
- **Referenced for**: Gateway implementation, Rich Presence, Voice support, Interaction handling

### discord.py-self
- **Repository**: [github.com/dolfies/discord.py-self](https://github.com/dolfies/discord.py-self)
- **Author**: dolfies
- **Referenced for**: REST API endpoints, User Settings, Relationship management, Authentication flows

We extend our gratitude to the maintainers of these projects for their work in documenting and implementing Discord's user API.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

**Note**: This library is provided as-is for educational purposes. The authors assume no responsibility for any consequences resulting from its use.
