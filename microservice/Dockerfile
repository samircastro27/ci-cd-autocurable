# Etapa 1: build
FROM golang:1.20 as builder
WORKDIR /app

COPY . .

# Compilar binario estático para evitar errores con GLIBC
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o server

# Etapa 2: runtime ultra mínima (Alpine o scratch)
FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/server .

RUN chmod +x ./server

EXPOSE 8080
CMD ["./server"]