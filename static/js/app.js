let currentSort = 'added_at';

async function loadData() {
    try {
        const response = await fetch('/data?sort=' + currentSort);
        const data = await response.json();

        document.getElementById('stat-total').textContent = data.stats.total_sites;
        document.getElementById('stat-active').textContent = data.stats.active_sites;
        document.getElementById('stat-broken').textContent = data.stats.broken_sites;
        document.getElementById('stat-dead').textContent = data.stats.dead_sites;
        document.getElementById('stat-clicks').textContent = data.stats.total_clicks;

        if (data.traffic && data.traffic.length > 0) {
            const trafficHtml = '<div class="traffic"><strong>top traffic sources:</strong>' +
                data.traffic.map(t => `<div class="traffic-item">${t.referrer} (${t.count} clicks)</div>`).join('') +
                '</div>';
            document.getElementById('traffic').innerHTML = trafficHtml;
        }

        const sitesHtml = data.sites.length > 0 ? data.sites.map(site => `
            <div class="site">
                <div class="site-header">
                    <div class="site-name">
                        ${site.name} <span class="status-${site.DisplayStatus}">[${site.DisplayStatus}]</span>
                        ${site.manual_status_override ? '<small>(manual override)</small>' : ''}
                    </div>
                    <div class="stat-value">${site.click_count} clicks</div>
                </div>
                <div class="site-url"><a href="${site.url}" target="_blank">${site.url}</a></div>
                ${site.description ? `<div class="site-desc">${site.description}</div>` : ''}
                <div class="site-meta">
                    checked: ${site.LastCheckedStr} | joined: ${site.AddedAtStr}
                    ${site.contact ? ` | contact: ${site.contact}` : ''}
                </div>
            </div>
        `).join('') : '<p>no sites yet</p>';

        document.getElementById('sites').innerHTML = sitesHtml;

    } catch (error) {
        console.error('Error loading data:', error);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    loadData();

    const params = new URLSearchParams(window.location.search);
    const msg = params.get('msg');
    const type = params.get('type');

    if (msg) {
        const messageDiv = document.getElementById('message');
        messageDiv.className = 'message ' + (type || 'error');
        messageDiv.textContent = decodeURIComponent(msg.replace(/\+/g, ' '));

        window.history.replaceState({}, document.title, window.location.pathname);
    }

    document.querySelectorAll('.sort-link').forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            currentSort = e.target.dataset.sort;

            document.querySelectorAll('.sort-link').forEach(l => l.classList.remove('active'));
            e.target.classList.add('active');

            loadData();
        });
    });

    const siteNameInput = document.getElementById('site-name-input');
    const webringCode = document.getElementById('webring-code');
    const { baseURL, siteTitle } = window.webringConfig;

    siteNameInput.addEventListener('input', function(e) {
        const siteName = e.target.value || 'YOUR_SITE_NAME';
        webringCode.textContent = `<a href="${baseURL}/prev/${siteName}">←</a> ( <a href="${baseURL}">${siteTitle}</a> ) <a href="${baseURL}/rand/${siteName}">?</a> <a href="${baseURL}/next/${siteName}">→</a>`;
    });
});
