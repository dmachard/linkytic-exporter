# Use the official Golang image as a builder
FROM golang:1.25 as builder

# Set the working directory inside the container
WORKDIR /build
COPY . .

# Build the binary
RUN CGO_ENABLED=0 go build -o linkytic-exporter main.go



# Use a minimal base image
FROM alpine:3.22.1

# Copy the binary from the builder
WORKDIR /app
COPY --from=builder /build/linkytic-exporter /app/linkytic-exporter
RUN chmod +x /app/linkytic-exporter

# Expose the metrics port
EXPOSE 9100/tcp

# Run the exporter
ENTRYPOINT ["/app/linkytic-exporter"]
