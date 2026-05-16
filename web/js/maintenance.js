// ── TYPING ──
const typingEl = document.querySelector('.hero-tag .typed');
const text = 'sudo maintenance --enable';
if (typingEl) {
    let i = 0;
    setTimeout(() => {
        const interval = setInterval(() => {
            typingEl.textContent += text[i]; i++;
            if (i >= text.length) clearInterval(interval);
        }, 65);
    }, 400);
}

// ── COUNTDOWN ──
// Modifie endTime pour définir la fin de la maintenance
const endTime = new Date();
endTime.setHours(endTime.getHours() + 2); // 2h à partir de maintenant

function updateCountdown() {
    const now  = new Date();
    const diff = Math.max(0, endTime - now);
    const h    = Math.floor(diff / (1000*60*60));
    const m    = Math.floor((diff % (1000*60*60)) / (1000*60));
    const s    = Math.floor((diff % (1000*60)) / 1000);

    const cdH = document.getElementById('cdH');
    const cdM = document.getElementById('cdM');
    const cdS = document.getElementById('cdS');
    if (cdH) cdH.textContent = String(h).padStart(2,'0');
    if (cdM) cdM.textContent = String(m).padStart(2,'0');
    if (cdS) cdS.textContent = String(s).padStart(2,'0');

    const returnEl = document.getElementById('returnTime');
    if (returnEl) {
        returnEl.textContent = endTime.toLocaleTimeString('fr-FR', {
            hour: '2-digit', minute: '2-digit'
        });
    }
}
updateCountdown();
setInterval(updateCountdown, 1000);

// ── PROGRESS BAR ──
const fillEl = document.getElementById('progressFill');
const pctEl  = document.getElementById('progressPct');

setTimeout(() => {
    if (fillEl) fillEl.style.width = '40%';
    if (pctEl)  pctEl.textContent  = '40%';
    document.getElementById('step1')?.classList.remove('active');
    document.getElementById('step1')?.classList.add('done');
    document.getElementById('step2')?.classList.add('active');
}, 800);