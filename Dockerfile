# Start with a base image containing the Go runtime
FROM golang:alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container
COPY cmd ./cmd
COPY pkg ./pkg

# Build the Go server
RUN go build -o server ./cmd

# Start a new stage with a minimal base image
FROM alpine:latest

WORKDIR /app
COPY --from=0 /app/server ./server

# Install ffmpeg
RUN apk update
RUN apk upgrade
RUN apk add --no-cache ffmpeg

# Expose the port that the server listens on
EXPOSE 8080

# Set the entry point for the container
CMD ["./server"]