package content

// Canonical media library paths (relative to each user's private Home).
// Clients MUST use these exact relative paths (no leading slash, no homes/{id} prefix).
// Physical layout: {FILES_STORAGE_ROOT}/homes/{userId}/Fotos/iPhone
const (
	MediaLibraryFolder = "Fotos"
	MediaLibraryRel    = "Fotos/iPhone"
)

// NormalizeClientRelPath strips leading/trailing slashes for storage APIs.
func NormalizeClientRelPath(p string) string {
	for len(p) > 0 && (p[0] == '/' || p[0] == '\\') {
		p = p[1:]
	}
	for len(p) > 0 && (p[len(p)-1] == '/' || p[len(p)-1] == '\\') {
		p = p[:len(p)-1]
	}
	return p
}
