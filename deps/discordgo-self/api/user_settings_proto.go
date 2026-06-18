package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hytams/discordgo-self/types"
)

// GetUserSettingsProto returns the User Settings Proto for a specific type
func (c *Client) GetUserSettingsProto(ctx context.Context, protoType types.UserSettingsProtoType) (*types.UserSettingsProtoResponse, error) {
	var result types.UserSettingsProtoResponse
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: fmt.Sprintf("/users/@me/settings-proto/%d", protoType),
	}, &result)
	return &result, err
}

// ModifyUserSettingsProto modifies the User Settings Proto
// Note: 'settings' must be the base64 encoded serialized protobuf string.
// Be very careful when using this, as incorrect protobuf data can reset settings.
func (c *Client) ModifyUserSettingsProto(ctx context.Context, protoType types.UserSettingsProtoType, settingsBase64 string, requiredDataVersion int) (*types.UserSettingsProtoResponse, error) {
	body := map[string]interface{}{
		"settings": settingsBase64,
	}
	if requiredDataVersion > 0 {
		body["required_data_version"] = requiredDataVersion
	}

	var result types.UserSettingsProtoResponse
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPatch,
		Endpoint: fmt.Sprintf("/users/@me/settings-proto/%d", protoType),
		Body:     body,
	}, &result)
	return &result, err
}
