# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o d3 ./cmd/main.go

FROM scratch

COPY --from=builder /build/d3 /d3

EXPOSE 8080

ENTRYPOINT ["/d3"]
