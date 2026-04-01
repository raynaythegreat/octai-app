package providers

import (
	"github.com/raynaythegreat/octai-app/pkg/auth"
)

var getCredential = auth.GetCredential
var setCredential = auth.SetCredential
var anthropicOAuthConfig = auth.AnthropicOAuthConfig
var refreshAccessToken = auth.RefreshAccessToken
