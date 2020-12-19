package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	envoy_service_auth_v2 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v2"
	envoy_service_auth_v3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"google.golang.org/grpc"
)

func main() {
	shouldExit := make(chan os.Signal, 1)
	signal.Notify(shouldExit, syscall.SIGINT, syscall.SIGTERM)

	var c Config
	err := envconfig.Process("forwardeaz_service", &c)
	if err != nil {
		panic(errors.Wrap(err, "failed to process configuration"))
	}

	if c.Version == "" {
		c.Version = "unknown"
	}

	logConfig := zap.NewProductionConfig()
	err = logConfig.Level.UnmarshalText([]byte(c.LogLevel))
	if err != nil {
		panic(errors.Wrapf(err, "failed to determine log-level: %s", c.LogLevel))
	}
	commonLogFields := zap.Fields(
		zap.String("version", c.Version),
	)
	logger, err := logConfig.Build(commonLogFields)
	if err != nil {
		panic(errors.Wrap(err, "failed to set-up logging"))
	}

	grpc_zap.ReplaceGrpcLoggerV2(logger)

	listener, err := net.Listen("tcp", c.Address)
	if err != nil {
		logger.Panic("Failed to bind service to address", zap.Error(err), zap.String("address", c.Address))
	}

	var opts []grpc_zap.Option
	server := grpc.NewServer(
		grpc_middleware.WithUnaryServerChain(
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(logger, opts...),
		),
		grpc_middleware.WithStreamServerChain(
			grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.StreamServerInterceptor(logger, opts...),
		),
	)

	checker := &forwardAuthChecker{
		logger:              logger,
		forwardAuthAddress:  c.ForwardAuthAddress,
		authRequestHeaders:  c.AuthRequestHeaders,
		authResponseHeaders: c.AuthResponseHeaders,
		trustForwardHeader:  c.TrustForwardHeader,
	}

	// DEPRECATED: included to support Contour <= v1.10.0
	v2 := &authV2{
		logger:  logger,
		checker: checker,
	}
	envoy_service_auth_v2.RegisterAuthorizationServer(server, v2)

	v3 := &authV3{
		logger:  logger,
		checker: checker,
	}
	envoy_service_auth_v3.RegisterAuthorizationServer(server, v3)

	go func() {
		logger.Info("Starting service", zap.Any("config", c))
		err := server.Serve(listener)
		if err != nil {
			logger.Panic("Failed to start service", zap.Error(err))
		}
	}()

	<-shouldExit
	logger.Info("Stopping service")
	server.GracefulStop()
	_ = logger.Sync()
}
