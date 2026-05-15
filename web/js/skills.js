// ── CURRENTLY LEARNING TICKER ──
const learningItems = [
    'Administration Linux — Samba & annuaire LDAP...',
    'Cybersécurité — CTF HackTheBox Labs...',
    'Go — PostgreSQL & sécurité applicative...',
    'Docker — hardening & multi-stage builds...',
    'CI/CD — GitHub Actions & gosec...',
    'Cloudflare — WAF & SSL configuration...',
];

const learningEl = document.getElementById('learningText');
let lIdx = 0, lChar = 0, lDeleting = false;

function typeLearning() {
    if (!learningEl) return;
    const current = learningItems[lIdx];
    if (!lDeleting) {
        learningEl.textContent = current.slice(0, lChar + 1); lChar++;
        if (lChar === current.length) { lDeleting = true; setTimeout(typeLearning, 1800); return; }
    } else {
        learningEl.textContent = current.slice(0, lChar - 1); lChar--;
        if (lChar === 0) { lDeleting = false; lIdx = (lIdx + 1) % learningItems.length; }
    }
    setTimeout(typeLearning, lDeleting ? 40 : 70);
}
setTimeout(typeLearning, 1500);

// ── SKILL ICON COLORS ──
document.querySelectorAll('.skill-icon').forEach(icon => {
    const color = getComputedStyle(icon).getPropertyValue('--ic').trim() || '#00f5a0';
    icon.style.background = color + '18';
    icon.style.borderColor = color + '33';
    icon.style.color = color;
});

// ── ANIMATION BARRES AU SCROLL ──
const barObserver = new IntersectionObserver((entries) => {
    entries.forEach(e => {
        if (e.isIntersecting) {
            e.target.classList.add('bar-animated');
            barObserver.unobserve(e.target);
        }
    });
}, { threshold: 0.3 });
document.querySelectorAll('.skill-card').forEach(card => barObserver.observe(card));