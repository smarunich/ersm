package main

import (
	"context"
	"fmt"
	"net/http"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type headerModifier struct{}

func (h *headerModifier) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	sessionHeader := req.Attributes.Request.Http.Headers["session"]
	if sessionHeader == "" {
		return &authv3.CheckResponse{
			Status: status.New(codes.PermissionDenied, "Missing session header").Proto(),
		}, nil
	}

	newHeader := fmt.Sprintf("session-%s", sessionHeader)
	headers := []*core.HeaderValueOption{
		{
			Header: &core.HeaderValue{
				Key:   "x-new-header",
				Value: newHeader,
			},
		},
	}

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
	server := grpc.NewServer()
	authv3.RegisterAuthorizationServer(server, &headerModifier{})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Envoy External Authorization Server"))
	})

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
