// ── PAGE ACTIVE ──
const currentPath = window.location.pathname;
document.querySelectorAll('.nav-link').forEach(link => {
    if (link.getAttribute('href') === currentPath) link.classList.add('active');
});

// ── DROPDOWN "PLUS" ──
const navMore    = document.querySelector('.nav-more');
const navMoreBtn = document.getElementById('navMoreBtn');

navMoreBtn?.addEventListener('click', (e) => {
    e.stopPropagation();
    navMore.classList.toggle('open');
});
document.addEventListener('click', () => navMore?.classList.remove('open'));
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') navMore?.classList.remove('open');
});

// ── HAMBURGER MOBILE ──
const navBurger  = document.getElementById('navBurger');
const mobileMenu = document.getElementById('mobileMenu');

navBurger?.addEventListener('click', () => {
    navBurger.classList.toggle('open');
    mobileMenu?.classList.toggle('open');
});
document.querySelectorAll('.mobile-link').forEach(link => {
    link.addEventListener('click', () => {
        navBurger?.classList.remove('open');
        mobileMenu?.classList.remove('open');
    });
});

// ── SCROLL REVEAL ──
const revealObserver = new IntersectionObserver((entries) => {
    entries.forEach(e => { if (e.isIntersecting) e.target.classList.add('visible'); });
}, { threshold: 0.1 });
document.querySelectorAll('.reveal').forEach(el => revealObserver.observe(el));

// ── TYPING ANIMATION ──
// Usage : <div class="hero-tag" data-typed="cat about.md">
(function initTyping() {
    const tag = document.querySelector('.hero-tag');
    const el  = tag?.querySelector('.typed');
    if (!el) return;
    const text  = tag.dataset.typed || '';
    const delay = parseInt(tag.dataset.typingDelay || '400');
    let i = 0;
    setTimeout(() => {
        const interval = setInterval(() => {
            el.textContent += text[i]; i++;
            if (i >= text.length) clearInterval(interval);
        }, 80);
    }, delay);
})();

// ── EASTER EGG — Konami Code ──
const konami = ['ArrowUp','ArrowUp','ArrowDown','ArrowDown','ArrowLeft','ArrowRight','ArrowLeft','ArrowRight'];
let kIdx = 0;
document.addEventListener('keydown', (e) => {
    kIdx = (e.key === konami[kIdx]) ? kIdx + 1 : 0;
    if (kIdx === konami.length) { kIdx = 0; triggerKonami(); }
});

function triggerKonami() {
    const overlay = document.createElement('div');
    overlay.style.cssText = 'position:fixed;inset:0;background:rgba(8,11,15,0.95);z-index:9999;display:flex;align-items:center;justify-content:center;font-family:"DM Mono",monospace;color:#00f5a0;text-align:center;cursor:pointer;';
    overlay.innerHTML = `<div>
        <pre style="font-size:clamp(8px,1.5vw,13px);line-height:1.4;margin-bottom:2rem;">
  ██████╗ ██████╗ ██╗      █████╗ ███╗   ██╗██████╗  ██████╗
 ██╔═══██╗██╔══██╗██║     ██╔══██╗████╗  ██║██╔══██╗██╔═══██╗
 ██║   ██║██████╔╝██║     ███████║██╔██╗ ██║██║  ██║██║   ██║
 ██║   ██║██╔══██╗██║     ██╔══██║██║╚██╗██║██║  ██║██║   ██║
 ╚██████╔╝██║  ██║███████╗██║  ██║██║ ╚████║██████╔╝╚██████╔╝
  ╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝  ╚═════╝</pre>
        <p style="font-size:14px;color:#e8f0f8;margin-bottom:0.5rem;">Tu as trouvé l'easter egg du site</p>
        <p style="font-size:12px;color:#5a7080;">Clique pour fermer</p>
    </div>`;
    overlay.addEventListener('click', () => overlay.remove());
    document.body.appendChild(overlay);
}

// ── EASTER EGG — Console ──
console.log('%c Orlando Cogo — Portfolio ', 'background:#00f5a0;color:#080b0f;font-size:14px;font-weight:bold;padding:8px 16px;');
console.log('%c Étudiant en cybersécurité · Ynov Campus · B1 ', 'color:#00c8f5;font-size:12px;');
console.log('%c [ Konami code activé ] ', 'color:#3a4a5a;font-size:10px;');