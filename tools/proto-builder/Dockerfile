FROM golang:1.25-alpine

RUN apk add --no-cache protobuf-dev git nodejs npm

# Install Go plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Install TS plugins
RUN npm install -g ts-proto

WORKDIR /workspace
