// ── SCROLL REVEAL ──
const observer = new IntersectionObserver((entries) => {
    entries.forEach(e => { if (e.isIntersecting) e.target.classList.add('visible'); });
}, { threshold: 0.1 });
document.querySelectorAll('.reveal').forEach(el => observer.observe(el));

// ── LIGHTBOX ──
const images = window.projectImages || [
    { src: '/img/projects/zoo/zoo1.png', title: '01 — Menu principal — ASCII art & navigation' },
    { src: '/img/projects/zoo/zoo2.png', title: '02 — Menu zoo — Jouer, infos, carte' },
    { src: '/img/projects/zoo/zoo3.png', title: '03 — Tableau de bord — Argent & animaux' },
    { src: '/img/projects/zoo/zoo4.png', title: '04 — Registre des animaux — âge & sexe' },
];

const lightbox      = document.getElementById('lightbox');
const lightboxImg   = document.getElementById('lightboxImg');
const lightboxTitle = document.getElementById('lightboxTitle');
const lightboxCounter = document.getElementById('lightboxCounter');
let currentIndex = 0;

function openLightbox(index) {
    currentIndex = index;
    updateLightbox();
    lightbox.classList.add('active');
    document.body.style.overflow = 'hidden';
}

function closeLightbox() {
    lightbox.classList.remove('active');
    document.body.style.overflow = '';
}

function updateLightbox() {
    const item = images[currentIndex];
    lightboxImg.src   = item.src;
    lightboxImg.alt   = item.title;
    lightboxTitle.textContent   = item.title;
    lightboxCounter.textContent = `${currentIndex + 1} / ${images.length}`;
}

function prevImage() {
    currentIndex = (currentIndex - 1 + images.length) % images.length;
    updateLightbox();
}

function nextImage() {
    currentIndex = (currentIndex + 1) % images.length;
    updateLightbox();
}

// Clic sur les cartes
document.querySelectorAll('.gallery-card').forEach(card => {
    card.addEventListener('click', () => openLightbox(parseInt(card.dataset.index)));
});

// Boutons nav
document.getElementById('lightboxPrev').addEventListener('click', (e) => { e.stopPropagation(); prevImage(); });
document.getElementById('lightboxNext').addEventListener('click', (e) => { e.stopPropagation(); nextImage(); });
document.getElementById('lightboxClose').addEventListener('click', closeLightbox);
document.getElementById('lightboxOverlay').addEventListener('click', closeLightbox);

// Clavier
document.addEventListener('keydown', (e) => {
    if (!lightbox.classList.contains('active')) return;
    if (e.key === 'ArrowLeft')  prevImage();
    if (e.key === 'ArrowRight') nextImage();
    if (e.key === 'Escape')     closeLightbox();
});