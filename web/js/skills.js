/* ══════════════════════════════════════════
   SKILLS.JS — page compétences (/skills)
   Dépend de core.js
   (la couleur des icônes est maintenant gérée en CSS via --ic)
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $$, onReady, reducedMotion } = window.Site;

    onReady(() => {
        // ── BARRES DE PROGRESSION : se remplissent quand la card arrive à l'écran ──
        const cards = $$('.skill-card');
        if (reducedMotion || !('IntersectionObserver' in window)) {
            cards.forEach((card) => card.classList.add('bar-animated'));
            return;
        }
        const observer = new IntersectionObserver((entries) => {
            entries.forEach((entry) => {
                if (!entry.isIntersecting) return;
                entry.target.classList.add('bar-animated');
                observer.unobserve(entry.target);
            });
        }, { threshold: 0.3 });
        cards.forEach((card) => observer.observe(card));
    });
})();