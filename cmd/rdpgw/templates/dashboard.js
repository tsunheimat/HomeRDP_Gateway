let dashboardUser = null;
let allEntries = [];
const dashboardMessages = window.dashboardMessages || {};

function t(key, fallback) {
    const value = dashboardMessages[key];
    return typeof value === 'string' && value.length > 0 ? value : fallback;
}

function formatMessage(template, value) {
    return template.replace('%s', value);
}

function entryTypeLabel(type) {
    const entryTypes = dashboardMessages.entryTypes || {};
    return entryTypes[type] || type;
}

function setDashboardError(message, showRetry = false) {
    const el = document.getElementById('dashboardError');
    document.getElementById('dashboardErrorText').textContent = message;
    
    const retryBtn = document.getElementById('dashboardRetryBtn');
    if (retryBtn) {
        retryBtn.hidden = !showRetry;
    }
    
    el.style.display = 'flex';
}

function clearDashboardError() {
    document.getElementById('dashboardError').style.display = 'none';
    const retryBtn = document.getElementById('dashboardRetryBtn');
    if (retryBtn) {
        retryBtn.hidden = true;
    }
}

function setDashboardSuccess(message) {
    const el = document.getElementById('dashboardSuccess');
    el.textContent = message;
    el.style.display = 'flex';
}

function clearDashboardSuccess() {
    document.getElementById('dashboardSuccess').style.display = 'none';
}

function userInitials(value) {
    if (!value) {
        return 'U';
    }
    return value.split(' ').map((word) => word.trim().charAt(0)).filter(Boolean).slice(0, 2).join('').toUpperCase();
}

async function apiGetJSON(url) {
    const response = await fetch(url);
    if (response.redirected) {
        window.location.href = '/';
        throw new Error('authentication required');
    }
    if (!response.ok) {
        if (response.status === 401 || response.status === 403) {
            window.location.href = '/';
            throw new Error('authentication required');
        }
        throw new Error(await response.text() || `request failed (${response.status})`);
    }
    return response.json();
}

function updateSummary(entries) {
    const strip = document.getElementById('summaryStrip');
    if (!entries || entries.length === 0) {
        strip.hidden = true;
        document.getElementById('dashboardControls').hidden = true;
        return;
    }
    
    const hosts = entries.filter(e => e.type === 'host').length;
    const templates = entries.filter(e => e.type === 'template').length;
    const total = entries.length;

    strip.innerHTML = `
        <div style="font-size: 0.9rem; color: var(--muted-foreground)">
            <strong style="color: var(--foreground)">${total}</strong> ${t('summaryTotal', 'Total entries')}
        </div>
        <div style="font-size: 0.9rem; color: var(--muted-foreground)">
            <strong style="color: var(--foreground)">${hosts}</strong> ${t('summaryHosts', 'Hosts')}
        </div>
        <div style="font-size: 0.9rem; color: var(--muted-foreground)">
            <strong style="color: var(--foreground)">${templates}</strong> ${t('summaryTemplates', 'Templates')}
        </div>
    `;
    strip.hidden = false;
    document.getElementById('dashboardControls').hidden = false;
}

function renderEntries(entries) {
    const grid = document.getElementById('entriesGrid');
    const empty = document.getElementById('entriesEmpty');
    const loading = document.getElementById('entriesLoading');
    grid.innerHTML = '';
    loading.hidden = true;

    if (!entries || entries.length === 0) {
        empty.hidden = false;
        return;
    }

    empty.hidden = true;
    entries.forEach((entry) => {
        const card = document.createElement('article');
        card.className = 'entry-card';

        const header = document.createElement('header');
        header.className = 'entry-header';

        const titleInfo = document.createElement('div');
        titleInfo.className = 'entry-title-info';

        const title = document.createElement('h3');
        title.className = 'entry-name';
        title.textContent = entry.name;

        const meta = document.createElement('span');
        meta.className = 'entry-meta-badge';
        meta.textContent = entryTypeLabel(entry.type);

        titleInfo.appendChild(title);
        titleInfo.appendChild(meta);
        header.appendChild(titleInfo);

        const targetDiv = document.createElement('div');
        targetDiv.className = 'entry-target';
        
        let targetText = '';
        if (entry.target) {
            targetText = entry.target;
        } else if (entry.hasUploadedTemplate) {
            targetText = t('uploadedTemplateLabel', 'Uploaded template');
        }
        
        if (targetText) {
            targetDiv.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align: middle; margin-right: 4px;"><path d="M5 12h14"></path><path d="m12 5 7 7-7 7"></path></svg>${targetText}`;
        } else {
            targetDiv.innerHTML = '&nbsp;';
        }

        const description = document.createElement('p');
        description.className = 'entry-description';
        description.textContent = entry.description || t('noDescription', 'No description provided.');

        const body = document.createElement('div');
        body.className = 'entry-body';
        body.appendChild(targetDiv);
        body.appendChild(description);

        const button = document.createElement('button');
        button.type = 'button';
        button.className = 'primary-button';
        button.textContent = t('downloadButton', 'Download RDP');

        button.addEventListener('click', () => {
            clearDashboardError();
            clearDashboardSuccess();
            window.location.href = entry.downloadUrl;
            setDashboardSuccess(formatMessage(t('downloading', 'Downloading %s.'), entry.name));
        });

        card.appendChild(header);
        card.appendChild(body);
        card.appendChild(button);
        grid.appendChild(card);
    });
}

async function loadDashboard() {
    clearDashboardError();
    clearDashboardSuccess();
    document.getElementById('entriesLoading').hidden = false;
    document.getElementById('entriesEmpty').hidden = true;
    document.getElementById('entriesGrid').innerHTML = '';

    try {
        dashboardUser = await apiGetJSON('/api/v1/user');
        const entries = await apiGetJSON('/api/v1/entries');
        allEntries = entries;

        const username = dashboardUser.displayName || dashboardUser.username || t('unknownUser', 'Unknown User');
        document.getElementById('dashboardUsername').textContent = username;
        document.getElementById('userAvatar').textContent = userInitials(username);
        
        if (dashboardUser.isAdmin) {
            document.getElementById('adminLink').hidden = false;
        }

        updateSummary(entries);
        
        // Apply filter if one exists (e.g. on retry)
        const searchInput = document.getElementById('entrySearch');
        if (searchInput && searchInput.value) {
            const query = searchInput.value.toLowerCase();
            renderEntries(allEntries.filter(entry => 
                entry.name.toLowerCase().includes(query) || 
                (entry.description && entry.description.toLowerCase().includes(query))
            ));
        } else {
            renderEntries(entries);
        }
    } catch (error) {
        document.getElementById('entriesLoading').hidden = true;
        if (error.message !== 'authentication required') {
            setDashboardError(`${t('loadErrorPrefix', 'Unable to load dashboard:')} ${error.message}`, true);
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    loadDashboard();
    
    const searchInput = document.getElementById('entrySearch');
    if (searchInput) {
        searchInput.addEventListener('input', (e) => {
            const query = e.target.value.toLowerCase();
            const filtered = allEntries.filter(entry => 
                entry.name.toLowerCase().includes(query) || 
                (entry.description && entry.description.toLowerCase().includes(query))
            );
            renderEntries(filtered);
        });
    }

    const retryBtn = document.getElementById('dashboardRetryBtn');
    if (retryBtn) {
        retryBtn.addEventListener('click', () => {
            loadDashboard();
        });
    }
});
