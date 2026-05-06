# ── BUILD STAGE ──────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Télécharge les dépendances en premier (cache Docker)
COPY go.mod ./
RUN go mod download

# Copie tout le code source
COPY . .

# Build optimisé
RUN go build -ldflags="-s -w" -o main .

# ── RUN STAGE ────────────────────────────────────────────────────────────────
FROM alpine:latest

WORKDIR /app

# Utilisateur non-root pour la sécurité
RUN adduser -D appuser

# Copie le binaire compilé
COPY --from=builder /app/main .

# Copie les fichiers statiques et templates
COPY --from=builder /app/web ./web

# Crée le dossier data pour les messages JSON (vide au départ)
RUN mkdir -p data && chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

ENV PORT=8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget --spider -q http://localhost:8080/health || exit 1

CMD ["./main"]