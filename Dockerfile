# ====== Stage 1: Build frontend ======
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ====== Stage 2: Build Go backend + mail-receiver ======
FROM golang:1.25-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app/backend

# Copy go mod files and download deps
COPY backend/go.mod ./
RUN go mod download 2>/dev/null || true

COPY backend/ ./

# Copy frontend dist for embed
COPY --from=frontend-builder /app/frontend/dist ./dist

# Build main server
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /app/mailer-server .

# Build pipe receiver (separate binary)
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /app/mail-receiver ./pipe/

# ====== Stage 3: Backend runtime ======
FROM alpine:3.20 AS backend
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /app/mailer-server /app/mailer-server
ENV DB_PATH=/data/mailer.db
ENV LISTEN_ADDR=:8080
ENV GIN_MODE=release
EXPOSE 8080
VOLUME /data
CMD ["/app/mailer-server"]

# ====== Stage 4: Postfix runtime ======
FROM alpine:3.20 AS postfix
RUN apk add --no-cache postfix ca-certificates tzdata
COPY --from=backend-builder /app/mail-receiver /usr/local/bin/mail-receiver
RUN chmod +x /usr/local/bin/mail-receiver
COPY postfix/main.cf /etc/postfix/main.cf
COPY postfix/master.cf /etc/postfix/master.cf
COPY postfix/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 25
CMD ["/entrypoint.sh"]
