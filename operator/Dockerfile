FROM golang:1.22 AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o healing-operator .

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/healing-operator .
USER nonroot:nonroot

ENTRYPOINT ["/healing-operator"]
