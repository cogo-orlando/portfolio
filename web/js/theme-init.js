/* ══════════════════════════════════════════
   THEME-INIT.JS — choisit le thème AVANT l'affichage de la page
   Chargé dans le <head> (sans defer) pour éviter un "flash" de la mauvaise couleur.
   Fichier externe : compatible avec la CSP stricte (aucun script inline).

   Ordre de priorité :
     1. le choix du visiteur (bouton soleil/lune), mémorisé
     2. sinon, le réglage de son appareil (clair / sombre)
══════════════════════════════════════════ */
(function () {
    'use strict';
    var theme = null;
    try { theme = localStorage.getItem('theme'); } catch (e) { /* navigation privée : on ignore */ }
    if (theme !== 'light' && theme !== 'dark') {
        theme = window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
    }
    document.documentElement.setAttribute('data-theme', theme);
})();
