const adminMessages = window.adminMessages || {};

function t(key, fallback) {
    const value = adminMessages[key];
    return typeof value === 'string' && value.length > 0 ? value : fallback;
}

function formatMessage(template, value) {
    return template.replace('%s', value);
}

function entryTypeLabel(type) {
    const entryTypes = adminMessages.entryTypes || {};
    return entryTypes[type] || type;
}

function statusLabel(enabled) {
    return enabled ? t('enabledStatus', 'enabled') : t('disabledStatus', 'disabled');
}

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
        empty.textContent = t('noEntriesConfigured', 'No entries configured yet.');
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
        type.textContent = entryTypeLabel(entry.type);
        mainRow.appendChild(type);

        const enabled = document.createElement('span');
        enabled.className = 'entry-meta';
        enabled.textContent = statusLabel(entry.enabled);
        mainRow.appendChild(enabled);

        const description = document.createElement('p');
        description.className = 'entry-description';
        description.textContent = entry.description || t('noDescription', 'No description provided.');

        const target = document.createElement('p');
        target.className = 'entry-meta';
        target.textContent = entry.host || entry.targetHostOverride || (entry.hasUploadedTemplate ? t('uploadedTemplateLabel', 'Uploaded template') : '');

        const groups = document.createElement('p');
        groups.className = 'entry-meta';
        groups.textContent = `${t('allowedGroupsLabel', 'Allowed Groups (comma separated)')}: ${(entry.allowedGroups || []).join(', ')}`;

        const actions = document.createElement('div');
        actions.className = 'entry-actions';

        const nameLabel = document.createElement('label');
        nameLabel.className = 'muted';
        nameLabel.textContent = t('nameLabel', 'Name');
        const nameInput = document.createElement('input');
        nameInput.type = 'text';
        nameInput.value = entry.name || '';
        nameLabel.appendChild(nameInput);

        const descriptionLabel = document.createElement('label');
        descriptionLabel.className = 'muted';
        descriptionLabel.textContent = t('descriptionLabel', 'Description');
        const descriptionInput = document.createElement('input');
        descriptionInput.type = 'text';
        descriptionInput.value = entry.description || '';
        descriptionLabel.appendChild(descriptionInput);

        const groupsLabel = document.createElement('label');
        groupsLabel.className = 'muted';
        groupsLabel.textContent = t('allowedGroupsLabel', 'Allowed Groups (comma separated)');
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
            hostLabel.textContent = t('hostLabel', 'Host (host:port)');
            hostInput = document.createElement('input');
            hostInput.type = 'text';
            hostInput.value = entry.host || '';
            hostLabel.appendChild(hostInput);
            wrapper.appendChild(hostLabel);
        } else {
            const targetLabel = document.createElement('label');
            targetLabel.className = 'muted';
            targetLabel.textContent = t('targetHostOverrideLabel', 'Target Host Override (optional)');
            targetHostInput = document.createElement('input');
            targetHostInput.type = 'text';
            targetHostInput.value = entry.targetHostOverride || '';
            targetLabel.appendChild(targetHostInput);
            wrapper.appendChild(targetLabel);
        }

        const saveButton = document.createElement('button');
        saveButton.type = 'button';
        saveButton.className = 'primary-button';
        saveButton.textContent = t('saveButton', 'Save');

        const toggleButton = document.createElement('button');
        toggleButton.type = 'button';
        toggleButton.className = 'secondary-button';
        toggleButton.dataset.action = 'toggle';
        toggleButton.textContent = t('toggleEnabledButton', 'Toggle Enabled');

        const deleteButton = document.createElement('button');
        deleteButton.type = 'button';
        deleteButton.className = 'danger-button';
        deleteButton.dataset.action = 'delete';
        deleteButton.textContent = t('deleteButton', 'Delete');

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
                setAdminSuccess(formatMessage(t('saveSuccess', 'Saved %s.'), entry.name));
                await loadEntries();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAdminError(`${t('saveEntryErrorPrefix', 'Unable to save entry:')} ${error.message}`);
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
                setAdminSuccess(formatMessage(t('updateSuccess', 'Updated %s.'), entry.name));
                await loadEntries();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAdminError(`${t('updateEntryErrorPrefix', 'Unable to update entry:')} ${error.message}`);
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
                setAdminSuccess(formatMessage(t('deleteSuccess', 'Deleted %s.'), entry.name));
                await loadEntries();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAdminError(`${t('deleteEntryErrorPrefix', 'Unable to delete entry:')} ${error.message}`);
                }
            }
        });

        root.appendChild(wrapper);
    });
}

function renderAdminAuthUsers(users) {
    const root = document.getElementById('adminAuthUsers');
    root.innerHTML = '';

    if (!users || users.length === 0) {
        const empty = document.createElement('p');
        empty.className = 'muted';
        empty.textContent = t('noAuthUsersConfigured', 'No direct auth users configured yet.');
        root.appendChild(empty);
        return;
    }

    users.forEach((user) => {
        const wrapper = document.createElement('article');
        wrapper.className = 'entry-row';

        const mainRow = document.createElement('div');
        mainRow.className = 'entry-row-main';

        const title = document.createElement('strong');
        title.textContent = user.username;
        mainRow.appendChild(title);

        const enabled = document.createElement('span');
        enabled.className = 'entry-meta';
        enabled.textContent = statusLabel(user.enabled);
        mainRow.appendChild(enabled);

        const passwordLabel = document.createElement('label');
        passwordLabel.className = 'muted';
        passwordLabel.textContent = t('newPasswordLabel', 'New Password');
        const passwordInput = document.createElement('input');
        passwordInput.type = 'password';
        passwordInput.placeholder = t('passwordKeepPlaceholder', 'Leave blank to keep current password');
        passwordLabel.appendChild(passwordInput);

        const actions = document.createElement('div');
        actions.className = 'entry-actions';

        const saveButton = document.createElement('button');
        saveButton.type = 'button';
        saveButton.className = 'primary-button';
        saveButton.textContent = t('saveButton', 'Save');

        const toggleButton = document.createElement('button');
        toggleButton.type = 'button';
        toggleButton.className = 'secondary-button';
        toggleButton.textContent = t('toggleEnabledButton', 'Toggle Enabled');

        const deleteButton = document.createElement('button');
        deleteButton.type = 'button';
        deleteButton.className = 'danger-button';
        deleteButton.textContent = t('deleteButton', 'Delete');

        actions.appendChild(saveButton);
        actions.appendChild(toggleButton);
        actions.appendChild(deleteButton);

        wrapper.appendChild(mainRow);
        wrapper.appendChild(passwordLabel);
        wrapper.appendChild(actions);

        saveButton.addEventListener('click', async () => {
            clearAdminError();
            clearAdminSuccess();

            const payload = {enabled: user.enabled};
            if (passwordInput.value) {
                payload.password = passwordInput.value;
            }

            try {
                await adminRequest(`/api/v1/admin/auth-users/${encodeURIComponent(user.username)}`, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload),
                });
                setAdminSuccess(formatMessage(t('saveSuccess', 'Saved %s.'), user.username));
                await loadAuthUsers();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAdminError(`${t('saveAuthUserErrorPrefix', 'Unable to save auth user:')} ${error.message}`);
                }
            }
        });

        toggleButton.addEventListener('click', async () => {
            clearAdminError();
            clearAdminSuccess();

            try {
                await adminRequest(`/api/v1/admin/auth-users/${encodeURIComponent(user.username)}`, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({enabled: !user.enabled}),
                });
                setAdminSuccess(formatMessage(t('updateSuccess', 'Updated %s.'), user.username));
                await loadAuthUsers();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAdminError(`${t('updateAuthUserErrorPrefix', 'Unable to update auth user:')} ${error.message}`);
                }
            }
        });

        deleteButton.addEventListener('click', async () => {
            clearAdminError();
            clearAdminSuccess();

            try {
                await adminRequest(`/api/v1/admin/auth-users/${encodeURIComponent(user.username)}`, {
                    method: 'DELETE',
                });
                setAdminSuccess(formatMessage(t('deleteSuccess', 'Deleted %s.'), user.username));
                await loadAuthUsers();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAdminError(`${t('deleteAuthUserErrorPrefix', 'Unable to delete auth user:')} ${error.message}`);
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
            setAdminError(`${t('loadEntriesErrorPrefix', 'Unable to load entries:')} ${error.message}`);
        }
    }
}

async function loadAuthUsers() {
    try {
        const users = await adminRequest('/api/v1/admin/auth-users');
        renderAdminAuthUsers(users);
    } catch (error) {
        if (error.message !== 'authentication required') {
            setAdminError(`${t('loadAuthUsersErrorPrefix', 'Unable to load auth users:')} ${error.message}`);
        }
    }
}

async function loadCurrentUser() {
    try {
        const user = await adminRequest('/api/v1/user');
        document.getElementById('adminUsername').textContent = user.displayName || user.username || t('unknownUser', 'Unknown User');
    } catch (error) {
        if (error.message !== 'authentication required') {
            setAdminError(`${t('loadUserErrorPrefix', 'Unable to load user information:')} ${error.message}`);
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
            setAdminSuccess(t('createHostEntrySuccess', 'Host entry created.'));
            form.reset();
            await loadEntries();
        } catch (error) {
            if (error.message !== 'authentication required') {
                setAdminError(`${t('createHostEntryErrorPrefix', 'Unable to create host entry:')} ${error.message}`);
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
            setAdminSuccess(t('uploadTemplateEntrySuccess', 'Template entry uploaded.'));
            form.reset();
            await loadEntries();
        } catch (error) {
            if (error.message !== 'authentication required') {
                setAdminError(`${t('uploadTemplateEntryErrorPrefix', 'Unable to upload template entry:')} ${error.message}`);
            }
        }
    });
}

function bindAuthUserForm() {
    const form = document.getElementById('authUserForm');
    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        clearAdminError();
        clearAdminSuccess();

        const data = new FormData(form);
        const payload = {
            username: String(data.get('username') || ''),
            password: String(data.get('password') || ''),
            enabled: data.get('enabled') === 'on',
        };

        try {
            await adminRequest('/api/v1/admin/auth-users', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(payload),
            });
            setAdminSuccess(t('createDirectAuthUserSuccess', 'Direct auth user created.'));
            form.reset();
            const enabled = form.querySelector('input[name="enabled"]');
            if (enabled) {
                enabled.checked = true;
            }
            await loadAuthUsers();
        } catch (error) {
            if (error.message !== 'authentication required') {
                setAdminError(`${t('createAuthUserErrorPrefix', 'Unable to create auth user:')} ${error.message}`);
            }
        }
    });
}

document.addEventListener('DOMContentLoaded', async () => {
    bindHostForm();
    bindTemplateForm();
    bindAuthUserForm();
    await loadCurrentUser();
    await loadEntries();
    await loadAuthUsers();
});
