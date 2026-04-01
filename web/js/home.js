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

// Active nav link on scroll
const sections = document.querySelectorAll("section[id]");
const navLinks = document.querySelectorAll(".nav-links a");

window.addEventListener("scroll", () => {
    let current = "";
    sections.forEach(section => {
        if (window.scrollY >= section.offsetTop - 120) {
            current = section.getAttribute("id");
        }
    });
    navLinks.forEach(link => {
        link.style.color = link.getAttribute("href") === `#${current}`
            ? "var(--accent)"
            : "";
    });
});