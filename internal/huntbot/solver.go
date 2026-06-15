package huntbot

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"sort"
	"strings"
)

type rgbaImage struct {
	data   []byte
	width  int
	height int
}

type letterMatch struct {
	x, y   int
	letter string
	width  int
	height int
}

// SolvePasswordCaptcha downloads the captcha image and matches letter templates.
func SolvePasswordCaptcha(captchaURL, token string) (string, error) {
	checks, err := buildTemplateChecks()
	if err != nil {
		return "", err
	}

	large, err := fetchCaptchaImage(captchaURL, token)
	if err != nil {
		return "", err
	}

	var matches []letterMatch
	for _, check := range checks {
		for y := 0; y <= large.height-check.height; y++ {
			for x := 0; x <= large.width-check.width; x++ {
				if !pixelsMatch(large, check.img, x, y) {
					continue
				}
				if overlaps(matches, x, y, check.width, check.height) {
					continue
				}
				matches = append(matches, letterMatch{
					x: x, y: y, letter: check.letter,
					width: check.width, height: check.height,
				})
			}
		}
	}

	sort.Slice(matches, func(i, j int) bool { return matches[i].x < matches[j].x })
	var sb strings.Builder
	for _, m := range matches {
		sb.WriteString(m.letter)
	}
	return sb.String(), nil
}

type templateCheck struct {
	img    rgbaImage
	letter string
	width  int
	height int
}

func buildTemplateChecks() ([]templateCheck, error) {
	var checks []templateCheck
	for _, group := range priorityGroups {
		for _, letter := range group {
			b64, ok := LetterTemplates[letter]
			if !ok {
				continue
			}
			img, err := decodeBase64PNG(b64)
			if err != nil {
				return nil, err
			}
			checks = append(checks, templateCheck{
				img: img, letter: letter,
				width: img.width, height: img.height,
			})
		}
	}
	return checks, nil
}

func decodeBase64PNG(b64 string) (rgbaImage, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return rgbaImage{}, err
	}
	img, err := pngToRGBA(raw)
	if err != nil {
		return rgbaImage{}, err
	}
	return img, nil
}

func pngToRGBA(data []byte) (rgbaImage, error) {
	src, err := pngDecodeRGBA(data)
	if err != nil {
		return rgbaImage{}, err
	}
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	out := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			i := (y*w + x) * 4
			out[i] = byte(r >> 8)
			out[i+1] = byte(g >> 8)
			out[i+2] = byte(b >> 8)
			out[i+3] = byte(a >> 8)
		}
	}
	return rgbaImage{data: out, width: w, height: h}, nil
}

func pngDecodeRGBA(data []byte) (image.Image, error) {
	return pngDecode(bytes.NewReader(data))
}

func pngDecode(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	return img, err
}

func fetchCaptchaImage(url, token string) (rgbaImage, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return rgbaImage{}, err
	}
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return rgbaImage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return rgbaImage{}, io.ErrUnexpectedEOF
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return rgbaImage{}, err
	}
	return pngToRGBA(body)
}

func pixelsMatch(large, small rgbaImage, x, y int) bool {
	for sy := 0; sy < small.height; sy++ {
		for sx := 0; sx < small.width; sx++ {
			si := (sy*small.width + sx) * 4
			if small.data[si+3] == 0 {
				continue
			}
			li := ((y+sy)*large.width + (x + sx)) * 4
			if large.data[li] != small.data[si] ||
				large.data[li+1] != small.data[si+1] ||
				large.data[li+2] != small.data[si+2] ||
				large.data[li+3] != small.data[si+3] {
				return false
			}
		}
	}
	return true
}

func overlaps(matches []letterMatch, x, y, w, h int) bool {
	for _, m := range matches {
		if m.x-w < x && x < m.x+w && m.y-h < y && y < m.y+h {
			return true
		}
	}
	return false
}
