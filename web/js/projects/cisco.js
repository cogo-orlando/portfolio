// ── TYPING ──
const typingEl = document.querySelector('.hero-tag .typed');
const text = 'cat reseau-multi-sites.md';
if (typingEl) {
    let i = 0;
    setTimeout(() => {
        const interval = setInterval(() => {
            typingEl.textContent += text[i]; i++;
            if (i >= text.length) clearInterval(interval);
        }, 70);
    }, 400);
}

// ── SCROLL REVEAL ──
const observer = new IntersectionObserver((entries) => {
    entries.forEach(e => { if (e.isIntersecting) e.target.classList.add('visible'); });
}, { threshold: 0.1 });
document.querySelectorAll('.reveal').forEach(el => observer.observe(el));