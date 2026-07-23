package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/semptpanick/sempatowo/internal/config/schemagen"
)

// The schema is generated, checked in, and embedded. Nothing forces a
// regeneration when a setting is added, so this test is what notices.
func TestSchemaFileIsUpToDate(t *testing.T) {
	// Tests run with the package directory as the working directory, which is
	// the same place the //go:generate line runs from.
	want, err := schemagen.Generate(".")
	if err != nil {
		t.Fatalf("schemagen.Generate: %v", err)
	}
	if string(schemaJSON) != string(want) {
		t.Error("config.schema.json is stale — run `go generate ./internal/config`")
	}
}

func TestSchemaDescribesTheCurrentShape(t *testing.T) {
	var doc struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(schemaJSON, &doc); err != nil {
		t.Fatalf("the embedded schema is not valid JSON: %v", err)
	}

	for _, key := range []string{"$schema", "schemaVersion", "label", "trackBalance", "discord", "humanize", "features"} {
		if _, ok := doc.Properties[key]; !ok {
			t.Errorf("schema has no property for %q", key)
		}
	}
	// The keys that moved in v2 must not still be described at the top level.
	for _, gone := range []string{"prefix", "owoBotId", "defaultChannel", "typing", "sendMessageInterval", "stopWhenChecklistDone"} {
		if _, ok := doc.Properties[gone]; ok {
			t.Errorf("schema still lists %q at the top level", gone)
		}
	}
}

// The schema sets "additionalProperties": false so that editors underline typos.
// That cuts both ways: a key the generator failed to describe would be flagged in
// a file the bot itself wrote. This walks the default config against the schema
// and fails on any key the schema does not know about.
func TestSchemaAcceptsTheConfigWeWrite(t *testing.T) {
	data, err := marshalReadable(Defaults())
	if err != nil {
		t.Fatalf("marshalReadable: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var schema map[string]any
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	checkAgainstSchema(t, "", doc, schema)
}

func checkAgainstSchema(t *testing.T, path string, value any, schema map[string]any) {
	t.Helper()

	obj, ok := value.(map[string]any)
	if !ok {
		return // scalars are covered by "type", which the editor checks
	}

	// A Range is written either way, so follow whichever branch is an object.
	if branches, ok := schema["oneOf"].([]any); ok {
		for _, b := range branches {
			if bs, ok := b.(map[string]any); ok {
				if _, hasProps := bs["properties"]; hasProps {
					schema = bs
					break
				}
			}
		}
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Errorf("%s: schema describes no properties for an object", path)
		return
	}
	for key, child := range obj {
		sub, ok := props[key].(map[string]any)
		if !ok {
			t.Errorf("%s: schema has no property %q, so an editor would flag it as a typo", path, key)
			continue
		}
		checkAgainstSchema(t, path+"/"+key, child, sub)
	}
}

// -check-config runs EnsureDirs and then exits without ever loading an account,
// so the schema has to be written here rather than when a config is opened.
func TestEnsureDirsWritesTheSchema(t *testing.T) {
	root := t.TempDir()
	e := &Env{Dirs: Dirs{
		Root:            root,
		Config:          filepath.Join(root, "config"),
		Data:            filepath.Join(root, "data"),
		BrowserProfiles: filepath.Join(root, "browser-profiles"),
	}}
	if err := e.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	if !fileExists(filepath.Join(e.Dirs.Config, SchemaFileName)) {
		t.Errorf("EnsureDirs did not write %s", SchemaFileName)
	}
}

func TestWriteSchemaFile(t *testing.T) {
	dir := t.TempDir()
	if err := WriteSchemaFile(dir); err != nil {
		t.Fatalf("WriteSchemaFile: %v", err)
	}

	// The "$schema" every config names is this bare filename, so the file has to
	// land as a sibling of the configs for an editor to resolve it.
	got, err := os.ReadFile(filepath.Join(dir, Defaults().Schema))
	if err != nil {
		t.Fatalf("reading back the schema by the path configs point at: %v", err)
	}
	if string(got) != string(schemaJSON) {
		t.Error("written schema differs from the embedded one")
	}
}
