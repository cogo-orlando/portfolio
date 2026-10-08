// ── SKILL ICON COLORS ──
document.querySelectorAll('.skill-icon').forEach(icon => {
    const color = getComputedStyle(icon).getPropertyValue('--ic').trim() || 'var(--accent)';
    icon.style.background = color + '18';
    icon.style.borderColor = color + '33';
    icon.style.color = color;
});

// ── ANIMATION BARRES AU SCROLL ──
const barObserver = new IntersectionObserver((entries) => {
    entries.forEach(e => {
        if (e.isIntersecting) {
            e.target.classList.add('bar-animated');
            barObserver.unobserve(e.target);
        }
    });
}, { threshold: 0.3 });
document.querySelectorAll('.skill-card').forEach(card => barObserver.observe(card));