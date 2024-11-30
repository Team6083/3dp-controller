package web

import (
	"context"
	"errors"
	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"time"
	"v400_monitor/docs"
	"v400_monitor/moonraker"
)

type Server struct {
	r      *gin.Engine
	logger *zap.SugaredLogger
	srv    *http.Server

	monitors map[string]*moonraker.Monitor

	ctx context.Context
}

func NewServer(ctx context.Context, isDevMode bool, logger *zap.SugaredLogger, monitors map[string]*moonraker.Monitor) *Server {
	var engine *gin.Engine

	if !isDevMode {
		gin.SetMode(gin.ReleaseMode)
		engine = gin.New()

		desugar := logger.Desugar()
		ginzapConfig := ginzap.Config{
			TimeFormat:   time.RFC3339,
			UTC:          true,
			DefaultLevel: zapcore.InfoLevel,
			Skipper: func(c *gin.Context) bool {
				if c.Writer.Status() < http.StatusBadRequest {
					return true
				}

				return false
			},
		}

		engine.Use(ginzap.GinzapWithConfig(desugar, &ginzapConfig))
		engine.Use(ginzap.RecoveryWithZap(desugar, true))
	} else {
		engine = gin.Default()
	}

	engine.Use(cors.Default())

	docs.SwaggerInfo.BasePath = "/api/v1"
	if isDevMode {
		engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	server := Server{
		r:        engine,
		logger:   logger,
		monitors: monitors,
		ctx:      ctx,
	}

	server.registerAPIRoutes(engine.Group(docs.SwaggerInfo.BasePath))

	feFS := http.FileServer(noListFileSystem{http.Dir("./frontend/dist")})
	engine.NoRoute(func(c *gin.Context) {
		feFS.ServeHTTP(c.Writer, c.Request)
	})

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
