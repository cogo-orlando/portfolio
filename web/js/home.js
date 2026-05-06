// ── NAV — Liens directs + dropdown "Plus" + hamburger mobile ──

// Active le lien de la page courante
const currentPath = window.location.pathname;
document.querySelectorAll('.nav-link').forEach(link => {
    if (link.getAttribute('href') === currentPath) {
        link.classList.add('active');
    }
});

// Dropdown "Plus"
const navMore     = document.querySelector('.nav-more');
const navMoreBtn  = document.getElementById('navMoreBtn');
const navMoreMenu = document.getElementById('navMoreMenu');

navMoreBtn?.addEventListener('click', (e) => {
    e.stopPropagation();
    navMore.classList.toggle('open');
});
document.addEventListener('click', () => navMore?.classList.remove('open'));
document.addEventListener('keydown', (e) => { if (e.key === 'Escape') navMore?.classList.remove('open'); });

// Hamburger mobile
const navBurger   = document.getElementById('navBurger');
const mobileMenu  = document.getElementById('mobileMenu');

navBurger?.addEventListener('click', () => {
    navBurger.classList.toggle('open');
    mobileMenu.classList.toggle('open');
});

// Ferme le menu mobile en cliquant sur un lien
document.querySelectorAll('.mobile-link').forEach(link => {
    link.addEventListener('click', () => {
        navBurger.classList.remove('open');
        mobileMenu.classList.remove('open');
    });
});

// ── BOOT SCREEN ──
const bootScreen = document.getElementById('bootScreen');
const bootLines  = document.getElementById('bootLines');

const bootMessages = [
    { text: 'Initialisation du système...', cls: 'dim' },
    { text: 'Démarrage des services réseau...', cls: 'ok' },
    { text: 'Connexion à orlando.cogo...', cls: 'ok' },
    { text: 'Système prêt.', cls: '' },
];

function runBoot() {
    let i = 0;
    const interval = setInterval(() => {
        if (i >= bootMessages.length) {
            clearInterval(interval);
            setTimeout(() => {
                bootScreen.classList.add('hidden');
                setTimeout(() => bootScreen.remove(), 600);
            }, 400);
            return;
        }
        const line = document.createElement('div');
        line.className = 'boot-line ' + bootMessages[i].cls;
        line.textContent = bootMessages[i].text;
        bootLines.appendChild(line);
        i++;
    }, 180);
}
runBoot();

// ── MATRIX RAIN ──
const canvas = document.getElementById('matrix');
const ctx    = canvas.getContext('2d');

function resizeMatrix() { canvas.width = window.innerWidth; canvas.height = window.innerHeight; }
resizeMatrix();
window.addEventListener('resize', resizeMatrix);

const chars = 'アイウエオカキクケコサシスセソタチツテトナニヌネノ0123456789ABCDEF';
const colW  = 18;
let cols  = Math.floor(canvas.width / colW);
let drops = Array(cols).fill(1);

function drawMatrix() {
    ctx.fillStyle = 'rgba(8,11,15,0.05)';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.fillStyle = '#00f5a0';
    ctx.font = '13px DM Mono, monospace';
    cols = Math.floor(canvas.width / colW);
    if (drops.length < cols) drops = [...drops, ...Array(cols - drops.length).fill(1)];
    for (let i = 0; i < cols; i++) {
        const char = chars[Math.floor(Math.random() * chars.length)];
        ctx.fillText(char, i * colW, drops[i] * colW);
        if (drops[i] * colW > canvas.height && Math.random() > 0.975) drops[i] = 0;
        drops[i]++;
    }
}
setInterval(drawMatrix, 80);

// ── TYPING ANIMATION ──
const typingEl = document.querySelector('.hero-tag .typed');
const text = 'ls -la';
if (typingEl) {
    let i = 0;
    setTimeout(() => {
        const interval = setInterval(() => {
            typingEl.textContent += text[i]; i++;
            if (i >= text.length) clearInterval(interval);
        }, 80);
    }, 1800);
}

// ── SCROLL REVEAL ──
const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => { if (entry.isIntersecting) entry.target.classList.add('visible'); });
}, { threshold: 0.1 });
document.querySelectorAll('.reveal').forEach(el => observer.observe(el));

// ── COUNTER ANIMATION ──
const counterObserver = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (!entry.isIntersecting) return;
        const el = entry.target;
        const target = parseInt(el.dataset.target);
        if (!target) return;
        let current = 0;
        const step = Math.ceil(target / 40);
        const timer = setInterval(() => {
            current += step;
            if (current >= target) { current = target; clearInterval(timer); }
            el.textContent = current;
        }, 40);
        counterObserver.unobserve(el);
    });
}, { threshold: 0.5 });
document.querySelectorAll('.stat-val[data-target]').forEach(el => counterObserver.observe(el));

// ── LIVE TIMER ──
const startDate = new Date('2025-09-01');
const liveEl = document.getElementById('live-timer');
if (liveEl) {
    function updateTimer() {
        const days = Math.floor((new Date() - startDate) / (1000 * 60 * 60 * 24));
        liveEl.textContent = days;
    }
    updateTimer();
    setInterval(updateTimer, 60000);
}

// ── CURRENTLY LEARNING TICKER ──
const learningItems = [
    'Sécurité des réseaux TCP/IP...',
    'SQL...',
    'Administration Linux...',
    'Go — architecture web...',
    'CTF challenges HackTheBox...',
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
setTimeout(typeLearning, 2500);

// ── EASTER EGG — Konami Code ──
const konami = ['ArrowUp','ArrowUp','ArrowDown','ArrowDown','ArrowLeft','ArrowRight','ArrowLeft','ArrowRight'];
let kIdx = 0;
document.addEventListener('keydown', (e) => {
    if (e.key === konami[kIdx]) { kIdx++; if (kIdx === konami.length) { kIdx = 0; triggerEasterEgg(); } }
    else kIdx = 0;
});

function triggerEasterEgg() {
    const overlay = document.createElement('div');
    overlay.style.cssText = `position:fixed;inset:0;background:rgba(8,11,15,0.95);z-index:9999;display:flex;align-items:center;justify-content:center;font-family:'DM Mono',monospace;color:#00f5a0;text-align:center;cursor:pointer;`;
    overlay.innerHTML = `<div><pre style="font-size:clamp(8px,1.5vw,13px);line-height:1.4;margin-bottom:2rem;">
  ██████╗ ██████╗ ██╗      █████╗ ███╗   ██╗██████╗  ██████╗
 ██╔═══██╗██╔══██╗██║     ██╔══██╗████╗  ██║██╔══██╗██╔═══██╗
 ██║   ██║██████╔╝██║     ███████║██╔██╗ ██║██║  ██║██║   ██║
 ██║   ██║██╔══██╗██║     ██╔══██║██║╚██╗██║██║  ██║██║   ██║
 ╚██████╔╝██║  ██║███████╗██║  ██║██║ ╚████║██████╔╝╚██████╔╝
  ╚═════╝ ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝  ╚═════╝</pre>
    <p style="font-size:14px;color:#e8f0f8;margin-bottom:0.5rem;">Tu as trouvé l'easter egg du site</p>
    <p style="font-size:12px;color:#5a7080;">Clique pour fermer</p></div>`;
    overlay.addEventListener('click', () => overlay.remove());
    document.body.appendChild(overlay);
}

// ── EASTER EGG — Console ──
console.log('%c Orlando Cogo — Portfolio ', 'background:#00f5a0;color:#080b0f;font-size:14px;font-weight:bold;padding:8px 16px;');
console.log('%c Étudiant en cybersécurité · Ynov Campus · B1 ', 'color:#00c8f5;font-size:12px;');
console.log('%c [ Konami code activé dans la page ] ', 'color:#3a4a5a;font-size:10px;');