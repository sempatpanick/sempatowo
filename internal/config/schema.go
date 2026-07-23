package config

import (
	_ "embed"
	"os"
	"path/filepath"
)

// SchemaFileName is where the JSON Schema is written, next to the config files
// that reference it. The "$schema" key in each config is this bare name — a
// sibling-relative path — rather than something reaching back into the repo:
// DATA_DIR can point anywhere, and a path out of the data root would break the
// moment someone moved it.
const SchemaFileName = "config.schema.json"

// schemaJSON is generated from the structs in config.go by
// `go generate ./internal/config`. schema_test.go fails if the two drift.
//
//go:embed config.schema.json
var schemaJSON []byte

// WriteSchemaFile puts the schema next to the config files, so editors can find
// it by the relative path each config names. It is rewritten on every start
// rather than only when missing: the settings it describes change with the
// binary, and a stale schema would report valid keys as typos.
func WriteSchemaFile(dir string) error {
	return os.WriteFile(filepath.Join(dir, SchemaFileName), schemaJSON, 0o644)
}
