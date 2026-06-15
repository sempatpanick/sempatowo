package quest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

var progressRe = regexp.MustCompile(`^\d+/\d+$`)

type ocrLine struct {
	Text  string
	XAxis int
	YAxis int
}

type ocrAPIResponse struct {
	Error                 string          `json:"error"`
	IsErroredOnProcessing bool            `json:"IsErroredOnProcessing"`
	ErrorMessage          json.RawMessage `json:"ErrorMessage"`
	ParsedResults         []struct {
		TextOverlay struct {
			Lines []struct {
				LineText string `json:"LineText"`
				Words    []struct {
					Left int `json:"Left"`
					Top  int `json:"Top"`
				} `json:"Words"`
			} `json:"Lines"`
		} `json:"TextOverlay"`
	} `json:"ParsedResults"`
}

// FetchQuestDetails downloads a quest-rows image and parses quest rows via OCR.space.
func FetchQuestDetails(imageURL, apiKey string) ([]ParsedQuest, error) {
	if apiKey == "" {
		apiKey = "helloworld"
	}
	imageData, filename, err := downloadQuestImage(imageURL)
	if err != nil {
		return nil, fmt.Errorf("download quest image: %w", err)
	}
	lines, err := ocrImageBytes(imageData, filename, apiKey)
	if err != nil {
		return nil, err
	}
	return extractQuests(lines), nil
}

func downloadQuestImage(imageURL string) ([]byte, string, error) {
	req, err := http.NewRequest(http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("empty image response")
	}

	filename := questImageFilename(imageURL)
	return data, filename, nil
}

func questImageFilename(imageURL string) string {
	u, err := url.Parse(imageURL)
	if err != nil {
		return "quest-rows.png"
	}
	name := path.Base(u.Path)
	if name == "" || name == "." || name == "/" {
		return "quest-rows.png"
	}
	return name
}

func ocrImageBytes(imageData []byte, filename, apiKey string) ([]ocrLine, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("apikey", apiKey)
	_ = w.WriteField("language", "eng")
	_ = w.WriteField("isOverlayRequired", "true")
	_ = w.WriteField("ocrengine", "2")
	_ = w.WriteField("filetype", "png")

	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(imageData); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.ocr.space/parse/image", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Referer", "https://ocr.space/")
	req.Header.Set("Origin", "https://ocr.space")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return parseOCRResponse(raw, apiKey)
}

func parseOCRResponse(raw []byte, apiKey string) ([]ocrLine, error) {
	var result ocrAPIResponse
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("ocr response decode: %w", err)
	}

	if msg := strings.TrimSpace(result.Error); msg != "" {
		return nil, formatOCRError(msg, apiKey)
	}
	if result.IsErroredOnProcessing {
		msg := parseOCRErrorMessage(result.ErrorMessage)
		if msg == "" {
			msg = "processing error"
		}
		return nil, formatOCRError(msg, apiKey)
	}
	if len(result.ParsedResults) == 0 {
		return nil, formatOCRError("no parsed results", apiKey)
	}

	var out []ocrLine
	alphaNum := regexp.MustCompile(`[^a-zA-Z0-9 /]`)
	for _, page := range result.ParsedResults {
		for _, line := range page.TextOverlay.Lines {
			x, y := 0, 0
			if len(line.Words) > 0 {
				x = line.Words[0].Left
				y = line.Words[0].Top
			}
			text := strings.TrimSpace(alphaNum.ReplaceAllString(line.LineText, ""))
			if text != "" {
				out = append(out, ocrLine{Text: text, XAxis: x, YAxis: y})
			}
		}
	}
	if len(out) == 0 {
		return nil, formatOCRError("no text overlay returned", apiKey)
	}
	return out, nil
}

func parseOCRErrorMessage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return s
	}
	var arr []string
	if json.Unmarshal(raw, &arr) == nil {
		return strings.Join(arr, "; ")
	}
	return strings.TrimSpace(string(raw))
}

func formatOCRError(msg, apiKey string) error {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "rate limit") || strings.Contains(lower, "e553") {
		if strings.EqualFold(strings.TrimSpace(apiKey), "helloworld") {
			return fmt.Errorf("%s — the default ocrApi key is rate-limited; set your free key in config ocrApi (https://ocr.space/ocrapi/freekey)", msg)
		}
		return fmt.Errorf("%s — OCR.space rate limit hit; try again later or use your own API key", msg)
	}
	return fmt.Errorf("%s", msg)
}

func extractQuests(lines []ocrLine) []ParsedQuest {
	blacklist := []string{"claimed", "rewards earned", "you can claim a new quest", "free a slot to receive it"}

	var progress []ocrLine
	var candidates []ocrLine
	for _, item := range lines {
		t := strings.ReplaceAll(item.Text, ",", "")
		if progressRe.MatchString(t) {
			progress = append(progress, item)
			continue
		}
		if len(item.Text) > 4 {
			candidates = append(candidates, item)
		}
	}

	filtered := candidates[:0]
	for _, c := range candidates {
		lower := strings.ToLower(c.Text)
		skip := false
		for _, b := range blacklist {
			if strings.Contains(lower, b) {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, c)
		}
	}
	candidates = filtered

	sortByY(candidates)
	sortByY(progress)

	var quests []ParsedQuest
	for _, p := range progress {
		parts := strings.Split(strings.ReplaceAll(p.Text, ",", ""), "/")
		if len(parts) != 2 {
			continue
		}
		current := atoi(parts[0])
		total := atoi(parts[1])
		for i, c := range candidates {
			if intGap(c.YAxis, p.YAxis, 22, 27) {
				quests = append(quests, ParsedQuest{
					Text:     c.Text,
					Current:  current,
					Total:    total,
					Complete: current >= total,
				})
				candidates = append(candidates[:i], candidates[i+1:]...)
				break
			}
		}
	}
	return quests
}

func intGap(a, b, min, max int) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d >= min && d <= max
}

func sortByY(items []ocrLine) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].YAxis < items[i].YAxis {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
