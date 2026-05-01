import assert from 'node:assert/strict';
import test from 'node:test';
import { normalizeOverviewRange } from './store.ts';

test('normalizeOverviewRange defaults to 7d when value is invalid', () => {
    assert.equal(normalizeOverviewRange('unexpected'), '7d');
});

test('normalizeOverviewRange preserves supported values', () => {
    assert.equal(normalizeOverviewRange('7d'), '7d');
    assert.equal(normalizeOverviewRange('30d'), '30d');
    assert.equal(normalizeOverviewRange('90d'), '90d');
});
