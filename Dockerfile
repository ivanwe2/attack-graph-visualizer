# Build stage: run the tests, then compile one static binary.
# The web UI (static/) is embedded into the binary with go:embed.
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY . .
RUN go vet ./... && go test ./...
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /mst-demo .

# Runtime stage: an empty image that contains only the binary.
FROM scratch
COPY --from=build /mst-demo /mst-demo
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/mst-demo"]
