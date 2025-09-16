package handler

import (
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/zgsm-ai/oidc-auth/internal/middleware"
	"github.com/zgsm-ai/oidc-auth/pkg/log"
)

type Server struct {
	ServerPort        string
	BaseURL           string
	HTTPClient        *http.Client
	IsPrivate         bool
	InviteCodeHandler *InviteCodeHandler
}

type ParameterCarrier struct {
	TokenHash     string `json:"token_hash"`
	Provider      string `json:"provider"`
	Platform      string `json:"platform"`
	State         string `form:"state" binding:"required"`
	MachineCode   string `form:"machine_code"`
	UriScheme     string `form:"uri_scheme"`
	PluginVersion string `form:"plugin_version"`
	VscodeVersion string `form:"vscode_version"`
	InviteCode    string `form:"invite_code"`
}

func (s *Server) SetupRouter(r *gin.Engine) {
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.RequestLogger())
	
	// 注入邀请码处理器中间件
	if s.InviteCodeHandler != nil {
		r.Use(middleware.InjectInviteCodeHandler(s.InviteCodeHandler))
	}

	pluginOauthServer := r.Group("/oidc-auth/api/v1/plugin",
		middleware.SetPlatform("plugin"),
	)
	{
		pluginOauthServer.GET("login", s.loginHandler)
		pluginOauthServer.GET("login/callback", s.callbackHandler)
		pluginOauthServer.GET("login/token", tokenHandler)
		pluginOauthServer.GET("login/logout", logoutHandler)
		pluginOauthServer.GET("login/status", statusHandler)
	}
	webOauthServer := r.Group("/oidc-auth/api/v1/manager",
		middleware.SetPlatform("web"),
	)
	{
		webOauthServer.GET("token", getTokenByHash)
		webOauthServer.GET("bind/account", s.bindAccount)
		webOauthServer.GET("bind/account/callback", s.bindAccountCallback)
		webOauthServer.GET("userinfo", s.userInfoHandler)
	}
	
	// 邀请码相关API
	inviteCodeAPI := r.Group("/oidc-auth/api/v1/invite")
	{
		inviteCodeAPI.POST("generate", s.generateInviteCodeHandler)
		inviteCodeAPI.GET("list", s.getInviteCodesHandler)
	}
	
	r.POST("/oidc-auth/api/v1/send/sms", s.SMSHandler)
	health := r.Group("/health")
	{
		health.GET("ready", readinessHandler)
	}
}

func (s *Server) StartServer() error {
	r := gin.Default()
	s.SetupRouter(r)

	port := ":" + s.ServerPort
	log.Info(nil, "Starting server on port %s", port)

	if err := r.Run(port); err != nil {
		log.Error(nil, "Server failed to start: %v", err)
		return err
	}
	return nil
}
