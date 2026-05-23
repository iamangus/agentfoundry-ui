FROM node:22-alpine AS frontend-builder

WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /src/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 go build -o /agentfoundry-ui ./cmd/server/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /agentfoundry-ui /usr/local/bin/agentfoundry-ui

EXPOSE 8080

ENTRYPOINT ["agentfoundry-ui"]
