# 🛡️ Portfolio — Orlando Cogo

---

## 👤 À propos

Étudiant en **Bachelor Informatique — Spécialisation Cybersécurité** à Ynov Campus Toulouse.  
Ce portfolio présente mon parcours, mes projets et mes compétences.  
Reconverti de l'industrie aéronautique (Airbus Toulouse & Airbus Helicopters Marseille) vers l'informatique.

---

## 🚀 Stack technique

| Couche | Technologie |
|--------|-------------|
| Backend | Go (serveur HTTP natif, pas de framework) |
| Frontend | HTML · CSS · JavaScript vanilla |
| Déploiement | Render (free tier) |
| Stockage | JSON files (pas de base de données) |
| Formulaire | Formspree |
| Contenu blog | Fichiers Markdown parsés par Go |

---

## 📁 Structure du projet

```
portfolio/
├── cmd/
│   └── main.go              # Point d'entrée
├── server/
│   ├── server.go            # Routeur principal + middleware maintenance
│   ├── handler.go           # Handlers des pages HTML
│   ├── contact_handler.go   # API contact + rate limiting + JSON storage
│   ├── faq_handler.go       # API soumission questions FAQ
├── web/
│   ├── html/                # Templates HTML
│   ├── css/                 # Feuilles de style
│   ├── js/                  # Scripts JavaScript
│   └── img/                 # Images
├── data/
│   ├── messages.json        # Messages de contact
│   ├── faq_questions.json   # Questions FAQ soumises
└── go.mod
```

---

## 📄 Pages

| Route | Description |
|-------|-------------|
| `/` | Page de bienvenue |
| `/home` | Accueil principal |
| `/about` | Parcours & reconversion |
| `/skills` | Compétences + roadmap |
| `/project` | Projets réalisés |
| `/contact` | Formulaire de contact |
| `/cv` | CV interactif |
| `/status` | État du système en temps réel |
| `/faq` | Questions fréquentes |
| `/maintenance` | Page de maintenance |

---

## ⚙️ Fonctionnalités

- **Serveur Go custom** — routing manuel, middleware, gestion des fichiers statiques
- **Mode maintenance** — activable page par page ou pour tout le site via une variable
- **Formulaire de contact** — validation front + back, rate limiting par IP, stockage JSON
- **Page de statut** — météo en temps réel (Open-Meteo), horloge live, compteurs animés
- **FAQ interactive** — mode interview, easter egg, filtres par catégorie, soumission de questions
- **CV interactif** — zones cliquables avec zoom sur les sections

---

## 🛠️ Lancer en local

```bash
# Cloner le repo
git clone https://github.com/cogo-orlando/portfolio.git
cd portfolio

# Lancer le serveur
go run ./cmd/

# Le site est accessible sur
http://localhost:8080
```

> ⚠️ Le working directory doit être `portfolio/` pour que les chemins relatifs (`web/`, `content/`, `data/`) fonctionnent correctement.

---

## 🔧 Configuration

### Mode maintenance

Dans `server/server.go` :

```go
// Tout le site en maintenance
var MaintenanceMode = true

// Pages spécifiques en maintenance
var maintenancePages = map[string]bool{
    "/blog": true,
    "/ctf":  false,
}

---

## 📦 Déploiement sur Render

1. Fork / clone ce repo sur GitHub
2. Connecte ton repo sur [render.com](https://render.com)
3. Configure le service :
   - **Language** : Go
   - **Build Command** : `go build -o main ./cmd/`
   - **Start Command** : `./main`
4. Ajoute la variable d'environnement `ADMIN_PASSWORD`
5. Deploy !

---

## 📬 Contact

| Canal | Lien |
|-------|------|
| Email | orlando.cogo.pro@gmail.com |
| LinkedIn | [orlando-liautard-cogo](https://www.linkedin.com/in/orlando-liautard-cogo) |
| GitHub | [@cogo-orlando](https://github.com/cogo-orlando) |

---

## 📜 Licence

Ce projet est sous licence MIT — voir le fichier [LICENSE](LICENSE) pour plus de détails.

---
>>>>>>> b20eeb3 (Update README.md)
