// ── NAV DROPDOWN ──
const dropdown    = document.querySelector('.nav-dropdown');
const dropdownBtn = document.querySelector('.nav-dropdown-btn');
if (dropdown && dropdownBtn) {
    dropdownBtn.addEventListener('click', (e) => { e.stopPropagation(); dropdown.classList.toggle('open'); });
    document.addEventListener('click', () => dropdown.classList.remove('open'));
    document.addEventListener('keydown', (e) => { if (e.key === 'Escape') dropdown.classList.remove('open'); });
    const currentPath = window.location.pathname;
    document.querySelectorAll('.dropdown-item').forEach(item => {
        if (item.getAttribute('href') === currentPath) {
            item.querySelector('.dropdown-name').style.color = 'var(--accent)';
            item.style.background = 'rgba(0, 245, 160, 0.04)';
        }
    });
}

// ── TYPING ──
const typingEl = document.querySelector('.hero-tag .typed');
const text = 'cat skills.md';
if (typingEl) {
    let i = 0;
    setTimeout(() => {
        const interval = setInterval(() => {
            typingEl.textContent += text[i]; i++;
            if (i >= text.length) clearInterval(interval);
        }, 80);
    }, 400);
}

// ── SCROLL REVEAL ──
const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => { if (entry.isIntersecting) entry.target.classList.add('visible'); });
}, { threshold: 0.1 });
document.querySelectorAll('.reveal').forEach(el => observer.observe(el));

// ── CURRENTLY LEARNING TICKER ──
const learningItems = [
    'Sécurité des réseaux TCP/IP...',
    'Cryptographie appliquée...',
    'SQL injection & prévention...',
    'Administration Linux avancée...',
    'Go — architecture web...',
    'CTF challenges TryHackMe...',
    'OWASP Top 10...',
    'Bash scripting...',
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

// ── SKILL ICON COLORS (fallback pour navigateurs sans color-mix) ──
document.querySelectorAll('.skill-icon').forEach(icon => {
    const color = getComputedStyle(icon).getPropertyValue('--ic').trim() || '#00f5a0';
    icon.style.background = color + '18';
    icon.style.borderColor = color + '33';
    icon.style.color = color;
});