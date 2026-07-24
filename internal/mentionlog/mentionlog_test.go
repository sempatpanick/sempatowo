package mentionlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendBuildsValidJSONArray(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, "123")

	if err := s.Append(Record{Event: "create", ChannelID: "c1", MessageID: "m1", Content: "hello"}); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := s.Append(Record{Event: "update", ChannelID: "c1", MessageID: "m1", Content: "hi", Embeds: []Embed{{Title: "t", Description: "d"}}}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "123_mentions.log"))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}

	// The whole file must parse as a JSON array with no massaging.
	var recs []Record
	if err := json.Unmarshal(data, &recs); err != nil {
		t.Fatalf("file is not a valid JSON array: %v\n%s", err, data)
	}

	if len(recs) != 2 {
		t.Fatalf("want 2 records, got %d", len(recs))
	}
	if recs[0].Content != "hello" || recs[0].Event != "create" {
		t.Errorf("record 0 = %+v", recs[0])
	}
	if recs[1].Time.IsZero() {
		t.Error("Append should stamp Time when zero")
	}
	if len(recs[1].Embeds) != 1 || recs[1].Embeds[0].Title != "t" {
		t.Errorf("record 1 embeds = %+v", recs[1].Embeds)
	}
}
