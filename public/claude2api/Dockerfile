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
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./main.go  

# Create a minimal production image  
FROM alpine:latest  

# Create app directory and set permissions  
WORKDIR /app  
COPY --from=build /app/main .  

# Command to run the executable  
CMD ["./main"]  