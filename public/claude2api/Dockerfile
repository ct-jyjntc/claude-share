# Start from the official Golang image
FROM golang:1.23-alpine AS build

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files first for better caching
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o claude2api ./main.go

# Create a minimal production image
FROM alpine:latest

# Create app directory and set permissions
WORKDIR /app
COPY --from=build /app/claude2api .

# Create data directory for sessionKeys.json
RUN mkdir -p /app/data

# Command to run the executable
CMD ["./claude2api"]