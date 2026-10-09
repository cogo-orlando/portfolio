/* ══════════════════════════════════════════
   HOME.JS — page d'accueil (/home)
   Dépend de core.js
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, $$, onReady, countUp, fetchJSON } = window.Site;

    // "26h3m4.5s" (format Go) → "1j 2h" / "3h 12min" / "45min"
    const formatUptime = (raw) => {
        if (typeof raw !== 'string') return '—';
        const h = Number(raw.match(/(\d+)h/)?.[1] || 0);
        const m = Number(raw.match(/(\d+)m(?!s)/)?.[1] || 0);
        const days = Math.floor(h / 24);
        if (days > 0) return `${days}j ${h % 24}h`;
        if (h > 0) return `${h}h ${m}min`;
        return `${m}min`;
    };

    onReady(() => {
        // ── COMPTEURS : les chiffres (venus de _data.html) montent de 0 à leur valeur ──
        const counters = $$('.stat-val').filter((el) => /^\d+$/.test(el.textContent.trim()));
        if ('IntersectionObserver' in window) {
            const observer = new IntersectionObserver((entries) => {
                entries.forEach((entry) => {
                    if (!entry.isIntersecting) return;
                    const el = entry.target;
                    countUp(el, el.textContent.trim());
                    observer.unobserve(el);
                });
            }, { threshold: 0.5 });
            counters.forEach((el) => observer.observe(el));
        }

        // ── JOURS DEPUIS LA RECONVERSION (si l'élément existe) ──
        const liveEl = $('#live-timer');
        if (liveEl) {
            const startDate = new Date('2025-09-01');
            const update = () => { liveEl.textContent = Math.floor((Date.now() - startDate) / 864e5); };
            update();
            setInterval(update, 60000);
        }

        // ── STATS LIVE depuis /health ──
        const gorEl = $('#stat-goroutines');
        const upEl = $('#stat-uptime');
        if (gorEl || upEl) {
            fetchJSON('/health')
                .then((data) => {
                    if (gorEl) gorEl.textContent = data.goroutines ?? '—';
                    if (upEl) upEl.textContent = formatUptime(data.uptime);
                })
                .catch((err) => console.warn('stats live :', err.message));
        }
    });
})();