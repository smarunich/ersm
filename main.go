package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type headerModifier struct {
	authv3.UnimplementedAuthorizationServer
}

func (h *headerModifier) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	// Extract the "session" header from the incoming request
	sessionHeader := req.Attributes.Request.Http.Headers["session"]
	if sessionHeader == "" {
		log.Println("Missing session header")
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

	log.Printf("Session header modified: %s", newHeader)
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
