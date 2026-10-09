/* ══════════════════════════════════════════
   PROJECT.JS — page liste des projets (/project)
   Dépend de core.js (l'apparition au scroll est gérée par nav.js)
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, $$, onReady } = window.Site;

    onReady(() => {
        // ── FILTRES ──
        const cards = $$('.project-card');
        const noResults = $('#noResults');
        const buttons = $$('.filter-btn');

        buttons.forEach((btn) => {
            btn.addEventListener('click', () => {
                buttons.forEach((b) => {
                    b.classList.toggle('active', b === btn);
                    b.setAttribute('aria-pressed', String(b === btn));
                });
                const filter = btn.dataset.filter;
                let visible = 0;
                cards.forEach((card) => {
                    // découpe en mots : "c" ne doit pas matcher "cs" ou "react"
                    const tags = (card.dataset.filter || '').split(/\s+/);
                    const show = filter === 'all' || tags.includes(filter);
                    card.hidden = !show;
                    if (show) visible++;
                });
                if (noResults) noResults.hidden = visible > 0;
            });
        });

        // ── REPLI DES ANNÉES (B1, B2…) ──
        $$('.year-card-header[role="button"]').forEach((header) => {
            const content = document.getElementById(header.getAttribute('aria-controls'));
            if (!content) return;

            const toggle = () => {
                const willOpen = content.classList.contains('collapsed');
                content.classList.toggle('collapsed', !willOpen);
                header.setAttribute('aria-expanded', String(willOpen));
                // la rotation du chevron est gérée en CSS via aria-expanded
            };

            header.addEventListener('click', toggle);
            header.addEventListener('keydown', (e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    toggle();
                }
            });
        });
    });
})();