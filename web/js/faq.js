/* ══════════════════════════════════════════
   FAQ.JS — page FAQ (/faq)
   Dépend de core.js
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, $$, onReady, onEscape, reducedMotion } = window.Site;

    onReady(() => {
        const items = $$('.faq-item');
        if (!items.length) return;

        const totalEl = $('#totalQ');
        const totalCountEl = $('#totalCount');
        const openCountEl = $('#openCount');
        const progressFill = $('#progressFill');
        const progressBar = $('.progress-bar');
        const searchInput = $('#searchInput');
        const expandBtn = $('#expandAllBtn');

        if (totalEl) totalEl.textContent = items.length;

        // État courant des filtres : catégorie + recherche se combinent
        const state = { cat: 'all', query: '' };
        const opened = new Set();

        // ── OUVRIR / FERMER UNE QUESTION ──
        const setOpen = (item, open) => {
            item.classList.toggle('open', open);
            item.querySelector('.faq-q')?.setAttribute('aria-expanded', String(open));
            if (open) opened.add(item); else opened.delete(item);
        };

        const isVisible = (item) => !item.classList.contains('hidden');

        // ── BARRE DE PROGRESSION ──
        const updateProgress = () => {
            const visible = items.filter(isVisible);
            const openedVisible = visible.filter((i) => opened.has(i)).length;
            const pct = visible.length ? Math.round((openedVisible / visible.length) * 100) : 0;
            if (openCountEl) openCountEl.textContent = openedVisible;
            if (totalCountEl) totalCountEl.textContent = visible.length;
            if (progressFill) progressFill.style.width = `${pct}%`;
            progressBar?.setAttribute('aria-valuenow', String(pct));
        };

        // ── APPLIQUER CATÉGORIE + RECHERCHE ──
        const applyFilters = () => {
            const q = state.query;
            items.forEach((item) => {
                const inCat = state.cat === 'all' || item.dataset.cat === state.cat;
                const text = `${item.querySelector('.faq-text')?.textContent || ''} ${item.querySelector('.faq-a-inner p')?.textContent || ''}`.toLowerCase();
                const inSearch = !q || text.includes(q);
                item.classList.toggle('hidden', !(inCat && inSearch));
            });
            updateProgress();
        };

        // ── ACCORDÉON ──
        items.forEach((item) => {
            item.querySelector('.faq-q')?.addEventListener('click', () => {
                setOpen(item, !item.classList.contains('open'));
                updateProgress();
            });
        });

        // ── FILTRES PAR CATÉGORIE ──
        const catButtons = $$('.cat-btn');
        catButtons.forEach((btn) => {
            btn.setAttribute('aria-pressed', String(btn.classList.contains('active')));
            btn.addEventListener('click', () => {
                catButtons.forEach((b) => {
                    b.classList.toggle('active', b === btn);
                    b.setAttribute('aria-pressed', String(b === btn));
                });
                state.cat = btn.dataset.cat;
                applyFilters();
            });
        });

        // ── RECHERCHE (+ easter egg : un mot secret ouvre sa question) ──
        searchInput?.addEventListener('input', () => {
            state.query = searchInput.value.toLowerCase().trim();
            const secret = items.find((i) => i.dataset.secret && i.dataset.secret === state.query);
            if (secret) {
                items.forEach((i) => i.classList.toggle('hidden', i !== secret));
                setOpen(secret, true);
                secret.scrollIntoView({ behavior: reducedMotion ? 'auto' : 'smooth', block: 'center' });
                updateProgress();
                return;
            }
            applyFilters();
        });

        // ── QUESTION ALÉATOIRE ──
        $('#randomBtn')?.addEventListener('click', () => {
            const visible = items.filter(isVisible);
            if (!visible.length) return;
            const pick = visible[Math.floor(Math.random() * visible.length)];
            items.forEach((i) => setOpen(i, false));
            setOpen(pick, true);
            pick.scrollIntoView({ behavior: reducedMotion ? 'auto' : 'smooth', block: 'center' });
            updateProgress();
        });

        // ── TOUT OUVRIR / TOUT FERMER ──
        let allOpen = false;
        expandBtn?.addEventListener('click', () => {
            allOpen = !allOpen;
            items.filter(isVisible).forEach((i) => setOpen(i, allOpen));
            expandBtn.textContent = allOpen ? 'Tout fermer' : 'Tout ouvrir';
            updateProgress();
        });

        // ── MODE INTERVIEW ──
        const overlay = $('#interviewOverlay');
        const ivBody = $('#interviewBody');
        const questions = items.map((item) => ({
            q: item.querySelector('.faq-text')?.textContent.trim() || '',
            a: item.querySelector('.faq-a-inner p')?.textContent.trim() || '',
        }));
        let ivIndex = 0;
        let ivTimer = null;
        let lastFocus = null;

        const renderQuestion = (idx) => {
            const data = questions[idx];
            if (!data || !ivBody) return;
            clearInterval(ivTimer); // stoppe l'écriture de la question précédente
            ivBody.replaceChildren();

            const head = document.createElement('div');
            head.className = 'iv-question';
            head.textContent = `Question ${idx + 1}/${questions.length}`;

            const text = document.createElement('div');
            text.className = 'iv-text';
            text.textContent = data.q;

            const answer = document.createElement('div');
            answer.className = 'iv-answer';

            ivBody.append(head, text, answer);

            if (reducedMotion) { answer.textContent = data.a; return; }
            const words = data.a.split(' ');
            let i = 0;
            ivTimer = setInterval(() => {
                if (i >= words.length) { clearInterval(ivTimer); return; }
                answer.textContent += (i > 0 ? ' ' : '') + words[i++];
            }, 40);
        };

        const openInterview = () => {
            if (!overlay) return;
            lastFocus = document.activeElement;
            ivIndex = 0;
            overlay.hidden = false;                       // retire l'attribut hidden du HTML
            requestAnimationFrame(() => overlay.classList.add('active'));
            document.body.style.overflow = 'hidden';      // bloque le scroll derrière
            renderQuestion(ivIndex);
            $('#ivNext')?.focus();
        };

        const closeInterview = () => {
            if (!overlay || overlay.hidden) return;
            clearInterval(ivTimer);
            overlay.classList.remove('active');
            document.body.style.overflow = '';
            setTimeout(() => { overlay.hidden = true; }, 300); // laisse le fondu se terminer
            lastFocus?.focus();                           // rend le focus au bouton d'origine
        };

        $('#interviewBtn')?.addEventListener('click', openInterview);
        $('#interviewClose')?.addEventListener('click', closeInterview);
        $('#ivExit')?.addEventListener('click', closeInterview);
        overlay?.addEventListener('click', (e) => { if (e.target === overlay) closeInterview(); });
        $('#ivNext')?.addEventListener('click', () => {
            ivIndex = (ivIndex + 1) % questions.length;
            renderQuestion(ivIndex);
        });
        onEscape(closeInterview);

        // ── INIT ──
        items.forEach((i) => i.querySelector('.faq-q')?.setAttribute('aria-expanded', 'false'));
        updateProgress();
    });
})();