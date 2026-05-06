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

// ── TYPING (spécifique home — délai plus long à cause du boot) ──
const typingEl = document.querySelector('.hero-tag .typed');
if (typingEl) {
    const text = 'ls -la';
    let i = 0;
    setTimeout(() => {
        const interval = setInterval(() => {
            typingEl.textContent += text[i]; i++;
            if (i >= text.length) clearInterval(interval);
        }, 80);
    }, 1800);
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
        learningEl.textContent = current.slice(0, ++lChar);
        if (lChar === current.length) { lDeleting = true; setTimeout(typeLearning, 1800); return; }
    } else {
        learningEl.textContent = current.slice(0, --lChar);
        if (lChar === 0) { lDeleting = false; lIdx = (lIdx + 1) % learningItems.length; }
    }
    setTimeout(typeLearning, lDeleting ? 40 : 70);
}
setTimeout(typeLearning, 2500);