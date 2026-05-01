import assert from 'node:assert/strict';
import test from 'node:test';
import type { ModelMarketItem } from '@/api/endpoints/model';
import { sortModelMarketItems } from './sort.ts';

function createModel(overrides: Partial<ModelMarketItem> = {}): ModelMarketItem {
    return {
        name: 'model-a',
        input: 0,
        output: 0,
        cache_read: 0,
        cache_write: 0,
        channel_count: 1,
        enabled_key_count: 1,
        average_latency_ms: 0,
        success_rate: 0,
        request_success: 0,
        request_failed: 0,
        channels: [],
        ...overrides,
    };
}

test('sortModelMarketItems defaults to success rate, then success count, then total requests', () => {
    const items = [
        createModel({ name: 'gamma', success_rate: 0, request_success: 0, request_failed: 0 }),
        createModel({ name: 'alpha', success_rate: 1, request_success: 3, request_failed: 0 }),
        createModel({ name: 'beta', success_rate: 1, request_success: 5, request_failed: 0 }),
        createModel({ name: 'delta', success_rate: 0.9, request_success: 9, request_failed: 1 }),
    ];

    const sorted = sortModelMarketItems(items, 'success-rate');

    assert.deepEqual(
        sorted.map((item) => item.name),
        ['beta', 'alpha', 'delta', 'gamma'],
    );
});

test('sortModelMarketItems can rank by request count with success rate as tie-breaker', () => {
    const items = [
        createModel({ name: 'alpha', success_rate: 0.8, request_success: 8, request_failed: 2 }),
        createModel({ name: 'beta', success_rate: 1, request_success: 6, request_failed: 0 }),
        createModel({ name: 'gamma', success_rate: 0.5, request_success: 1, request_failed: 1 }),
        createModel({ name: 'delta', success_rate: 1, request_success: 10, request_failed: 0 }),
    ];

    const sorted = sortModelMarketItems(items, 'request-count');

    assert.deepEqual(
        sorted.map((item) => item.name),
        ['delta', 'alpha', 'beta', 'gamma'],
    );
});
