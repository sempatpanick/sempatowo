package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// inlineWidth is the column past which an object or array is broken across
// lines. It is generous on purpose: the point is to keep a whole feature —
// `"hunt": { "enabled": true, "delay": { "min": "50s", "max": "3m20s" } }` — on
// one line, so the top of the file reads as a list of features rather than a
// wall of two-token lines.
const inlineWidth = 100

// marshalReadable renders v the way a person would have written it: anything
// that fits on one line stays on one line, everything else is indented.
//
// json.MarshalIndent puts every scalar on its own line, which turned
// `"crate": {"enabled": true}` into three lines and the features block into
// roughly a hundred and eighty. The file is meant to be opened and edited by
// hand, so it is worth the printer.
func marshalReadable(v any) ([]byte, error) {
	// Marshal once and reformat the result, rather than walking v with
	// reflection: this way every MarshalJSON on the way down (Duration, Range)
	// has already had its say, and string escaping is the standard library's
	// problem.
	compact, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := writeValue(&buf, compact, 0); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}

// writeValue renders one JSON value at the given indent depth. raw is exact
// bytes from the compact encoding, so numbers keep the formatting the encoder
// chose instead of round-tripping through float64.
func writeValue(buf *bytes.Buffer, raw json.RawMessage, depth int) error {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return fmt.Errorf("empty JSON value")
	}

	switch trimmed[0] {
	case '{', '[':
		// The budget is measured from where the value actually starts, which is
		// after its key — measuring from the indent alone let a long key push
		// the line past the limit.
		if column(buf)+len(trimmed)+inlineSpacing(trimmed)+1 <= inlineWidth {
			return writeInline(buf, trimmed)
		}
		return writeBlock(buf, trimmed, depth)
	default:
		buf.Write(trimmed)
		return nil
	}
}

// column is how far into the current line the buffer has got.
func column(buf *bytes.Buffer) int {
	b := buf.Bytes()
	return len(b) - bytes.LastIndexByte(b, '\n') - 1
}

// inlineSpacing is the extra width the inline form costs over the compact one:
// a space after every comma and colon, and one inside each brace.
func inlineSpacing(raw json.RawMessage) int {
	n := 2 // the padding just inside { }
	for _, c := range raw {
		if c == ',' || c == ':' {
			n++
		}
	}
	return n
}

// writeInline renders a container on a single line: { "a": 1, "b": 2 }. The
// padding inside the braces is written around the elements rather than with the
// delimiters, so an empty object comes out as {} and not { }.
func writeInline(buf *bytes.Buffer, raw json.RawMessage) error {
	return walk(raw,
		func(open byte) { buf.WriteByte(open) },
		func(i int, key *string, val json.RawMessage) error {
			switch {
			case i > 0:
				buf.WriteString(", ")
			case key != nil:
				buf.WriteByte(' ')
			}
			if key != nil {
				writeString(buf, *key)
				buf.WriteString(": ")
			}
			return writeInline(buf, bytes.TrimSpace(val))
		},
		func(close byte, n int) {
			if close == '}' && n > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteByte(close)
		},
		func(scalar json.RawMessage) { buf.Write(scalar) },
	)
}

// writeBlock renders a container across lines, recursing on each element.
func writeBlock(buf *bytes.Buffer, raw json.RawMessage, depth int) error {
	pad := strings.Repeat("  ", depth+1)
	return walk(raw,
		func(open byte) { buf.WriteByte(open) },
		func(i int, key *string, val json.RawMessage) error {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
			buf.WriteString(pad)
			if key != nil {
				writeString(buf, *key)
				buf.WriteString(": ")
			}
			return writeValue(buf, val, depth+1)
		},
		func(close byte, n int) {
			if n > 0 {
				buf.WriteByte('\n')
				buf.WriteString(strings.Repeat("  ", depth))
			}
			buf.WriteByte(close)
		},
		func(scalar json.RawMessage) { buf.Write(scalar) },
	)
}

// walk streams one container, calling elem for each member in file order. Keys
// come from json.Decoder.Token, which preserves the order they were encoded in;
// values are decoded as json.RawMessage so their bytes survive untouched.
func walk(
	raw json.RawMessage,
	open func(byte),
	elem func(i int, key *string, val json.RawMessage) error,
	closed func(byte, int),
	scalar func(json.RawMessage),
) error {
	if len(raw) == 0 {
		return fmt.Errorf("empty JSON value")
	}
	if raw[0] != '{' && raw[0] != '[' {
		scalar(raw)
		return nil
	}

	isObject := raw[0] == '{'
	openByte, closeByte := byte('['), byte(']')
	if isObject {
		openByte, closeByte = '{', '}'
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.Token(); err != nil { // consume the opening delimiter
		return err
	}

	open(openByte)
	i := 0
	for dec.More() {
		var key *string
		if isObject {
			tok, err := dec.Token()
			if err != nil {
				return err
			}
			k, ok := tok.(string)
			if !ok {
				return fmt.Errorf("object key is not a string: %v", tok)
			}
			key = &k
		}
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return err
		}
		if err := elem(i, key, val); err != nil {
			return err
		}
		i++
	}
	closed(closeByte, i)
	return nil
}

// writeString emits s as a JSON string. It goes through the encoder rather than
// quoting by hand so escaping matches the rest of the document exactly.
func writeString(buf *bytes.Buffer, s string) {
	b, err := json.Marshal(s)
	if err != nil {
		// json.Marshal of a string cannot fail; fall back to something valid
		// rather than propagating an error that can never happen.
		b = []byte(`""`)
	}
	buf.Write(b)
}
