'use client';

import { useEffect, useMemo } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { GroupCard } from './Card';
import { useGroupList } from '@/api/endpoints/group';
import { usePaginationStore, useSearchStore, useSortStore } from '@/components/modules/toolbar';
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
    const sortConfig = useSortStore((s) => s.getSortConfig(pageKey));
    const page = usePaginationStore((s) => s.getPage(pageKey));
    const setPage = usePaginationStore((s) => s.setPage);
    const setTotalItems = usePaginationStore((s) => s.setTotalItems);
    const setPageSize = usePaginationStore((s) => s.setPageSize);
    const direction = usePaginationStore((s) => s.getDirection(pageKey));

    const filteredGroups = useMemo(() => {
        if (!groups) return [];

        // 先筛选
        let result = [...groups];
        if (searchTerm.trim()) {
            const term = searchTerm.toLowerCase();
            result = result.filter((g) => g.name.toLowerCase().includes(term));
        }

        // 再排序
        result.sort((a, b) => {
            let compareResult = 0;
            if (sortConfig.field === 'id') {
                compareResult = (a.id || 0) - (b.id || 0);
            } else if (sortConfig.field === 'name') {
                compareResult = a.name.localeCompare(b.name, 'zh-CN');
            }
            return sortConfig.order === 'asc' ? compareResult : -compareResult;
        });

        return result;
    }, [groups, searchTerm, sortConfig.field, sortConfig.order]);

    // Sync to store for Toolbar to display pagination info
    useEffect(() => {
        setTotalItems(pageKey, filteredGroups.length);
        setPageSize(pageKey, pageSize);
    }, [filteredGroups.length, pageSize, pageKey, setTotalItems, setPageSize]);

    // Reset to page 1 when search term or sort config changes
    useEffect(() => {
        setPage(pageKey, 1);
    }, [searchTerm, sortConfig.field, sortConfig.order, pageKey, setPage]);

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
