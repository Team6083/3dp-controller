package web

import (
	"context"
	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"time"
	"v400_monitor/docs"
	"v400_monitor/moonraker"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	r      *gin.Engine
	logger *zap.SugaredLogger

	monitors map[string]*moonraker.Monitor

	ctx context.Context
}

func NewServer(ctx context.Context, logger *zap.SugaredLogger, monitors map[string]*moonraker.Monitor) *Server {
	engine := gin.New()

	desugar := logger.Desugar()
	engine.Use(ginzap.Ginzap(desugar, time.RFC3339, true))
	engine.Use(ginzap.RecoveryWithZap(desugar, true))
	engine.Use(cors.Default())

	docs.SwaggerInfo.BasePath = "/api/v1"
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	server := Server{
		r:        engine,
		logger:   logger,
		monitors: monitors,
		ctx:      ctx,
	}

	server.registerAPIRoutes(engine.Group(docs.SwaggerInfo.BasePath))

	return &server
}

func (s *Server) Run() {
	// TODO: addr
	err := s.r.Run(":8080")
	if err != nil {
		panic(err)
	}
}
