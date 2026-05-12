# Orlando Cogo — Portfolio

![CI](https://github.com/cogo-orlando/portfolio/actions/workflows/ci.yml/badge.svg)

Portfolio personnel développé **from scratch** en Go — sans framework, sans CMS, sans template.  
Étudiant en **Bachelor Informatique — Spécialisation Cybersécurité** à Ynov Campus Toulouse.  
Reconverti de l'industrie aéronautique (Airbus Toulouse & Airbus Helicopters Marseille).

🌐 **[orlandocogo.com](https://orlandocogo.com)**

---

## Stack technique

| Couche | Technologie |
|--------|-------------|
| Backend | Go — serveur HTTP natif, zéro framework |
| Frontend | HTML · CSS · JavaScript vanilla |
| Déploiement | Render (free tier) + Cloudflare CDN |
| Formulaire | Formspree |
| Sécurité | Cloudflare WAF · Rate limiting · Honeypot |
| Monitoring | UptimeRobot · Google Search Console |
| CI/CD | GitHub Actions |

---

## Structure du projet

```
portfolio/
├── main.go                      # Point d'entrée
├── server/
│   ├── server.go                # Routeur + graceful shutdown + sitemap + health
│   ├── handler/
│   │   ├── pages.go             # Handlers HTML + template cache
│   │   └── pages_test.go        # Tests unitaires handlers
│   └── middleware/
│       ├── middleware.go        # Logger · Gzip · Cache · RateLimit · Honeypot · Recovery · RequestID
│       └── middleware_test.go   # Tests unitaires middleware
├── web/
│   ├── html/                    # Templates HTML
│   ├── css/                     # Feuilles de style
│   ├── js/                      # Scripts JavaScript
│   └── img/                     # Images + favicon
├── .github/
│   └── workflows/
│       └── ci.yml               # GitHub Actions — tests automatiques
├── Dockerfile                   # Multi-stage build — image ~60MB
├── Dockerfile.dev               # Dev avec Air hot-reload
├── docker-compose.yml           # Production
├── docker-compose.dev.yml       # Développement
└── go.mod
```

---

## Pages

| Route | Description |
|-------|-------------|
| `/` | Page de bienvenue |
| `/home` | Accueil principal |
| `/about` | Parcours & reconversion |
| `/skills` | Compétences + roadmap |
| `/project` | Projets réalisés |
| `/contact` | Formulaire de contact (Formspree) |
| `/cv` | CV interactif téléchargeable |
| `/status` | État du système en temps réel |
| `/faq` | Questions fréquentes |
| `/sitemap.xml` | Sitemap généré automatiquement |
| `/health` | Métriques Go (uptime, goroutines, mémoire) |

---

## Fonctionnalités techniques

**Backend Go**
- Serveur HTTP custom — routing manuel, zéro dépendance externe
- Template cache avec `sync.RWMutex` — compile une seule fois
- Graceful shutdown sur `SIGINT`/`SIGTERM` — arrêt propre en 10s
- Logs structurés JSON avec `slog` — compatibles Render Logs
- Sitemap XML généré dynamiquement

**Sécurité**
- Honeypot — 15 routes fausses (`/wp-admin`, `/.env`, etc.) qui blacklistent les IPs 24h
- Rate limiting global — 120 req/min/IP
- Panic recovery middleware — le serveur ne crash jamais
- Request ID unique par requête — tracabilité dans les logs
- IP réelle via `CF-Connecting-IP` (Cloudflare)
- Headers de sécurité via Cloudflare — score A sur securityheaders.com

**Cloudflare**
- SSL Full Strict · Always Use HTTPS
- WAF rate limiting sur `/admin`
- Headers : HSTS · X-Frame-Options · CSP · Permissions-Policy
- DDoS protection automatique

**Tests**
- 47 tests unitaires — handler + middleware
- Couverture : validation méthodes HTTP, cache templates, rate limiting, honeypot, panic recovery, request ID

---

## Lancer en local

### Sans Docker

```bash
git clone https://github.com/cogo-orlando/portfolio.git
cd portfolio
go run .
# http://localhost:8080
```

### Avec Docker (production)

```bash
docker compose up -d
docker compose logs -f
```

### Avec Docker (développement — hot reload)

```bash
docker compose -f docker-compose.dev.yml up
```

---

## Tests

```bash
# Lancer tous les tests
go test ./server/handler/... ./server/middleware/... -v

# Avec couverture
go test ./server/handler/... ./server/middleware/... -cover
```

---

## Mode maintenance

Dans `server/server.go` :

```go
// Tout le site en maintenance
var MaintenanceMode = true

// Pages spécifiques en maintenance
var maintenancePages = map[string]bool{
    "/blog": true,
    "/about": false,
}
```

---

## Déploiement sur Render

1. Fork ce repo sur GitHub
2. Connecte ton repo sur [render.com](https://render.com)
3. Configure le service :
    - **Language** : Go
    - **Build Command** : `go build -o main .`
    - **Start Command** : `./main`
4. Deploy

---

## Contact

| Canal | Lien |
|-------|------|
| Site | [orlandocogo.com](https://orlandocogo.com) |
| Email | orlando.cogo.pro@gmail.com |
| LinkedIn | [orlando-liautard-cogo](https://www.linkedin.com/in/orlando-liautard-cogo) |
| GitHub | [@cogo-orlando](https://github.com/cogo-orlando) |

---

## Licence

MIT — voir [LICENSE](LICENSE)