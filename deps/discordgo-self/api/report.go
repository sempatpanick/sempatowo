package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hytams/discordgo-self/types"
)

// GetReportReasons returns a list of report reasons (V1 API)
func (c *Client) GetReportReasons(ctx context.Context, channelID, messageID, userID types.Snowflake) ([]types.ReportReason, error) {
	var reasons []types.ReportReason

	endpoint := "/report"
	params := url.Values{}
	if messageID != 0 && channelID != 0 {
		params.Set("message_id", messageID.String())
		params.Set("channel_id", channelID.String())
	} else if userID != 0 {
		params.Set("user_id", userID.String())
	}

	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &reasons)
	return reasons, err
}

// CreateReport creates a report (V1 API)
func (c *Client) CreateReport(ctx context.Context, messageID, channelID, userID types.Snowflake, reason int) (*types.Snowflake, error) {
	body := map[string]interface{}{
		"reason": reason,
	}
	if messageID != 0 {
		body["message_id"] = messageID
		body["channel_id"] = channelID
	} else if userID != 0 {
		body["user_id"] = userID
	}

	var result struct {
		ID types.Snowflake `json:"id"`
	}

	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/report",
		Body:     body,
	}, &result)
	return &result.ID, err
}

// GetReportOptions returns report options (V2 API)
func (c *Client) GetReportOptions(ctx context.Context) ([]types.ReportOption, error) {
	var options []types.ReportOption
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: "/report/options",
	}, &options)
	return options, err
}

// StageReport stages a report for a message (V2 API)
func (c *Client) StageReport(ctx context.Context, channelID, messageID types.Snowflake) (string, error) {
	var result struct {
		Token string `json:"token"`
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/reports/channels/%s/messages/%s", channelID, messageID),
	}, &result)
	return result.Token, err
}

// CreateStagedReport creates a staged report (V2 API)
func (c *Client) CreateStagedReport(ctx context.Context, token, reportType, reportSubtype, subject, description string) error {
	body := map[string]string{
		"token":       token,
		"report_type": reportType,
		"subject":     subject,
		"description": description,
	}
	if reportSubtype != "" {
		body["report_subtype"] = reportSubtype
	}

	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/reports",
		Body:     body,
	})
	return err
}

// GetReportMenu returns a report menu (V3 API)
func (c *Client) GetReportMenu(ctx context.Context, menuType, variant string) (*types.ReportMenu, error) {
	var menu types.ReportMenu

	endpoint := fmt.Sprintf("/reporting/menu/%s", menuType)
	if variant != "" {
		endpoint += "?variant=" + variant
	}

	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &menu)
	return &menu, err
}

// SubmitReportMenu submits a report menu (V3 API)
func (c *Client) SubmitReportMenu(ctx context.Context, menuType string, data types.ReportSubmitData) (*types.Snowflake, error) {
	var result struct {
		ReportID types.Snowflake `json:"report_id"`
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/reporting/%s", menuType),
		Body:     data,
	}, &result)
	return &result.ReportID, err
}

// ============================================
// Unauthenticated / DSA Reporting API
// ============================================

// QueryUnauthenticatedReportEligibility checks if the user can use unauthenticated reporting
func (c *Client) QueryUnauthenticatedReportEligibility(ctx context.Context) error {
	_, err := c.Do(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: "/reporting/unauthenticated/experiment",
	})
	return err
}

// GetUnauthenticatedReportCapabilities returns available unauthenticated report types
func (c *Client) GetUnauthenticatedReportCapabilities(ctx context.Context) ([]string, error) {
	var result struct {
		Capabilities []string `json:"capabilities"`
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: "/reporting/unauthenticated/capabilities",
	}, &result)
	return result.Capabilities, err
}

// GetUnauthenticatedReportVerificationCode sends a verification code to email
func (c *Client) GetUnauthenticatedReportVerificationCode(ctx context.Context, reportType, email string) error {
	body := map[string]string{
		"name":  reportType,
		"email": email,
	}
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/reporting/unauthenticated/%s/code", reportType),
		Body:     body,
	})
	return err
}

// VerifyUnauthenticatedReport verifies the email code
func (c *Client) VerifyUnauthenticatedReport(ctx context.Context, reportType, email, code string) (string, error) {
	body := map[string]string{
		"name":  reportType,
		"email": email,
		"code":  code,
	}
	var result struct {
		Token string `json:"token"`
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/reporting/unauthenticated/%s/verify", reportType),
		Body:     body,
	}, &result)
	return result.Token, err
}

// GetUnauthenticatedReportMenu returns a report menu for unauthenticated users
func (c *Client) GetUnauthenticatedReportMenu(ctx context.Context, reportType, variant string) (*types.ReportMenu, error) {
	var menu types.ReportMenu

	endpoint := fmt.Sprintf("/reporting/unauthenticated/menu/%s", reportType)
	if variant != "" {
		endpoint += "?variant=" + variant
	}

	err := c.DoJSON(ctx, Request{
		Method:   http.MethodGet,
		Endpoint: endpoint,
	}, &menu)
	return &menu, err
}

// SubmitUnauthenticatedReportMenu submits an unauthenticated report
func (c *Client) SubmitUnauthenticatedReportMenu(ctx context.Context, reportType string, data types.ReportSubmitData) (*types.Snowflake, error) {
	var result struct {
		ReportID types.Snowflake `json:"report_id"`
	}
	err := c.DoJSON(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: fmt.Sprintf("/reporting/unauthenticated/%s", reportType),
		Body:     data,
	}, &result)
	return &result.ReportID, err
}

// RequestReportReview submits a request to review a report
func (c *Client) RequestReportReview(ctx context.Context, token string) error {
	body := map[string]string{
		"token": token,
	}
	_, err := c.Do(ctx, Request{
		Method:   http.MethodPost,
		Endpoint: "/reporting/review",
		Body:     body,
	})
	return err
}
