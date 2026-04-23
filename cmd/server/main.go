package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/uncle3dev/velotrax-core-go/internal/config"
	"github.com/uncle3dev/velotrax-core-go/internal/db"
	orderpb "github.com/uncle3dev/velotrax-core-go/internal/gen/order"
	orderService "github.com/uncle3dev/velotrax-core-go/internal/service/order"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger, err := initLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting velotrax-core-go",
		zap.String("env", cfg.AppEnv),
		zap.Int("grpc_port", cfg.GRPCPort),
	)

	mongoDB, err := db.Connect(context.Background(), cfg.MongoURI)
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoDB.Disconnect(shutdownCtx); err != nil {
			logger.Error("MongoDB disconnect error", zap.Error(err))
		}
	}()

	if err := db.EnsureIndexes(context.Background(), mongoDB.Database); err != nil {
		logger.Fatal("Failed to ensure MongoDB indexes", zap.Error(err))
	}

	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		logger.Fatal("Failed to listen on gRPC port", zap.Error(err))
	}

	grpcSrv := grpc.NewServer()
	orderpb.RegisterOrderServiceServer(grpcSrv, orderService.NewService(logger, cfg.JWTSecret))

	go func() {
		logger.Info("gRPC server listening", zap.String("addr", grpcListener.Addr().String()))
		if err := grpcSrv.Serve(grpcListener); err != nil {
			logger.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	grpcSrv.GracefulStop()

	logger.Info("Server stopped")
}

func initLogger(level string) (*zap.Logger, error) {
	var cfg zap.Config
	if level == "debug" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	cfg.Level = zap.NewAtomicLevelAt(parseLogLevel(level))
	return cfg.Build()
}

func parseLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}
