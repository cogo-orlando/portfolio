/* ══════════════════════════════════════════
   CORE.JS — boîte à outils partagée par tout le site
   Chargé AVANT nav.js et les scripts de page (via le partial footer).
   Tout est rangé dans window.Site → aucune variable globale qui traîne.

   Utilisation dans un autre JS :
     Site.reveal(element)
     Site.typeText(el, 'texte')
     Site.countUp(el, 129)
     Site.fetchJSON('/health').then(data => ...)
     Site.escapeHTML(texteUtilisateur)
══════════════════════════════════════════ */
// Signale que le JS tourne : le CSS ne cache les éléments animés QUE dans ce cas
document.documentElement.classList.add('js');

(function () {
    'use strict';

    const Site = {};

    // ── Sélecteurs raccourcis ──
    Site.$ = (sel, root = document) => root.querySelector(sel);
    Site.$$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

    // ── Lancer du code quand le DOM est prêt ──
    Site.onReady = (fn) => {
        if (document.readyState !== 'loading') fn();
        else document.addEventListener('DOMContentLoaded', fn, { once: true });
    };

    // ── Préférence "réduire les animations" (mise à jour en direct) ──
    const motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    Site.reducedMotion = motionQuery.matches;
    motionQuery.addEventListener?.('change', (e) => { Site.reducedMotion = e.matches; });

    // ── Sécurité : échapper du texte avant de l'insérer en HTML ──
    // À utiliser dès qu'un contenu vient de l'utilisateur ou d'une API
    // et qu'on ne peut pas utiliser textContent.
    Site.escapeHTML = (str) => String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');

    // ── Attendre x ms (avec await) ──
    Site.sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

    // ── Limiter un handler à 1 exécution par frame (scroll, pointermove…) ──
    Site.rafThrottle = (fn) => {
        let queued = false;
        return (...args) => {
            if (queued) return;
            queued = true;
            requestAnimationFrame(() => { queued = false; fn(...args); });
        };
    };

    // ── Stockage local sans plantage (navigation privée, quota plein…) ──
    Site.storage = {
        get(key, fallback = null) {
            try { const v = localStorage.getItem(key); return v === null ? fallback : JSON.parse(v); }
            catch { return fallback; }
        },
        set(key, value) {
            try { localStorage.setItem(key, JSON.stringify(value)); return true; }
            catch { return false; }
        },
    };

    // ── fetch JSON avec timeout et erreurs propres ──
    Site.fetchJSON = async (url, { timeout = 5000, ...options } = {}) => {
        const controller = new AbortController();
        const timer = setTimeout(() => controller.abort(), timeout);
        try {
            const res = await fetch(url, { ...options, signal: controller.signal });
            if (!res.ok) throw new Error(`${url} → HTTP ${res.status}`);
            return await res.json();
        } finally {
            clearTimeout(timer);
        }
    };

    // ── Apparition au scroll (.reveal → .visible) ──
    // Un seul observer pour tout le site. Site.reveal(el) pour un élément ajouté en JS.
    let revealObserver = null;
    Site.reveal = (target) => {
        const els = target instanceof Element ? [target] : Site.$$(target || '.reveal');
        if (Site.reducedMotion || !('IntersectionObserver' in window)) {
            els.forEach((el) => el.classList.add('visible'));
            return;
        }
        if (!revealObserver) {
            revealObserver = new IntersectionObserver((entries) => {
                entries.forEach((entry) => {
                    if (!entry.isIntersecting) return;
                    entry.target.classList.add('visible');
                    revealObserver.unobserve(entry.target); // plus besoin de le surveiller
                });
            }, { threshold: 0.05 }); // 5 % suffit : évite que les très grands blocs restent invisibles
        }
        els.forEach((el) => revealObserver.observe(el));
    };

    // ── Effet machine à écrire ──
    // Renvoie une promesse : await Site.typeText(el, 'ls -la')
    Site.typeText = async (el, text, { speed = 80, delay = 0 } = {}) => {
        if (!el || !text) return;
        if (Site.reducedMotion) { el.textContent = text; return; }
        if (delay) await Site.sleep(delay);
        el.textContent = '';
        for (const char of text) {
            el.textContent += char;
            await Site.sleep(speed);
        }
    };

    // ── Compteur animé de 0 à target ──
    Site.countUp = (el, target, { duration = 1200 } = {}) => {
        const end = Number(target);
        if (!el || Number.isNaN(end)) return;
        if (Site.reducedMotion) { el.textContent = end; return; }
        const start = performance.now();
        const tick = (now) => {
            const progress = Math.min((now - start) / duration, 1);
            const eased = 1 - Math.pow(1 - progress, 3); // ralentit à la fin
            el.textContent = Math.round(end * eased);
            if (progress < 1) requestAnimationFrame(tick);
        };
        requestAnimationFrame(tick);
    };

    // ── Durée Go ("26h3m4.5s") → texte lisible ("1j 2h", "3h 12min", "45min") ──
    Site.formatUptime = (raw) => {
        if (typeof raw !== 'string') return '—';
        const h = Number(raw.match(/(\d+)h/)?.[1] || 0);
        const m = Number(raw.match(/(\d+)m(?!s)/)?.[1] || 0);
        const days = Math.floor(h / 24);
        if (days > 0) return `${days}j ${h % 24}h`;
        if (h > 0) return `${h}h ${m}min`;
        return `${m}min`;
    };

    // ── Fermer un élément avec la touche Échap ──
    Site.onEscape = (fn) => {
        document.addEventListener('keydown', (e) => { if (e.key === 'Escape') fn(e); });
    };

    window.Site = Site;
})();