# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source
COPY . .

# Build specified service
ARG SERVICE
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/${SERVICE} ./cmd/${SERVICE}

# Run stage
FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/bin/${SERVICE} .

EXPOSE 1883 4222

ENTRYPOINT ["/app/${SERVICE}"]
