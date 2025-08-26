# Build stage
FROM golang:1.20 as builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o server

# Final image
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
