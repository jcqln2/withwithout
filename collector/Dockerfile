FROM golang:1.26.10-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/collector ./cmd/collector

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=build /out/collector /collector
USER nobody
ENTRYPOINT ["/collector"]
