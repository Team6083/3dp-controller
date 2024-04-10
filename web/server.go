package web

import (
	"context"
	"errors"
	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"time"
	"v400_monitor/docs"
	"v400_monitor/moonraker"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	r      *gin.Engine
	logger *zap.SugaredLogger
	srv    *http.Server

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

	engine.Static("/ui", "./frontend/out")

	return &server
}

func (s *Server) Run() {
	// TODO: addr
	s.srv = &http.Server{
		Addr:    ":8080",
		Handler: s.r,
	}

	if err := s.srv.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		s.logger.Fatalf("listen: %s\n", err)
	}
}

func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		s.logger.Fatalf("Server Shutdown: %s\n", err)
	}
	s.logger.Infoln("Server exiting")
}
