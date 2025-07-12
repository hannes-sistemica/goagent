package api

import (
	"log/slog"
	"os"

	"agent-server/internal/api/handlers"
	"agent-server/internal/api/middleware"
	"agent-server/internal/config"
	contextpkg "agent-server/internal/context"
	"agent-server/internal/llm"
	"agent-server/internal/services"
	"agent-server/internal/storage"

	"github.com/gin-gonic/gin"
)

// Server represents the HTTP server
type Server struct {
	router            *gin.Engine
	config            *config.Config
	repo              storage.Repository
	ctxRegistry       *contextpkg.StrategyRegistry
	llmRegistry       *llm.Registry
	toolService     *services.ToolService
	chatService     *services.ChatService
	logger          *slog.Logger
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config, repo storage.Repository, ctxRegistry *contextpkg.StrategyRegistry, llmRegistry *llm.Registry) *Server {
	// Set Gin mode based on config
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	
	// Initialize tool service
	toolService := services.NewToolService(repo, logger)
	
	// Initialize prompt service
	promptService := services.NewPromptService(toolService)
	
	// Initialize unified chat service with tool support
	chatService := services.NewChatService(repo, llmRegistry, ctxRegistry, toolService, promptService, logger)
	
	return &Server{
		router:      router,
		config:      cfg,
		repo:        repo,
		ctxRegistry: ctxRegistry,
		llmRegistry: llmRegistry,
		toolService: toolService,
		chatService: chatService,
		logger:      logger,
	}
}

// SetupRoutes configures all routes and middleware
func (s *Server) SetupRoutes() {
	// Global middleware
	s.router.Use(middleware.Logger())
	s.router.Use(middleware.Recovery())
	s.router.Use(middleware.CORS())

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Tool routes
		toolHandler := handlers.NewToolsHandler(s.toolService)
		tools := v1.Group("/tools")
		{
			tools.GET("", toolHandler.ListTools)
			tools.GET("/:tool_name", toolHandler.GetTool)
			tools.POST("/:tool_name/test", toolHandler.TestTool)
			tools.POST("/:tool_name/execute", toolHandler.ExecuteTool)
			tools.GET("/schemas", toolHandler.GetToolSchemas)
			tools.GET("/stats", toolHandler.GetToolUsageStats)
		}

		// Agent routes
		agentHandler := handlers.NewAgentHandler(s.repo.Agent())
		agents := v1.Group("/agents")
		{
			agents.POST("", agentHandler.Create)
			agents.GET("", agentHandler.List)
			agents.GET("/:id", agentHandler.GetByID)
			agents.PUT("/:id", agentHandler.Update)
			agents.DELETE("/:id", agentHandler.Delete)

			// Session routes under agents
			sessionHandler := handlers.NewSessionHandler(s.repo.Session(), s.repo.Agent())
			agents.POST("/:id/sessions", sessionHandler.Create)
			agents.GET("/:id/sessions", sessionHandler.ListByAgent)
		}

		// Session routes
		sessionHandler := handlers.NewSessionHandler(s.repo.Session(), s.repo.Agent())
		sessions := v1.Group("/sessions")
		{
			sessions.GET("/:id", sessionHandler.GetByID)
			sessions.PUT("/:id", sessionHandler.Update)
			sessions.DELETE("/:id", sessionHandler.Delete)

			// Message routes under sessions
			messageHandler := handlers.NewMessageHandler(s.repo.Message())
			sessions.POST("/:id/messages", messageHandler.Create)
			sessions.GET("/:id/messages", messageHandler.ListBySession)
			sessions.DELETE("/:id/messages", messageHandler.DeleteBySession)

			// Chat routes with tool calling support
			chatHandler := handlers.NewChatHandler(s.chatService, s.toolService, s.logger)
			sessions.POST("/:id/chat", chatHandler.Chat)
			sessions.POST("/:id/stream", chatHandler.Stream)
			sessions.POST("/:id/chat/tools", chatHandler.ChatWithTools)
			sessions.POST("/:id/chat/auto-tools", chatHandler.ChatWithAutoTools)
			
			// Tool-related routes for sessions
			sessions.GET("/:id/tools", chatHandler.ListAvailableTools)
			sessions.GET("/:id/tools/:tool_name/schema", chatHandler.GetToolSchema)
			sessions.POST("/:id/tools/:tool_name/test", chatHandler.TestToolForSession)
			sessions.GET("/:id/tool-calls", chatHandler.GetToolCallHistory)
		}
	}
}

// GetRouter returns the Gin router
func (s *Server) GetRouter() *gin.Engine {
	return s.router
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.router.Run(s.config.GetAddress())
}