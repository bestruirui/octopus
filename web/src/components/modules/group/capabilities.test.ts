import assert from 'node:assert/strict';
import test from 'node:test';

import { inferCapabilities, matchesGroupEndpointFilter } from './capabilities.ts';

test('inferCapabilities classifies embedding models as embeddings instead of chat fallback', () => {
    assert.deepEqual(inferCapabilities('text-embedding-3-small'), ['embeddings']);
    assert.deepEqual(inferCapabilities('bge-m3'), ['embeddings']);
});

test('matchesGroupEndpointFilter excludes embedding-only wildcard groups from chat filter', () => {
    assert.equal(matchesGroupEndpointFilter('chat', '*', ['text-embedding-3-small']), false);
    assert.equal(matchesGroupEndpointFilter('embeddings', '*', ['text-embedding-3-small']), true);
});

test('matchesGroupEndpointFilter keeps conversation endpoint aliases under chat filter', () => {
    assert.equal(matchesGroupEndpointFilter('chat', 'responses', ['anything']), true);
    assert.equal(matchesGroupEndpointFilter('chat', 'messages', ['anything']), true);
});
