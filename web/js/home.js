// Typing animation
const typingEl = document.querySelector(".hero-tag .typed");
const text = "ls -la";

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
}, { threshold: 0.15 });

document.querySelectorAll(".skill-card, .project-card, .contact-card").forEach(el => {
    el.classList.add("reveal");
    observer.observe(el);
});