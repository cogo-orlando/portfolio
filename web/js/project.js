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

// ── Year card toggle ──
document.querySelectorAll('.year-card-header[role="button"]').forEach(btn => {
    const content = document.getElementById(btn.getAttribute('aria-controls'));
    if (!content) return;

    btn.addEventListener('click', () => {
        const isCollapsed = content.classList.contains('collapsed');
        content.classList.toggle('collapsed');
        btn.setAttribute('aria-expanded', String(isCollapsed));
        btn.querySelector('.year-chevron').style.transform = isCollapsed ? '' : 'rotate(-90deg)';
    });

    btn.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            btn.click();
        }
    });
});

// ── REVEAL OBSERVER ──
const projectRevealObserver = new IntersectionObserver((entries) => {
    entries.forEach(e => {
        if (e.isIntersecting) {
            e.target.classList.add('visible');
            projectRevealObserver.unobserve(e.target);
        }
    });
}, { threshold: 0.05 }); // ← réduit à 0.05 au lieu de 0.1

document.querySelectorAll('.reveal').forEach(el => projectRevealObserver.observe(el));
