package batch

import "fmt"

// NewBatchClient creates the right BatchClient based on the provider name.
// Supported providers: "anthropic", "openai".
func NewBatchClient(provider, apiKey string) (BatchClient, error) {
	switch provider {
	case "anthropic":
		return NewAnthropicBatchClient(apiKey), nil
	case "openai":
		return NewOpenAIBatchClient(apiKey), nil
	default:
		return nil, fmt.Errorf("batch not supported for provider %q (supported: anthropic, openai)", provider)
	}
}
