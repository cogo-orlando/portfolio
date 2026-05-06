// ═══════════════════════════════════════════════
//  BACKGROUND.JS — Matrix rain partagé
//  Ajoute <canvas id="matrix" class="matrix-canvas"></canvas>
//  dans le body de chaque page, puis charge ce script
// ═══════════════════════════════════════════════

(function () {
    const canvas = document.getElementById('matrix');
    if (!canvas) return;

    const ctx   = canvas.getContext('2d');
    const chars = 'アイウエオカキクケコサシスセソタチツテトナニヌネノ0123456789ABCDEF';
    const colW  = 18;
    let drops   = [];

    function resize() {
        canvas.width  = window.innerWidth;
        canvas.height = window.innerHeight;
        const cols = Math.floor(canvas.width / colW);
        // Conserve les drops existants, remplit les nouveaux à 1
        if (drops.length < cols) {
            drops = [...drops, ...Array(cols - drops.length).fill(1)];
        } else {
            drops = drops.slice(0, cols);
        }
    }

    function draw() {
        const cols = Math.floor(canvas.width / colW);
        ctx.fillStyle = 'rgba(8,11,15,0.05)';
        ctx.fillRect(0, 0, canvas.width, canvas.height);
        ctx.fillStyle = '#00f5a0';
        ctx.font = '13px DM Mono, monospace';

        for (let i = 0; i < cols; i++) {
            const char = chars[Math.floor(Math.random() * chars.length)];
            ctx.fillText(char, i * colW, drops[i] * colW);
            if (drops[i] * colW > canvas.height && Math.random() > 0.975) drops[i] = 0;
            drops[i]++;
        }
    }

    resize();
    window.addEventListener('resize', resize);
    setInterval(draw, 80);
})();