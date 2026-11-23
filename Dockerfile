# Frontend build stage
FROM node:20-alpine AS web-builder

WORKDIR /app/web
# Build-time frontend settings (Vite inlines VITE_* variables in the bundle).
ARG VITE_SUPPORT_EMAIL=""
ARG VITE_PAYPAL_DONATE_URL=""
ENV VITE_SUPPORT_EMAIL=$VITE_SUPPORT_EMAIL \
    VITE_PAYPAL_DONATE_URL=$VITE_PAYPAL_DONATE_URL
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Backend build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o manawise ./cmd/server

# Runtime stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/manawise .
COPY --from=web-builder /app/web/dist ./web/dist

EXPOSE 8080
ENTRYPOINT ["./manawise"]
