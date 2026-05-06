// ── HORLOGE LIVE ──
function updateClock() {
    const now = new Date();
    const h = String(now.getHours()).padStart(2,'0');
    const m = String(now.getMinutes()).padStart(2,'0');
    const s = String(now.getSeconds()).padStart(2,'0');
    const timeStr = `${h}:${m}:${s}`;
    const navClock  = document.getElementById('navClock');
    const localTime = document.getElementById('localTime');
    if (navClock)  navClock.textContent  = timeStr;
    if (localTime) localTime.textContent = timeStr;
}
updateClock();
setInterval(updateClock, 1000);

// ── UPTIME (depuis le chargement de la page) ──
const pageStart = Date.now();
function updateUptime() {
    const el   = document.getElementById('uptime');
    if (!el) return;
    const diff = Math.floor((Date.now() - pageStart) / 1000);
    const h    = Math.floor(diff / 3600);
    const m    = Math.floor((diff % 3600) / 60);
    const s    = diff % 60;
    el.textContent = `${String(h).padStart(2,'0')}:${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')}`;
}
updateUptime();
setInterval(updateUptime, 1000);

// ── COMPTEUR RECONVERSION ──
const reconvEl = document.getElementById('reconvDays');
if (reconvEl) {
    reconvEl.textContent = Math.floor((new Date() - new Date('2025-09-01')) / (1000*60*60*24));
}

// ── DERNIÈRE MISE À JOUR ──
const lastUpdateEl = document.getElementById('lastUpdate');
if (lastUpdateEl) {
    lastUpdateEl.textContent = new Date().toLocaleDateString('fr-FR', { day:'2-digit', month:'long', year:'numeric' });
}

// ── MÉTÉO TOULOUSE via Open-Meteo ──
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
        const set = (id, val) => { const el = document.getElementById(id); if (el) el.textContent = val; };
        set('weatherTemp',  `${Math.round(c.temperature_2m)}°C`);
        set('weatherDesc',  codes[c.weather_code] || 'Inconnu');
        set('weatherFeels', `${Math.round(c.apparent_temperature)}°C`);
        set('weatherHumid', `${c.relative_humidity_2m}%`);
        set('weatherWind',  `${Math.round(c.wind_speed_10m)} km/h`);
    } catch {
        const el = document.getElementById('weatherDesc');
        if (el) el.textContent = 'Données indisponibles';
    }
}
fetchWeather();

// ── OBJECTIF PROGRESSION ──
const steps     = ['done', 'done', 'inprog', 'pending'];
const pct       = Math.round(steps.filter(s => s === 'done').length / steps.length * 100);
setTimeout(() => {
    const goalFill = document.getElementById('goalFill');
    const goalPct  = document.getElementById('goalPct');
    if (goalFill) goalFill.style.width = pct + '%';
    if (goalPct)  goalPct.textContent  = pct + '%';
}, 600);

// ── CURRENTLY LEARNING TICKER ──
const learningItems = [
    'Sécurité des réseaux TCP/IP...',
    'Cryptographie appliquée...',
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
setTimeout(typeLearning, 1500);

// ── COMPTEUR DE VISITES ──
fetch('/api/visits')
    .then(r => r.json())
    .then(data => {
        const el = document.getElementById('visitCount');
        if (el) el.textContent = data.visits;
    })
    .catch(() => {
        const el = document.getElementById('visitCount');
        if (el) el.textContent = '--';
    });