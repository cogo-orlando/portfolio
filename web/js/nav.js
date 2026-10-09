(function () {
    'use strict';
    const { $, $$, onReady, onEscape, rafThrottle, reveal } = window.Site;

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

        // ── PAGE ACTIVE (desktop + mobile, et /projects/* → "Projets") ──
        const path = window.location.pathname.replace(/\/$/, '') || '/';
        $$('.nav-link, .mobile-link').forEach((link) => {
            const href = link.getAttribute('href');
            const isActive = href === path || (href === '/project' && path.startsWith('/projects/'));
            if (isActive) {
                link.classList.add('active');
                link.setAttribute('aria-current', 'page');
            }
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
            navBurger?.setAttribute('aria-label', open ? 'Fermer le menu' : 'Ouvrir le menu');
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

        // ── APPARITION AU SCROLL ──
        reveal('.reveal');

    });
})();