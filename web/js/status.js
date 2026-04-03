// ── TYPING ──
const typingEl = document.querySelector('.hero-tag .typed');
const text = 'systemctl status portfolio';
if (typingEl) {
    let i = 0;
    setTimeout(() => {
        const interval = setInterval(() => {
            typingEl.textContent += text[i]; i++;
            if (i >= text.length) clearInterval(interval);
        }, 60);
    }, 500);
}

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
            item.style.background = 'rgba(0,245,160,0.04)';
        }
    });
}

// ── HORLOGE LIVE ──
function updateClock() {
    const now = new Date();
    const h = String(now.getHours()).padStart(2,'0');
    const m = String(now.getMinutes()).padStart(2,'0');
    const s = String(now.getSeconds()).padStart(2,'0');
    const timeStr = `${h}:${m}:${s}`;
    const navClock = document.getElementById('navClock');
    const localTime = document.getElementById('localTime');
    if (navClock) navClock.textContent = timeStr;
    if (localTime) localTime.textContent = timeStr;
}
updateClock();
setInterval(updateClock, 1000);

// ── UPTIME (depuis le démarrage de la page) ──
const pageStart = Date.now();
function updateUptime() {
    const el = document.getElementById('uptime');
    if (!el) return;
    const diff = Math.floor((Date.now() - pageStart) / 1000);
    const h = Math.floor(diff / 3600);
    const m = Math.floor((diff % 3600) / 60);
    const s = diff % 60;
    el.textContent = `${String(h).padStart(2,'0')}:${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')}`;
}
updateUptime();
setInterval(updateUptime, 1000);

// ── COMPTEUR RECONVERSION ──
const startDate   = new Date('2025-09-01');
const reconvEl    = document.getElementById('reconvDays');
if (reconvEl) {
    reconvEl.textContent = Math.floor((new Date() - startDate) / (1000*60*60*24));
}

// ── DERNIÈRE MISE À JOUR ──
const lastUpdateEl = document.getElementById('lastUpdate');
if (lastUpdateEl) {
    const now = new Date();
    lastUpdateEl.textContent = now.toLocaleDateString('fr-FR', { day:'2-digit', month:'long', year:'numeric' });
}

// ── MÉTÉO TOULOUSE via Open-Meteo (gratuit, sans clé) ──
async function fetchWeather() {
    try {
        const res  = await fetch('https://api.open-meteo.com/v1/forecast?latitude=43.6047&longitude=1.4442&current=temperature_2m,apparent_temperature,relative_humidity_2m,wind_speed_10m,weather_code&wind_speed_unit=kmh&timezone=Europe/Paris');
        const data = await res.json();
        const c    = data.current;

        const codes = {
            0:'Ciel dégagé', 1:'Généralement dégagé', 2:'Partiellement nuageux', 3:'Couvert',
            45:'Brouillard', 48:'Brouillard givrant',
            51:'Bruine légère', 61:'Pluie légère', 63:'Pluie modérée', 65:'Pluie forte',
            71:'Neige légère', 80:'Averses légères', 81:'Averses modérées',
            95:'Orage', 96:'Orage avec grêle'
        };

        const tempEl   = document.getElementById('weatherTemp');
        const descEl   = document.getElementById('weatherDesc');
        const feelsEl  = document.getElementById('weatherFeels');
        const humidEl  = document.getElementById('weatherHumid');
        const windEl   = document.getElementById('weatherWind');

        if (tempEl)  tempEl.textContent  = `${Math.round(c.temperature_2m)}°C`;
        if (descEl)  descEl.textContent  = codes[c.weather_code] || 'Inconnu';
        if (feelsEl) feelsEl.textContent = `${Math.round(c.apparent_temperature)}°C`;
        if (humidEl) humidEl.textContent = `${c.relative_humidity_2m}%`;
        if (windEl)  windEl.textContent  = `${Math.round(c.wind_speed_10m)} km/h`;

    } catch {
        const descEl = document.getElementById('weatherDesc');
        if (descEl) descEl.textContent = 'Données indisponibles';
    }
}
fetchWeather();

// ── OBJECTIF PROGRESSION ──
// Calcule automatiquement selon les étapes
const steps     = ['done', 'done', 'inprog', 'pending'];
const doneCount = steps.filter(s => s === 'done').length;
const pct       = Math.round((doneCount / steps.length) * 100);
const goalFill  = document.getElementById('goalFill');
const goalPct   = document.getElementById('goalPct');
setTimeout(() => {
    if (goalFill) goalFill.style.width = pct + '%';
    if (goalPct)  goalPct.textContent  = pct + '%';
}, 600);

// ── CURRENTLY LEARNING TICKER ──
const learningItems = [
    'Sécurité des réseaux TCP/IP...',
    'Cryptographie appliquée...',
    'SQL injection & prévention...',
    'Administration Linux avancée...',
    'Go — architecture web...',
    'CTF challenges TryHackMe...',
    'OWASP Top 10...',
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

// ── SCROLL REVEAL ──
const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => { if (entry.isIntersecting) entry.target.classList.add('visible'); });
}, { threshold: 0.1 });
document.querySelectorAll('.reveal').forEach(el => observer.observe(el));