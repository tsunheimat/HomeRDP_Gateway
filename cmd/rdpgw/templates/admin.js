const adminMessages = window.adminMessages || {};
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
let uploadedEntryIcons = [];
const APP_NAME_ALIASES = {
    chrome: 'Google Chrome',
    cmd: 'Command Prompt',
    dbeaver: 'DBeaver',
    edge: 'Microsoft Edge',
    excel: 'Excel',
    explorer: 'File Explorer',
    firefox: 'Firefox',
    iexplore: 'Internet Explorer',
    msedge: 'Microsoft Edge',
    outlook: 'Outlook',
    powerpnt: 'PowerPoint',
    powerpoint: 'PowerPoint',
    powershell: 'PowerShell',
    pwsh: 'PowerShell',
    sqlcmd: 'SQLCMD',
    ssms: 'SQL Server Management Studio',
    terminal: 'Terminal',
    winword: 'Word',
    word: 'Word',
    wt: 'Windows Terminal',
};

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

function entryIconLabels() {
    const labels = adminMessages.entryIcons;
    if (labels && typeof labels === 'object' && Object.keys(labels).length > 0) {
        return labels;
    }
    return DEFAULT_ENTRY_ICONS;
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

function uploadedEntryIconFor(icon) {
    const id = uploadedEntryIconId(icon);
    if (!id) return null;
    const uploadedIcon = uploadedEntryIcons.find((candidate) => candidate.id === id);
    return {
        id,
        originalName: uploadedIcon ? uploadedIcon.originalName : t('customIconLabel', 'Custom icon'),
        url: `/assets/icons/${id}`,
    };
}

function entryIconLabel(icon) {
    const normalized = normalizeEntryIcon(icon);
    const uploadedIcon = uploadedEntryIconFor(normalized);
    if (uploadedIcon) return uploadedIcon.originalName || t('customIconLabel', 'Custom icon');
    const labels = entryIconLabels();
    return labels[normalized] || DEFAULT_ENTRY_ICONS[normalized] || normalized;
}

function entryIconSVG(icon) {
    const uploadedIcon = uploadedEntryIconFor(icon);
    if (uploadedIcon) {
        return `<img src="${uploadedIcon.url}" alt="" loading="lazy">`;
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

function setEntryIconOptions(icons) {
    uploadedEntryIcons = (Array.isArray(icons) ? icons : [])
        .filter((icon) => icon && uploadedEntryIconId(`${UPLOADED_ENTRY_ICON_PREFIX}${icon.id}`))
        .map((icon) => ({
            id: String(icon.id).trim().toLowerCase(),
            originalName: icon.originalName || t('customIconLabel', 'Custom icon'),
            url: `/assets/icons/${String(icon.id).trim().toLowerCase()}`,
        }));
    initializeEntryIconSelects();
}

function createEntryIconElement(icon, className, decorative = false) {
    const el = document.createElement('span');
    el.className = className;
    el.innerHTML = entryIconSVG(icon);
    if (decorative) {
        el.setAttribute('aria-hidden', 'true');
    } else {
        el.setAttribute('role', 'img');
        el.setAttribute('aria-label', entryIconLabel(icon));
    }
    return el;
}

function populateEntryIconSelect(select, selectedIcon = 'window') {
    if (!select) return;

    const normalizedSelected = normalizeEntryIcon(selectedIcon);
    const labels = entryIconLabels();
    select.innerHTML = '';

    ENTRY_ICON_ORDER.forEach((icon) => {
        const option = document.createElement('option');
        option.value = icon;
        option.textContent = labels[icon] || DEFAULT_ENTRY_ICONS[icon] || icon;
        option.selected = icon === normalizedSelected;
        select.appendChild(option);
    });

    uploadedEntryIcons.forEach((icon) => {
        const value = `${UPLOADED_ENTRY_ICON_PREFIX}${icon.id}`;
        const option = document.createElement('option');
        option.value = value;
        option.textContent = icon.originalName || t('customIconLabel', 'Custom icon');
        option.selected = value === normalizedSelected;
        select.appendChild(option);
    });

    const selectedUploadedIcon = uploadedEntryIconFor(normalizedSelected);
    if (selectedUploadedIcon && !uploadedEntryIcons.some((icon) => `${UPLOADED_ENTRY_ICON_PREFIX}${icon.id}` === normalizedSelected)) {
        const option = document.createElement('option');
        option.value = normalizedSelected;
        option.textContent = selectedUploadedIcon.originalName || t('customIconLabel', 'Custom icon');
        option.selected = true;
        select.appendChild(option);
    }
}

function buildEntryIconSelect(selectedIcon = 'window') {
    const select = document.createElement('select');
    populateEntryIconSelect(select, selectedIcon);
    return select;
}

function initializeEntryIconSelects() {
    document.querySelectorAll('[data-entry-icon-select]').forEach((select) => {
        populateEntryIconSelect(select, select.value || 'window');
    });
}

function statusLabel(enabled) {
    return enabled ? t('enabledStatus', 'enabled') : t('disabledStatus', 'disabled');
}

function statusBadgeClass(enabled) {
    return enabled ? 'entry-status-badge is-enabled' : 'entry-status-badge is-disabled';
}

function toggleEnabledButtonLabel(enabled) {
    return enabled ? t('disableButton', 'Disable') : t('enableButton', 'Enable');
}

function setEntriesError(message) {
    const el = document.getElementById('entriesError');
    if (el) {
        el.textContent = message;
        el.style.display = 'block';
    }
}

function clearEntriesError() {
    const el = document.getElementById('entriesError');
    if (el) el.style.display = 'none';
}

function setEntriesSuccess(message) {
    const el = document.getElementById('entriesSuccess');
    if (el) {
        el.textContent = message;
        el.style.display = 'block';
    }
}

function clearEntriesSuccess() {
    const el = document.getElementById('entriesSuccess');
    if (el) el.style.display = 'none';
}

function setAuthUsersError(message) {
    const el = document.getElementById('authUsersError');
    if (el) {
        el.textContent = message;
        el.style.display = 'block';
    }
}

function clearAuthUsersError() {
    const el = document.getElementById('authUsersError');
    if (el) el.style.display = 'none';
}

function setAuthUsersSuccess(message) {
    const el = document.getElementById('authUsersSuccess');
    if (el) {
        el.textContent = message;
        el.style.display = 'block';
    }
}

function clearAuthUsersSuccess() {
    const el = document.getElementById('authUsersSuccess');
    if (el) el.style.display = 'none';
}

function setBrandingError(message) {
    const el = document.getElementById('brandingError');
    if (el) {
        el.textContent = message;
        el.style.display = 'block';
    }
}

function clearBrandingError() {
    const el = document.getElementById('brandingError');
    if (el) el.style.display = 'none';
}

function setBrandingSuccess(message) {
    const el = document.getElementById('brandingSuccess');
    if (el) {
        el.textContent = message;
        el.style.display = 'block';
    }
}

function clearBrandingSuccess() {
    const el = document.getElementById('brandingSuccess');
    if (el) el.style.display = 'none';
}

function splitGroups(value) {
    return value.split(',').map((part) => part.trim()).filter(Boolean);
}

function normalizeAppToken(value) {
    const cleaned = String(value || '').trim().replace(/^\|\|/, '');
    if (!cleaned) {
        return '';
    }

    const lastSegment = cleaned.replace(/[\\/]+/g, '/').split('/').filter(Boolean).pop() || cleaned;
    return lastSegment.replace(/\.(exe|msc|bat|cmd|lnk)$/i, '').trim();
}

function humanizeAppName(value) {
    const token = normalizeAppToken(value);
    if (!token) {
        return '';
    }

    const alias = APP_NAME_ALIASES[token.toLowerCase()];
    if (alias) {
        return alias;
    }

    return token.replace(/[_-]+/g, ' ').trim();
}

function hostFromAddress(value) {
    const raw = String(value || '').trim();
    if (!raw) {
        return '';
    }
    if (raw.startsWith('[')) {
        const end = raw.indexOf(']');
        if (end !== -1) {
            return raw.slice(1, end);
        }
    }
    const lastColon = raw.lastIndexOf(':');
    if (lastColon > -1 && raw.indexOf(':') === lastColon) {
        return raw.slice(0, lastColon);
    }
    return raw;
}

function parseRdpTemplateMetadata(content) {
    const metadata = {};
    const lines = String(content || '').replace(/^\uFEFF/, '').replace(/\u0000/g, '').split(/\r?\n/);

    lines.forEach((line) => {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith('#')) {
            return;
        }

        const match = /^([^:]+):([sib]):(.*)$/.exec(trimmed);
        if (!match) {
            return;
        }

        metadata[match[1].trim().toLowerCase()] = match[3].trim();
    });

    return metadata;
}

function suggestEntryNameFromMetadata(metadata) {
    const remoteName = String(metadata.remoteapplicationname || '').trim();
    if (remoteName) {
        return remoteName;
    }

    const programFields = [
        metadata.remoteapplicationprogram,
        metadata.remoteapplicationfile,
        metadata['alternate shell'],
    ];

    for (const value of programFields) {
        const name = humanizeAppName(value);
        if (name) {
            return name;
        }
    }

    return hostFromAddress(metadata['full address'] || metadata['alternate full address']);
}

function inferEntryIcon(metadata, suggestedName = '') {
    const haystack = [
        suggestedName,
        metadata.remoteapplicationname,
        metadata.remoteapplicationprogram,
        metadata.remoteapplicationfile,
        metadata.remoteapplicationicon,
        metadata['alternate shell'],
    ].join(' ').toLowerCase();

    if (!haystack.trim()) {
        return 'window';
    }
    if (/(winword|\bword\b|\.docx?\b)/.test(haystack)) {
        return 'word';
    }
    if (/(excel|\.xlsx?\b|\.csv\b)/.test(haystack)) {
        return 'excel';
    }
    if (/(powerpnt|powerpoint|\.pptx?\b)/.test(haystack)) {
        return 'powerpoint';
    }
    if (/(outlook|mail|\.msg\b)/.test(haystack)) {
        return 'outlook';
    }
    if (/(chrome|msedge|\bedge\b|firefox|iexplore|browser|webview)/.test(haystack)) {
        return 'browser';
    }
    if (/(powershell|\bpwsh\b|\bcmd\b|terminal|shell|bash|wt\.exe|windows terminal)/.test(haystack)) {
        return 'terminal';
    }
    if (/(explorer|folder|files|share|onedrive)/.test(haystack)) {
        return 'folder';
    }
    if (/(database|\bsql\b|oracle|mysql|postgres|pgadmin|dbeaver|ssms)/.test(haystack)) {
        return 'database';
    }
    return 'window';
}

function deriveTemplateSuggestions(content) {
    const metadata = parseRdpTemplateMetadata(content);
    const name = suggestEntryNameFromMetadata(metadata);
    const icon = inferEntryIcon(metadata, name);

    return {
        icon,
        iconSuggested: icon !== 'window',
        metadata,
        name,
    };
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

function setTemplateSuggestionStatus(message, tone = 'neutral') {
    const el = document.getElementById('templateSuggestionStatus');
    if (!el) return;

    el.textContent = message;
    el.classList.remove('is-success', 'is-error');
    if (tone === 'success') {
        el.classList.add('is-success');
    } else if (tone === 'error') {
        el.classList.add('is-error');
    }
}

function resetTemplateSuggestionState(form) {
    if (!form) return;

    delete form.dataset.autofilledName;
    delete form.dataset.autofilledIcon;
    setTemplateSuggestionStatus(
        t('templateSuggestionHint', 'Selecting an .rdp file can suggest a name and icon when possible.')
    );
}

async function suggestTemplateFields(file, form) {
    const nameInput = form.querySelector('input[name="name"]');
    const iconSelect = form.querySelector('select[name="icon"]');
    if (!file || !nameInput || !iconSelect) {
        return;
    }

    try {
        const suggestions = deriveTemplateSuggestions(await file.text());
        const applied = [];

        if (suggestions.name) {
            const currentName = nameInput.value.trim();
            const previousAutoName = form.dataset.autofilledName || '';
            if (currentName === '' || currentName === previousAutoName) {
                nameInput.value = suggestions.name;
                form.dataset.autofilledName = suggestions.name;
                applied.push(`${t('nameLabel', 'Name')}: ${suggestions.name}`);
            }
        }

        if (suggestions.iconSuggested) {
            const currentIcon = normalizeEntryIcon(iconSelect.value);
            const previousAutoIcon = normalizeEntryIcon(form.dataset.autofilledIcon || 'window');
            if (currentIcon === 'window' || currentIcon === previousAutoIcon) {
                populateEntryIconSelect(iconSelect, suggestions.icon);
                form.dataset.autofilledIcon = suggestions.icon;
                applied.push(`${t('entryIconLabel', 'Entry Icon')}: ${entryIconLabel(suggestions.icon)}`);
            }
        }

        if (applied.length > 0) {
            setTemplateSuggestionStatus(
                formatMessage(t('templateSuggestionApplied', 'Suggested from template: %s.'), applied.join(' · ')),
                'success'
            );
            return;
        }

        if (suggestions.name || suggestions.iconSuggested) {
            setTemplateSuggestionStatus(
                t('templateSuggestionHint', 'Selecting an .rdp file can suggest a name and icon when possible.')
            );
            return;
        }

        setTemplateSuggestionStatus(
            t('templateSuggestionUnavailable', 'The template did not include a recognizable app name or icon hint.')
        );
    } catch (error) {
        setTemplateSuggestionStatus(
            t('templateSuggestionReadError', 'Unable to inspect the selected template in the browser.'),
            'error'
        );
    }
}

function renderAdminEntries(entries) {
    const root = document.getElementById('adminEntries');
    if (!root) return;
    root.innerHTML = '';

    if (!entries || entries.length === 0) {
        const empty = document.createElement('p');
        empty.className = 'muted';
        empty.textContent = t('noEntriesConfigured', 'No entries configured yet.');
        root.appendChild(empty);
        return;
    }

    const list = document.createElement('div');
    list.className = 'inventory-list';

    entries.forEach((entry) => {
        const wrapper = document.createElement('article');
        wrapper.className = 'entry-row';

        const summary = document.createElement('div');
        summary.className = 'entry-row-summary';

        const info = document.createElement('div');
        info.className = 'entry-row-info';

        const icon = createEntryIconElement(entry.icon, 'entry-row-icon');
        info.appendChild(icon);

        const title = document.createElement('span');
        title.className = 'entry-row-title';
        title.textContent = entry.name;
        info.appendChild(title);

        const type = document.createElement('span');
        type.className = 'entry-meta-badge';
        type.textContent = entryTypeLabel(entry.type);
        info.appendChild(type);

        const iconBadge = document.createElement('span');
        iconBadge.className = 'entry-meta-badge';
        iconBadge.textContent = entryIconLabel(entry.icon);
        info.appendChild(iconBadge);

        const enabled = document.createElement('span');
        enabled.className = statusBadgeClass(entry.enabled);
        enabled.textContent = statusLabel(entry.enabled);
        info.appendChild(enabled);

        const target = document.createElement('span');
        target.className = 'entry-row-target';
        target.textContent = entry.forceTargetIPOverride && entry.targetIPOverride
            ? entry.targetIPOverride
            : (entry.host || entry.targetHostOverride || (entry.hasUploadedTemplate ? t('uploadedTemplateLabel', 'Uploaded template') : ''));
        info.appendChild(target);

        const actions = document.createElement('div');
        actions.className = 'entry-row-actions';

        const editToggle = document.createElement('button');
        editToggle.type = 'button';
        editToggle.className = 'secondary-button';
        editToggle.textContent = t('editButton', 'Edit');

        const toggleEnabledButton = document.createElement('button');
        toggleEnabledButton.type = 'button';
        toggleEnabledButton.className = 'secondary-button';
        toggleEnabledButton.textContent = toggleEnabledButtonLabel(entry.enabled);

        actions.appendChild(editToggle);
        actions.appendChild(toggleEnabledButton);

        summary.appendChild(info);
        summary.appendChild(actions);

        const editPanel = document.createElement('div');
        editPanel.className = 'entry-edit-panel';
        editPanel.hidden = true;

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

        const iconLabel = document.createElement('label');
        iconLabel.className = 'muted';
        iconLabel.textContent = t('entryIconLabel', 'Entry Icon');
        const iconSelect = buildEntryIconSelect(entry.icon);
        iconLabel.appendChild(iconSelect);

        const groupsLabel = document.createElement('label');
        groupsLabel.className = 'muted';
        groupsLabel.textContent = t('allowedGroupsLabel', 'Allowed Groups (comma separated)');
        const groupsInput = document.createElement('input');
        groupsInput.type = 'text';
        groupsInput.value = (entry.allowedGroups || []).join(', ');
        groupsLabel.appendChild(groupsInput);

        editPanel.appendChild(nameLabel);
        editPanel.appendChild(descriptionLabel);
        editPanel.appendChild(iconLabel);
        editPanel.appendChild(groupsLabel);

        let hostInput = null;
        let targetHostInput = null;
        let targetIPInput = null;
        let forceTargetIPInput = null;
        if (entry.type === 'host') {
            const hostLabel = document.createElement('label');
            hostLabel.className = 'muted';
            hostLabel.textContent = t('hostLabel', 'Host (host:port)');
            hostInput = document.createElement('input');
            hostInput.type = 'text';
            hostInput.value = entry.host || '';
            hostLabel.appendChild(hostInput);
            editPanel.appendChild(hostLabel);
        } else {
            const targetLabel = document.createElement('label');
            targetLabel.className = 'muted';
            targetLabel.textContent = t('targetHostOverrideLabel', 'Target Host Override (optional)');
            targetHostInput = document.createElement('input');
            targetHostInput.type = 'text';
            targetHostInput.value = entry.targetHostOverride || '';
            targetLabel.appendChild(targetHostInput);
            editPanel.appendChild(targetLabel);
        }
        const targetIPLabel = document.createElement('label');
        targetIPLabel.className = 'muted';
        targetIPLabel.textContent = t('targetIPOverrideLabel', 'Target IP Override (optional)');
        targetIPInput = document.createElement('input');
        targetIPInput.type = 'text';
        targetIPInput.value = entry.targetIPOverride || '';
        targetIPLabel.appendChild(targetIPInput);
        editPanel.appendChild(targetIPLabel);

        const forceTargetIPLabel = document.createElement('label');
        forceTargetIPLabel.className = 'checkbox-row';
        forceTargetIPInput = document.createElement('input');
        forceTargetIPInput.type = 'checkbox';
        forceTargetIPInput.checked = Boolean(entry.forceTargetIPOverride);
        forceTargetIPLabel.appendChild(forceTargetIPInput);
        forceTargetIPLabel.appendChild(document.createTextNode(t('forceTargetIPOverrideLabel', 'Force target IP override')));
        editPanel.appendChild(forceTargetIPLabel);

        const editActions = document.createElement('div');
        editActions.className = 'entry-edit-actions';

        const saveButton = document.createElement('button');
        saveButton.type = 'button';
        saveButton.className = 'primary-button';
        saveButton.textContent = t('saveButton', 'Save');

        const deleteButton = document.createElement('button');
        deleteButton.type = 'button';
        deleteButton.className = 'danger-button';
        deleteButton.textContent = t('deleteButton', 'Delete');

        editActions.appendChild(saveButton);
        editActions.appendChild(deleteButton);

        editPanel.appendChild(editActions);

        wrapper.appendChild(summary);
        wrapper.appendChild(editPanel);

        editToggle.addEventListener('click', () => {
            editPanel.hidden = !editPanel.hidden;
            if (!editPanel.hidden) {
                nameInput.focus();
            }
        });

        saveButton.addEventListener('click', async () => {
            clearEntriesError();
            clearEntriesSuccess();
            const payload = {
                name: nameInput.value,
                description: descriptionInput.value,
                icon: iconSelect.value,
                allowedGroups: splitGroups(groupsInput.value),
                enabled: entry.enabled,
            };
            if (hostInput) {
                payload.host = hostInput.value;
            }
            if (targetHostInput) {
                payload.targetHostOverride = targetHostInput.value;
            }
            payload.targetIPOverride = targetIPInput ? targetIPInput.value : '';
            payload.forceTargetIPOverride = forceTargetIPInput ? forceTargetIPInput.checked : false;
            try {
                await adminRequest(`/api/v1/admin/entries/${encodeURIComponent(entry.id)}`, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload),
                });
                setEntriesSuccess(formatMessage(t('saveSuccess', 'Saved %s.'), entry.name));
                await loadEntries();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setEntriesError(`${t('saveEntryErrorPrefix', 'Unable to save entry:')} ${error.message}`);
                }
            }
        });

        toggleEnabledButton.addEventListener('click', async () => {
            clearEntriesError();
            clearEntriesSuccess();
            try {
                await adminRequest(`/api/v1/admin/entries/${encodeURIComponent(entry.id)}`, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({enabled: !entry.enabled}),
                });
                setEntriesSuccess(formatMessage(t('updateSuccess', 'Updated %s.'), entry.name));
                await loadEntries();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setEntriesError(`${t('updateEntryErrorPrefix', 'Unable to update entry:')} ${error.message}`);
                }
            }
        });

        deleteButton.addEventListener('click', async () => {
            if (!confirm(formatMessage(t('deleteConfirm', 'Are you sure you want to delete %s?'), entry.name))) return;
            clearEntriesError();
            clearEntriesSuccess();
            try {
                await adminRequest(`/api/v1/admin/entries/${encodeURIComponent(entry.id)}`, {
                    method: 'DELETE',
                });
                setEntriesSuccess(formatMessage(t('deleteSuccess', 'Deleted %s.'), entry.name));
                await loadEntries();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setEntriesError(`${t('deleteEntryErrorPrefix', 'Unable to delete entry:')} ${error.message}`);
                }
            }
        });

        list.appendChild(wrapper);
    });

    root.appendChild(list);
}

function renderAdminAuthUsers(users) {
    const root = document.getElementById('adminAuthUsers');
    if (!root) return;
    root.innerHTML = '';

    if (!users || users.length === 0) {
        const empty = document.createElement('p');
        empty.className = 'muted';
        empty.textContent = t('noAuthUsersConfigured', 'No direct auth users configured yet.');
        root.appendChild(empty);
        return;
    }

    const list = document.createElement('div');
    list.className = 'inventory-list';

    users.forEach((user) => {
        const wrapper = document.createElement('article');
        wrapper.className = 'entry-row';

        const summary = document.createElement('div');
        summary.className = 'entry-row-summary';

        const info = document.createElement('div');
        info.className = 'entry-row-info';

        const title = document.createElement('span');
        title.className = 'entry-row-title';
        title.textContent = user.username;
        info.appendChild(title);

        const enabled = document.createElement('span');
        enabled.className = statusBadgeClass(user.enabled);
        enabled.textContent = statusLabel(user.enabled);
        info.appendChild(enabled);

        const actions = document.createElement('div');
        actions.className = 'entry-row-actions';

        const editToggle = document.createElement('button');
        editToggle.type = 'button';
        editToggle.className = 'secondary-button';
        editToggle.textContent = t('editButton', 'Edit');

        const toggleEnabledButton = document.createElement('button');
        toggleEnabledButton.type = 'button';
        toggleEnabledButton.className = 'secondary-button';
        toggleEnabledButton.textContent = toggleEnabledButtonLabel(user.enabled);

        actions.appendChild(editToggle);
        actions.appendChild(toggleEnabledButton);

        summary.appendChild(info);
        summary.appendChild(actions);

        const editPanel = document.createElement('div');
        editPanel.className = 'entry-edit-panel';
        editPanel.hidden = true;

        const passwordLabel = document.createElement('label');
        passwordLabel.textContent = t('newPasswordLabel', 'New Password');
        const passwordInput = document.createElement('input');
        passwordInput.type = 'password';
        passwordInput.placeholder = t('passwordKeepPlaceholder', 'Leave blank to keep current password');
        passwordLabel.appendChild(passwordInput);

        editPanel.appendChild(passwordLabel);

        const editActions = document.createElement('div');
        editActions.className = 'entry-edit-actions';

        const saveButton = document.createElement('button');
        saveButton.type = 'button';
        saveButton.className = 'primary-button';
        saveButton.textContent = t('saveButton', 'Save');

        const deleteButton = document.createElement('button');
        deleteButton.type = 'button';
        deleteButton.className = 'danger-button';
        deleteButton.textContent = t('deleteButton', 'Delete');

        editActions.appendChild(saveButton);
        editActions.appendChild(deleteButton);
        editPanel.appendChild(editActions);

        wrapper.appendChild(summary);
        wrapper.appendChild(editPanel);

        editToggle.addEventListener('click', () => {
            editPanel.hidden = !editPanel.hidden;
            if (!editPanel.hidden) {
                passwordInput.focus();
            }
        });

        saveButton.addEventListener('click', async () => {
            clearAuthUsersError();
            clearAuthUsersSuccess();

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
                setAuthUsersSuccess(formatMessage(t('saveSuccess', 'Saved %s.'), user.username));
                await loadAuthUsers();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAuthUsersError(`${t('saveAuthUserErrorPrefix', 'Unable to save auth user:')} ${error.message}`);
                }
            }
        });

        toggleEnabledButton.addEventListener('click', async () => {
            clearAuthUsersError();
            clearAuthUsersSuccess();

            try {
                await adminRequest(`/api/v1/admin/auth-users/${encodeURIComponent(user.username)}`, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({enabled: !user.enabled}),
                });
                setAuthUsersSuccess(formatMessage(t('updateSuccess', 'Updated %s.'), user.username));
                await loadAuthUsers();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAuthUsersError(`${t('updateAuthUserErrorPrefix', 'Unable to update auth user:')} ${error.message}`);
                }
            }
        });

        deleteButton.addEventListener('click', async () => {
            if (!confirm(formatMessage(t('deleteConfirm', 'Are you sure you want to delete %s?'), user.username))) return;
            clearAuthUsersError();
            clearAuthUsersSuccess();

            try {
                await adminRequest(`/api/v1/admin/auth-users/${encodeURIComponent(user.username)}`, {
                    method: 'DELETE',
                });
                setAuthUsersSuccess(formatMessage(t('deleteSuccess', 'Deleted %s.'), user.username));
                await loadAuthUsers();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setAuthUsersError(`${t('deleteAuthUserErrorPrefix', 'Unable to delete auth user:')} ${error.message}`);
                }
            }
        });

        list.appendChild(wrapper);
    });

    root.appendChild(list);
}

function refreshDisplayedAppIcon() {
    const cacheBustUrl = `/assets/app-icon?v=${Date.now()}`;
    document.querySelectorAll('img[src^="/assets/app-icon"]').forEach((image) => {
        image.src = cacheBustUrl;
    });
}

function renderBrandingIcons(icons) {
    const root = document.getElementById('brandingIcons');
    const activeName = document.getElementById('activeIconName');
    if (!root) return;

    root.innerHTML = '';
    const active = icons.find((icon) => icon.active);
    if (activeName) {
        activeName.textContent = active ? active.originalName : t('defaultIconLabel', 'Default RDP Gateway icon');
    }

    if (!icons.length) {
        const empty = document.createElement('p');
        empty.className = 'muted';
        empty.textContent = t('defaultIconLabel', 'Default RDP Gateway icon');
        root.appendChild(empty);
        return;
    }

    const list = document.createElement('div');
    list.className = 'inventory-list branding-icon-list';
    icons.forEach((icon) => {
        const row = document.createElement('div');
        row.className = 'entry-row branding-icon-row';

        const summary = document.createElement('div');
        summary.className = 'entry-row-summary';

        const info = document.createElement('div');
        info.className = 'entry-row-info';

        const preview = document.createElement('span');
        preview.className = 'branding-icon-thumb';
        const image = document.createElement('img');
        image.src = icon.active ? `/assets/app-icon?v=${Date.now()}` : icon.url;
        image.alt = '';
        image.setAttribute('aria-hidden', 'true');
        preview.appendChild(image);
        info.appendChild(preview);

        const title = document.createElement('span');
        title.className = 'entry-row-title';
        title.textContent = icon.originalName;
        info.appendChild(title);

        const type = document.createElement('span');
        type.className = 'entry-meta-badge';
        type.textContent = icon.contentType;
        info.appendChild(type);

        if (icon.active) {
            const activeBadge = document.createElement('span');
            activeBadge.className = 'entry-status-badge is-enabled';
            activeBadge.textContent = t('enabledStatus', 'Enabled');
            info.appendChild(activeBadge);
        }

        const actions = document.createElement('div');
        actions.className = 'entry-row-actions';

        const selectButton = document.createElement('button');
        selectButton.type = 'button';
        selectButton.className = 'secondary-button';
        selectButton.textContent = t('selectIconButton', 'Use This Icon');
        selectButton.disabled = icon.active;
        selectButton.addEventListener('click', async () => {
            clearBrandingError();
            clearBrandingSuccess();
            try {
                await adminRequest('/api/v1/admin/icon/active', {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({id: icon.id}),
                });
                setBrandingSuccess(formatMessage(t('selectIconSuccess', 'Selected %s.'), icon.originalName));
                await loadIcons();
                refreshDisplayedAppIcon();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setBrandingError(`${t('selectIconErrorPrefix', 'Unable to select icon:')} ${error.message}`);
                }
            }
        });
        actions.appendChild(selectButton);

        const deleteButton = document.createElement('button');
        deleteButton.type = 'button';
        deleteButton.className = 'danger-button';
        deleteButton.textContent = t('deleteButton', 'Delete');
        deleteButton.addEventListener('click', async () => {
            if (!confirm(formatMessage(t('deleteConfirm', 'Are you sure you want to delete %s?'), icon.originalName))) {
                return;
            }
            clearBrandingError();
            clearBrandingSuccess();
            try {
                await adminRequest(`/api/v1/admin/icons/${encodeURIComponent(icon.id)}`, {
                    method: 'DELETE',
                });
                setBrandingSuccess(formatMessage(t('deleteSuccess', 'Deleted %s.'), icon.originalName));
                await loadIcons();
                refreshDisplayedAppIcon();
            } catch (error) {
                if (error.message !== 'authentication required') {
                    setBrandingError(`${t('deleteIconErrorPrefix', 'Unable to delete icon:')} ${error.message}`);
                }
            }
        });
        actions.appendChild(deleteButton);

        summary.appendChild(info);
        summary.appendChild(actions);
        row.appendChild(summary);
        list.appendChild(row);
    });

    root.appendChild(list);
}

async function loadEntries() {
    try {
        const entries = await adminRequest('/api/v1/admin/entries');
        renderAdminEntries(entries);
    } catch (error) {
        if (error.message !== 'authentication required') {
            setEntriesError(`${t('loadEntriesErrorPrefix', 'Unable to load entries:')} ${error.message}`);
        }
    }
}

async function loadAuthUsers() {
    try {
        const users = await adminRequest('/api/v1/admin/auth-users');
        renderAdminAuthUsers(users);
    } catch (error) {
        if (error.message !== 'authentication required') {
            setAuthUsersError(`${t('loadAuthUsersErrorPrefix', 'Unable to load auth users:')} ${error.message}`);
        }
    }
}

async function loadIcons() {
    try {
        const icons = await adminRequest('/api/v1/admin/icons');
        setEntryIconOptions(icons);
        renderBrandingIcons(icons);
    } catch (error) {
        if (error.message !== 'authentication required') {
            setBrandingError(`${t('loadIconsErrorPrefix', 'Unable to load icons:')} ${error.message}`);
        }
    }
}

async function loadCurrentUser() {
    try {
        const user = await adminRequest('/api/v1/user');
        const el = document.getElementById('adminUsername');
        if (el) el.textContent = user.displayName || user.username || t('unknownUser', 'Unknown User');
    } catch (error) {
        if (error.message !== 'authentication required') {
            setEntriesError(`${t('loadUserErrorPrefix', 'Unable to load user information:')} ${error.message}`);
        }
    }
}

function bindHostForm() {
    const form = document.getElementById('hostForm');
    if (!form) return;
    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        clearEntriesError();
        clearEntriesSuccess();

        const data = new FormData(form);
        const payload = {
            name: data.get('name') || '',
            description: data.get('description') || '',
            icon: data.get('icon') || 'window',
            allowedGroups: splitGroups(String(data.get('allowedGroups') || '')),
            host: data.get('host') || '',
            targetIPOverride: data.get('targetIPOverride') || '',
            forceTargetIPOverride: data.get('forceTargetIPOverride') === 'on',
        };

        try {
            await adminRequest('/api/v1/admin/entries/host', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify(payload),
            });
            setEntriesSuccess(t('createHostEntrySuccess', 'Host entry created.'));
            form.reset();
            populateEntryIconSelect(form.querySelector('select[name="icon"]'), 'window');
            await loadEntries();
        } catch (error) {
            if (error.message !== 'authentication required') {
                setEntriesError(`${t('createHostEntryErrorPrefix', 'Unable to create host entry:')} ${error.message}`);
            }
        }
    });
}

function bindTemplateForm() {
    const form = document.getElementById('templateForm');
    if (!form) return;

    const fileInput = form.querySelector('input[name="template"]');
    if (fileInput) {
        fileInput.addEventListener('change', async () => {
            const [file] = fileInput.files || [];
            if (!file) {
                resetTemplateSuggestionState(form);
                return;
            }
            await suggestTemplateFields(file, form);
        });
    }

    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        clearEntriesError();
        clearEntriesSuccess();

        const formData = new FormData(form);

        try {
            await adminRequest('/api/v1/admin/entries/template', {
                method: 'POST',
                body: formData,
            });
            setEntriesSuccess(t('uploadTemplateEntrySuccess', 'Template entry uploaded.'));
            form.reset();
            populateEntryIconSelect(form.querySelector('select[name="icon"]'), 'window');
            resetTemplateSuggestionState(form);
            await loadEntries();
        } catch (error) {
            if (error.message !== 'authentication required') {
                setEntriesError(`${t('uploadTemplateEntryErrorPrefix', 'Unable to upload template entry:')} ${error.message}`);
            }
        }
    });

    resetTemplateSuggestionState(form);
}

function bindAuthUserForm() {
    const form = document.getElementById('authUserForm');
    if (!form) return;
    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        clearAuthUsersError();
        clearAuthUsersSuccess();

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
            setAuthUsersSuccess(t('createDirectAuthUserSuccess', 'Direct auth user created.'));
            form.reset();
            const enabled = form.querySelector('input[name="enabled"]');
            if (enabled) {
                enabled.checked = true;
            }
            await loadAuthUsers();
        } catch (error) {
            if (error.message !== 'authentication required') {
                setAuthUsersError(`${t('createAuthUserErrorPrefix', 'Unable to create auth user:')} ${error.message}`);
            }
        }
    });
}

function bindIconForm() {
    const form = document.getElementById('iconForm');
    if (!form) return;
    form.addEventListener('submit', async (event) => {
        event.preventDefault();
        clearBrandingError();
        clearBrandingSuccess();

        const formData = new FormData(form);

        try {
            await adminRequest('/api/v1/admin/icon', {
                method: 'POST',
                headers: {'Accept': 'application/json'},
                body: formData,
            });
            setBrandingSuccess(t('uploadIconSuccess', 'Icon uploaded and selected.'));
            form.reset();
            await loadIcons();
            refreshDisplayedAppIcon();
        } catch (error) {
            if (error.message !== 'authentication required') {
                setBrandingError(`${t('uploadIconErrorPrefix', 'Unable to upload icon:')} ${error.message}`);
            }
        }
    });
}

function bindTabSwitching() {
    const tabs = document.querySelectorAll('.switcher-tab');
    const sections = document.querySelectorAll('.admin-section');

    tabs.forEach((tab) => {
        tab.addEventListener('click', () => {
            tabs.forEach((current) => current.classList.remove('is-active'));
            tab.classList.add('is-active');

            const targetId = tab.dataset.target;
            sections.forEach((section) => {
                if (section.id === targetId) {
                    section.removeAttribute('hidden');
                } else {
                    section.setAttribute('hidden', '');
                }
            });
        });
    });
}

function activateInitialAdminSection() {
    const section = new URLSearchParams(window.location.search).get('section');
    const targetId = section === 'branding' ? 'section-branding' : section === 'auth-users' ? 'section-auth-users' : '';
    if (!targetId) return;

    const tab = document.querySelector(`.switcher-tab[data-target="${targetId}"]`);
    if (tab) tab.click();
}

document.addEventListener('DOMContentLoaded', async () => {
    initializeEntryIconSelects();
    bindTabSwitching();
    activateInitialAdminSection();
    bindHostForm();
    bindTemplateForm();
    bindAuthUserForm();
    bindIconForm();
    await loadCurrentUser();
    await loadIcons();
    await loadEntries();
    await loadAuthUsers();
});
