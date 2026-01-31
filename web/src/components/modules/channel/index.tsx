'use client';

import { useEffect, useMemo } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { useChannelList } from '@/api/endpoints/channel';
import { Card } from './Card';
import { usePaginationStore, useSearchStore, useLayoutStore, useSortStore } from '@/components/modules/toolbar';
import { EASING } from '@/lib/animations/fluid-transitions';
import { useGridPageSize } from '@/hooks/use-grid-page-size';

/** Channel card height: h-54 = 216px */
const CHANNEL_CARD_HEIGHT = 216;

export function Channel() {
    const { data: channelsData } = useChannelList();
    const pageKey = 'channel' as const;
    const currentLayout = useLayoutStore((s) => s.getLayout(pageKey));
    const pageSize = useGridPageSize({
        itemHeight: CHANNEL_CARD_HEIGHT,
        gap: 16,
        columns: { default: 1, md: 2, lg: 3, xl: 4 },
    });
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const sortConfig = useSortStore((s) => s.getSortConfig(pageKey));
    const page = usePaginationStore((s) => s.getPage(pageKey));
    const setPage = usePaginationStore((s) => s.setPage);
    const setTotalItems = usePaginationStore((s) => s.setTotalItems);
    const setPageSize = usePaginationStore((s) => s.setPageSize);
    const direction = usePaginationStore((s) => s.getDirection(pageKey));

    const filteredChannels = useMemo(() => {
        if (!channelsData) return [];

        // 先筛选
        let result = [...channelsData];
        if (searchTerm.trim()) {
            const term = searchTerm.toLowerCase();
            result = result.filter((c) => c.raw.name.toLowerCase().includes(term));
        }

        // 再排序
        result.sort((a, b) => {
            let compareResult = 0;
            if (sortConfig.field === 'id') {
                compareResult = a.raw.id - b.raw.id;
            } else if (sortConfig.field === 'name') {
                compareResult = a.raw.name.localeCompare(b.raw.name, 'zh-CN');
            }
            return sortConfig.order === 'asc' ? compareResult : -compareResult;
        });

        return result;
    }, [channelsData, searchTerm, sortConfig.field, sortConfig.order]);

    // Sync to store for Toolbar to display pagination info
    useEffect(() => {
        setTotalItems(pageKey, filteredChannels.length);
        setPageSize(pageKey, pageSize);
    }, [filteredChannels.length, pageSize, pageKey, setTotalItems, setPageSize]);

    // Reset to page 1 when search term or sort config changes
    useEffect(() => {
        setPage(pageKey, 1);
    }, [searchTerm, sortConfig.field, sortConfig.order, pageKey, setPage]);

    const pagedChannels = useMemo(() => {
        const start = (page - 1) * pageSize;
        return filteredChannels.slice(start, start + pageSize);
    }, [filteredChannels, page, pageSize]);

    // 根据布局类型确定网格类名
    const gridClassName = useMemo(() => {
        switch (currentLayout) {
            case 'grid':
                return 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4';
            case 'single-column':
                return 'grid grid-cols-1 gap-4';
            default:
                return 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4';
        }
    }, [currentLayout]);

    return (
        <AnimatePresence mode="popLayout" initial={false} custom={direction}>
            <motion.div
                key={`channel-page-${page}-${currentLayout}`}
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
                <div className={gridClassName}>
                    <AnimatePresence mode="popLayout">
                        {pagedChannels.map((channel, index) => (
                            <motion.div
                                key={"channel-" + channel.raw.id}
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
                                <Card channel={channel.raw} stats={channel.formatted} layout={currentLayout} />
                            </motion.div>
                        ))}
                    </AnimatePresence>
                </div>
            </motion.div>
        </AnimatePresence>
    );
}
