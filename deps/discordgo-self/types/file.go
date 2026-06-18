package types

import (
	"io"
	"os"
	"path/filepath"
)

type File struct {
	Name        string
	ContentType string
	Reader      io.Reader
	Spoiler     bool
	Description string
}

func NewFile(name string, reader io.Reader) *File {
	return &File{
		Name:   name,
		Reader: reader,
	}
}

func NewFileFromPath(path string) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &File{
		Name:   filepath.Base(path),
		Reader: file,
	}, nil
}

func NewFileFromBytes(name string, data []byte) *File {
	return &File{
		Name:   name,
		Reader: &bytesReader{data: data},
	}
}

type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (f *File) WithSpoiler() *File {
	f.Spoiler = true
	if f.Name != "" && len(f.Name) > 0 && f.Name[:9] != "SPOILER_" {
		f.Name = "SPOILER_" + f.Name
	}
	return f
}

func (f *File) WithDescription(desc string) *File {
	f.Description = desc
	return f
}

func (f *File) WithContentType(ct string) *File {
	f.ContentType = ct
	return f
}

func (f *File) Close() error {
	if closer, ok := f.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

type MessageSendDataWithFiles struct {
	Content          string            `json:"content,omitempty"`
	TTS              bool              `json:"tts,omitempty"`
	Embeds           []*Embed          `json:"embeds,omitempty"`
	AllowedMentions  *AllowedMentions  `json:"allowed_mentions,omitempty"`
	MessageReference *MessageReference `json:"message_reference,omitempty"`
	Components       []Component       `json:"components,omitempty"`
	StickerIDs       []Snowflake       `json:"sticker_ids,omitempty"`
	Flags            MessageFlags      `json:"flags,omitempty"`
	Nonce            string            `json:"nonce,omitempty"`
	Files            []*File           `json:"-"`
}

type AttachmentPayload struct {
	ID          int    `json:"id"`
	Filename    string `json:"filename"`
	Description string `json:"description,omitempty"`
}

const (
	MIMETypeJPEG = "image/jpeg"
	MIMETypePNG  = "image/png"
	MIMETypeGIF  = "image/gif"
	MIMETypeWEBP = "image/webp"
	MIMETypeMP4  = "video/mp4"
	MIMETypeMOV  = "video/quicktime"
	MIMETypeMP3  = "audio/mpeg"
	MIMETypeOGG  = "audio/ogg"
	MIMETypePDF  = "application/pdf"
	MIMETypeZIP  = "application/zip"
	MIMETypeText = "text/plain"
	MIMETypeJSON = "application/json"
)

func GuessContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".jpg", ".jpeg":
		return MIMETypeJPEG
	case ".png":
		return MIMETypePNG
	case ".gif":
		return MIMETypeGIF
	case ".webp":
		return MIMETypeWEBP
	case ".mp4":
		return MIMETypeMP4
	case ".mov":
		return MIMETypeMOV
	case ".mp3":
		return MIMETypeMP3
	case ".ogg":
		return MIMETypeOGG
	case ".pdf":
		return MIMETypePDF
	case ".zip":
		return MIMETypeZIP
	case ".txt":
		return MIMETypeText
	case ".json":
		return MIMETypeJSON
	default:
		return "application/octet-stream"
	}
}
