import assert from 'node:assert/strict';
import test from 'node:test';
import { DEFAULT_NAV_ORDER, normalizeNavOrder } from './nav-order.ts';

test('normalizeNavOrder drops unknown ids and appends missing defaults', () => {
    const got = normalizeNavOrder(['group', 'group', 'unknown', 'setting'], DEFAULT_NAV_ORDER);
    assert.deepEqual(got, ['group', 'setting', 'home', 'channel', 'model', 'analytics', 'log', 'alert', 'ops', 'apikey', 'user']);
});

test('normalizeNavOrder preserves default order when input is empty', () => {
    assert.deepEqual(normalizeNavOrder([], DEFAULT_NAV_ORDER), DEFAULT_NAV_ORDER);
});

test('normalizeNavOrder trims items before filtering and de-duplicating', () => {
    const got = normalizeNavOrder([' group ', ' ', 'setting', ' setting '], DEFAULT_NAV_ORDER);
    assert.deepEqual(got, ['group', 'setting', 'home', 'channel', 'model', 'analytics', 'log', 'alert', 'ops', 'apikey', 'user']);
});
