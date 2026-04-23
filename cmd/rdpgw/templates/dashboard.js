let dashboardUser = null;
let allEntries = [];
const dashboardMessages = window.dashboardMessages || {};
const ENTRY_ICON_ORDER = ['window', 'browser', 'terminal', 'folder', 'database', 'word', 'excel', 'powerpoint', 'outlook'];
const UPLOADED_ENTRY_ICON_PREFIX = 'uploaded:';
const DEFAULT_ENTRY_ICONS = {
    window: 'Window',
    browser: 'Browser',
    terminal: 'Terminal',
    folder: 'File Explorer',
    database: 'Database',
    word: 'Word',
    excel: 'Excel',
    powerpoint: 'PowerPoint',
    outlook: 'Outlook',
};

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

function normalizeEntryIcon(icon) {
    const normalized = String(icon || '').trim().toLowerCase();
    if (uploadedEntryIconId(normalized)) return normalized;
    return ENTRY_ICON_ORDER.includes(normalized) ? normalized : 'window';
}

function uploadedEntryIconId(icon) {
    const normalized = String(icon || '').trim().toLowerCase();
    if (!normalized.startsWith(UPLOADED_ENTRY_ICON_PREFIX)) return '';
    const id = normalized.slice(UPLOADED_ENTRY_ICON_PREFIX.length);
    return /^[a-f0-9]{32}$/.test(id) ? id : '';
}

function entryIconLabel(icon) {
    if (uploadedEntryIconId(icon)) return t('customIconLabel', 'Custom icon');
    const labels = dashboardMessages.entryIcons || DEFAULT_ENTRY_ICONS;
    const normalized = normalizeEntryIcon(icon);
    return labels[normalized] || DEFAULT_ENTRY_ICONS[normalized] || normalized;
}

function entryIconSVG(icon) {
    const uploadedIconId = uploadedEntryIconId(icon);
    if (uploadedIconId) {
        return `<img src="/assets/icons/${uploadedIconId}" alt="" loading="lazy">`;
    }

    switch (normalizeEntryIcon(icon)) {
        case 'browser':
            return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="5" width="18" height="14" rx="2"></rect><path d="M3 9h18"></path><path d="M8 15h8"></path></svg>';
        case 'terminal':
            return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="2"></rect><path d="m7 9 3 3-3 3"></path><path d="M12.5 15H17"></path></svg>';
        case 'folder':
            return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 7.5A2.5 2.5 0 0 1 5.5 5H10l2 2h6.5A2.5 2.5 0 0 1 21 9.5v7A2.5 2.5 0 0 1 18.5 19h-13A2.5 2.5 0 0 1 3 16.5z"></path></svg>';
        case 'database':
            return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="12" cy="6" rx="7" ry="3"></ellipse><path d="M5 6v12c0 1.7 3.1 3 7 3s7-1.3 7-3V6"></path><path d="M5 12c0 1.7 3.1 3 7 3s7-1.3 7-3"></path></svg>';
        case 'word':
            return '<svg viewBox="0 0 24 24" fill="none"><path d="M6 4h9l3 3v13H6z" fill="currentColor" opacity="0.16"></path><path d="M15 4v4h4" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"></path><path d="M8 9.5 9.5 16l2-4.2L13.5 16 15 9.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"></path><path d="M6 4h9l3 3v13H6z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"></path></svg>';
        case 'excel':
            return '<svg viewBox="0 0 24 24" fill="none"><path d="M6 4h9l3 3v13H6z" fill="currentColor" opacity="0.16"></path><path d="M15 4v4h4" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"></path><path d="M8.5 9.5 14.5 15.5M14.5 9.5l-6 6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"></path><path d="M6 4h9l3 3v13H6z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"></path></svg>';
        case 'powerpoint':
            return '<svg viewBox="0 0 24 24" fill="none"><path d="M6 4h9l3 3v13H6z" fill="currentColor" opacity="0.16"></path><path d="M15 4v4h4" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"></path><path d="M9 16v-6h3a2 2 0 1 1 0 4H9" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"></path><path d="M6 4h9l3 3v13H6z" stroke="currentColor" stroke-width="1.8" stroke-linejoin="round"></path></svg>';
        case 'outlook':
            return '<svg viewBox="0 0 24 24" fill="none"><rect x="4" y="6" width="16" height="12" rx="2" fill="currentColor" opacity="0.12"></rect><path d="M6 8.5 12 13l6-4.5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"></path><rect x="4" y="6" width="16" height="12" rx="2" stroke="currentColor" stroke-width="1.8"></rect><circle cx="8" cy="12" r="2.2" stroke="currentColor" stroke-width="1.6"></circle></svg>';
        case 'window':
        default:
            return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="4" width="18" height="16" rx="2"></rect><path d="M3 8h18"></path><path d="M8 4v16"></path></svg>';
    }
}

function createEntryIconElement(icon) {
    const el = document.createElement('span');
    el.className = 'entry-card-icon';
    el.innerHTML = entryIconSVG(icon);
    el.setAttribute('role', 'img');
    el.setAttribute('aria-label', entryIconLabel(icon));
    return el;
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
        <div class="summary-chip">
            <span class="summary-value">${total}</span>
            <span class="summary-label">${t('summaryTotal', 'Total entries')}</span>
        </div>
        <div class="summary-chip">
            <span class="summary-value">${hosts}</span>
            <span class="summary-label">${t('summaryHosts', 'Hosts')}</span>
        </div>
        <div class="summary-chip">
            <span class="summary-value">${templates}</span>
            <span class="summary-label">${t('summaryTemplates', 'Templates')}</span>
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

        const cardIcon = createEntryIconElement(entry.icon);
        header.appendChild(cardIcon);

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
            targetDiv.innerHTML = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="vertical-align: middle; margin-right: 4px;"><path d="M5 12h14"></path><path d="m12 5 7 7-7 7"></path></svg>`;
            targetDiv.appendChild(document.createTextNode(targetText));
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
