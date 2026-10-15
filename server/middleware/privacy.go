package middleware

// ══════════════════════════════════════════
//  VIE PRIVÉE — anonymisation des adresses IP dans les journaux
//  IPv4 : le dernier octet est effacé  (203.0.113.42 → 203.0.113.0)
//  IPv6 : seul le préfixe /48 est gardé (le réseau, pas la machine)
//  Assez pour diagnostiquer un problème, plus assez pour identifier quelqu'un.
// ══════════════════════════════════════════

import (
	"net"
	"strings"
)

// AnonymizeIP renvoie l'adresse IP sans la partie qui identifie la machine
func AnonymizeIP(raw string) string {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil {
		return "unknown"
	}
	if v4 := ip.To4(); v4 != nil {
		return net.IPv4(v4[0], v4[1], v4[2], 0).String()
	}
	return ip.Mask(net.CIDRMask(48, 128)).String()
}
