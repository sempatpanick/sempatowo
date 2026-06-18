package props

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// SuperProperties represents the X-Super-Properties header data.
type SuperProperties struct {
	OS                     string  `json:"os"`
	Browser                string  `json:"browser"`
	Device                 string  `json:"device"`
	SystemLocale           string  `json:"system_locale"`
	BrowserUserAgent       string  `json:"browser_user_agent"`
	BrowserVersion         string  `json:"browser_version"`
	OSVersion              string  `json:"os_version"`
	Referrer               string  `json:"referrer"`
	ReferringDomain        string  `json:"referring_domain"`
	ReferrerCurrent        string  `json:"referrer_current"`
	ReferringDomainCurrent string  `json:"referring_domain_current"`
	ReleaseChannel         string  `json:"release_channel"`
	ClientBuildNumber      int     `json:"client_build_number"`
	ClientEventSource      *string `json:"client_event_source"`
	DesignID               int     `json:"design_id,omitempty"`
}

const (
	DefaultChromeVersion     = "131"
	DefaultChromeFull        = "131.0.0.0"
	DefaultUserAgentTemplate = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Safari/537.36"
)

var (
	buildNumberCache     int
	buildNumberCacheMu   sync.RWMutex
	buildNumberCacheTime time.Time
	buildNumberCacheTTL  = 1 * time.Hour
)

// NewSuperProperties creates a new SuperProperties with sensible defaults.
func NewSuperProperties() *SuperProperties {
	osName := "Windows"
	osVersion := "10"

	switch runtime.GOOS {
	case "darwin":
		osName = "Mac OS X"
		osVersion = "10.15.7"
	case "linux":
		osName = "Linux"
		osVersion = "x86_64"
	}

	userAgent := fmt.Sprintf(DefaultUserAgentTemplate, DefaultChromeFull)

	return &SuperProperties{
		OS:                     osName,
		Browser:                "Chrome",
		Device:                 "",
		SystemLocale:           "en-US",
		BrowserUserAgent:       userAgent,
		BrowserVersion:         DefaultChromeFull,
		OSVersion:              osVersion,
		Referrer:               "",
		ReferringDomain:        "",
		ReferrerCurrent:        "",
		ReferringDomainCurrent: "",
		ReleaseChannel:         "stable",
		ClientBuildNumber:      0,
		ClientEventSource:      nil,
	}
}

// Encode returns the base64-encoded JSON representation.
func (sp *SuperProperties) Encode() (string, error) {
	data, err := json.Marshal(sp)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// MustEncode returns the base64-encoded JSON, panicking on error.
func (sp *SuperProperties) MustEncode() string {
	encoded, err := sp.Encode()
	if err != nil {
		panic(err)
	}
	return encoded
}

// UserAgent returns the browser user agent string.
func (sp *SuperProperties) UserAgent() string {
	return sp.BrowserUserAgent
}

// GatewayProperties returns properties for gateway identify payload.
func (sp *SuperProperties) GatewayProperties() map[string]interface{} {
	return map[string]interface{}{
		"os":                       sp.OS,
		"browser":                  sp.Browser,
		"device":                   sp.Device,
		"system_locale":            sp.SystemLocale,
		"browser_user_agent":       sp.BrowserUserAgent,
		"browser_version":          sp.BrowserVersion,
		"os_version":               sp.OSVersion,
		"referrer":                 sp.Referrer,
		"referring_domain":         sp.ReferringDomain,
		"referrer_current":         sp.ReferrerCurrent,
		"referring_domain_current": sp.ReferringDomainCurrent,
		"release_channel":          sp.ReleaseChannel,
		"client_build_number":      sp.ClientBuildNumber,
		"client_event_source":      sp.ClientEventSource,
	}
}

// FetchBuildNumber fetches the current Discord client build number.
func FetchBuildNumber() (int, error) {
	buildNumberCacheMu.RLock()
	if buildNumberCache > 0 && time.Since(buildNumberCacheTime) < buildNumberCacheTTL {
		cached := buildNumberCache
		buildNumberCacheMu.RUnlock()
		return cached, nil
	}
	buildNumberCacheMu.RUnlock()

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://discord.com/app")
	if err != nil {
		return 0, fmt.Errorf("failed to fetch discord app: %w", err)
	}
	defer resp.Body.Close()

	buf := make([]byte, 1024*1024)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])

	assetRegex := regexp.MustCompile(`/assets/([a-f0-9]+)\.js`)
	matches := assetRegex.FindAllStringSubmatch(body, -1)

	if len(matches) == 0 {
		return getDefaultBuildNumber(), nil
	}

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		assetURL := "https://discord.com/assets/" + match[1] + ".js"
		assetResp, err := client.Get(assetURL)
		if err != nil {
			continue
		}

		assetBuf := make([]byte, 512*1024)
		n, _ := assetResp.Body.Read(assetBuf)
		assetResp.Body.Close()

		assetBody := string(assetBuf[:n])

		buildRegex := regexp.MustCompile(`buildNumber["\s:]+(\d{6})`)
		buildMatch := buildRegex.FindStringSubmatch(assetBody)
		if len(buildMatch) >= 2 {
			var buildNum int
			fmt.Sscanf(buildMatch[1], "%d", &buildNum)
			if buildNum > 0 {
				buildNumberCacheMu.Lock()
				buildNumberCache = buildNum
				buildNumberCacheTime = time.Now()
				buildNumberCacheMu.Unlock()
				return buildNum, nil
			}
		}
	}

	return getDefaultBuildNumber(), nil
}

func getDefaultBuildNumber() int {
	return 345678
}

// UpdateBuildNumber fetches and updates the build number in SuperProperties.
func (sp *SuperProperties) UpdateBuildNumber() error {
	buildNum, err := FetchBuildNumber()
	if err != nil {
		return err
	}
	sp.ClientBuildNumber = buildNum
	return nil
}

// Clone creates a copy of the SuperProperties.
func (sp *SuperProperties) Clone() *SuperProperties {
	clone := *sp
	if sp.ClientEventSource != nil {
		ces := *sp.ClientEventSource
		clone.ClientEventSource = &ces
	}
	return &clone
}

// SetLocale sets the system locale.
func (sp *SuperProperties) SetLocale(locale string) *SuperProperties {
	sp.SystemLocale = locale
	return sp
}

// SetReleaseChannel sets the release channel.
func (sp *SuperProperties) SetReleaseChannel(channel string) *SuperProperties {
	sp.ReleaseChannel = strings.ToLower(channel)
	return sp
}

// WithCustomUserAgent sets a custom user agent.
func (sp *SuperProperties) WithCustomUserAgent(ua string) *SuperProperties {
	sp.BrowserUserAgent = ua

	chromeMatch := regexp.MustCompile(`Chrome/(\d+\.\d+\.\d+\.\d+)`).FindStringSubmatch(ua)
	if len(chromeMatch) >= 2 {
		sp.BrowserVersion = chromeMatch[1]
	}

	return sp
}

// ClientType represents the platform/client type.
type ClientType string

const (
	ClientWindows  ClientType = "windows"
	ClientMacOS    ClientType = "macos"
	ClientLinux    ClientType = "linux"
	ClientiOS      ClientType = "ios"
	ClientAndroid  ClientType = "android"
	ClientEmbedded ClientType = "embedded"
	ClientDiscord  ClientType = "discord"
)

// NewPropertiesForClient creates SuperProperties for a specific client type.
func NewPropertiesForClient(clientType ClientType) *SuperProperties {
	switch clientType {
	case ClientiOS:
		return NewIOSProperties()
	case ClientAndroid:
		return NewAndroidProperties()
	case ClientLinux:
		return NewLinuxProperties()
	case ClientMacOS:
		return NewMacOSProperties()
	case ClientDiscord:
		return NewDiscordDesktopProperties()
	case ClientEmbedded:
		return NewEmbeddedProperties()
	default:
		return NewSuperProperties()
	}
}

// NewIOSProperties creates iOS Discord app properties.
func NewIOSProperties() *SuperProperties {
	return &SuperProperties{
		OS:                     "iOS",
		Browser:                "Discord iOS",
		Device:                 "iPhone",
		SystemLocale:           "en-US",
		BrowserUserAgent:       "",
		BrowserVersion:         "251.0",
		OSVersion:              "17.4.1",
		Referrer:               "",
		ReferringDomain:        "",
		ReferrerCurrent:        "",
		ReferringDomainCurrent: "",
		ReleaseChannel:         "stable",
		ClientBuildNumber:      0,
		ClientEventSource:      nil,
	}
}

// NewAndroidProperties creates Android Discord app properties.
func NewAndroidProperties() *SuperProperties {
	return &SuperProperties{
		OS:                     "Android",
		Browser:                "Discord Android",
		Device:                 "samsung SM-S918B",
		SystemLocale:           "en-US",
		BrowserUserAgent:       "",
		BrowserVersion:         "251.7",
		OSVersion:              "14",
		Referrer:               "",
		ReferringDomain:        "",
		ReferrerCurrent:        "",
		ReferringDomainCurrent: "",
		ReleaseChannel:         "stable",
		ClientBuildNumber:      0,
		ClientEventSource:      nil,
	}
}

// NewLinuxProperties creates Linux browser properties.
func NewLinuxProperties() *SuperProperties {
	userAgent := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	return &SuperProperties{
		OS:                     "Linux",
		Browser:                "Chrome",
		Device:                 "",
		SystemLocale:           "en-US",
		BrowserUserAgent:       userAgent,
		BrowserVersion:         "131.0.0.0",
		OSVersion:              "x86_64",
		Referrer:               "",
		ReferringDomain:        "",
		ReferrerCurrent:        "",
		ReferringDomainCurrent: "",
		ReleaseChannel:         "stable",
		ClientBuildNumber:      0,
		ClientEventSource:      nil,
	}
}

// NewMacOSProperties creates macOS browser properties.
func NewMacOSProperties() *SuperProperties {
	userAgent := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	return &SuperProperties{
		OS:                     "Mac OS X",
		Browser:                "Chrome",
		Device:                 "",
		SystemLocale:           "en-US",
		BrowserUserAgent:       userAgent,
		BrowserVersion:         "131.0.0.0",
		OSVersion:              "10.15.7",
		Referrer:               "",
		ReferringDomain:        "",
		ReferrerCurrent:        "",
		ReferringDomainCurrent: "",
		ReleaseChannel:         "stable",
		ClientBuildNumber:      0,
		ClientEventSource:      nil,
	}
}

// NewDiscordDesktopProperties creates Discord Desktop client properties.
func NewDiscordDesktopProperties() *SuperProperties {
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) discord/1.0.9168 Chrome/128.0.6613.186 Electron/32.2.2 Safari/537.36"
	return &SuperProperties{
		OS:                     "Windows",
		Browser:                "Discord Client",
		Device:                 "",
		SystemLocale:           "en-US",
		BrowserUserAgent:       userAgent,
		BrowserVersion:         "32.2.2",
		OSVersion:              "10.0.22631",
		Referrer:               "",
		ReferringDomain:        "",
		ReferrerCurrent:        "",
		ReferringDomainCurrent: "",
		ReleaseChannel:         "stable",
		ClientBuildNumber:      0,
		ClientEventSource:      nil,
	}
}

// NewEmbeddedProperties creates embedded/console/bot properties.
func NewEmbeddedProperties() *SuperProperties {
	return &SuperProperties{
		OS:                     "Windows",
		Browser:                "Discord Embedded",
		Device:                 "embedded",
		SystemLocale:           "en-US",
		BrowserUserAgent:       "",
		BrowserVersion:         "",
		OSVersion:              "10",
		Referrer:               "",
		ReferringDomain:        "",
		ReferrerCurrent:        "",
		ReferringDomainCurrent: "",
		ReleaseChannel:         "stable",
		ClientBuildNumber:      0,
		ClientEventSource:      nil,
	}
}

// SetDevice sets a custom device name.
func (sp *SuperProperties) SetDevice(device string) *SuperProperties {
	sp.Device = device
	return sp
}

// SetOSVersion sets a custom OS version.
func (sp *SuperProperties) SetOSVersion(version string) *SuperProperties {
	sp.OSVersion = version
	return sp
}

// SetBrowser sets a custom browser name.
func (sp *SuperProperties) SetBrowser(browser string) *SuperProperties {
	sp.Browser = browser
	return sp
}

// SetOS sets a custom OS name.
func (sp *SuperProperties) SetOS(os string) *SuperProperties {
	sp.OS = os
	return sp
}
