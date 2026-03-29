function setAdminError(message) {
    const el = document.getElementById('adminError');
    el.textContent = message;
    el.style.display = 'block';
}

function clearAdminError() {
    document.getElementById('adminError').style.display = 'none';
}

function setAdminSuccess(message) {
    const el = document.getElementById('adminSuccess');
    el.textContent = message;
    el.style.display = 'block';
}

function clearAdminSuccess() {
    document.getElementById('adminSuccess').style.display = 'none';
}

function splitGroups(value) {
    return value.split(',').map((part) => part.trim()).filter(Boolean);
}

async function adminRequest(url, init = {}) {
    const response = await fetch(url, init);
    if (!response.ok) {
        if (response.status === 401 || response.status === 403) {
            window.location.href = '/';
            throw new Error('authentication required');
        }
        const body = await response.text();
        throw new Error(body || `request failed (${response.status})`);
    }

    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
        return response.json();
    }
    return null;
}

function renderAdminEntries(entries) {
    const root = document.getElementById('adminEntries');
    root.innerHTML = '';

    if (!entries || entries.length === 0) {
        root.innerHTML = '<p class="muted">No entries configured yet.</p>';
        return;
    }

    entries.forEach((entry) => {
        const wrapper = document.createElement('article');
        wrapper.className = 'entry-row';
        wrapper.innerHTML = `
            <div class="entry-row-main">
                <strong>${entry.name}</strong>
                <span class="entry-meta">${entry.type}</span>
                <span class="entry-meta">${entry.enabled ? 'enabled' : 'disabled'}</span>
            </div>
            <p class="entry-description">${entry.description || 'No description provided.'}</p>
            <p class="entry-meta">${entry.host || entry.targetHostOverride || entry.uploadedTemplatePath || ''}</p>
            <p class="entry-meta">Allowed groups: ${(entry.allowedGroups || []).join(', ')}</p>
            <div class="entry-actions">
                <button type="button" class="secondary-button" data-action="toggle">Toggle Enabled</button>
                <button type="button" class="danger-button" data-action="delete">Delete</button>
            </div>
        `;

        wrapper.querySelector('[data-action="toggle"]').addEventListener('click', async () => {
            clearAdminError();
            clearAdminSuccess();
            try {
                await adminRequest(`/api/v1/admin/entries/${encodeURIComponent(entry.id)}`, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({enabled: !entry.enabled}),
                });
                setAdminSuccess(`Updated ${entry.name}.`);
                await loadEntries();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAdminError(`Unable to update entry: ${error.message}`);
                }
            }
        });

        wrapper.querySelector('[data-action="delete"]').addEventListener('click', async () => {
            clearAdminError();
            clearAdminSuccess();
            try {
                await adminRequest(`/api/v1/admin/entries/${encodeURIComponent(entry.id)}`, {
                    method: 'DELETE',
                });
                setAdminSuccess(`Deleted ${entry.name}.`);
                await loadEntries();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAdminError(`Unable to delete entry: ${error.message}`);
                }
            }
        });

        root.appendChild(wrapper);
    });
}

async function loadEntries() {
    try {
        const entries = await adminRequest('/api/v1/admin/entries');
        renderAdminEntries(entries);
    } catch (error) {
        if (error.message !== 'authentication required') {
            setAdminError(`Unable to load entries: ${error.message}`);
        }
    }
}

async function loadCurrentUser() {
    try {
        const user = await adminRequest('/api/v1/user');
        document.getElementById('adminUsername').textContent = user.displayName || user.username || 'Unknown User';
    } catch (error) {
        if (error.message !== 'authentication required') {
            setAdminError(`Unable to load user information: ${error.message}`);
        }
    }
}

function bindHostForm() {
    const form = document.getElementById('hostForm');
    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        clearAdminError();
        clearAdminSuccess();

        const data = new FormData(form);
        const payload = {
            name: data.get('name') || '',
            description: data.get('description') || '',
            allowedGroups: splitGroups(String(data.get('allowedGroups') || '')),
            host: data.get('host') || '',
        };

        try {
            await adminRequest('/api/v1/admin/entries/host', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(payload),
            });
            setAdminSuccess('Host entry created.');
            form.reset();
            await loadEntries();
        } catch (error) {
            if (error.message !== 'authentication required') {
                setAdminError(`Unable to create host entry: ${error.message}`);
            }
        }
    });
}

function bindTemplateForm() {
    const form = document.getElementById('templateForm');
    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        clearAdminError();
        clearAdminSuccess();

        const formData = new FormData(form);

        try {
            await adminRequest('/api/v1/admin/entries/template', {
                method: 'POST',
                body: formData,
            });
            setAdminSuccess('Template entry uploaded.');
            form.reset();
            await loadEntries();
        } catch (error) {
            if (error.message !== 'authentication required') {
                setAdminError(`Unable to upload template entry: ${error.message}`);
            }
        }
    });
}

document.addEventListener('DOMContentLoaded', async () => {
    bindHostForm();
    bindTemplateForm();
    await loadCurrentUser();
    await loadEntries();
});
