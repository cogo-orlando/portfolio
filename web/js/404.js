/* ══════════════════════════════════════════
   404.JS — page introuvable
   Autonome (la 404 ne charge pas core.js)
══════════════════════════════════════════ */
(function () {
    'use strict';

    const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const typedCmd = document.querySelector('.typed-cmd');
    const termOutput = document.getElementById('term-output');
    const countdownEl = document.getElementById('countdown');
    const redirectEl = document.querySelector('.subtitle-redirect');

    // ── COMMANDE + SORTIE DU TERMINAL (sans effet de frappe) ──
    if (typedCmd) typedCmd.textContent = 'cat 404.log';

    const lines = [
        { cls: 't-err',  text: '[ERROR] 404 — Page introuvable' },
        { cls: 't-dim',  text: `> Chemin : ${window.location.pathname}` }, // textContent → aucun risque XSS
        { cls: 't-dim',  text: '> Statut : NOT_FOUND' },
        { cls: 't-acc',  text: '> Suggestion : retour à l\u2019accueil' },
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
            window.location.href = '/home';
        }
    }, 1000);

    const cancel = () => {
        clearInterval(timer);
        if (redirectEl) redirectEl.textContent = 'Redirection annulée — choisis une page ci-dessous.';
        ['pointerdown', 'keydown', 'wheel', 'touchstart'].forEach((ev) => window.removeEventListener(ev, cancel));
    };
    ['pointerdown', 'keydown', 'wheel', 'touchstart'].forEach((ev) => window.addEventListener(ev, cancel, { passive: true }));
})();