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
	log "github.com/sirupsen/logrus" // Import logrus for advanced logging
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type headerModifier struct {
	authv3.UnimplementedAuthorizationServer
}

func (h *headerModifier) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	// Log the detailed request at debug level
	log.Debugf("Received CheckRequest: %+v", req)

	// Extract the "session" header from the incoming request
	sessionHeader := req.Attributes.Request.Http.Headers["session"]
	if sessionHeader == "" {
		log.Warn("Missing session header")
		return &authv3.CheckResponse{
			Status: status.New(codes.PermissionDenied, "Missing session header").Proto(),
		}, nil
	}

	// Modify the session header and create a new header
	newHeader := fmt.Sprintf("session-%s", sessionHeader)
	headers := []*corev3.HeaderValueOption{
		{
			Header: &corev3.HeaderValue{
				Key:   "x-new-header",
				Value: newHeader,
			},
		},
	}

	log.Debugf("Session header modified to: %s", newHeader)
	return &authv3.CheckResponse{
		Status: status.New(codes.OK, "").Proto(),
		HttpResponse: &authv3.CheckResponse_OkResponse{
			OkResponse: &authv3.OkHttpResponse{
				Headers: headers,
			},
		},
	}, nil
}

func main() {
	// Set log level to debug for detailed output
	log.SetLevel(log.DebugLevel)
	// Optionally, format logs with timestamps
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

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
