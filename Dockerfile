# Use the official Golang image as a builder
FROM golang:1.24 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy the source code
COPY . .

# Build the binary
RUN go build -o linkytic-exporter main.go

# Use a minimal base image
FROM alpine:3.16

# Set the working directory
WORKDIR /root/

# Copy the binary from the builder
COPY --from=builder /app/linkytic-exporter .

# Expose the metrics port
EXPOSE 9100

# Run the exporter
CMD ["./linkytic-exporter"]