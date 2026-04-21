import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import { fileURLToPath } from 'node:url';

const templatesDir = path.dirname(fileURLToPath(import.meta.url));

function loadAdminContext(messages = {}) {
    const source = fs.readFileSync(path.join(templatesDir, 'admin.js'), 'utf8');
    const context = {
        window: { adminMessages: messages },
        document: {
            addEventListener() {},
            getElementById() {
                return null;
            },
            querySelectorAll() {
                return [];
            },
        },
        console,
    };

    vm.createContext(context);
    vm.runInContext(source, context, {filename: 'admin.js'});
    return context;
}

test('style.css keeps hidden elements hidden even when classes set display', () => {
    const css = fs.readFileSync(path.join(templatesDir, 'style.css'), 'utf8');

    assert.match(css, /\[hidden\][^{]*\{[^}]*display:\s*none\s*!important/i);
});

test('admin status helpers expose distinct tone and action labels', () => {
    const context = loadAdminContext({
        enabledStatus: 'Enabled',
        disabledStatus: 'Disabled',
        enableButton: 'Enable',
        disableButton: 'Disable',
    });

    assert.equal(context.statusLabel(true), 'Enabled');
    assert.equal(context.statusLabel(false), 'Disabled');
    assert.equal(context.statusBadgeClass(true), 'entry-status-badge is-enabled');
    assert.equal(context.statusBadgeClass(false), 'entry-status-badge is-disabled');
    assert.equal(context.toggleEnabledButtonLabel(true), 'Disable');
    assert.equal(context.toggleEnabledButtonLabel(false), 'Enable');
});

test('admin template suggestions parse remote app metadata into name and icon hints', () => {
    const context = loadAdminContext({
        entryIcons: {
            window: 'Window',
            word: 'Word',
        },
    });

    const suggestions = context.deriveTemplateSuggestions([
        'remoteapplicationname:s:Quarterly Report',
        'remoteapplicationprogram:s:||WINWORD',
        'full address:s:rds.internal:3389',
    ].join('\r\n'));

    assert.equal(suggestions.name, 'Quarterly Report');
    assert.equal(suggestions.icon, 'word');
    assert.equal(suggestions.iconSuggested, true);
});

test('admin template suggestions fall back to window icon for plain desktop templates', () => {
    const context = loadAdminContext();

    const suggestions = context.deriveTemplateSuggestions('full address:s:desktop.internal:3389\r\n');

    assert.equal(suggestions.name, 'desktop.internal');
    assert.equal(suggestions.icon, 'window');
    assert.equal(suggestions.iconSuggested, false);
});
