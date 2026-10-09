/* ══════════════════════════════════════════
   DEMO-PROJECT.JS — commun à toutes les pages projets
   (visionneuse d'images + description dépliable)
   Dépend de core.js — l'apparition au scroll est gérée par nav.js

   Les images viennent de la page :
     <script> window.projectImages = [{ src, title }, …] </script>
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, $$, onReady } = window.Site;

    onReady(() => {
        // ── VISIONNEUSE ──
        const images = Array.isArray(window.projectImages) ? window.projectImages : [];
        const lightbox = $('#lightbox');
        const img = $('#lightboxImg');
        const title = $('#lightboxTitle');
        const counter = $('#lightboxCounter');

        if (lightbox && img && images.length) {
            let current = 0;
            let lastFocus = null;

            lightbox.setAttribute('aria-hidden', 'true');

            const show = (index) => {
                current = (index + images.length) % images.length;
                const item = images[current];
                img.src = item.src;
                img.alt = item.title || '';
                if (title) title.textContent = item.title || '';
                if (counter) counter.textContent = `${current + 1} / ${images.length}`;
                // Précharge les images voisines → navigation instantanée
                [current - 1, current + 1].forEach((i) => {
                    new Image().src = images[(i + images.length) % images.length].src;
                });
            };

            const open = (index) => {
                lastFocus = document.activeElement;
                show(index);
                lightbox.classList.add('active');
                lightbox.setAttribute('aria-hidden', 'false');
                document.body.style.overflow = 'hidden';
                $('#lightboxClose')?.focus();
            };

            const close = () => {
                if (!lightbox.classList.contains('active')) return;
                lightbox.classList.remove('active');
                lightbox.setAttribute('aria-hidden', 'true');
                document.body.style.overflow = '';
                lastFocus?.focus(); // le focus revient sur la capture cliquée
            };

            // Les captures deviennent utilisables au clavier (Tab + Entrée)
            $$('.gallery-card').forEach((card) => {
                const index = parseInt(card.dataset.index, 10) || 0;
                card.setAttribute('tabindex', '0');
                card.setAttribute('role', 'button');
                card.setAttribute('aria-label', `Agrandir : ${images[index]?.title || 'capture'}`);
                card.addEventListener('click', () => open(index));
                card.addEventListener('keydown', (e) => {
                    if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); open(index); }
                });
            });

            $('#lightboxPrev')?.addEventListener('click', (e) => { e.stopPropagation(); show(current - 1); });
            $('#lightboxNext')?.addEventListener('click', (e) => { e.stopPropagation(); show(current + 1); });
            $('#lightboxClose')?.addEventListener('click', close);
            $('#lightboxOverlay')?.addEventListener('click', close);

            document.addEventListener('keydown', (e) => {
                if (!lightbox.classList.contains('active')) return;
                if (e.key === 'ArrowLeft') show(current - 1);
                if (e.key === 'ArrowRight') show(current + 1);
                if (e.key === 'Escape') close();
            });

            // Swipe sur mobile
            let startX = null;
            lightbox.addEventListener('touchstart', (e) => { startX = e.touches[0].clientX; }, { passive: true });
            lightbox.addEventListener('touchend', (e) => {
                if (startX === null) return;
                const dx = e.changedTouches[0].clientX - startX;
                if (Math.abs(dx) > 50) show(current + (dx < 0 ? 1 : -1));
                startX = null;
            });
        }

        // ── DESCRIPTION "LIRE LA SUITE" ──
        const content = $('#descContent');
        const toggle = $('#descToggle');
        const label = $('#descToggleText');
        const arrow = $('#descArrow');

        toggle?.addEventListener('click', () => {
            const expanded = content.classList.toggle('expanded');
            toggle.setAttribute('aria-expanded', String(expanded));
            if (label) label.textContent = expanded ? 'Réduire' : 'Lire la suite';
            arrow?.classList.toggle('rotated', expanded);
        });
    });
})();