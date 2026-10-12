/* ══════════════════════════════════════════
   CONTACT.JS — formulaire de contact (/contact)
   Dépend de core.js
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, onReady, t } = window.Site;

    // Champs du formulaire : id de l'input → id du message d'erreur + règle de validation
    const FIELDS = {
        firstname: { err: 'firstnameErr', label: t('Prénom', 'First name'), check: (v) => v ? '' : t('Le prénom est requis', 'First name is required') },
        lastname:  { err: 'lastnameErr',  label: t('Nom', 'Last name'),     check: (v) => v ? '' : t('Le nom est requis', 'Last name is required') },
        email:     { err: 'mailErr',      label: 'Email',                  check: (v) => !v ? t('Un email est requis', 'Email is required')
                                                                              : /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test(v) ? '' : t('Adresse email invalide', 'Invalid email address') },
        subject:   { err: 'subjectErr',   label: t('Sujet', 'Subject'),    check: (v) => v ? '' : t('Choisis un sujet', 'Please choose a subject') },
        message:   { err: 'messageErr',   label: 'Message',                check: (v) => v.length >= 10 ? '' : t('Message trop court (min 10 caractères)', 'Message too short (10 characters minimum)') },
    };

    onReady(() => {
        const form = $('#contactForm');
        if (!form) return;

        const ftBody = $('#ftBody');
        const charCount = $('#charCount');
        const messageEl = $('#message');
        const submitBtn = $('#submitBtn');
        const submitText = submitBtn?.querySelector('.submit-text');
        const submitLoad = submitBtn?.querySelector('.submit-loading');
        const submitArr = submitBtn?.querySelector('.submit-arrow');
        const successBox = $('#formSuccess');
        const errorBox = $('#formError');
        const errorMsg = $('#errorMsg');

        // ── TERMINAL : textContent uniquement → aucune injection HTML possible ──
        const addTermLine = (cls, prefix, msg) => {
            if (!ftBody) return;
            const line = document.createElement('div');
            line.className = 'ft-line';
            const p = document.createElement('span');
            p.className = 'ft-dim';
            p.textContent = prefix;
            const m = document.createElement('span');
            m.className = cls;
            m.textContent = msg;
            line.append(p, m);
            ftBody.append(line);
            ftBody.scrollTop = ftBody.scrollHeight;
        };

        const resetTerminal = () => {
            if (!ftBody) return;
            ftBody.replaceChildren();
            addTermLine('ft-acc', '$', './send_message');
            addTermLine('ft-muted', '#', t('En attente de saisie...', 'Waiting for input...'));
        };

        // ── ERREURS PAR CHAMP ──
        const setError = (id, msg) => {
            const input = document.getElementById(id);
            const err = document.getElementById(FIELDS[id].err);
            input?.classList.toggle('invalid', Boolean(msg));
            input?.setAttribute('aria-invalid', String(Boolean(msg)));
            if (err) err.textContent = msg ? `[ERR] ${msg}` : '';
        };

        const validate = (id) => {
            const value = (document.getElementById(id)?.value || '').trim();
            const msg = FIELDS[id].check(value);
            setError(id, msg);
            return msg;
        };

        // ── COMPTEUR DE CARACTÈRES ──
        const updateCount = () => {
            if (!charCount || !messageEl) return;
            const len = messageEl.value.length;
            charCount.textContent = len;
            charCount.style.color = len > 900 ? 'var(--err)' : len > 700 ? '#fbbf24' : '';
        };
        messageEl?.addEventListener('input', updateCount);

        // ── VALIDATION EN DIRECT (en quittant un champ) ──
        Object.keys(FIELDS).forEach((id) => {
            const input = document.getElementById(id);
            const event = input?.tagName === 'SELECT' ? 'change' : 'blur';
            input?.addEventListener(event, () => {
                const msg = validate(id);
                if (msg) addTermLine('ft-warn', '#', `${FIELDS[id].label} : ${msg}`);
                else addTermLine('ft-ok', '#', `${FIELDS[id].label} : OK`);
            });
            // L'erreur disparaît dès que l'utilisateur corrige
            input?.addEventListener('input', () => { if (input.classList.contains('invalid')) validate(id); });
        });

        // ── ÉTAT DU BOUTON ──
        const setLoading = (loading) => {
            if (!submitBtn) return;
            submitBtn.disabled = loading;
            submitBtn.setAttribute('aria-busy', String(loading));
            if (submitText) submitText.style.display = loading ? 'none' : 'inline';
            if (submitArr) submitArr.style.display = loading ? 'none' : 'inline';
            if (submitLoad) submitLoad.style.display = loading ? 'flex' : 'none';
        };

        // ── ENVOI VERS FORMSPREE ──
        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            if (errorBox) errorBox.style.display = 'none';

            // Honeypot : un bot a rempli le champ caché → on fait semblant que tout va bien
            if (form.querySelector('input[name="_gotcha"]')?.value) {
                form.style.display = 'none';
                if (successBox) successBox.style.display = 'block';
                return;
            }

            const errors = Object.keys(FIELDS).filter((id) => validate(id));
            if (errors.length) {
                addTermLine('ft-err', '#', t('[ERR] Validation échouée — corrige les champs', '[ERR] Validation failed — please fix the fields'));
                document.getElementById(errors[0])?.focus(); // on emmène l'utilisateur au premier champ en erreur
                return;
            }

            setLoading(true);
            addTermLine('ft-warn', '$', t('Connexion à Formspree...', 'Connecting to Formspree...'));

            const controller = new AbortController();
            const timer = setTimeout(() => controller.abort(), 15000);

            try {
                const res = await fetch(form.action, {
                    method: 'POST',
                    headers: { Accept: 'application/json' },
                    body: new FormData(form),
                    signal: controller.signal,
                });

                if (!res.ok) {
                    // Formspree renvoie le détail : { errors: [{ message: "..." }] }
                    const data = await res.json().catch(() => ({}));
                    const detail = data.errors?.map((x) => x.message).join(', ');
                    throw new Error(detail || `${t('Erreur serveur', 'Server error')} (${res.status})`);
                }

                addTermLine('ft-ok', '#', t('[OK] Message envoyé avec succès', '[OK] Message sent successfully'));
                form.reset();
                updateCount();
                form.style.display = 'none';
                if (successBox) successBox.style.display = 'block';
            } catch (err) {
                const msg = err.name === 'AbortError'
                    ? t('Le serveur met trop de temps à répondre. Réessaie dans un instant.', 'The server is taking too long to respond. Please try again shortly.')
                    : err.message || t('Une erreur est survenue. Réessaie.', 'Something went wrong. Please try again.');
                addTermLine('ft-err', '#', `[ERR] ${msg}`);
                if (errorMsg) errorMsg.textContent = msg;
                if (errorBox) errorBox.style.display = 'block';
            } finally {
                clearTimeout(timer);
                setLoading(false);
            }
        });

        // ── ENVOYER UN AUTRE MESSAGE ──
        $('#formReset')?.addEventListener('click', () => {
            form.reset();
            Object.keys(FIELDS).forEach((id) => setError(id, ''));
            updateCount();
            form.style.display = 'block';
            if (successBox) successBox.style.display = 'none';
            resetTerminal();
            document.getElementById('firstname')?.focus();
        });
    });
})();
