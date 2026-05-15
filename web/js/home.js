// ── TYPING ──
const typingEl = document.querySelector('.hero-tag .typed');
if (typingEl) {
    const text = 'ls -la';
    let i = 0;
    setTimeout(() => {
        const interval = setInterval(() => {
            typingEl.textContent += text[i]; i++;
            if (i >= text.length) clearInterval(interval);
        }, 80);
    }, 400); // délai réduit — plus de boot screen
}

// ── COUNTER ANIMATION ──
const counterObserver = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (!entry.isIntersecting) return;
        const el     = entry.target;
        const target = parseInt(el.dataset.target);
        if (!target) return;
        let current  = 0;
        const step   = Math.ceil(target / 40);
        const timer  = setInterval(() => {
            current += step;
            if (current >= target) { current = target; clearInterval(timer); }
            el.textContent = current;
        }, 40);
        counterObserver.unobserve(el);
    });
}, { threshold: 0.5 });
document.querySelectorAll('.stat-val[data-target]').forEach(el => counterObserver.observe(el));

// ── LIVE TIMER (jours depuis reconversion) ──
const liveEl = document.getElementById('live-timer');
if (liveEl) {
    const startDate = new Date('2025-09-01');
    const update = () => {
        liveEl.textContent = Math.floor((new Date() - startDate) / 864e5);
    };
    update();
    setInterval(update, 60000);
}

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
        learningEl.textContent = current.slice(0, ++lChar);
        if (lChar === current.length) { lDeleting = true; setTimeout(typeLearning, 1800); return; }
    } else {
        learningEl.textContent = current.slice(0, --lChar);
        if (lChar === 0) { lDeleting = false; lIdx = (lIdx + 1) % learningItems.length; }
    }
    setTimeout(typeLearning, lDeleting ? 40 : 70);
}
setTimeout(typeLearning, 800);

// ── STATS LIVE depuis /health ──
async function loadLiveStats() {
    try {
        const res = await fetch('/health');
        if (!res.ok) return;
        const data = await res.json();
        const gorEl = document.getElementById('stat-goroutines');
        const upEl  = document.getElementById('stat-uptime');
        if (gorEl) gorEl.textContent = data.goroutines ?? '—';
        if (upEl)  upEl.textContent  = data.uptime ?? '—';
    } catch (e) {
        console.warn('loadLiveStats:', e);
    }
}
loadLiveStats();