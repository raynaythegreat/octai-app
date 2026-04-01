package files

import "fmt"

// NewFileStore creates the appropriate FileStore based on the provider name.
//
//   - "anthropic" → AnthropicFileStore (Anthropic Files API)
//   - "openai"    → OpenAIFileStore (OpenAI Files API)
//   - "local" or any unsupported provider → LocalFileStore stored at localBaseDir
func NewFileStore(provider, apiKey, localBaseDir string) (FileStore, error) {
	switch provider {
	case "anthropic":
		return NewAnthropicFileStore(apiKey), nil
	case "openai":
		return NewOpenAIFileStore(apiKey), nil
	default:
		store, err := NewLocalFileStore(localBaseDir)
		if err != nil {
			return nil, fmt.Errorf("files/factory: create local store: %w", err)
		}
		return store, nil
	}
}
