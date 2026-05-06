// ── FERME ZOOM avec Escape ──
// (nav.js gère déjà Escape pour le dropdown, on ajoute closeZoom ici)
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') closeZoom?.();
});

// ── ZOOM CV ──
// Ajoute ici ta logique de zoom si elle existe dans cv.js
// function closeZoom() { ... }