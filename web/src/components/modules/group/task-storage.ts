'use client';

import type { AIRouteScope } from '@/api/endpoints/group';

const AI_ROUTE_PROGRESS_STORAGE_KEY = 'octopus.ai-route-progress';
const GROUP_TEST_PROGRESS_STORAGE_KEY = 'octopus.group-test-progress';

export type StoredAIRouteTask = {
    id: string;
    scope: AIRouteScope;
    groupId?: number;
};

export type StoredGroupTestTask = {
    id: string;
    groupId: number;
};

function getSessionStorage() {
    if (typeof window === 'undefined') {
        return null;
    }
    return window.sessionStorage;
}

function isPositiveNumber(value: unknown): value is number {
    return typeof value === 'number' && Number.isFinite(value) && value > 0;
}

function readStoredTask<T>(key: string, isValid: (value: unknown) => value is T): T | null {
    const storage = getSessionStorage();
    if (!storage) {
        return null;
    }

    const raw = storage.getItem(key);
    if (!raw) {
        return null;
    }

    try {
        const parsed = JSON.parse(raw) as unknown;
        if (!isValid(parsed)) {
            storage.removeItem(key);
            return null;
        }
        return parsed;
    } catch {
        storage.removeItem(key);
        return null;
    }
}

function writeStoredTask<T>(key: string, task: T) {
    const storage = getSessionStorage();
    if (!storage) {
        return;
    }

    storage.setItem(key, JSON.stringify(task));
}

function removeStoredTask(key: string) {
    const storage = getSessionStorage();
    if (!storage) {
        return;
    }

    storage.removeItem(key);
}

function isStoredAIRouteTask(value: unknown): value is StoredAIRouteTask {
    if (!value || typeof value !== 'object') {
        return false;
    }

    const task = value as Partial<StoredAIRouteTask>;
    if (typeof task.id !== 'string' || (task.scope !== 'group' && task.scope !== 'table')) {
        return false;
    }

    if (task.scope === 'group') {
        return isPositiveNumber(task.groupId);
    }

    return task.groupId === undefined || isPositiveNumber(task.groupId);
}

function isStoredGroupTestTask(value: unknown): value is StoredGroupTestTask {
    if (!value || typeof value !== 'object') {
        return false;
    }

    const task = value as Partial<StoredGroupTestTask>;
    return typeof task.id === 'string' && isPositiveNumber(task.groupId);
}

export function readStoredAIRouteTask() {
    return readStoredTask(AI_ROUTE_PROGRESS_STORAGE_KEY, isStoredAIRouteTask);
}

export function writeStoredAIRouteTask(task: StoredAIRouteTask) {
    writeStoredTask(AI_ROUTE_PROGRESS_STORAGE_KEY, task);
}

export function clearStoredAIRouteTask(id?: string) {
    if (!id) {
        removeStoredTask(AI_ROUTE_PROGRESS_STORAGE_KEY);
        return;
    }

    const current = readStoredAIRouteTask();
    if (current?.id === id) {
        removeStoredTask(AI_ROUTE_PROGRESS_STORAGE_KEY);
    }
}

export function matchesStoredAIRouteTask(task: StoredAIRouteTask | null, scope: AIRouteScope, groupId: number) {
    if (!task || task.scope !== scope) {
        return false;
    }

    if (scope === 'group') {
        return task.groupId === groupId && groupId > 0;
    }

    return true;
}

export function readStoredGroupTestTask() {
    return readStoredTask(GROUP_TEST_PROGRESS_STORAGE_KEY, isStoredGroupTestTask);
}

export function writeStoredGroupTestTask(task: StoredGroupTestTask) {
    writeStoredTask(GROUP_TEST_PROGRESS_STORAGE_KEY, task);
}

export function clearStoredGroupTestTask(id?: string) {
    if (!id) {
        removeStoredTask(GROUP_TEST_PROGRESS_STORAGE_KEY);
        return;
    }

    const current = readStoredGroupTestTask();
    if (current?.id === id) {
        removeStoredTask(GROUP_TEST_PROGRESS_STORAGE_KEY);
    }
}

export function matchesStoredGroupTestTask(task: StoredGroupTestTask | null, groupId?: number) {
    return Boolean(task && isPositiveNumber(groupId) && task.groupId === groupId);
}
