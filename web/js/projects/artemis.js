// ── TERMINAL ANIMATION ──
const termLines = [
    { cls:'t-acc',   prompt:'$', text:'java -jar ArtemisIII.jar' },
    { cls:'t-dim',   prompt:'#', text:'Chargement du simulateur...' },
    { cls:'t-white', prompt:'',  text:'==============================' },
    { cls:'t-acc',   prompt:'',  text:'        ARTEMIS III' },
    { cls:'t-white', prompt:'',  text:'==============================' },
    { cls:'t-white', prompt:'',  text:'' },
    { cls:'t-white', prompt:'',  text:'1. Nouvelle mission' },
    { cls:'t-white', prompt:'',  text:'2. Historique des missions' },
    { cls:'t-white', prompt:'',  text:'0. Quitter' },
    { cls:'t-dim',   prompt:'',  text:'Votre choix : 1' },
    { cls:'t-white', prompt:'',  text:'' },
    { cls:'t-dim',   prompt:'#', text:'Sélection du lanceur...' },
    { cls:'t-white', prompt:'',  text:'1. Saturn V    — 1 650M€ · Poussée : 35 MN' },
    { cls:'t-white', prompt:'',  text:'2. Ariane 5    — 510M€  · Poussée : 13 MN' },
    { cls:'t-white', prompt:'',  text:'3. Starship    — 312M€  · Poussée : 74 MN' },
    { cls:'t-dim',   prompt:'',  text:'Votre choix : 1' },
    { cls:'t-dim',   prompt:'#', text:'Sélection de la capsule...' },
    { cls:'t-white', prompt:'',  text:'1. Orion       — 8 astronautes · 26t' },
    { cls:'t-white', prompt:'',  text:'2. Crew Dragon — 7 astronautes · 12t' },
    { cls:'t-dim',   prompt:'',  text:'Votre choix : 2' },
    { cls:'t-dim',   prompt:'#', text:'Sélection de la mission...' },
    { cls:'t-white', prompt:'',  text:'1. ISS            — Orbite basse' },
    { cls:'t-white', prompt:'',  text:'2. Orbite terrestre — Orbite moyenne' },
    { cls:'t-white', prompt:'',  text:'3. Nibiru          — Mission lointaine' },
    { cls:'t-dim',   prompt:'',  text:'Votre choix : 1' },
    { cls:'t-white', prompt:'',  text:'' },
    { cls:'t-dim',   prompt:'#', text:'Initialisation du lancement...' },
    { cls:'t-acc',   prompt:'',  text:'🚀 T-10... T-9... T-8... T-7...' },
    { cls:'t-acc',   prompt:'',  text:'🔥 Allumage des moteurs...' },
    { cls:'t-acc',   prompt:'',  text:'⬆  Décollage confirmé !' },
    { cls:'t-white', prompt:'',  text:'' },
    { cls:'t-ok',    prompt:'',  text:'✓ SUCCÈS — Lancement réussi' },
    { cls:'t-cost',  prompt:'',  text:'  Lanceur  : Saturn V' },
    { cls:'t-cost',  prompt:'',  text:'  Capsule  : Crew Dragon' },
    { cls:'t-cost',  prompt:'',  text:'  Mission  : ISS' },
    { cls:'t-cost',  prompt:'',  text:'  Coût     : 1 650,01M€' },
    { cls:'t-dim',   prompt:'',  text:'  Date     : 06/05/2026 15:36' },
];

const termBody = document.getElementById('termBody');
let animTimeout = null;

function clearTermAnim() {
    if (animTimeout) clearTimeout(animTimeout);
    termBody.innerHTML = '';
}

function runTermAnim() {
    clearTermAnim();
    let delay = 0;
    termLines.forEach((line, idx) => {
        const d = idx < 5 ? 60 : idx < 10 ? 120 : idx < 25 ? 90 : 200;
        delay += d;
        animTimeout = setTimeout(() => {
            const div = document.createElement('div');
            div.className = 't-line';
            if (line.prompt) {
                div.innerHTML = `<span class="t-prompt">${line.prompt}</span><span class="${line.cls}">${line.text}</span>`;
            } else {
                div.innerHTML = `<span class="${line.cls}">${line.text}</span>`;
            }
            termBody.appendChild(div);
            termBody.scrollTop = termBody.scrollHeight;
        }, delay);
    });
}

runTermAnim();
document.getElementById('replayBtn')?.addEventListener('click', runTermAnim);

// ── CAPTURE LIGHTBOX ──
const lightbox = document.getElementById('lightbox');
document.getElementById('captureImg')?.addEventListener('click', () => lightbox.classList.add('active'));
document.getElementById('lightboxOverlay')?.addEventListener('click', () => lightbox.classList.remove('active'));
document.getElementById('lightboxClose')?.addEventListener('click', () => lightbox.classList.remove('active'));
document.addEventListener('keydown', (e) => { if (e.key === 'Escape') lightbox.classList.remove('active'); });