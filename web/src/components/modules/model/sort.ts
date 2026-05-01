import type { ModelMarketItem } from '@/api/endpoints/model';

export type ModelSortMode = 'success-rate' | 'request-count';

function getRequestCount(item: ModelMarketItem): number {
    return item.request_success + item.request_failed;
}

function compareBySuccessRate(left: ModelMarketItem, right: ModelMarketItem): number {
    const leftRequests = getRequestCount(left);
    const rightRequests = getRequestCount(right);

    if (leftRequests === 0 && rightRequests > 0) return 1;
    if (leftRequests > 0 && rightRequests === 0) return -1;

    if (leftRequests > 0 && rightRequests > 0) {
        const leftRatio = left.request_success * rightRequests;
        const rightRatio = right.request_success * leftRequests;
        if (leftRatio !== rightRatio) {
            return rightRatio - leftRatio;
        }
    }

    if (left.request_success !== right.request_success) {
        return right.request_success - left.request_success;
    }

    if (leftRequests !== rightRequests) {
        return rightRequests - leftRequests;
    }

    return left.name.localeCompare(right.name);
}

function compareByRequestCount(left: ModelMarketItem, right: ModelMarketItem): number {
    const leftRequests = getRequestCount(left);
    const rightRequests = getRequestCount(right);

    if (leftRequests !== rightRequests) {
        return rightRequests - leftRequests;
    }

    if (left.success_rate !== right.success_rate) {
        return right.success_rate - left.success_rate;
    }

    if (left.request_success !== right.request_success) {
        return right.request_success - left.request_success;
    }

    return left.name.localeCompare(right.name);
}

export function sortModelMarketItems(items: ModelMarketItem[], mode: ModelSortMode): ModelMarketItem[] {
    const sorted = [...items];
    sorted.sort((left, right) => {
        if (mode === 'request-count') {
            return compareByRequestCount(left, right);
        }
        return compareBySuccessRate(left, right);
    });
    return sorted;
}
