package api

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/raynaythegreat/octai-app/pkg/agent"
	"github.com/raynaythegreat/octai-app/pkg/marketplace"
	"github.com/raynaythegreat/octai-app/pkg/tenant"
	"github.com/raynaythegreat/octai-app/web/backend/launcherconfig"
)

var postInitFuncs []func(h *Handler)

type Handler struct {
	configPath           string
	serverPort           int
	serverPublic         bool
	serverPublicExplicit bool
	serverCIDRs          []string
	oauthMu              sync.Mutex
	oauthFlows           map[string]*oauthFlow
	oauthState           map[string]string
	weixinMu             sync.Mutex
	weixinFlows          map[string]*weixinFlow
	wecomMu              sync.Mutex
	wecomFlows           map[string]*wecomFlow
	tenantStore          tenant.TenantStore
	marketplaceStore     marketplace.MarketplaceStore
	analyticsCache       map[string]interface{}
	membershipIDs        map[string]string
	auditCache           map[string]interface{}
	complianceHandler    *ComplianceHandler
	ssoMu                sync.Mutex
	ssoFlows             map[string]*ssoFlow
	ssoStates            map[string]string
	loopSched            *agent.LoopScheduler
	loopSchedOnce        sync.Once
}

func NewHandler(configPath string) *Handler {
	h := &Handler{
		configPath:     configPath,
		serverPort:     launcherconfig.DefaultPort,
		oauthFlows:     make(map[string]*oauthFlow),
		oauthState:     make(map[string]string),
		weixinFlows:    make(map[string]*weixinFlow),
		wecomFlows:     make(map[string]*wecomFlow),
		analyticsCache: make(map[string]interface{}),
		membershipIDs:  make(map[string]string),
		auditCache:     make(map[string]interface{}),
		ssoFlows:       make(map[string]*ssoFlow),
		ssoStates:      make(map[string]string),
	}
	for _, fn := range postInitFuncs {
		fn(h)
	}
	return h
}

func (h *Handler) SetServerOptions(port int, public bool, publicExplicit bool, allowedCIDRs []string) {
	h.serverPort = port
	h.serverPublic = public
	h.serverPublicExplicit = publicExplicit
	h.serverCIDRs = append([]string(nil), allowedCIDRs...)
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	h.registerConfigRoutes(mux)
	h.registerPicoRoutes(mux)
	h.registerGatewayRoutes(mux)
	h.registerSessionRoutes(mux)
	h.registerOAuthRoutes(mux)
	h.registerModelRoutes(mux)
	h.registerChannelRoutes(mux)
	h.registerSkillRoutes(mux)
	h.registerToolRoutes(mux)
	h.registerStartupRoutes(mux)
	h.registerLauncherConfigRoutes(mux)
	h.registerWeixinRoutes(mux)
	h.registerWecomRoutes(mux)
	h.registerOrganizationRoutes(mux)
	h.registerMembershipRoutes(mux)
	h.registerSubscriptionRoutes(mux)
	h.registerAnalyticsRoutes(mux)
	h.registerMarketplaceRoutes(mux)
	h.registerTeamRoutes(mux)
	h.registerWorkflowRoutes(mux)
	h.registerLoopRoutes(mux)
	h.registerScannerRoutes(mux)
	h.registerReferenceURLRoutes(mux)
	h.registerPluginRoutes(mux)
	h.registerImageModelRoutes(mux)
	h.registerVideoModelRoutes(mux)
	h.registerMediaRoutes(mux)
	h.registerImageGenRoutes(mux)
	h.registerTailscaleRoutes(mux)
	h.registerMobileRoutes(mux)
}

func (h *Handler) Shutdown() {
	h.StopGateway()
}

const maxRequestBodySize = 1 << 20

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, maxRequestBodySize)).Decode(v)
}
