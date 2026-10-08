// ── GLASSMORPHISM NAV AU SCROLL ──
const nav = document.querySelector('.nav');
window.addEventListener('scroll', () => {
    nav?.classList.toggle('scrolled', window.scrollY > 50);
}, { passive: true });

// ── PAGE ACTIVE ──
const currentPath = window.location.pathname;
document.querySelectorAll('.nav-link').forEach(link => {
    if (link.getAttribute('href') === currentPath) link.classList.add('active');
});

// ── DROPDOWN "PLUS" ──
const navMore = document.querySelector('.nav-more');
const navMoreBtn = document.getElementById('navMoreBtn');

navMoreBtn?.addEventListener('click', (e) => {
    e.stopPropagation();
    const open = navMore.classList.toggle('open');
    navMoreBtn.setAttribute('aria-expanded', open);
});
document.addEventListener('click', () => {
    navMore?.classList.remove('open');
    navMoreBtn?.setAttribute('aria-expanded', 'false');
});
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        navMore?.classList.remove('open');
        navMoreBtn?.setAttribute('aria-expanded', 'false');
    }
});

// ── HAMBURGER MOBILE ──
const navBurger = document.getElementById('navBurger');
const mobileMenu = document.getElementById('mobileMenu');

navBurger?.addEventListener('click', () => {
    const open = navBurger.classList.toggle('open');
    mobileMenu?.classList.toggle('open');
    navBurger.setAttribute('aria-expanded', open);
    mobileMenu?.setAttribute('aria-hidden', !open);
});
document.querySelectorAll('.glass-light').forEach(el => {
    el.addEventListener('pointermove', e => {
        const r = el.getBoundingClientRect();
        el.style.setProperty('--mx', `${e.clientX - r.left}px`);
        el.style.setProperty('--my', `${e.clientY - r.top}px`);
    });
});

// ── SCROLL REVEAL ──
const revealObserver = new IntersectionObserver((entries) => {
    entries.forEach(e => { if (e.isIntersecting) e.target.classList.add('visible'); });
}, { threshold: 0.1 });
document.querySelectorAll('.reveal').forEach(el => revealObserver.observe(el));