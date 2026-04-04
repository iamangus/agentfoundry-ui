FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /agentfoundry-ui ./cmd/server/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /agentfoundry-ui /usr/local/bin/agentfoundry-ui

EXPOSE 8080

ENTRYPOINT ["agentfoundry-ui"]
