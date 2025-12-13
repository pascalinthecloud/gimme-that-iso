# ---- Builder Stage ----
FROM golang:1.22-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code
COPY . .

# Build the Go app
# -ldflags="-w -s" reduces the size of the binary by removing debug information.
# CGO_ENABLED=0 creates a statically linked binary.
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /gimme-that-iso ./cmd/gimme-that-iso

# ---- Final Stage ----
FROM alpine:latest

# Install gnupg for GPG verification
RUN apk --no-cache add gnupg

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /gimme-that-iso /gimme-that-iso

# Set the entrypoint
ENTRYPOINT ["/gimme-that-iso"]

# Default command can be overridden
CMD ["--help"]
