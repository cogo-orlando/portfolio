/* ══════════════════════════════════════════
   INDEX.JS — page d'atterrissage (/)
   Dépend de core.js
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, onReady, fetchJSON } = window.Site;

    onReady(() => {
        // ── UPTIME LIVE ──
        const el = $('#idx-uptime');
        if (!el) return;
        fetchJSON('/health')
            .then((data) => { if (data.uptime) el.textContent = `uptime ${data.uptime}`; })
            .catch(() => { /* pas grave : on garde le texte par défaut */ });
    });
})();