import assert from 'node:assert/strict';
import test from 'node:test';
import { getMorphingDialogLifecycleEvent } from './morphing-dialog-state.ts';

test('getMorphingDialogLifecycleEvent reports opened transitions', () => {
    assert.equal(getMorphingDialogLifecycleEvent(false, true), 'opened');
});

test('getMorphingDialogLifecycleEvent reports closed transitions', () => {
    assert.equal(getMorphingDialogLifecycleEvent(true, false), 'closed');
});

test('getMorphingDialogLifecycleEvent ignores stable states', () => {
    assert.equal(getMorphingDialogLifecycleEvent(false, false), null);
    assert.equal(getMorphingDialogLifecycleEvent(true, true), null);
});
