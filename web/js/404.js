/* ══════════════════════════════════════════
   404.JS — page introuvable
   Autonome (la 404 ne charge pas core.js)
══════════════════════════════════════════ */
(function () {
    'use strict';

    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    // Langue de la page (404.js est autonome : il ne charge pas core.js)
    const en = (document.documentElement.lang || '').toLowerCase().startsWith('en');
    const t = (fr, english) => (en ? english : fr);
    const typedCmd = document.querySelector('.typed-cmd');
    const termOutput = document.getElementById('term-output');
    const countdownEl = document.getElementById('countdown');
    const redirectEl = document.querySelector('.subtitle-redirect');

    // ── COMMANDE + SORTIE DU TERMINAL (sans effet de frappe) ──
    if (typedCmd) typedCmd.textContent = 'cat 404.log';

    const lines = [
        { cls: 't-err',  text: t('[ERROR] 404 — Page introuvable', '[ERROR] 404 — Page not found') },
        { cls: 't-dim',  text: `> ${t('Chemin', 'Path')} : ${window.location.pathname}` }, // textContent → aucun risque XSS
        { cls: 't-dim',  text: `> ${t('Statut', 'Status')} : NOT_FOUND` },
        { cls: 't-acc',  text: t('> Suggestion : retour à l\u2019accueil', '> Suggestion: back to home') },
    ];

    lines.forEach((line, i) => {
        const show = () => {
            if (!termOutput) return;
            const el = document.createElement('div');
            el.className = `t-line ${line.cls}`;
            el.textContent = line.text;
            termOutput.append(el);
        };
        if (reducedMotion) show();
        else setTimeout(show, 300 + i * 180);
    });

    // ── REDIRECTION AUTOMATIQUE ──
    // Annulée dès que le visiteur interagit : il est peut-être en train de lire.
    let seconds = 10;
    const timer = setInterval(() => {
        if (document.hidden) return; // en pause si l'onglet n'est pas affiché
        seconds -= 1;
        if (countdownEl) countdownEl.textContent = seconds;
        if (seconds <= 0) {
            clearInterval(timer);
            window.location.href = en ? '/en/home' : '/home';
        }
    }, 1000);

    const cancel = () => {
        clearInterval(timer);
        if (redirectEl) redirectEl.textContent = t('Redirection annulée — choisis une page ci-dessous.', 'Redirect cancelled — pick a page below.');
        ['pointerdown', 'keydown', 'wheel', 'touchstart'].forEach((ev) => window.removeEventListener(ev, cancel));
    };
    ['pointerdown', 'keydown', 'wheel', 'touchstart'].forEach((ev) => window.addEventListener(ev, cancel, { passive: true }));
})();
