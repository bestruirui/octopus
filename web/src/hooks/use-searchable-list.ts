'use client';

import { useMemo } from 'react';
import { useSearchStore, useToolbarViewOptionsStore, normalizeGroupFilterValue } from '@/components/modules/toolbar';
import type { ToolbarCreatedSortablePage, ChannelFilter, GroupFilter } from '@/components/modules/toolbar/view-options-store';

type SortableItem = {
    id: number;
    name: string;
};

type ChannelItem = SortableItem & {
    enabled: boolean;
};

type GroupItem = SortableItem & {
    items?: { length: number }[] | unknown[];
    endpoint_type: string;
};

type FilterPredicate<T> = (item: T, filter: string) => boolean;

interface UseSearchableListOptions<T extends SortableItem> {
    data: T[] | undefined;
    pageKey: ToolbarCreatedSortablePage;
    filter?: string;
    filterPredicate?: FilterPredicate<T>;
}

function getSortComparator<T extends SortableItem>(sortField: 'name' | 'created', sortOrder: 'asc' | 'desc') {
    return (a: T, b: T) => {
        const diff = sortField === 'name'
            ? a.name.localeCompare(b.name)
            : a.id - b.id;
        return sortOrder === 'asc' ? diff : -diff;
    };
}

export function useSearchableList<T extends SortableItem>({
    data,
    pageKey,
    filter,
    filterPredicate,
}: UseSearchableListOptions<T>) {
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const sortField = useToolbarViewOptionsStore((s) => s.getSortField(pageKey));
    const sortOrder = useToolbarViewOptionsStore((s) => s.getSortOrder(pageKey));

    const sortedItems = useMemo(() => {
        if (!data) return [];
        return [...data].sort(getSortComparator(sortField, sortOrder));
    }, [data, sortField, sortOrder]);

    const visibleItems = useMemo(() => {
        const term = searchTerm.toLowerCase().trim();
        const byName = !term ? sortedItems : sortedItems.filter((item) => item.name.toLowerCase().includes(term));

        if (filter && filterPredicate) {
            return byName.filter((item) => filterPredicate(item, filter));
        }

        return byName;
    }, [sortedItems, searchTerm, filter, filterPredicate]);

    return { visibleItems, sortedItems, searchTerm, sortField, sortOrder };
}

export function useChannelFilter() {
    return useToolbarViewOptionsStore((s) => s.channelFilter);
}

export function useGroupFilter() {
    return useToolbarViewOptionsStore((s) => normalizeGroupFilterValue(s.groupFilter));
}

export function createChannelFilterPredicate(filter: ChannelFilter) {
    return (item: ChannelItem) => {
        if (filter === 'enabled') return item.enabled;
        if (filter === 'disabled') return !item.enabled;
        return true;
    };
}

export function createGroupFilterPredicate(filter: GroupFilter) {
    return (item: GroupItem, _filterValue: string) => {
        if (filter === 'with-members') return (item.items?.length || 0) > 0;
        if (filter === 'empty') return (item.items?.length || 0) === 0;
        return true;
    };
}
