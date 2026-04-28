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
    assert.deepEqual(
        createAlertChannelDraft({
            name: 'ops',
            url: 'https://example.com/webhook',
            secret: 'abc',
            type: 'webhook',
            headers: '{"x-test":"1"}',
        }),
        {
            name: 'ops',
            url: 'https://example.com/webhook',
            secret: 'abc',
        }
    );
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
        },
        {
            name: 'ops-new',
            url: 'https://new.example.com',
            secret: 'new-secret',
        }
    );

    assert.deepEqual(updated, {
        id: 7,
        name: 'ops-new',
        type: 'webhook',
        url: 'https://new.example.com',
        secret: 'new-secret',
        headers: '{"x-test":"1"}',
    });
});
