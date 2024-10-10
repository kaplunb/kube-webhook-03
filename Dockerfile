# Build stage
FROM golang:1.23.2 AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY src/ .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o validate-label .

# Final stage
FROM alpine:latest  

WORKDIR /root/

# Copy the pre-built binary file from the previous stage
COPY --from=builder /app/validate-label .

RUN mkdir -p /etc/webhook/tls

# Command to run the executable
CMD ["./validate-label"]