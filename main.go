package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	log "github.com/sirupsen/logrus" // Advanced logging with logrus
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type headerModifier struct {
	authv3.UnimplementedAuthorizationServer
}

func (h *headerModifier) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	// Log method entry
	log.Debug("Entered Check method")

	// Log the full request at trace level
	log.Tracef("Full CheckRequest: %+v", req)

	// Log request attributes
	log.Debugf("Request Attributes: %+v", req.Attributes)

	// Extract the "session" header
	sessionHeader := req.Attributes.Request.Http.Headers["session"]
	if sessionHeader == "" {
		log.Warn("Missing session header")
		// Log the response before returning
		resp := &authv3.CheckResponse{
			Status: status.New(codes.PermissionDenied, "Missing session header").Proto(),
		}
		log.Debugf("Returning CheckResponse: %+v", resp)
		return resp, nil
	}

	// Modify the session header
	newHeader := fmt.Sprintf("session-%s", sessionHeader)
	headers := []*corev3.HeaderValueOption{
		{
			Header: &corev3.HeaderValue{
				Key:   "x-new-header",
				Value: newHeader,
			},
		},
	}

	log.Debugf("Modified session header to: %s", newHeader)

	// Construct response
	response := &authv3.CheckResponse{
		Status: status.New(codes.OK, "").Proto(),
		HttpResponse: &authv3.CheckResponse_OkResponse{
			OkResponse: &authv3.OkHttpResponse{
				Headers: headers,
			},
		},
	}

	// Log the response before returning
	log.Debugf("Returning CheckResponse: %+v", response)

	return response, nil
}

func init() {
	// Set log level from environment variable or default to debug
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "debug" // Default to debug level
	}

	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatalf("Invalid LOG_LEVEL: %v", err)
	}
	log.SetLevel(level)

	// Format logs with full timestamps
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

func main() {
	// Listen on a TCP port for gRPC connections
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a new gRPC server and register the authorization service
	server := grpc.NewServer()
	authv3.RegisterAuthorizationServer(server, &headerModifier{})

	// Channel to listen for OS interrupt signals for graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start the gRPC server in a goroutine
	go func() {
		log.Println("Starting gRPC server on :50051")
		if err := server.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down gRPC server...")
	server.GracefulStop()
}
