/* ══════════════════════════════════════════
   STATUS.JS — page Status (/status)
   Dépend de core.js
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, $$, onReady, fetchJSON } = window.Site;

    const set = (id, value) => {
        const el = document.getElementById(id);
        if (el) el.textContent = value;
    };
    const pad = (n) => String(n).padStart(2, '0');

    // "26h3m4.5s" (format Go) → "1j 2h" / "3h 12min" / "45min"
    const formatUptime = (raw) => {
        if (typeof raw !== 'string') return '—';
        const h = Number(raw.match(/(\d+)h/)?.[1] || 0);
        const m = Number(raw.match(/(\d+)m(?!s)/)?.[1] || 0);
        const days = Math.floor(h / 24);
        if (days > 0) return `${days}j ${h % 24}h`;
        if (h > 0) return `${h}h ${m}min`;
        return `${m}min`;
    };

    const WEATHER = {
        0: 'Ciel dégagé', 1: 'Généralement dégagé', 2: 'Partiellement nuageux', 3: 'Couvert',
        45: 'Brouillard', 48: 'Brouillard givrant',
        51: 'Bruine légère', 53: 'Bruine', 55: 'Bruine forte',
        56: 'Bruine verglaçante', 57: 'Bruine verglaçante forte',
        61: 'Pluie légère', 63: 'Pluie modérée', 65: 'Pluie forte',
        66: 'Pluie verglaçante', 67: 'Pluie verglaçante forte',
        71: 'Neige légère', 73: 'Neige modérée', 75: 'Neige forte', 77: 'Grains de neige',
        80: 'Averses légères', 81: 'Averses modérées', 82: 'Averses violentes',
        85: 'Averses de neige', 86: 'Fortes averses de neige',
        95: 'Orage', 96: 'Orage avec grêle', 99: 'Orage avec forte grêle',
    };

    onReady(() => {
        // ── HEURE DE TOULOUSE (et pas celle du visiteur) ──
        const clock = () => set('localTime', new Date().toLocaleTimeString('fr-FR', { timeZone: 'Europe/Paris' }));

        // ── DURÉE DE LA VISITE ──
        const pageStart = Date.now();
        const session = () => {
            const diff = Math.floor((Date.now() - pageStart) / 1000);
            set('uptime', `${pad(Math.floor(diff / 3600))}:${pad(Math.floor((diff % 3600) / 60))}:${pad(diff % 60)}`);
        };

        clock(); session();
        setInterval(() => { clock(); session(); }, 1000);

        // ── DATE D'AFFICHAGE ──
        const lastUpdate = $('#lastUpdate');
        if (lastUpdate) {
            const now = new Date();
            lastUpdate.dateTime = now.toISOString();
            lastUpdate.textContent = now.toLocaleString('fr-FR', {
                day: '2-digit', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit',
                timeZone: 'Europe/Paris',
            });
        }

        // ── MÉTRIQUES GO depuis /health ──
        const loadMetrics = () => fetchJSON('/health')
            .then((d) => {
                set('m-goroutines', d.goroutines ?? '—');
                set('m-uptime', formatUptime(d.uptime));
                set('m-alloc', d.memory_mb != null ? `${d.memory_mb} Mo` : '—');
                set('m-gc', d.gc_cycles ?? '—');
                set('m-runtime', d.go_version ?? '—');
            })
            .catch((err) => console.warn('métriques Go :', err.message));

        loadMetrics();
        // Rafraîchit toutes les 30 s, mais seulement quand l'onglet est visible
        setInterval(() => { if (!document.hidden) loadMetrics(); }, 30000);

        // ── ÉTAT RÉEL DES PAGES (/api/status, vérifié par le serveur toutes les minutes) ──
        const routesBox = $('#routesStatus');
        const summary = $('#routesSummary');
        const OVERALL = {
            operational: { text: 'tout est opérationnel', cls: 'ok' },
            degraded:    { text: 'service dégradé',       cls: 'warn' },
            down:        { text: 'service indisponible',  cls: 'down' },
            pending:     { text: 'vérification…',         cls: 'warn' },
        };

        const span = (cls, text) => {
            const el = document.createElement('span');
            el.className = cls;
            el.textContent = text;
            return el;
        };

        const renderRoutes = (snap) => {
            if (summary) {
                const o = OVERALL[snap.overall] || OVERALL.pending;
                summary.textContent = o.text;
                summary.className = `page-status ${o.cls} routes-summary`;
            }
            if (!routesBox || !snap.routes?.length) return;

            const rows = snap.routes.map((r) => {
                const row = document.createElement('div');
                row.className = 'page-row';
                row.setAttribute('role', 'listitem');
                row.title = `Vérifié à ${new Date(r.checked_at).toLocaleTimeString('fr-FR', { timeZone: 'Europe/Paris' })}`;

                const dot = span(`page-dot ${r.ok ? 'online' : 'offline'}`, '');
                dot.setAttribute('aria-hidden', 'true');

                row.append(
                    dot,
                    span('page-name', r.path),
                    span('page-meta', `${Math.round(r.latency_ms)} ms · ${r.uptime}%`),
                    span(`page-status ${r.ok ? 'ok' : 'down'}`, r.ok ? 'online' : `erreur ${r.status}`),
                );
                return row;
            });
            routesBox.replaceChildren(...rows);
        };

        const loadRoutes = () => fetchJSON('/api/status')
            .then(renderRoutes)
            .catch(() => {
                if (!summary) return;
                summary.textContent = 'état indisponible';
                summary.className = 'page-status warn routes-summary';
            });

        loadRoutes();
        setInterval(() => { if (!document.hidden) loadRoutes(); }, 60000);

        // ── MÉTÉO TOULOUSE (Open-Meteo) ──
        const url = 'https://api.open-meteo.com/v1/forecast?latitude=43.6047&longitude=1.4442'
            + '&current=temperature_2m,apparent_temperature,relative_humidity_2m,wind_speed_10m,weather_code'
            + '&wind_speed_unit=kmh&timezone=Europe/Paris';
        fetchJSON(url, { timeout: 8000 })
            .then(({ current: c }) => {
                set('weatherTemp', `${Math.round(c.temperature_2m)}°C`);
                set('weatherDesc', WEATHER[c.weather_code] || 'Conditions inconnues');
                set('weatherFeels', `${Math.round(c.apparent_temperature)}°C`);
                set('weatherHumid', `${c.relative_humidity_2m}%`);
                set('weatherWind', `${Math.round(c.wind_speed_10m)} km/h`);
            })
            .catch(() => set('weatherDesc', 'Données indisponibles'));

        // ── OBJECTIF : % calculé depuis les étapes affichées dans le HTML ──
        const steps = $$('.goal-block .sys-row');
        const done = steps.filter((row) => row.querySelector('.done')).length;
        const pct = steps.length ? Math.round((done / steps.length) * 100) : 0;
        setTimeout(() => {
            const fill = $('#goalFill');
            if (fill) fill.style.width = `${pct}%`;
            set('goalPct', `${pct}%`);
            $('.goal-bar')?.setAttribute('aria-valuenow', String(pct));
        }, 600);

        // ── VISITES : la ligne est masquée tant que le compteur n'est pas en place ──
        const visitRow = $('#visitCount')?.closest('.sys-row');
        fetchJSON('/api/visits')
            .then((d) => {
                if (d.visits > 0) set('visitCount', d.visits);
                else if (visitRow) visitRow.hidden = true;
            })
            .catch(() => { if (visitRow) visitRow.hidden = true; });
    });
})();