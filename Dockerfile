# syntax=docker/dockerfile:1
# ══════════════════════════════════════════
#  STAGE 1 — Build
#  SDK Go complet pour compiler le binaire
# ══════════════════════════════════════════
FROM golang:1.23-alpine AS builder

# Certificats TLS (copiés dans l'image finale pour Supabase et les appels HTTPS)
RUN apk add --no-cache ca-certificates

WORKDIR /src

# Dépendances d'abord : cette couche reste en cache tant que go.mod/go.sum ne changent pas.
# go.sum garantit que les modules téléchargés sont exactement ceux attendus (intégrité).
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Code source
COPY . .

# Architecture fournie automatiquement par Docker (amd64 sur Render, arm64 sur un Mac M1…)
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Binaire 100 % statique :
#   CGO_ENABLED=0     pas de libc → fonctionne dans une image vide (scratch)
#   -ldflags="-s -w"  sans symboles de debug → plus léger
#   -trimpath         sans les chemins de ta machine dans le binaire
#   -tags timetzdata  base des fuseaux horaires intégrée au binaire (~450 Ko)
#                     → plus besoin de copier /usr/share/zoneinfo
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -tags timetzdata \
    -ldflags="-s -w" \
    -trimpath \
    -o /out/portfolio .

# ══════════════════════════════════════════
#  STAGE 2 — Runtime
#  Image vide : uniquement le binaire, les certificats et le site
# ══════════════════════════════════════════
FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app
COPY --from=builder /out/portfolio /app/portfolio
COPY --from=builder /src/web /app/web

# Utilisateur non-root. Les fichiers restent la propriété de root :
# le serveur peut les lire mais pas les modifier, même s'il était compromis.
USER 1001:1001

ENV PORT=8080 \
    TZ=Europe/Paris

EXPOSE 8080

# scratch n'a ni shell ni wget : c'est le binaire lui-même qui vérifie /health
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/app/portfolio", "healthcheck"]

ENTRYPOINT ["/app/portfolio"]