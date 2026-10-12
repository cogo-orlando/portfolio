/* ══════════════════════════════════════════
   STATUS.JS — page Status (/status)
   Dépend de core.js
══════════════════════════════════════════ */
(function () {
    'use strict';
    const { $, $$, onReady, fetchJSON, t, lang, formatUptime } = window.Site;

    const set = (id, value) => {
        const el = document.getElementById(id);
        if (el) el.textContent = value;
    };
    const pad = (n) => String(n).padStart(2, '0');

    // Codes météo Open-Meteo → [français, anglais]
    const WEATHER = {
        0: ['Ciel dégagé', 'Clear sky'], 1: ['Généralement dégagé', 'Mainly clear'],
        2: ['Partiellement nuageux', 'Partly cloudy'], 3: ['Couvert', 'Overcast'],
        45: ['Brouillard', 'Fog'], 48: ['Brouillard givrant', 'Freezing fog'],
        51: ['Bruine légère', 'Light drizzle'], 53: ['Bruine', 'Drizzle'], 55: ['Bruine forte', 'Heavy drizzle'],
        56: ['Bruine verglaçante', 'Freezing drizzle'], 57: ['Bruine verglaçante forte', 'Heavy freezing drizzle'],
        61: ['Pluie légère', 'Light rain'], 63: ['Pluie modérée', 'Moderate rain'], 65: ['Pluie forte', 'Heavy rain'],
        66: ['Pluie verglaçante', 'Freezing rain'], 67: ['Pluie verglaçante forte', 'Heavy freezing rain'],
        71: ['Neige légère', 'Light snow'], 73: ['Neige modérée', 'Moderate snow'], 75: ['Neige forte', 'Heavy snow'],
        77: ['Grains de neige', 'Snow grains'],
        80: ['Averses légères', 'Light showers'], 81: ['Averses modérées', 'Moderate showers'], 82: ['Averses violentes', 'Violent showers'],
        85: ['Averses de neige', 'Snow showers'], 86: ['Fortes averses de neige', 'Heavy snow showers'],
        95: ['Orage', 'Thunderstorm'], 96: ['Orage avec grêle', 'Thunderstorm with hail'], 99: ['Orage avec forte grêle', 'Thunderstorm with heavy hail'],
    };

    onReady(() => {
        // ── HEURE DE TOULOUSE (et pas celle du visiteur) ──
        const clock = () => set('localTime', new Date().toLocaleTimeString(t('fr-FR', 'en-GB'), { timeZone: 'Europe/Paris' }));

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
            lastUpdate.textContent = now.toLocaleString(t('fr-FR', 'en-GB'), {
                day: '2-digit', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit',
                timeZone: 'Europe/Paris',
            });
        }

        // ── MÉTRIQUES GO depuis /health ──
        const loadMetrics = () => fetchJSON('/health')
            .then((d) => {
                set('m-goroutines', d.goroutines ?? '—');
                set('m-uptime', formatUptime(d.uptime));
                set('m-alloc', d.memory_mb != null ? `${d.memory_mb} ${t('Mo', 'MB')}` : '—');
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
            operational: { text: t('tout est opérationnel', 'all systems operational'), cls: 'ok' },
            degraded:    { text: t('service dégradé', 'degraded service'), cls: 'warn' },
            down:        { text: t('service indisponible', 'service unavailable'), cls: 'down' },
            pending:     { text: t('vérification…', 'checking…'), cls: 'warn' },
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
                row.title = `${t('Vérifié à', 'Checked at')} ${new Date(r.checked_at).toLocaleTimeString(t('fr-FR', 'en-GB'), { timeZone: 'Europe/Paris' })}`;

                const dot = span(`page-dot ${r.ok ? 'online' : 'offline'}`, '');
                dot.setAttribute('aria-hidden', 'true');

                row.append(
                    dot,
                    span('page-name', r.path),
                    span('page-meta', `${Math.round(r.latency_ms)} ms · ${r.uptime}%`),
                    span(`page-status ${r.ok ? 'ok' : 'down'}`, r.ok ? 'online' : `${t('erreur', 'error')} ${r.status}`),
                );
                return row;
            });
            routesBox.replaceChildren(...rows);
        };

        const loadRoutes = () => fetchJSON('/api/status')
            .then(renderRoutes)
            .catch(() => {
                if (!summary) return;
                summary.textContent = t('état indisponible', 'status unavailable');
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
                set('weatherDesc', WEATHER[c.weather_code]?.[lang === 'en' ? 1 : 0] || t('Conditions inconnues', 'Unknown conditions'));
                set('weatherFeels', `${Math.round(c.apparent_temperature)}°C`);
                set('weatherHumid', `${c.relative_humidity_2m}%`);
                set('weatherWind', `${Math.round(c.wind_speed_10m)} km/h`);
            })
            .catch(() => set('weatherDesc', t('Données indisponibles', 'Data unavailable')));

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
