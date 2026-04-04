// ── NAV DROPDOWN ──
const dropdown    = document.querySelector('.nav-dropdown');
const dropdownBtn = document.querySelector('.nav-dropdown-btn');
if (dropdown && dropdownBtn) {
    dropdownBtn.addEventListener('click', (e) => { e.stopPropagation(); dropdown.classList.toggle('open'); });
    document.addEventListener('click', () => dropdown.classList.remove('open'));
    document.addEventListener('keydown', (e) => { if (e.key === 'Escape') { dropdown.classList.remove('open'); closeInterview(); } });
    const currentPath = window.location.pathname;
    document.querySelectorAll('.dropdown-item').forEach(item => {
        if (item.getAttribute('href') === currentPath) {
            item.querySelector('.dropdown-name').style.color = 'var(--accent)';
            item.style.background = 'rgba(0,245,160,0.04)';
        }
    });
}

// ── TYPING ──
const typingEl = document.querySelector('.hero-tag .typed');
const text = 'cat faq.md';
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
const revealObserver = new IntersectionObserver((entries) => {
    entries.forEach(entry => { if (entry.isIntersecting) entry.target.classList.add('visible'); });
}, { threshold: 0.1 });
document.querySelectorAll('.reveal').forEach(el => revealObserver.observe(el));

// ── FAQ DATA ──
const items = document.querySelectorAll('.faq-item');
const totalEl   = document.getElementById('totalQ');
const totalCountEl = document.getElementById('totalCount');
const openCountEl  = document.getElementById('openCount');
const progressFill = document.getElementById('progressFill');

// Init compteur total
if (totalEl) totalEl.textContent = items.length;
if (totalCountEl) totalCountEl.textContent = items.length;

// Pulse sur les items non ouverts
items.forEach(item => item.classList.add('pulse'));

// ── ACCORDION ──
let openedIds = new Set();

function updateProgress() {
    const total  = document.querySelectorAll('.faq-item:not(.hidden)').length;
    const opened = [...openedIds].filter(id => {
        const el = document.getElementById(id);
        return el && !el.classList.contains('hidden');
    }).length;
    if (openCountEl) openCountEl.textContent = opened;
    if (totalCountEl) totalCountEl.textContent = total;
    if (progressFill) progressFill.style.width = total ? (opened / total * 100) + '%' : '0%';
}

items.forEach((item, idx) => {
    const id = 'faq-' + idx;
    item.id  = id;
    const btn = item.querySelector('.faq-q');
    btn.addEventListener('click', () => {
        const isOpen = item.classList.contains('open');
        if (isOpen) {
            item.classList.remove('open');
            openedIds.delete(id);
        } else {
            item.classList.add('open');
            item.classList.remove('pulse');
            openedIds.add(id);
        }
        updateProgress();
    });
});

// ── FILTRES CATÉGORIES ──
document.querySelectorAll('.cat-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.cat-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        const cat = btn.dataset.cat;
        items.forEach(item => {
            if (cat === 'all' || item.dataset.cat === cat) {
                item.classList.remove('hidden');
            } else {
                item.classList.add('hidden');
            }
        });
        updateProgress();
    });
});

// ── RECHERCHE ──
const searchInput = document.getElementById('searchInput');
const secretWords = { 'airbus': 'faq-2' }; // data-secret="airbus" → item index 2

searchInput?.addEventListener('input', () => {
    const query = searchInput.value.toLowerCase().trim();

    // Easter egg — mot clé secret
    if (secretWords[query]) {
        items.forEach(item => item.classList.add('hidden'));
        const secretItem = document.getElementById(secretWords[query]);
        if (secretItem) {
            secretItem.classList.remove('hidden');
            secretItem.classList.add('open');
            openedIds.add(secretWords[query]);
            setTimeout(() => secretItem.scrollIntoView({ behavior:'smooth', block:'center' }), 100);
        }
        updateProgress();
        return;
    }

    // Recherche normale
    if (!query) {
        items.forEach(item => item.classList.remove('hidden'));
        updateProgress();
        return;
    }

    items.forEach(item => {
        const qText = item.querySelector('.faq-text')?.textContent.toLowerCase() || '';
        const aText = item.querySelector('.faq-a-inner p')?.textContent.toLowerCase() || '';
        if (qText.includes(query) || aText.includes(query)) {
            item.classList.remove('hidden');
        } else {
            item.classList.add('hidden');
        }
    });
    updateProgress();
});

// ── QUESTION ALÉATOIRE ──
document.getElementById('randomBtn')?.addEventListener('click', () => {
    const visible = [...items].filter(i => !i.classList.contains('hidden'));
    if (!visible.length) return;
    const random = visible[Math.floor(Math.random() * visible.length)];

    // Ferme tous, ouvre le random
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
        if (allOpen) {
            item.classList.add('open');
            item.classList.remove('pulse');
            openedIds.add(item.id);
        } else {
            item.classList.remove('open');
            openedIds.delete(item.id);
        }
    });
    e.target.textContent = allOpen ? 'Tout fermer' : 'Tout ouvrir';
    updateProgress();
});

// ── MODE INTERVIEW ──
const interviewData = [...items].map(item => ({
    q: item.querySelector('.faq-text')?.textContent || '',
    a: item.querySelector('.faq-a-inner p')?.textContent || '',
    cat: item.dataset.cat || '',
}));

let ivIndex    = 0;
const overlay  = document.getElementById('interviewOverlay');
const ivBody   = document.getElementById('interviewBody');

function renderIVQuestion(idx) {
    const q = interviewData[idx];
    if (!q) return;
    ivBody.innerHTML = '';

    const qEl  = document.createElement('div');
    qEl.className = 'iv-question';
    qEl.textContent = `~/interview $ Question ${idx + 1}/${interviewData.length}`;
    ivBody.appendChild(qEl);

    const textEl = document.createElement('div');
    textEl.className = 'iv-text';
    textEl.textContent = q.q;
    ivBody.appendChild(textEl);

    // Anime la réponse lettre par lettre
    const ansEl = document.createElement('div');
    ansEl.className = 'iv-answer';
    ivBody.appendChild(ansEl);

    let i = 0;
    const words = q.a.split(' ');
    const interval = setInterval(() => {
        if (i >= words.length) { clearInterval(interval); return; }
        ansEl.textContent += (i > 0 ? ' ' : '') + words[i];
        i++;
    }, 40);
}

function openInterview() {
    ivIndex = 0;
    overlay.classList.add('active');
    renderIVQuestion(ivIndex);
}

function closeInterview() {
    overlay?.classList.remove('active');
}

document.getElementById('interviewBtn')?.addEventListener('click', openInterview);
document.getElementById('interviewClose')?.addEventListener('click', closeInterview);
document.getElementById('ivExit')?.addEventListener('click', closeInterview);
overlay?.addEventListener('click', (e) => { if (e.target === overlay) closeInterview(); });

document.getElementById('ivNext')?.addEventListener('click', () => {
    ivIndex = (ivIndex + 1) % interviewData.length;
    renderIVQuestion(ivIndex);
});

// ── SOUMETTRE UNE QUESTION ──
document.getElementById('askBtn')?.addEventListener('click', async () => {
    const input = document.getElementById('askInput');
    const val   = input?.value.trim();
    if (!val || val.length < 5) return;

    try {
        await fetch('/api/faq-question', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ question: val })
        });
    } catch {}

    // Affiche succès peu importe le résultat serveur
    document.getElementById('askForm').style.display   = 'none';
    document.getElementById('askSuccess').style.display = 'block';
});

// ── INITIALISATION ──
updateProgress();