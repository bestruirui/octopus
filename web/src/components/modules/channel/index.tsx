'use client';

import { useEffect, useMemo } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { useChannelList } from '@/api/endpoints/channel';
import { Card } from './Card';
import { usePaginationStore, useSearchStore } from '@/components/modules/toolbar';
import { useSortStore } from '@/components/modules/toolbar/sort-store';
import { EASING } from '@/lib/animations/fluid-transitions';
import { useGridPageSize } from '@/hooks/use-grid-page-size';

/** Channel card height: h-54 = 216px */
const CHANNEL_CARD_HEIGHT = 216;

export function Channel() {
    const { data: channelsData } = useChannelList();
    const pageKey = 'channel' as const;
    const pageSize = useGridPageSize({
        itemHeight: CHANNEL_CARD_HEIGHT,
        gap: 16,
        columns: { default: 1, md: 2, lg: 3, xl: 4 },
    });
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const page = usePaginationStore((s) => s.getPage(pageKey));
    const setPage = usePaginationStore((s) => s.setPage);
    const setTotalItems = usePaginationStore((s) => s.setTotalItems);
    const setPageSize = usePaginationStore((s) => s.setPageSize);
    const direction = usePaginationStore((s) => s.getDirection(pageKey));
    const sortField = useSortStore((s) => s.getSortField(pageKey));
    const sortOrder = useSortStore((s) => s.getSortOrder(pageKey));

    const filteredChannels = useMemo(() => {
        if (!channelsData) return [];
        const filtered = searchTerm.trim()
            ? channelsData.filter((c) => c.raw.name.toLowerCase().includes(searchTerm.toLowerCase()))
            : [...channelsData];

        // Sort by field and order
        return filtered.sort((a, b) => {
            let cmp = 0;
            if (sortField === 'name') {
                cmp = a.raw.name.localeCompare(b.raw.name);
            } else {
                cmp = a.raw.id - b.raw.id;
            }
            return sortOrder === 'desc' ? -cmp : cmp;
        });
    }, [channelsData, searchTerm, sortField, sortOrder]);

    // Sync to store for Toolbar to display pagination info
    useEffect(() => {
        setTotalItems(pageKey, filteredChannels.length);
        setPageSize(pageKey, pageSize);
    }, [filteredChannels.length, pageSize, pageKey, setTotalItems, setPageSize]);

    // Reset to page 1 when search term changes
    useEffect(() => {
        setPage(pageKey, 1);
    }, [searchTerm, pageKey, setPage]);

    const pagedChannels = useMemo(() => {
        const start = (page - 1) * pageSize;
        return filteredChannels.slice(start, start + pageSize);
    }, [filteredChannels, page, pageSize]);

    return (
        <AnimatePresence mode="popLayout" initial={false} custom={direction}>
            <motion.div
                key={`channel-page-${page}`}
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
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
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
                                <Card channel={channel.raw} stats={channel.formatted} />
                            </motion.div>
                        ))}
                    </AnimatePresence>
                </div>
            </motion.div>
        </AnimatePresence>
    );
}
