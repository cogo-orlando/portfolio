/* ══════════════════════════════════════════
   NAV.JS — comportements communs à toutes les pages
   Dépend de core.js (chargé juste avant dans le partial footer)
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, $$, onReady, onEscape, rafThrottle, reveal, t } = window.Site;

    onReady(() => {
        const nav = $('.nav');
        const navMore = $('.nav-more');
        const navMoreBtn = $('#navMoreBtn');
        const navBurger = $('#navBurger');
        const mobileMenu = $('#mobileMenu');

        // ── NAV PLUS OPAQUE AU SCROLL ──
        const onScroll = rafThrottle(() => nav?.classList.toggle('scrolled', window.scrollY > 50));
        window.addEventListener('scroll', onScroll, { passive: true });
        onScroll();

        // ── PAGE ACTIVE (desktop + mobile, FR et EN, et /projects/* → "Projets") ──
        // "/en/home" et "/home" désignent la même page : on compare sans le préfixe /en
        const stripLang = (p) => (p.replace(/^\/en(?=\/|$)/, '') || '/').replace(/\/$/, '') || '/';
        const path = stripLang(window.location.pathname);
        $$('.nav-link, .mobile-link').forEach((link) => {
            const href = stripLang(link.getAttribute('href') || '');
            const isActive = href === path || (href === '/project' && path.startsWith('/projects/'));
            if (isActive) {
                link.classList.add('active');
                link.setAttribute('aria-current', 'page');
            }
        });

        // ── BOUTON FR / EN : même page dans l'autre langue ──
        $$('[data-lang-switch]').forEach((link) => {
            const target = link.dataset.langSwitch;
            link.setAttribute('href', target === 'en' ? (path === '/' ? '/en' : '/en' + path) : path);
        });

        // ── DROPDOWN "PLUS" ──
        const closeMore = () => {
            navMore?.classList.remove('open');
            navMoreBtn?.setAttribute('aria-expanded', 'false');
        };
        navMoreBtn?.addEventListener('click', (e) => {
            e.stopPropagation();
            const open = navMore.classList.toggle('open');
            navMoreBtn.setAttribute('aria-expanded', String(open));
        });
        document.addEventListener('click', (e) => {
            if (navMore && !navMore.contains(e.target)) closeMore();
        });

        // ── MENU MOBILE ──
        const setMobile = (open) => {
            navBurger?.classList.toggle('open', open);
            mobileMenu?.classList.toggle('open', open);
            navBurger?.setAttribute('aria-expanded', String(open));
            navBurger?.setAttribute('aria-label', open ? t('Fermer le menu', 'Close menu') : t('Ouvrir le menu', 'Open menu'));
            mobileMenu?.setAttribute('aria-hidden', String(!open));
        };
        navBurger?.addEventListener('click', () => setMobile(!mobileMenu?.classList.contains('open')));
        $$('.mobile-link').forEach((link) => link.addEventListener('click', () => setMobile(false)));
        // Si on agrandit la fenêtre au-delà du mode mobile, on referme le menu
        window.addEventListener('resize', rafThrottle(() => { if (window.innerWidth > 900) setMobile(false); }));

        // Échap ferme tout
        onEscape(() => { closeMore(); setMobile(false); });

        // ── REFLET DE LUMIÈRE SUR LE VERRE ──
        $$('.glass-light').forEach((el) => {
            el.addEventListener('pointermove', rafThrottle((e) => {
                const r = el.getBoundingClientRect();
                el.style.setProperty('--mx', `${e.clientX - r.left}px`);
                el.style.setProperty('--my', `${e.clientY - r.top}px`);
            }));
        });

        // ── THÈME CLAIR / SOMBRE ──
        const themeBtn = $('#themeToggle');
        const metaTheme = document.querySelector('meta[name="theme-color"]');
        const applyThemeUI = (theme) => {
            themeBtn?.setAttribute('aria-label', theme === 'light'
                ? t('Activer le mode sombre', 'Switch to dark mode')
                : t('Activer le mode clair', 'Switch to light mode'));
            themeBtn?.setAttribute('aria-pressed', String(theme === 'light'));
            metaTheme?.setAttribute('content', theme === 'light' ? '#eef0fa' : '#0a0b16');
        };
        applyThemeUI(document.documentElement.dataset.theme || 'dark');

        themeBtn?.addEventListener('click', () => {
            const next = document.documentElement.dataset.theme === 'light' ? 'dark' : 'light';
            const root = document.documentElement;
            root.classList.add('theme-transition');           // fondu des couleurs
            root.dataset.theme = next;
            try { localStorage.setItem('theme', next); } catch { /* navigation privée */ }
            applyThemeUI(next);
            setTimeout(() => root.classList.remove('theme-transition'), 400);
        });

        // Si le visiteur n'a rien choisi, on suit le réglage de son appareil en direct
        window.matchMedia('(prefers-color-scheme: light)').addEventListener?.('change', (e) => {
            let saved = null;
            try { saved = localStorage.getItem('theme'); } catch { /* ignoré */ }
            if (saved) return;
            document.documentElement.dataset.theme = e.matches ? 'light' : 'dark';
            applyThemeUI(document.documentElement.dataset.theme);
        });

        // ── APPARITION AU SCROLL ──
        reveal('.reveal');

    });
})();
