/* ══════════════════════════════════════════
   INDEX.JS — page d'atterrissage (/)
   Dépend de core.js (à charger AVANT ce fichier)
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, onReady, fetchJSON, formatUptime } = window.Site;

    onReady(() => {
        // ── "En ligne depuis …" dans la pilule du bas ──
        const el = $('#idx-uptime');
        if (!el) return;
        fetchJSON('/health')
            .then((data) => { if (data.uptime) el.textContent = `en ligne depuis ${formatUptime(data.uptime)}`; })
            .catch(() => { /* pas grave : on garde "2026" */ });
    });
})();