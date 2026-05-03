import assert from 'node:assert/strict';
import test from 'node:test';

import {
    applyAlertChannelDraft,
    applyAlertRuleDraft,
    createAlertChannelDraft,
    createAlertRuleDraft,
} from './forms.ts';

test('createAlertRuleDraft returns expected defaults for a new rule', () => {
    assert.deepEqual(createAlertRuleDraft(), {
        name: '',
        condition_type: 'error_rate',
        threshold: 10,
        notif_channel_id: 0,
        cooldown_sec: 300,
    });
});

test('applyAlertRuleDraft updates editable fields and preserves hidden metadata', () => {
    const updated = applyAlertRuleDraft(
        {
            id: 12,
            enabled: true,
            name: 'old',
            condition_type: 'error_rate',
            threshold: 15,
            notif_channel_id: 2,
            cooldown_sec: 60,
            condition_json: '{"sample":true}',
            scope_channel_id: 8,
            scope_api_key_id: 9,
        },
        {
            name: 'new',
            condition_type: 'channel_down',
            threshold: 3,
            notif_channel_id: 4,
            cooldown_sec: 120,
        }
    );

    assert.deepEqual(updated, {
        id: 12,
        enabled: true,
        name: 'new',
        condition_type: 'channel_down',
        threshold: 3,
        notif_channel_id: 4,
        cooldown_sec: 120,
        condition_json: '{"sample":true}',
        scope_channel_id: 8,
        scope_api_key_id: 9,
    });
});

test('createAlertChannelDraft keeps editable channel fields only', () => {
    const draft = createAlertChannelDraft({
        name: 'ops',
        url: 'https://example.com/webhook',
        secret: 'abc',
        type: 'webhook',
        headers: '{"x-test":"1"}',
    });
    assert.equal(draft.name, 'ops');
    assert.equal(draft.url, 'https://example.com/webhook');
    assert.equal(draft.secret, 'abc');
    assert.equal(draft.type, 'webhook');
});

test('createAlertChannelDraft parses gotify config', () => {
    const draft = createAlertChannelDraft({
        name: 'my-gotify',
        type: 'gotify',
        url: 'https://gotify.example.com',
        secret: 'my-token',
        config: '{"server_url":"https://gotify.example.com","token":"my-token","priority":8}',
    });
    assert.equal(draft.type, 'gotify');
    assert.equal(draft.gotify.server_url, 'https://gotify.example.com');
    assert.equal(draft.gotify.token, 'my-token');
    assert.equal(draft.gotify.priority, 8);
});

test('createAlertChannelDraft parses email config', () => {
    const draft = createAlertChannelDraft({
        name: 'my-email',
        type: 'email',
        config: '{"smtp_host":"smtp.example.com","smtp_port":587,"username":"user","password":"pass","from":"a@b.com","to":"c@d.com","use_tls":true}',
    });
    assert.equal(draft.type, 'email');
    assert.equal(draft.email.smtp_host, 'smtp.example.com');
    assert.equal(draft.email.from, 'a@b.com');
    assert.equal(draft.email.to, 'c@d.com');
});

test('applyAlertChannelDraft updates editable fields and preserves channel metadata', () => {
    const updated = applyAlertChannelDraft(
        {
            id: 7,
            name: 'ops',
            type: 'webhook',
            url: 'https://old.example.com',
            secret: 'old-secret',
            headers: '{"x-test":"1"}',
            config: '',
        },
        {
            name: 'ops-new',
            type: 'webhook',
            url: 'https://new.example.com',
            secret: 'new-secret',
            gotify: { server_url: '', token: '' },
            email: { smtp_host: '', smtp_port: 587, username: '', password: '', from: '', to: '', use_tls: true },
        }
    );

    assert.equal(updated.name, 'ops-new');
    assert.equal(updated.type, 'webhook');
    assert.equal(updated.url, 'https://new.example.com');
    assert.equal(updated.secret, 'new-secret');
    assert.equal(updated.headers, '{"x-test":"1"}');
    assert.equal(updated.config, '');
});

test('applyAlertChannelDraft serializes gotify config', () => {
    const updated = applyAlertChannelDraft(
        { id: 1, name: '', type: 'gotify', url: '', secret: '', config: '' },
        {
            name: 'my-gotify',
            type: 'gotify',
            url: '',
            secret: '',
            gotify: { server_url: 'https://gotify.example.com', token: 'my-token', priority: 7 },
            email: { smtp_host: '', smtp_port: 587, username: '', password: '', from: '', to: '', use_tls: true },
        }
    );

    assert.equal(updated.type, 'gotify');
    assert.equal(updated.url, 'https://gotify.example.com');
    assert.equal(updated.secret, 'my-token');
    const config = JSON.parse(updated.config || '{}');
    assert.equal(config.server_url, 'https://gotify.example.com');
    assert.equal(config.token, 'my-token');
    assert.equal(config.priority, 7);
});

test('applyAlertChannelDraft serializes email config', () => {
    const updated = applyAlertChannelDraft(
        { id: 2, name: '', type: 'email', url: '', secret: '', config: '' },
        {
            name: 'my-email',
            type: 'email',
            url: '',
            secret: '',
            gotify: { server_url: '', token: '' },
            email: { smtp_host: 'smtp.example.com', smtp_port: 465, username: 'user', password: 'pass', from: 'a@b.com', to: 'c@d.com', use_tls: true },
        }
    );

    assert.equal(updated.type, 'email');
    const config = JSON.parse(updated.config || '{}');
    assert.equal(config.smtp_host, 'smtp.example.com');
    assert.equal(config.smtp_port, 465);
    assert.equal(config.from, 'a@b.com');
    assert.equal(config.to, 'c@d.com');
    assert.equal(config.use_tls, true);
});
