import assert from 'node:assert/strict';
import test from 'node:test';

import { AUTO_STRATEGY_FIELDS, RETRY_FIELDS } from './runtime-settings.ts';

test('retry fields expose cooldown and total-attempt controls', () => {
    assert.deepEqual(
        RETRY_FIELDS.map((field) => field.key),
        ['relay_retry_count', 'ratelimit_cooldown', 'relay_max_total_attempts']
    );
});

test('auto strategy fields expose latency weight with bounded range', () => {
    const latencyWeight = AUTO_STRATEGY_FIELDS.find((field) => field.key === 'auto_strategy_latency_weight');

    assert.ok(latencyWeight);
    assert.equal(latencyWeight.min, '0');
    assert.equal(latencyWeight.max, '100');
});
