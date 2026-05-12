# ══════════════════════════════════════════
#  STAGE 1 — Build
#  SDK Go complet pour compiler le binaire
# ══════════════════════════════════════════
FROM golang:1.23-alpine AS builder

# Dépendances système nécessaires pour la compilation
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copie les fichiers de dépendances en premier
# Docker cache cette couche si go.mod/go.sum n'ont pas changé
COPY go.mod ./
RUN go mod download

# Copie le code source
COPY . .

# Compilation statique — pas de dépendances dynamiques
# CGO_ENABLED=0 : pas de libc C → binaire 100% Go statique
# -ldflags="-s -w" : supprime les symboles de debug → binaire plus petit
# -trimpath : supprime les chemins locaux du binaire
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -trimpath \
    -o main .

# ══════════════════════════════════════════
#  STAGE 2 — Runtime
#  Image minimale sans SDK Go
# ══════════════════════════════════════════
FROM scratch

# Copie les certificats SSL (pour les appels HTTPS sortants)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copie les timezones (pour time.Now() en Europe/Paris)
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copie uniquement le binaire compilé
COPY --from=builder /app/main /main

# Copie les fichiers statiques du frontend
COPY --from=builder /app/web /web

# Port exposé
EXPOSE 8080

# Utilisateur non-root (UID 1001)
# scratch n'a pas useradd, on utilise USER directement
USER 1001

# Variables d'environnement par défaut
ENV PORT=8080 \
    TZ=Europe/Paris

# Lance le binaire
ENTRYPOINT ["/main"]