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
const text = 'cat project.md';
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

// ── FILTRES ──
const cards     = document.querySelectorAll('.project-card');
const noResults = document.getElementById('noResults');

document.querySelectorAll('.filter-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');

        const filter = btn.dataset.filter;
        let visible  = 0;

        cards.forEach(card => {
            const tags = card.dataset.filter || '';
            if (filter === 'all' || tags.includes(filter)) {
                card.style.display = '';
                visible++;
            } else {
                card.style.display = 'none';
            }
        });

        if (noResults) noResults.style.display = visible === 0 ? 'flex' : 'none';
    });
});