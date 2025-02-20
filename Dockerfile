# Use the official Golang image as a builder
FROM golang:1.24 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary
RUN go build -o linkytic-exporter main.go

# Use a minimal base image
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Set the working directory
WORKDIR /root/

# Copy the binary from the builder
COPY --from=builder /app/linkytic-exporter .

# Set environment variables
ENV LINKY_TIC_DEVICE="/dev/ttyACM0"
ENV LINKY_TIC_MODE="HISTORICAL"

# Expose the metrics port
EXPOSE 9100

# Run the exporter
CMD ["./linkytic-exporter"]