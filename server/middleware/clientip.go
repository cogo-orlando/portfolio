package middleware

// ══════════════════════════════════════════
//  IP RÉELLE DU VISITEUR — sans pouvoir être falsifiée
//
//  Le problème : l'en-tête CF-Connecting-IP est posé par Cloudflare… mais
//  n'importe qui peut l'écrire lui-même en appelant directement l'adresse
//  orlando-portfolio.onrender.com (sans passer par Cloudflare). Il pouvait
//  ainsi contourner le rate limit ou faire blacklister l'IP d'un autre.
//
//  La solution : on regarde QUI nous a transmis la requête (le "pair").
//    - Derrière Render, c'est la DERNIÈRE IP de X-Forwarded-For : elle est
//      ajoutée par le proxy de Render, le visiteur ne peut pas la choisir.
//    - En local (sans proxy), c'est RemoteAddr.
//  On ne fait confiance à CF-Connecting-IP QUE si ce pair est un serveur
//  Cloudflare (plages officielles : https://www.cloudflare.com/ips/).
//  Sinon, l'IP retenue est celle du pair : impossible à falsifier.
// ══════════════════════════════════════════

import (
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
)

// Plages d'IP officielles de Cloudflare (https://www.cloudflare.com/ips-v4 et ips-v6).
// Elles changent très rarement ; à revérifier de temps en temps.
var cloudflareRanges = mustPrefixes(
	// IPv4
	"173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
	"141.101.64.0/18", "108.162.192.0/18", "190.93.240.0/20", "188.114.96.0/20",
	"197.234.240.0/22", "198.41.128.0/17", "162.158.0.0/15", "104.16.0.0/13",
	"104.24.0.0/14", "172.64.0.0/13", "131.0.72.0/22",
	// IPv6
	"2400:cb00::/32", "2606:4700::/32", "2803:f800::/32", "2405:b500::/32",
	"2405:8100::/32", "2a06:98c0::/29", "2c0f:f248::/32",
)

func mustPrefixes(cidrs ...string) []netip.Prefix {
	out := make([]netip.Prefix, 0, len(cidrs))
	for _, c := range cidrs {
		out = append(out, netip.MustParsePrefix(c))
	}
	return out
}

// isCloudflare indique si l'IP appartient à Cloudflare
func isCloudflare(ip string) bool {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return false
	}
	addr = addr.Unmap() // "::ffff:1.2.3.4" → "1.2.3.4"
	for _, p := range cloudflareRanges {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// peerIP renvoie l'IP de la machine qui nous a transmis la requête
// (dernier saut de X-Forwarded-For, ajouté par le proxy de Render ; sinon RemoteAddr)
func peerIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if last := strings.TrimSpace(parts[len(parts)-1]); last != "" {
			return last
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// GetIP renvoie l'IP réelle du visiteur, sans faire confiance aux en-têtes
// qu'il pourrait écrire lui-même.
func GetIP(r *http.Request) string {
	peer := peerIP(r)
	if isCloudflare(peer) {
		if cf := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cf != "" {
			if _, err := netip.ParseAddr(cf); err == nil {
				return cf
			}
		}
	}
	return peer
}

// ══════════════════════════════════════════
//  CLOUDFLARE OBLIGATOIRE (optionnel)
//  REQUIRE_CLOUDFLARE=true → toute requête qui ne passe pas par Cloudflare
//  reçoit un 403 (sauf /health, utilisé par le health check de Render).
//  Ainsi le WAF et la protection DDoS de Cloudflare ne peuvent plus être contournés.
// ══════════════════════════════════════════

// isLocal : machine locale ou réseau privé (dev, docker compose) → jamais bloqué
func isLocal(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	return addr.IsLoopback() || addr.IsPrivate()
}

func requireCloudflare() bool { return os.Getenv("REQUIRE_CLOUDFLARE") == "true" }

func CloudflareOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer := peerIP(r)
		if r.URL.Path != "/health" && !isCloudflare(peer) && !isLocal(peer) {
			if requireCloudflare() {
				slog.Warn("accès direct refusé (hors Cloudflare)",
					"peer", peer, "path", r.URL.Path,
					"request_id", GetRequestID(r.Context()),
				)
				writeError(w, r, http.StatusForbidden)
				return
			}
			slog.Info("accès direct (hors Cloudflare)",
				"peer", peer, "path", r.URL.Path,
				"request_id", GetRequestID(r.Context()),
			)
		}
		next.ServeHTTP(w, r)
	})
}
