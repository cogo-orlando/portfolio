/* ══════════════════════════════════════════
   ARTEMIS.JS — page Artemis III
   (terminal de simulation + agrandissement de la capture)
   Dépend de core.js
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, onReady, onEscape, reducedMotion } = window.Site;

    // Sortie console rejouée dans le terminal
    const LINES = [
        { cls: 't-acc',   prompt: '$', text: 'java -jar ArtemisIII.jar' },
        { cls: 't-dim',   prompt: '#', text: 'Chargement du simulateur...' },
        { cls: 't-white', text: '==============================' },
        { cls: 't-acc',   text: '        ARTEMIS III' },
        { cls: 't-white', text: '==============================' },
        { cls: 't-white', text: '' },
        { cls: 't-white', text: '1. Nouvelle mission' },
        { cls: 't-white', text: '2. Historique des missions' },
        { cls: 't-white', text: '0. Quitter' },
        { cls: 't-dim',   text: 'Votre choix : 1' },
        { cls: 't-white', text: '' },
        { cls: 't-dim',   prompt: '#', text: 'Sélection du lanceur...' },
        { cls: 't-white', text: '1. Saturn V    — 1 650M€ · Poussée : 35 MN' },
        { cls: 't-white', text: '2. Ariane 5    — 510M€  · Poussée : 13 MN' },
        { cls: 't-white', text: '3. Starship    — 312M€  · Poussée : 74 MN' },
        { cls: 't-dim',   text: 'Votre choix : 1' },
        { cls: 't-dim',   prompt: '#', text: 'Sélection de la capsule...' },
        { cls: 't-white', text: '1. Orion       — 8 astronautes · 26t' },
        { cls: 't-white', text: '2. Crew Dragon — 7 astronautes · 12t' },
        { cls: 't-dim',   text: 'Votre choix : 2' },
        { cls: 't-dim',   prompt: '#', text: 'Sélection de la mission...' },
        { cls: 't-white', text: '1. ISS            — Orbite basse' },
        { cls: 't-white', text: '2. Orbite terrestre — Orbite moyenne' },
        { cls: 't-white', text: '3. Nibiru          — Mission lointaine' },
        { cls: 't-dim',   text: 'Votre choix : 1' },
        { cls: 't-white', text: '' },
        { cls: 't-dim',   prompt: '#', text: 'Initialisation du lancement...' },
        { cls: 't-acc',   text: '🚀 T-10... T-9... T-8... T-7...' },
        { cls: 't-acc',   text: '🔥 Allumage des moteurs...' },
        { cls: 't-acc',   text: '⬆  Décollage confirmé !' },
        { cls: 't-white', text: '' },
        { cls: 't-ok',    text: '✓ SUCCÈS — Lancement réussi' },
        { cls: 't-cost',  text: '  Lanceur  : Saturn V' },
        { cls: 't-cost',  text: '  Capsule  : Crew Dragon' },
        { cls: 't-cost',  text: '  Mission  : ISS' },
        { cls: 't-cost',  text: '  Coût     : 1 650,01M€' },
        { cls: 't-dim',   text: '  Date     : 06/05/2026 15:36' },
    ];

    onReady(() => {
        // ── TERMINAL ──
        const body = $('#termBody');
        let timers = [];

        const renderLine = (line) => {
            const div = document.createElement('div');
            div.className = 't-line';
            if (line.prompt) {
                const p = document.createElement('span');
                p.className = 't-prompt';
                p.textContent = line.prompt;
                div.append(p);
            }
            const t = document.createElement('span');
            t.className = line.cls;
            t.textContent = line.text || '\u00a0'; // ligne vide visible
            div.append(t);
            body.append(div);
            body.scrollTop = body.scrollHeight;
        };

        const run = () => {
            if (!body) return;
            timers.forEach(clearTimeout); // annule TOUTE l'animation en cours (pas seulement la dernière ligne)
            timers = [];
            body.replaceChildren();

            if (reducedMotion) { LINES.forEach(renderLine); return; }

            let delay = 0;
            LINES.forEach((line, i) => {
                delay += i < 5 ? 60 : i < 10 ? 120 : i < 25 ? 90 : 200;
                timers.push(setTimeout(() => renderLine(line), delay));
            });
        };

        // Démarre quand le terminal arrive à l'écran (sinon l'animation est finie avant qu'on la voie)
        if (body && 'IntersectionObserver' in window) {
            const io = new IntersectionObserver((entries) => {
                if (entries[0].isIntersecting) { run(); io.disconnect(); }
            }, { threshold: 0.3 });
            io.observe(body);
        } else {
            run();
        }
        $('#replayBtn')?.addEventListener('click', run);

        // ── AGRANDISSEMENT DE LA CAPTURE ──
        const lightbox = $('#lightbox');
        const capture = $('#captureImg');
        if (!lightbox || !capture) return;

        let lastFocus = null;
        lightbox.setAttribute('aria-hidden', 'true');

        const open = () => {
            lastFocus = document.activeElement;
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
            lastFocus?.focus();
        };

        // La capture devient utilisable au clavier
        const wrap = capture.closest('.capture-wrap') || capture;
        wrap.setAttribute('tabindex', '0');
        wrap.setAttribute('role', 'button');
        wrap.setAttribute('aria-label', 'Agrandir la capture');
        wrap.addEventListener('click', open);
        wrap.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); open(); }
        });

        $('#lightboxOverlay')?.addEventListener('click', close);
        $('#lightboxClose')?.addEventListener('click', close);
        onEscape(close);
    });
})();