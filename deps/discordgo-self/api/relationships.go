package api

import (
	"context"
	"fmt"

	"github.com/hytams/discordgo-self/types"
)

func (c *Client) GetRelationships(ctx context.Context) ([]*types.Relationship, error) {
	var relationships []*types.Relationship
	err := c.DoJSON(ctx, Request{
		Method:   "GET",
		Endpoint: "/users/@me/relationships",
	}, &relationships)
	return relationships, err
}

func (c *Client) GetRelationship(ctx context.Context, userID types.Snowflake) (*types.Relationship, error) {
	var relationship types.Relationship
	err := c.DoJSON(ctx, Request{
		Method:   "GET",
		Endpoint: fmt.Sprintf("/users/@me/relationships/%s", userID),
	}, &relationship)
	return &relationship, err
}

func (c *Client) AddFriend(ctx context.Context, userID types.Snowflake) error {
	return c.DoJSON(ctx, Request{
		Method:   "PUT",
		Endpoint: fmt.Sprintf("/users/@me/relationships/%s", userID),
		Body:     map[string]interface{}{},
	}, nil)
}

func (c *Client) SendFriendRequest(ctx context.Context, username string, discriminator string) error {
	body := map[string]string{
		"username": username,
	}
	if discriminator != "" && discriminator != "0" {
		body["discriminator"] = discriminator
	}
	return c.DoJSON(ctx, Request{
		Method:   "POST",
		Endpoint: "/users/@me/relationships",
		Body:     body,
	}, nil)
}

func (c *Client) RemoveFriend(ctx context.Context, userID types.Snowflake) error {
	return c.DoJSON(ctx, Request{
		Method:   "DELETE",
		Endpoint: fmt.Sprintf("/users/@me/relationships/%s", userID),
	}, nil)
}

func (c *Client) BlockUser(ctx context.Context, userID types.Snowflake) error {
	return c.DoJSON(ctx, Request{
		Method:   "PUT",
		Endpoint: fmt.Sprintf("/users/@me/relationships/%s", userID),
		Body:     map[string]int{"type": 2},
	}, nil)
}

func (c *Client) UnblockUser(ctx context.Context, userID types.Snowflake) error {
	return c.DoJSON(ctx, Request{
		Method:   "DELETE",
		Endpoint: fmt.Sprintf("/users/@me/relationships/%s", userID),
	}, nil)
}

func (c *Client) AcceptFriendRequest(ctx context.Context, userID types.Snowflake) error {
	return c.DoJSON(ctx, Request{
		Method:   "PUT",
		Endpoint: fmt.Sprintf("/users/@me/relationships/%s", userID),
		Body:     map[string]interface{}{},
	}, nil)
}

func (c *Client) SetUserNote(ctx context.Context, userID types.Snowflake, note string) error {
	return c.DoJSON(ctx, Request{
		Method:   "PUT",
		Endpoint: fmt.Sprintf("/users/@me/notes/%s", userID),
		Body:     map[string]string{"note": note},
	}, nil)
}

func (c *Client) GetFriends(ctx context.Context) ([]*types.Relationship, error) {
	relationships, err := c.GetRelationships(ctx)
	if err != nil {
		return nil, err
	}

	var friends []*types.Relationship
	for _, r := range relationships {
		if r.Type == types.RelationshipTypeFriend {
			friends = append(friends, r)
		}
	}
	return friends, nil
}

func (c *Client) GetBlockedUsers(ctx context.Context) ([]*types.Relationship, error) {
	relationships, err := c.GetRelationships(ctx)
	if err != nil {
		return nil, err
	}

	var blocked []*types.Relationship
	for _, r := range relationships {
		if r.Type == types.RelationshipTypeBlocked {
			blocked = append(blocked, r)
		}
	}
	return blocked, nil
}

func (c *Client) GetPendingFriendRequests(ctx context.Context) ([]*types.Relationship, []*types.Relationship, error) {
	relationships, err := c.GetRelationships(ctx)
	if err != nil {
		return nil, nil, err
	}

	var incoming, outgoing []*types.Relationship
	for _, r := range relationships {
		switch r.Type {
		case types.RelationshipTypePendingIncoming:
			incoming = append(incoming, r)
		case types.RelationshipTypePendingOutgoing:
			outgoing = append(outgoing, r)
		}
	}
	return incoming, outgoing, nil
}
