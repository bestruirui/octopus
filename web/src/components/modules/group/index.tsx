'use client';

import { useEffect, useMemo } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { GroupCard } from './Card';
import { useGroupList } from '@/api/endpoints/group';
import { usePaginationStore, useSearchStore } from '@/components/modules/toolbar';
import { useSortStore } from '@/components/modules/toolbar/sort-store';
import { EASING } from '@/lib/animations/fluid-transitions';
import { useGridPageSize } from '@/hooks/use-grid-page-size';

/** Group card approximate height (variable due to content) */
const GROUP_CARD_HEIGHT = 280;

export function Group() {
    const { data: groups } = useGroupList();
    const pageKey = 'group' as const;
    const pageSize = useGridPageSize({
        itemHeight: GROUP_CARD_HEIGHT,
        gap: 16,
        columns: { default: 1, md: 2, lg: 3 },
    });
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const page = usePaginationStore((s) => s.getPage(pageKey));
    const setPage = usePaginationStore((s) => s.setPage);
    const setTotalItems = usePaginationStore((s) => s.setTotalItems);
    const setPageSize = usePaginationStore((s) => s.setPageSize);
    const direction = usePaginationStore((s) => s.getDirection(pageKey));
    const sortField = useSortStore((s) => s.getSortField(pageKey));
    const sortOrder = useSortStore((s) => s.getSortOrder(pageKey));

    const filteredGroups = useMemo(() => {
        if (!groups) return [];
        const filtered = searchTerm.trim()
            ? groups.filter((g) => g.name.toLowerCase().includes(searchTerm.toLowerCase()))
            : [...groups];

        // Sort by field and order
        return filtered.sort((a, b) => {
            let cmp = 0;
            if (sortField === 'name') {
                cmp = a.name.localeCompare(b.name);
            } else {
                cmp = a.id! - b.id!;
            }
            return sortOrder === 'desc' ? -cmp : cmp;
        });
    }, [groups, searchTerm, sortField, sortOrder]);

    // Sync to store for Toolbar to display pagination info
    useEffect(() => {
        setTotalItems(pageKey, filteredGroups.length);
        setPageSize(pageKey, pageSize);
    }, [filteredGroups.length, pageSize, pageKey, setTotalItems, setPageSize]);

    // Reset to page 1 when search term changes
    useEffect(() => {
        setPage(pageKey, 1);
    }, [searchTerm, pageKey, setPage]);

    const pagedGroups = useMemo(() => {
        const start = (page - 1) * pageSize;
        return filteredGroups.slice(start, start + pageSize);
    }, [filteredGroups, page, pageSize]);

    return (
        <AnimatePresence mode="popLayout" initial={false} custom={direction}>
            <motion.div
                key={`group-page-${page}`}
                custom={direction}
                variants={{
                    enter: (d: number) => ({ x: d >= 0 ? 24 : -24, opacity: 0 }),
                    center: { x: 0, opacity: 1 },
                    exit: (d: number) => ({ x: d >= 0 ? -24 : 24, opacity: 0 }),
                }}
                initial="enter"
                animate="center"
                exit="exit"
                transition={{ duration: 0.25, ease: EASING.easeOutExpo }}
            >
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    <AnimatePresence mode="popLayout">
                        {pagedGroups.map((group, index) => (
                            <motion.div
                                key={group.id}
                                initial={{ opacity: 0, y: 20 }}
                                animate={{ opacity: 1, y: 0 }}
                                exit={{
                                    opacity: 0,
                                    scale: 0.95,
                                    transition: { duration: 0.2 }
                                }}
                                transition={{
                                    duration: 0.45,
                                    ease: EASING.easeOutExpo,
                                    delay: index === 0 ? 0 : Math.min(0.08 * Math.log2(index + 1), 0.4),
                                }}
                                layout={!searchTerm.trim()}
                            >
                                <GroupCard group={group} />
                            </motion.div>
                        ))}
                    </AnimatePresence>
                </div>
            </motion.div>
        </AnimatePresence>
    );
}
