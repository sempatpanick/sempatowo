package captcha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	owoClientID    = "408785106942164992"
	owoRedirectURI = "https://owobot.com/api/auth/discord/redirect"
	owoScope       = "identify guilds email guilds.members.read"
	owoSiteKey     = "a6a1d5ce-612d-472d-8e37-7601408fbc09"
	fallbackURL    = "https://owobot.com/captcha"
	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/111.0"
)

var discordOAuthURL = "https://discord.com/api/oauth2/authorize?response_type=code&redirect_uri=https%3A%2F%2Fowobot.com%2Fapi%2Fauth%2Fdiscord%2Fredirect&scope=identify%20guilds%20email%20guilds.members.read&client_id=408785106942164992"

var httpClient = &http.Client{Timeout: 30 * time.Second}

type Result struct {
	Success bool
	Message string
	URL     string
}

// GetURL returns the OwO captcha page URL for manual solving.
func GetURL(token string) string {
	url, err := getOAuthURL(token)
	if err != nil || url == "" {
		return fallbackURL
	}
	return url
}

// GetURLAsync fetches the captcha URL without blocking other work.
func GetURLAsync(token string, callback func(string)) {
	go func() {
		callback(GetURL(token))
	}()
}

// Solve tries automatic hCaptcha solving when CAPTCHA_API_KEY is set.
func Solve(token string) Result {
	apiKey := strings.TrimSpace(os.Getenv("CAPTCHA_API_KEY"))
	manualURL := GetURL(token)

	if apiKey == "" {
		return Result{Success: false, Message: "CAPTCHA_API_KEY not set — solve manually in browser", URL: manualURL}
	}

	timeoutSec := 90
	if v := strings.TrimSpace(os.Getenv("CAPTCHA_SOLVE_TIMEOUT")); v != "" {
		if n, err := fmt.Sscanf(v, "%d", &timeoutSec); n == 1 && err == nil && timeoutSec > 0 {
			// ok
		} else {
			timeoutSec = 90
		}
	}

	cookie, err := getOwoAuthCookie(token)
	if err != nil || cookie == "" {
		return Result{Success: false, Message: "Failed to authenticate with OwO (OAuth cookie)", URL: manualURL}
	}

	service := strings.TrimSpace(os.Getenv("CAPTCHA_SERVICE"))
	if service == "" {
		service = "capsolver"
	}

	hcToken, err := solveHcaptcha(apiKey, service, time.Duration(timeoutSec)*time.Second)
	if err != nil || hcToken == "" {
		return Result{Success: false, Message: fmt.Sprintf("hCaptcha solve failed (%s)", service), URL: manualURL}
	}

	if !verifyOwoCaptcha(cookie, hcToken) {
		return Result{Success: false, Message: "OwO rejected captcha token", URL: manualURL}
	}

	return Result{Success: true, Message: "Captcha solved and verified"}
}

func getOAuthURL(token string) (string, error) {
	body := map[string]any{
		"guild_id": "1119963281923248219", "permissions": "8", "authorize": true,
		"integration_type": 0,
		"location_context": map[string]any{"guild_id": "10000", "channel_id": "10000", "channel_type": 10000},
	}
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, discordOAuthURL, bytes.NewReader(data))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var out struct {
		Location string `json:"location"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out.Location, nil
}

func getOwoAuthCookie(token string) (string, error) {
	noRedirect := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, _ := http.NewRequest(http.MethodGet, "https://owobot.com/api/auth/discord", nil)
	resp, err := noRedirect.Do(req)
	if err != nil {
		return "", err
	}
	oauthURL := resp.Header.Get("Location")
	resp.Body.Close()
	if oauthURL == "" {
		return "", fmt.Errorf("no oauth redirect")
	}

	reqWarm, _ := http.NewRequest(http.MethodGet, oauthURL, nil)
	reqWarm.Header.Set("User-Agent", userAgent)
	_, _ = httpClient.Do(reqWarm)

	oauthBody := map[string]any{
		"permissions": "0", "authorize": true, "integration_type": 0,
		"location_context": map[string]any{"guild_id": "10000", "channel_id": "10000", "channel_type": 10000},
	}
	data, _ := json.Marshal(oauthBody)
	req, _ = http.NewRequest(http.MethodPost, oauthURL, bytes.NewReader(data))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err = httpClient.Do(req)
	if err != nil {
		return "", err
	}
	var oauthResp struct {
		Location string `json:"location"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&oauthResp)
	resp.Body.Close()

	req, _ = http.NewRequest(http.MethodGet, oauthResp.Location, nil)
	req.Header.Set("User-Agent", userAgent)
	resp, err = noRedirect.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	cookies := resp.Header.Values("Set-Cookie")
	if len(cookies) == 0 {
		return "", fmt.Errorf("no session cookie")
	}
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		parts = append(parts, strings.Split(c, ";")[0])
	}
	return strings.Join(parts, "; "), nil
}

func verifyOwoCaptcha(cookie, hcToken string) bool {
	body, _ := json.Marshal(map[string]string{"token": hcToken})
	req, _ := http.NewRequest(http.MethodPost, "https://owobot.com/api/captcha/verify", bytes.NewReader(body))
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

var serviceEndpoints = map[string][2]string{
	"capsolver":  {"https://api.capsolver.com/createTask", "https://api.capsolver.com/getTaskResult"},
	"capmonster": {"https://api.capmonster.cloud/createTask", "https://api.capmonster.cloud/getTaskResult"},
	"2captcha":   {"https://api.2captcha.com/createTask", "https://api.2captcha.com/getTaskResult"},
}

func solveHcaptcha(apiKey, service string, maxWait time.Duration) (string, error) {
	service = strings.ToLower(service)
	eps, ok := serviceEndpoints[service]
	if !ok {
		service = "capsolver"
		eps = serviceEndpoints[service]
	}

	createBody, _ := json.Marshal(map[string]any{
		"clientKey": apiKey,
		"task": map[string]string{
			"type": "HCaptchaTaskProxyLess", "websiteKey": owoSiteKey, "websiteURL": "https://owobot.com",
		},
	})

	resp, err := http.Post(eps[0], "application/json", bytes.NewReader(createBody))
	if err != nil {
		return "", err
	}
	var createResp struct {
		TaskID string `json:"taskId"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&createResp)
	resp.Body.Close()
	if createResp.TaskID == "" {
		return "", fmt.Errorf("no task id")
	}

	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		resultBody, _ := json.Marshal(map[string]string{"clientKey": apiKey, "taskId": createResp.TaskID})
		resp, err = http.Post(eps[1], "application/json", bytes.NewReader(resultBody))
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			Status   string `json:"status"`
			Solution struct {
				GRecaptchaResponse string `json:"gRecaptchaResponse"`
			} `json:"solution"`
		}
		_ = json.Unmarshal(body, &result)
		if result.Status == "ready" {
			return result.Solution.GRecaptchaResponse, nil
		}
		if result.Status == "failed" {
			return "", fmt.Errorf("solver failed")
		}
	}
	return "", fmt.Errorf("timeout")
}

// Suppress unused const warnings for OAuth fields used in future expansion.
var _ = owoClientID
var _ = owoRedirectURI
var _ = owoScope
