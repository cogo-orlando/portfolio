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

    // Init hauteur
    content.style.maxHeight = content.scrollHeight + 'px';

    btn.addEventListener('click', () => {
        const expanded = btn.getAttribute('aria-expanded') === 'true';
        btn.setAttribute('aria-expanded', !expanded);
        if (expanded) {
            content.style.maxHeight = content.scrollHeight + 'px';
            requestAnimationFrame(() => {
                content.style.maxHeight = '0';
            });
        } else {
            content.style.maxHeight = content.scrollHeight + 'px';
        }
    });

    // Keyboard support
    btn.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            btn.click();
        }
    });
});

// Reveal observer
const revealObserver = new IntersectionObserver(entries => {
    entries.forEach(e => {
        if (e.isIntersecting) {
            e.target.classList.add('visible');
            revealObserver.unobserve(e.target);
        }
    });
}, { threshold: 0.1 });

document.querySelectorAll('.reveal').forEach(el => revealObserver.observe(el));