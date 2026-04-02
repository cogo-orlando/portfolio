const cmdEl    = document.querySelector('.typed-cmd');
const outputEl = document.getElementById('term-output');
const cursorEl = document.querySelector('.cursor');

const command = 'cd ' + window.location.pathname;

const lines = [
    { cls: 'err', text: 'bash: cd: ' + window.location.pathname + ': Aucun fichier ou dossier de ce type' },
    { cls: 'dim', text: '─────────────────────────────────────────────────' },
    { cls: 'acc', text: 'Suggestion : retourner à /home' },
];

function addLine(cls, text) {
    const d = document.createElement('div');
    d.className = 't-line t-' + cls;
    d.textContent = text;
    outputEl.appendChild(d);
}

// Type the command first
let i = 0;
setTimeout(() => {
    const type = setInterval(() => {
        cmdEl.textContent += command[i];
        i++;
        if (i >= command.length) {
            clearInterval(type);

            // Hide cursor, show output
            setTimeout(() => {
                cursorEl.style.display = 'none';
                lines.forEach((l, idx) => {
                    setTimeout(() => addLine(l.cls, l.text), idx * 180);
                });
            }, 300);
        }
    }, 55);
}, 700);