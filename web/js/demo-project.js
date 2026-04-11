// ── TYPING ──
const typingEl = document.querySelector('.hero-tag .typed');
const text = window.typingText || 'cat project.md';
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
    entries.forEach(e => { if (e.isIntersecting) e.target.classList.add('visible'); });
}, { threshold: 0.1 });
document.querySelectorAll('.reveal').forEach(el => observer.observe(el));

// ── LIGHTBOX ──
const images = window.projectImages || [];
const lightbox        = document.getElementById('lightbox');
const lightboxImg     = document.getElementById('lightboxImg');
const lightboxTitle   = document.getElementById('lightboxTitle');
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
    if (!images.length) return;
    const item = images[currentIndex];
    lightboxImg.src             = item.src;
    lightboxImg.alt             = item.title;
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

// Clic sur les cartes galerie
document.querySelectorAll('.gallery-card').forEach(card => {
    card.addEventListener('click', () => openLightbox(parseInt(card.dataset.index)));
});

// Boutons
document.getElementById('lightboxPrev')?.addEventListener('click', (e) => { e.stopPropagation(); prevImage(); });
document.getElementById('lightboxNext')?.addEventListener('click', (e) => { e.stopPropagation(); nextImage(); });
document.getElementById('lightboxClose')?.addEventListener('click', closeLightbox);
document.getElementById('lightboxOverlay')?.addEventListener('click', closeLightbox);

// Clavier
document.addEventListener('keydown', (e) => {
    if (!lightbox?.classList.contains('active')) return;
    if (e.key === 'ArrowLeft')  prevImage();
    if (e.key === 'ArrowRight') nextImage();
    if (e.key === 'Escape')     closeLightbox();
});