/* ══════════════════════════════════════════
   INDEX.JS — page d'atterrissage (/)
   Dépend de core.js (à charger AVANT ce fichier)
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, onReady, fetchJSON, formatUptime, t } = window.Site;

    onReady(() => {
        // ── "En ligne depuis …" dans la pilule du bas ──
        const el = $('#idx-uptime');
        if (!el) return;
        fetchJSON('/health')
            .then((data) => { if (data.uptime) el.textContent = `${t('en ligne depuis', 'online for')} ${formatUptime(data.uptime)}`; })
            .catch(() => { /* pas grave : on garde "2026" */ });
    });
})();
