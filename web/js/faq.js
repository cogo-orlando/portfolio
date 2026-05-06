// ── FERME INTERVIEW avec Escape ──
// (nav.js gère déjà Escape pour le dropdown, on ajoute closeInterview ici)
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeInterview();
});

// ── FAQ DATA ──
const items        = document.querySelectorAll('.faq-item');
const totalEl      = document.getElementById('totalQ');
const totalCountEl = document.getElementById('totalCount');
const openCountEl  = document.getElementById('openCount');
const progressFill = document.getElementById('progressFill');

if (totalEl)      totalEl.textContent      = items.length;
if (totalCountEl) totalCountEl.textContent = items.length;

items.forEach(item => item.classList.add('pulse'));

// ── ACCORDION ──
let openedIds = new Set();

function updateProgress() {
    const total  = document.querySelectorAll('.faq-item:not(.hidden)').length;
    const opened = [...openedIds].filter(id => {
        const el = document.getElementById(id);
        return el && !el.classList.contains('hidden');
    }).length;
    if (openCountEl)  openCountEl.textContent  = opened;
    if (totalCountEl) totalCountEl.textContent = total;
    if (progressFill) progressFill.style.width = total ? (opened / total * 100) + '%' : '0%';
}

items.forEach((item, idx) => {
    const id = 'faq-' + idx;
    item.id  = id;
    item.querySelector('.faq-q').addEventListener('click', () => {
        const isOpen = item.classList.contains('open');
        if (isOpen) { item.classList.remove('open'); openedIds.delete(id); }
        else        { item.classList.add('open'); item.classList.remove('pulse'); openedIds.add(id); }
        updateProgress();
    });
});

// ── FILTRES CATÉGORIES ──
document.querySelectorAll('.cat-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.cat-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        const cat = btn.dataset.cat;
        items.forEach(item => item.classList.toggle('hidden', cat !== 'all' && item.dataset.cat !== cat));
        updateProgress();
    });
});

// ── RECHERCHE ──
const searchInput = document.getElementById('searchInput');
const secretWords = { 'airbus': 'faq-2' };

searchInput?.addEventListener('input', () => {
    const query = searchInput.value.toLowerCase().trim();

    // Easter egg
    if (secretWords[query]) {
        items.forEach(item => item.classList.add('hidden'));
        const secret = document.getElementById(secretWords[query]);
        if (secret) {
            secret.classList.remove('hidden');
            secret.classList.add('open');
            openedIds.add(secretWords[query]);
            setTimeout(() => secret.scrollIntoView({ behavior:'smooth', block:'center' }), 100);
        }
        updateProgress();
        return;
    }

    items.forEach(item => {
        if (!query) { item.classList.remove('hidden'); return; }
        const qText = item.querySelector('.faq-text')?.textContent.toLowerCase() || '';
        const aText = item.querySelector('.faq-a-inner p')?.textContent.toLowerCase() || '';
        item.classList.toggle('hidden', !qText.includes(query) && !aText.includes(query));
    });
    updateProgress();
});

// ── QUESTION ALÉATOIRE ──
document.getElementById('randomBtn')?.addEventListener('click', () => {
    const visible = [...items].filter(i => !i.classList.contains('hidden'));
    if (!visible.length) return;
    const random = visible[Math.floor(Math.random() * visible.length)];
    items.forEach(i => { i.classList.remove('open'); openedIds.delete(i.id); });
    random.classList.add('open');
    random.classList.remove('pulse');
    openedIds.add(random.id);
    random.scrollIntoView({ behavior:'smooth', block:'center' });
    updateProgress();
});

// ── TOUT OUVRIR / FERMER ──
let allOpen = false;
document.getElementById('expandAllBtn')?.addEventListener('click', (e) => {
    allOpen = !allOpen;
    items.forEach(item => {
        if (item.classList.contains('hidden')) return;
        item.classList.toggle('open', allOpen);
        if (allOpen) { item.classList.remove('pulse'); openedIds.add(item.id); }
        else         { openedIds.delete(item.id); }
    });
    e.target.textContent = allOpen ? 'Tout fermer' : 'Tout ouvrir';
    updateProgress();
});

// ── MODE INTERVIEW ──
const interviewData = [...items].map(item => ({
    q: item.querySelector('.faq-text')?.textContent || '',
    a: item.querySelector('.faq-a-inner p')?.textContent || '',
}));

let ivIndex   = 0;
const overlay = document.getElementById('interviewOverlay');
const ivBody  = document.getElementById('interviewBody');

function renderIVQuestion(idx) {
    const q = interviewData[idx];
    if (!q || !ivBody) return;
    ivBody.innerHTML = '';

    const qEl   = document.createElement('div');
    qEl.className   = 'iv-question';
    qEl.textContent = `~/interview $ Question ${idx + 1}/${interviewData.length}`;
    ivBody.appendChild(qEl);

    const textEl = document.createElement('div');
    textEl.className   = 'iv-text';
    textEl.textContent = q.q;
    ivBody.appendChild(textEl);

    const ansEl = document.createElement('div');
    ansEl.className = 'iv-answer';
    ivBody.appendChild(ansEl);

    let i = 0;
    const words    = q.a.split(' ');
    const interval = setInterval(() => {
        if (i >= words.length) { clearInterval(interval); return; }
        ansEl.textContent += (i > 0 ? ' ' : '') + words[i++];
    }, 40);
}

function openInterview()  { ivIndex = 0; overlay?.classList.add('active'); renderIVQuestion(ivIndex); }
function closeInterview() { overlay?.classList.remove('active'); }

document.getElementById('interviewBtn')?.addEventListener('click', openInterview);
document.getElementById('interviewClose')?.addEventListener('click', closeInterview);
document.getElementById('ivExit')?.addEventListener('click', closeInterview);
overlay?.addEventListener('click', (e) => { if (e.target === overlay) closeInterview(); });
document.getElementById('ivNext')?.addEventListener('click', () => {
    ivIndex = (ivIndex + 1) % interviewData.length;
    renderIVQuestion(ivIndex);
});

// ── INIT ──
updateProgress();