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
    if (response.redirected) {
        window.location.href = '/';
        throw new Error('authentication required');
    }
    if (!response.ok) {
        if (response.status === 401 || response.status === 403) {
            window.location.href = '/';
            throw new Error('authentication required');
        }
        const body = await response.text();
        if (response.status >= 500) {
            throw new Error(`server error (${response.status})`);
        }
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
        const empty = document.createElement('p');
        empty.className = 'muted';
        empty.textContent = 'No entries configured yet.';
        root.appendChild(empty);
        return;
    }

    entries.forEach((entry) => {
        const wrapper = document.createElement('article');
        wrapper.className = 'entry-row';

        const mainRow = document.createElement('div');
        mainRow.className = 'entry-row-main';

        const title = document.createElement('strong');
        title.textContent = entry.name;
        mainRow.appendChild(title);

        const type = document.createElement('span');
        type.className = 'entry-meta';
        type.textContent = entry.type;
        mainRow.appendChild(type);

        const enabled = document.createElement('span');
        enabled.className = 'entry-meta';
        enabled.textContent = entry.enabled ? 'enabled' : 'disabled';
        mainRow.appendChild(enabled);

        const description = document.createElement('p');
        description.className = 'entry-description';
        description.textContent = entry.description || 'No description provided.';

        const target = document.createElement('p');
        target.className = 'entry-meta';
        target.textContent = entry.host || entry.targetHostOverride || (entry.hasUploadedTemplate ? 'Uploaded template' : '');

        const groups = document.createElement('p');
        groups.className = 'entry-meta';
        groups.textContent = `Allowed groups: ${(entry.allowedGroups || []).join(', ')}`;

        const actions = document.createElement('div');
        actions.className = 'entry-actions';

        const nameLabel = document.createElement('label');
        nameLabel.className = 'muted';
        nameLabel.textContent = 'Name';
        const nameInput = document.createElement('input');
        nameInput.type = 'text';
        nameInput.value = entry.name || '';
        nameLabel.appendChild(nameInput);

        const descriptionLabel = document.createElement('label');
        descriptionLabel.className = 'muted';
        descriptionLabel.textContent = 'Description';
        const descriptionInput = document.createElement('input');
        descriptionInput.type = 'text';
        descriptionInput.value = entry.description || '';
        descriptionLabel.appendChild(descriptionInput);

        const groupsLabel = document.createElement('label');
        groupsLabel.className = 'muted';
        groupsLabel.textContent = 'Allowed Groups';
        const groupsInput = document.createElement('input');
        groupsInput.type = 'text';
        groupsInput.value = (entry.allowedGroups || []).join(', ');
        groupsLabel.appendChild(groupsInput);

        wrapper.appendChild(nameLabel);
        wrapper.appendChild(descriptionLabel);
        wrapper.appendChild(groupsLabel);

        let hostInput = null;
        let targetHostInput = null;
        if (entry.type === 'host') {
            const hostLabel = document.createElement('label');
            hostLabel.className = 'muted';
            hostLabel.textContent = 'Host';
            hostInput = document.createElement('input');
            hostInput.type = 'text';
            hostInput.value = entry.host || '';
            hostLabel.appendChild(hostInput);
            wrapper.appendChild(hostLabel);
        } else {
            const targetLabel = document.createElement('label');
            targetLabel.className = 'muted';
            targetLabel.textContent = 'Target Host Override';
            targetHostInput = document.createElement('input');
            targetHostInput.type = 'text';
            targetHostInput.value = entry.targetHostOverride || '';
            targetLabel.appendChild(targetHostInput);
            wrapper.appendChild(targetLabel);
        }

        const saveButton = document.createElement('button');
        saveButton.type = 'button';
        saveButton.className = 'primary-button';
        saveButton.textContent = 'Save';

        const toggleButton = document.createElement('button');
        toggleButton.type = 'button';
        toggleButton.className = 'secondary-button';
        toggleButton.dataset.action = 'toggle';
        toggleButton.textContent = 'Toggle Enabled';

        const deleteButton = document.createElement('button');
        deleteButton.type = 'button';
        deleteButton.className = 'danger-button';
        deleteButton.dataset.action = 'delete';
        deleteButton.textContent = 'Delete';

        actions.appendChild(saveButton);
        actions.appendChild(toggleButton);
        actions.appendChild(deleteButton);

        wrapper.appendChild(mainRow);
        wrapper.appendChild(description);
        wrapper.appendChild(target);
        wrapper.appendChild(groups);
        wrapper.appendChild(actions);

        saveButton.addEventListener('click', async () => {
            clearAdminError();
            clearAdminSuccess();
            const payload = {
                name: nameInput.value,
                description: descriptionInput.value,
                allowedGroups: splitGroups(groupsInput.value),
                enabled: entry.enabled,
            };
            if (hostInput) {
                payload.host = hostInput.value;
            }
            if (targetHostInput) {
                payload.targetHostOverride = targetHostInput.value;
            }
            try {
                await adminRequest(`/api/v1/admin/entries/${encodeURIComponent(entry.id)}`, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload),
                });
                setAdminSuccess(`Saved ${entry.name}.`);
                await loadEntries();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAdminError(`Unable to save entry: ${error.message}`);
                }
            }
        });

        toggleButton.addEventListener('click', async () => {
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

        deleteButton.addEventListener('click', async () => {
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
