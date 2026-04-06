// ── NAV DROPDOWN ──
const dropdown    = document.querySelector('.nav-dropdown');
const dropdownBtn = document.querySelector('.nav-dropdown-btn');
if (dropdown && dropdownBtn) {
    dropdownBtn.addEventListener('click', (e) => { e.stopPropagation(); dropdown.classList.toggle('open'); });
    document.addEventListener('click', () => dropdown.classList.remove('open'));
    document.addEventListener('keydown', (e) => { if (e.key === 'Escape') dropdown.classList.remove('open'); });
    const currentPath = window.location.pathname;
    document.querySelectorAll('.dropdown-item').forEach(item => {
        if (item.getAttribute('href') === currentPath) {
            item.querySelector('.dropdown-name').style.color = 'var(--accent)';
            item.style.background = 'rgba(0, 245, 160, 0.04)';
        }
    });
}

// ── TYPING ──
const typingEl = document.querySelector('.hero-tag .typed');
const text = 'cat contact.md';
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
    entries.forEach(entry => { if (entry.isIntersecting) entry.target.classList.add('visible'); });
}, { threshold: 0.1 });
document.querySelectorAll('.reveal').forEach(el => observer.observe(el));

// ── TERMINAL FEEDBACK ──
const ftBody = document.getElementById('ftBody');

function addTermLine(cls, prefix, msg) {
    const line = document.createElement('div');
    line.className = 'ft-line';
    line.innerHTML = `<span class="ft-dim">${prefix}</span><span class="${cls}">${msg}</span>`;
    ftBody.appendChild(line);
    ftBody.scrollTop = ftBody.scrollHeight;
}

function clearTerminal() {
    ftBody.innerHTML = '';
    addTermLine('ft-acc', '$', './send_message');
    addTermLine('ft-muted', '#', 'En attente de saisie...');
}

// ── COMPTEUR DE CARACTÈRES ──
const messageEl  = document.getElementById('message');
const charCount  = document.getElementById('charCount');
if (messageEl && charCount) {
    messageEl.addEventListener('input', () => {
        const len = messageEl.value.length;
        charCount.textContent = len;
        charCount.style.color = len > 900 ? 'var(--err)' : len > 700 ? '#f5a000' : 'var(--muted)';
    });
}

function setFieldError(inputId, errId, msg) {
    const input = document.getElementById(inputId);
    const err   = document.getElementById(errId);
    if (input) input.classList.add('invalid');
    if (err)   err.textContent = msg;
    return false;
}

function clearFieldError(inputId, errId) {
    const input = document.getElementById(inputId);
    const err   = document.getElementById(errId);
    if (input) input.classList.remove('invalid');
    if (err)   err.textContent = '';
}

// Live validation
document.getElementById('firstname')?.addEventListener('blur', () => {
    const val = document.getElementById('firstname').value.trim();
    if (!val) { setFieldError('name','nameErr','[ERR] Le prenom est requis'); addTermLine('ft-warn','#','Champ nom vide'); }
    else { clearFieldError('name','nameErr'); addTermLine('ft-ok','#','Prenom : OK'); }
});

document.getElementById('lastname')?.addEventListener('blur', () => {
    const val = document.getElementById('lastname').value.trim();
    if (!val) { setFieldError('name','nameErr','[ERR] Le nom de famille est requis'); addTermLine('ft-warn','#','Champ nom vide'); }
    else { clearFieldError('name','nameErr'); addTermLine('ft-ok','#','Nom de famille : OK'); }
});

document.getElementById('mail')?.addEventListener('blur', () => {
    const val = document.getElementById('mail').value.trim();
    if (!val) { setFieldError('email','nameErr','[ERR] Un mail est requis'); addTermLine('ft-warn','#','Champ nom vide'); }
    else { clearFieldError('email','nameErr'); addTermLine('ft-ok','#','email : OK'); }
});

document.getElementById('subject')?.addEventListener('change', () => {
    const val = document.getElementById('subject').value;
    if (!val) { setFieldError('subject','subjectErr','[ERR] Choisis un sujet'); }
    else { clearFieldError('subject','subjectErr'); addTermLine('ft-ok','#','Sujet : ' + val); }
});

document.getElementById('message')?.addEventListener('blur', () => {
    const val = document.getElementById('message').value.trim();
    if (val.length < 10) { setFieldError('message','messageErr','[ERR] Message trop court (min 10 caractères)'); }
    else { clearFieldError('message','messageErr'); addTermLine('ft-ok','#','Message : ' + val.length + ' caractères'); }
});

// ── SOUMISSION DU FORMULAIRE ──
const form       = document.getElementById('contactForm');
const submitBtn  = document.getElementById('submitBtn');
const submitText = submitBtn?.querySelector('.submit-text');
const submitLoad = submitBtn?.querySelector('.submit-loading');
const submitArr  = submitBtn?.querySelector('.submit-arrow');

form?.addEventListener('submit', async (e) => {
    e.preventDefault();

    // Reset erreurs
    ['firstname', 'lastname', 'subject','message', 'email'].forEach(id => clearFieldError(id, id+'Err'));
    document.getElementById('formError').style.display = 'none';

    // Récupère les valeurs
    const firstname    = document.getElementById('firstname').value.trim();
    const lastname    = document.getElementById('lastname').value.trim();
    const subject = document.getElementById('subject').value;
    const message = document.getElementById('message').value.trim();
    const email = document.getElementById('email').value.trim();
    const honey   = form.querySelector('input[name="website"]').value;

    // Honeypot check
    if (honey) return;

    // Validation
    let valid = true;
    if (!firstname)                        { setFieldError('firstname','firstnameErr','[ERR] Le prenom est requis'); valid = false; }
    if (!lastname)                        { setFieldError('lastname','lastnameErr','[ERR] Le nom est requis'); valid = false; }
    if (!subject)                     { setFieldError('subject','subjectErr','[ERR] Choisis un sujet'); valid = false; }
    if (!email)                     { setFieldError('email','subjectErr','[ERR] Un email est requis'); valid = false; }
    if (message.length < 10)          { setFieldError('message','messageErr','[ERR] Message trop court (min 10 caractères)'); valid = false; }

    if (!valid) {
        addTermLine('ft-err','#','[ERR] Validation échouée — corrige les champs');
        return;
    }

    try {
        const res = await fetch('/api/contact', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ firstname, lastname, subject, message, mail })
        });

        const data = await res.json();

        if (res.ok) {
            addTermLine('ft-ok','#','[OK] Message envoyé avec succès');
            addTermLine('ft-ok','#','[OK] Message sauvegardé dans messages.json');

            // Reset du formulaire
            form.reset();
            charCount.textContent = '0';

            // Reset bouton
            submitBtn.disabled       = false;
            submitText.style.display = 'inline';
            submitArr.style.display  = 'inline';
            submitLoad.style.display = 'none';

            // Affiche le succès
            document.getElementById('formSuccess').style.display = 'block';
            form.style.display = 'none';
        }

    } catch (err) {
        addTermLine('ft-err','#','[ERR] ' + err.message);
        document.getElementById('errorMsg').textContent = err.message;
        document.getElementById('formError').style.display = 'block';

        // Reset bouton
        submitBtn.disabled       = false;
        submitText.style.display = 'inline';
        submitArr.style.display  = 'inline';
        submitLoad.style.display = 'none';
    }
});

// ── RESET FORMULAIRE ──
document.getElementById('formReset')?.addEventListener('click', () => {
    form.reset();
    form.style.display = 'block';
    document.getElementById('formSuccess').style.display = 'none';
    charCount.textContent = '0';
    clearTerminal();
});