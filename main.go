package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	extprocv3 "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	log "github.com/sirupsen/logrus" // Advanced logging with logrus
	"google.golang.org/grpc"
)

type externalProcessorServer struct {
	extprocv3.UnimplementedExternalProcessorServer
}

func (s *externalProcessorServer) Process(stream extprocv3.ExternalProcessor_ProcessServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			log.Errorf("Error receiving stream: %v", err)
			return err
		}

		// Log the full request when debug mode is on
		if log.IsLevelEnabled(log.DebugLevel) {
			reqString := fmt.Sprintf("%+v", req)
			log.Debugf("Received ProcessingRequest:\n%s", reqString)
		}

		switch v := req.Request.(type) {
		case *extprocv3.ProcessingRequest_RequestHeaders:
			log.Debug("Processing RequestHeaders")

			// Extract the session header
			sessionValue := getSessionHeader(v.RequestHeaders.Headers.GetHeaders())
			if sessionValue == "" {
				log.Warn("Missing session header")
				// Optionally, you can send an immediate response to stop processing
				// For now, let's continue without modifying headers
			}

			// Modify the session header and create a new header
			newHeaderValue := fmt.Sprintf("session-%s", sessionValue)

			resp := &extprocv3.ProcessingResponse{
				Response: &extprocv3.ProcessingResponse_RequestHeaders{
					RequestHeaders: &extprocv3.HeadersResponse{
						Response: &extprocv3.CommonResponse{
							HeaderMutation: &extprocv3.HeaderMutation{
								SetHeaders: []*corev3.HeaderValueOption{
									{
										Header: &corev3.HeaderValue{
											Key:   "x-new-header",
											Value: newHeaderValue,
										},
									},
								},
							},
						},
					},
				},
			}

			// Log the full response when debug mode is on
			if log.IsLevelEnabled(log.DebugLevel) {
				respString := fmt.Sprintf("%+v", resp)
				log.Debugf("Sending ProcessingResponse:\n%s", respString)
			}

			if err := stream.Send(resp); err != nil {
				log.Errorf("Error sending response: %v", err)
				return err
			}

		case *extprocv3.ProcessingRequest_RequestBody:
			log.Debug("Processing RequestBody")
			// Handle request body if needed

		case *extprocv3.ProcessingRequest_ResponseHeaders:
			log.Debug("Processing ResponseHeaders")
			// Handle response headers if needed

		case *extprocv3.ProcessingRequest_ResponseBody:
			log.Debug("Processing ResponseBody")
			// Handle response body if needed

		default:
			log.Warnf("Received unknown request type: %T", v)
		}
	}
}

func getSessionHeader(headers []*corev3.HeaderValue) string {
	for _, headerValue := range headers {
		if headerValue.GetKey() == "session" {
			return headerValue.GetValue()
		}
	}
	return ""
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

	// Create a new gRPC server and register the external processor service
	server := grpc.NewServer()
	extprocv3.RegisterExternalProcessorServer(server, &externalProcessorServer{})

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
