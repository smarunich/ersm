# Envoy External Authorization Server (ERSM)

This project implements an Envoy External Authorization Server using Go. It provides a simple gRPC service that can be used with Envoy's external authorization filter to manage session-based header modifications.

## Project Structure

- `main.go`: Contains the main application logic including the gRPC server and HTTP server setup.
- `Dockerfile`: Defines the steps to containerize the application.
- `Taskfile.yaml`: Specifies tasks for building, running, and managing the application and Docker images.
- `deployment.yaml`: Kubernetes deployment and service configuration for orchestrating the application in a cluster.

## Prerequisites

- Go 1.21 or later: Required for building and running the Go application.
- Docker: Needed for creating Docker images and running containers.
- Kubernetes: Necessary for deploying the application to a Kubernetes cluster.
- Task: A task runner that simplifies and automates development workflows.

