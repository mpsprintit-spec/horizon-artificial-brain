FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/api-gateway ./cmd/api-gateway

FROM alpine:3.20
RUN adduser -D -H horizon
USER horizon
COPY --from=build /out/api-gateway /usr/local/bin/api-gateway
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api-gateway"]
