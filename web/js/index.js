// Typing animation for the whoami command
const text = "whoami";
const typingEl = document.querySelector(".typing-tag span:nth-child(2)");

if (typingEl) {
    typingEl.textContent = "";
    let i = 0;

    setTimeout(() => {
        const interval = setInterval(() => {
            typingEl.textContent += text[i];
            i++;
            if (i >= text.length) clearInterval(interval);
        }, 80);
    }, 600);
}

// ── UPTIME LIVE ──
async function loadUptime() {
    try {
        const res = await fetch('/health');
        if (!res.ok) return;
        const data = await res.json();
        const el = document.getElementById('idx-uptime');
        if (el && data.uptime) el.textContent = 'uptime ' + data.uptime;
    } catch {}
}
loadUptime();