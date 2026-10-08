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
// ── STATS LIVE depuis /health ──
async function loadLiveStats() {
    try {
        const res = await fetch('/health');
        if (!res.ok) return;
        const data = await res.json();
        const gorEl = document.getElementById('stat-goroutines');
        const upEl  = document.getElementById('stat-uptime');
        if (gorEl) gorEl.textContent = data.goroutines ?? '—';
        if (upEl)  upEl.textContent  = data.uptime ?? '—';
    } catch (e) {
        console.warn('loadLiveStats:', e);
    }
}
loadLiveStats();