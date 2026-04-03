// ── NAV DROPDOWN ──
// Ajoute ce bloc dans ton home.js (et tous tes autres JS de pages)

const dropdown = document.querySelector('.nav-dropdown');
const dropdownBtn = document.querySelector('.nav-dropdown-btn');

if (dropdown && dropdownBtn) {
    // Ouvre/ferme au clic
    dropdownBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        dropdown.classList.toggle('open');
    });

    // Ferme en cliquant ailleurs
    document.addEventListener('click', () => {
        dropdown.classList.remove('open');
    });

    // Ferme avec Escape
    document.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') dropdown.classList.remove('open');
    });

    // Surligne la page active dans le dropdown
    const currentPath = window.location.pathname;
    document.querySelectorAll('.dropdown-item').forEach(item => {
        if (item.getAttribute('href') === currentPath) {
            item.querySelector('.dropdown-name').style.color = 'var(--accent)';
            item.style.background = 'rgba(0, 245, 160, 0.04)';
        }
    });
}

// Typing animation
const typingEl = document.querySelector(".hero-tag .typed");
const text = "cat skills.md";

if (typingEl) {
    let i = 0;
    setTimeout(() => {
        const interval = setInterval(() => {
            typingEl.textContent += text[i];
            i++;
            if (i >= text.length) clearInterval(interval);
        }, 80);
    }, 400);
}

// Scroll reveal
const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            entry.target.classList.add("visible");
        }
    });
}, { threshold: 0.1 });

document.querySelectorAll(".reveal").forEach(el => {
    observer.observe(el);
});