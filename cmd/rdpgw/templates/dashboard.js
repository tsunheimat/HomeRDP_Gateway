let dashboardUser = null;
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

function setDashboardError(message) {
    const el = document.getElementById('dashboardError');
    el.textContent = message;
    el.style.display = 'block';
}

function clearDashboardError() {
    document.getElementById('dashboardError').style.display = 'none';
}

function setDashboardSuccess(message) {
    const el = document.getElementById('dashboardSuccess');
    el.textContent = message;
    el.style.display = 'block';
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

        const meta = document.createElement('div');
        meta.className = 'entry-meta';
        meta.textContent = entryTypeLabel(entry.type);

        const title = document.createElement('h3');
        title.className = 'entry-name';
        title.textContent = entry.name;

        const description = document.createElement('p');
        description.className = 'entry-description';
        description.textContent = entry.description || t('noDescription', 'No description provided.');

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

        card.appendChild(meta);
        card.appendChild(title);
        card.appendChild(description);
        card.appendChild(button);
        grid.appendChild(card);
    });
}

async function loadDashboard() {
    clearDashboardError();
    clearDashboardSuccess();
    document.getElementById('entriesLoading').hidden = false;
    document.getElementById('entriesEmpty').hidden = true;

    try {
        dashboardUser = await apiGetJSON('/api/v1/user');
        const entries = await apiGetJSON('/api/v1/entries');

        const username = dashboardUser.displayName || dashboardUser.username || t('unknownUser', 'Unknown User');
        document.getElementById('dashboardUsername').textContent = username;
        document.getElementById('userAvatar').textContent = userInitials(username);
        if (dashboardUser.isAdmin) {
            document.getElementById('adminLink').hidden = false;
        }

        renderEntries(entries);
    } catch (error) {
        document.getElementById('entriesLoading').hidden = true;
        if (error.message !== 'authentication required') {
            setDashboardError(`${t('loadErrorPrefix', 'Unable to load dashboard:')} ${error.message}`);
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    loadDashboard();
});
