let dashboardUser = null;

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
    grid.innerHTML = '';

    if (!entries || entries.length === 0) {
        empty.hidden = false;
        return;
    }

    empty.hidden = true;
    entries.forEach((entry) => {
        const card = document.createElement('article');
        card.className = 'entry-card';
        card.innerHTML = `
            <div class="entry-meta">${entry.type}</div>
            <h3 class="entry-name">${entry.name}</h3>
            <p class="entry-description">${entry.description || 'No description provided.'}</p>
            <button type="button" class="primary-button">Download RDP</button>
        `;

        card.querySelector('button').addEventListener('click', () => {
            clearDashboardError();
            clearDashboardSuccess();
            window.location.href = entry.downloadUrl;
            setDashboardSuccess(`Downloading ${entry.name}.`);
        });
        grid.appendChild(card);
    });
}

async function loadDashboard() {
    clearDashboardError();
    clearDashboardSuccess();

    try {
        dashboardUser = await apiGetJSON('/api/v1/user');
        const entries = await apiGetJSON('/api/v1/entries');

        const username = dashboardUser.displayName || dashboardUser.username || 'Unknown User';
        document.getElementById('dashboardUsername').textContent = username;
        document.getElementById('userAvatar').textContent = userInitials(username);
        if (dashboardUser.isAdmin) {
            document.getElementById('adminLink').hidden = false;
        }

        renderEntries(entries);
    } catch (error) {
        if (error.message !== 'authentication required') {
            setDashboardError(`Unable to load dashboard: ${error.message}`);
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    loadDashboard();
});
