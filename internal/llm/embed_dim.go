package llm

import "strings"

// EmbedDim returns the dimension for known embedding models.
// For unknown models, we infer at runtime by doing one embed call.
func EmbedDim(model string) int {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "text-embedding-3-small"):
		return 1536
	case strings.Contains(m, "text-embedding-3-large"):
		return 3072
	case strings.Contains(m, "text-embedding-ada-002"):
		return 1536
	case strings.Contains(m, "voyage-3-lite"):
		return 512
	case strings.Contains(m, "voyage-3"):
		return 1024
	case strings.Contains(m, "voyage-large-2"):
		return 1536
	case strings.Contains(m, "embed-english-v3"):
		return 1024
	case strings.Contains(m, "embed-multilingual-v3"):
		return 1024
	}
	return 0
}
