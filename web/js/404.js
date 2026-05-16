// ── TYPING CMD ──
const typedCmd = document.querySelector('.typed-cmd');
const termOutput = document.getElementById('term-output');
const cmd = 'cat 404.log';

let i = 0;
const typeCmd = setInterval(() => {
    if (!typedCmd) { clearInterval(typeCmd); return; }
    typedCmd.textContent += cmd[i++];
    if (i >= cmd.length) {
        clearInterval(typeCmd);
        setTimeout(showOutput, 400);
    }
}, 80);

// ── TERMINAL OUTPUT ──
function showOutput() {
    const lines = [
        { cls: 't-err', text: '[ERROR] 404 — Route introuvable' },
        { cls: 't-dim', text: `> Path: ${window.location.pathname}` },
        { cls: 't-dim', text: '> Status: NOT_FOUND' },
        { cls: 't-warn', text: '> Aucun handler Go enregistré pour cette route' },
        { cls: 't-acc', text: '> Suggestion: redirection vers /home' },
    ];

    let delay = 0;
    lines.forEach(line => {
        delay += 180;
        setTimeout(() => {
            if (!termOutput) return;
            const el = document.createElement('div');
            el.className = `t-line ${line.cls}`;
            el.textContent = line.text;
            termOutput.appendChild(el);
        }, delay);
    });
}

// ── COUNTDOWN REDIRECTION ──
const countdownEl = document.getElementById('countdown');
let seconds = 10;

const countdown = setInterval(() => {
    seconds--;
    if (countdownEl) countdownEl.textContent = seconds;
    if (seconds <= 0) {
        clearInterval(countdown);
        window.location.href = '/home';
    }
}, 1000);

// ── ANNULER REDIRECTION si click ──
document.querySelectorAll('a').forEach(a => {
    a.addEventListener('click', () => clearInterval(countdown));
});