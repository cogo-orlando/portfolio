/* ══════════════════════════════════════════
   HOME.JS — page d'accueil (/home)
   Dépend de core.js
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, $$, onReady, countUp, fetchJSON, rafThrottle, reducedMotion } = window.Site;

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

        // ── EFFETS VERRE (souris uniquement, pas sur mobile ni si "moins d'animations") ──
        const finePointer = window.matchMedia('(hover: hover) and (pointer: fine)').matches;
        if (finePointer && !reducedMotion) {
            // Reflet de lumière qui suit la souris sur chaque panneau
            $$('.stat-card, .whatido-card, .flagship-card, .quicknav-card, .infra-card, .setup-card, .certif-item, .arch-diagram, .security-checks, .learning-ticker')
                .forEach((el) => {
                    el.addEventListener('pointermove', rafThrottle((e) => {
                        const r = el.getBoundingClientRect();
                        el.style.setProperty('--mx', `${e.clientX - r.left}px`);
                        el.style.setProperty('--my', `${e.clientY - r.top}px`);
                    }));
                    el.addEventListener('pointerleave', () => {
                        el.style.removeProperty('--mx');
                        el.style.removeProperty('--my');
                    });
                });

            // Légère inclinaison 3D des cards projets, comme une plaque de verre qu'on tient
            $$('.flagship-card').forEach((card) => {
                card.addEventListener('pointermove', rafThrottle((e) => {
                    const r = card.getBoundingClientRect();
                    const x = (e.clientX - r.left) / r.width - 0.5;   // -0.5 → 0.5
                    const y = (e.clientY - r.top) / r.height - 0.5;
                    card.style.setProperty('--ry', `${(x * 6).toFixed(2)}deg`);
                    card.style.setProperty('--rx', `${(-y * 6).toFixed(2)}deg`);
                }));
                card.addEventListener('pointerleave', () => {
                    card.style.removeProperty('--rx');
                    card.style.removeProperty('--ry');
                });
            });
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