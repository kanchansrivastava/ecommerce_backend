# micro-1/Dockerfile

# Stage 1: Build the Go application
FROM golang:1.23.4-alpine3.21 AS builder

WORKDIR /app


# Set the target OS and architecture
ENV GOOS=linux
ENV GOARCH=amd64

# Copy go.mod and go.sum first to leverage layer caching
COPY go.mod go.sum ./
#if copy go.mod and go.sum before then run go mod tidy and download immediately
# it would cache the layer so both command would only run if there any changes in go.mod or go.sum
# Download and cache dependencies
RUN go mod download

# Run go mod tidy to ensure that dependencies are consistent
RUN go mod tidy

# Copy the source code into the container
COPY . .

# Download and cache dependencies
#RUN go mod download



# Build the Go application
RUN go build -o order-service .

# Stage 2: Create a lightweight runtime image
FROM alpine:latest

WORKDIR /app

# Copy the compiled binary from the previous stage
COPY --from=builder /app/order-service .

# Copy the migrations folder into the container
COPY --from=builder /app/internal/stores/postgres/migrations ./internal/stores/postgres/migrations


# Copy the .env file into the container
COPY .env .
COPY pubkey.pem .


EXPOSE 80
# Command to run the application
CMD ["./order-service"]