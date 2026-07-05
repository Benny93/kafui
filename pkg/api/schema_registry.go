package api

// SchemaVersion is the metadata for a single registered version of a subject.
// Schema holds the full definition text; it may be empty when only metadata was
// requested (fetch the text lazily via GetSchemaContent).
type SchemaVersion struct {
	Version    int    `json:"version"`
	ID         int    `json:"id"`
	SchemaType string `json:"schemaType"` // AVRO, PROTOBUF, JSON — empty means AVRO
	Schema     string `json:"schema"`
}

// CompatibilityLevel is a schema-registry compatibility setting. The zero value
// is invalid; use one of the constants below.
type CompatibilityLevel string

const (
	CompatibilityBackward           CompatibilityLevel = "BACKWARD"
	CompatibilityBackwardTransitive CompatibilityLevel = "BACKWARD_TRANSITIVE"
	CompatibilityForward            CompatibilityLevel = "FORWARD"
	CompatibilityForwardTransitive  CompatibilityLevel = "FORWARD_TRANSITIVE"
	CompatibilityFull               CompatibilityLevel = "FULL"
	CompatibilityFullTransitive     CompatibilityLevel = "FULL_TRANSITIVE"
	CompatibilityNone               CompatibilityLevel = "NONE"
)

// CompatibilityLevels lists every valid compatibility level in the order a UI
// selector should present them.
func CompatibilityLevels() []CompatibilityLevel {
	return []CompatibilityLevel{
		CompatibilityBackward,
		CompatibilityBackwardTransitive,
		CompatibilityForward,
		CompatibilityForwardTransitive,
		CompatibilityFull,
		CompatibilityFullTransitive,
		CompatibilityNone,
	}
}

// Valid reports whether l is one of the seven defined compatibility levels.
func (l CompatibilityLevel) Valid() bool {
	switch l {
	case CompatibilityBackward, CompatibilityBackwardTransitive,
		CompatibilityForward, CompatibilityForwardTransitive,
		CompatibilityFull, CompatibilityFullTransitive, CompatibilityNone:
		return true
	default:
		return false
	}
}
