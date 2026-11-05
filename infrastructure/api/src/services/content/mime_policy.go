package content

// =============================================================================
// MIME POLICY SERVICE
// =============================================================================

// Single Responsibility: Determine if a file is eligible for AI indexing

// aiIndexableMimeTypes defines which file types can be processed by the AI
var aiIndexableMimeTypes = map[string]bool{
	"text/plain":      true,
	"application/pdf": true,
	"text/markdown":   true,
	"text/csv":        true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/msword": true,
}

// MimePolicy determines if files should be indexed by AI
type MimePolicy struct{}

// NewMimePolicy creates a new MimePolicy instance
func NewMimePolicy() *MimePolicy {
	return &MimePolicy{}
}

// IsIndexable checks if a MIME type is eligible for AI indexing
func (p *MimePolicy) IsIndexable(mimeType string) bool {
	return aiIndexableMimeTypes[mimeType]
}

// IsIndexableStatic is a convenience function for stateless checks
func IsIndexable(mimeType string) bool {
	return aiIndexableMimeTypes[mimeType]
}

// GetIndexableMimeTypes returns a copy of all indexable MIME types
func (p *MimePolicy) GetIndexableMimeTypes() []string {
	result := make([]string, 0, len(aiIndexableMimeTypes))
	for mimeType := range aiIndexableMimeTypes {
		result = append(result, mimeType)
	}
	return result
}
