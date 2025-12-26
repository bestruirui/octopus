'use client';

import { useEffect, useMemo, useRef } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { useModelList } from '@/api/endpoints/model';
import { ModelItem } from './Item';
import { usePaginationStore, useSearchStore } from '@/components/modules/toolbar';
import { EASING } from '@/lib/animations/fluid-transitions';
import { useIsMobile } from '@/hooks/use-mobile';

export function Model() {
    const { data: models } = useModelList();
    const pageKey = 'model' as const;
    const isMobile = useIsMobile();
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const page = usePaginationStore((s) => s.getPage(pageKey));
    const setTotalItems = usePaginationStore((s) => s.setTotalItems);
    const setPage = usePaginationStore((s) => s.setPage);
    const pageSize = usePaginationStore((s) => s.getPageSize(pageKey));
    const setPageSize = usePaginationStore((s) => s.setPageSize);

    const filteredModels = useMemo(() => {
        if (!models) return [];
        const sortedModels = [...models].sort((a, b) => a.name.localeCompare(b.name));
        if (!searchTerm.trim()) return sortedModels;
        const term = searchTerm.toLowerCase();
        return sortedModels.filter((m) => m.name.toLowerCase().includes(term));
    }, [models, searchTerm]);

    useEffect(() => {
        setTotalItems(pageKey, filteredModels.length);
    }, [filteredModels.length, pageKey, setTotalItems]);

    useEffect(() => {
        setPage(pageKey, 1);
    }, [pageKey, searchTerm, setPage]);

    useEffect(() => {
        setPageSize(pageKey, isMobile ? 6 : 18);
    }, [isMobile, pageKey, setPageSize]);

    const pagedModels = useMemo(() => {
        const start = (page - 1) * pageSize;
        const end = start + pageSize;
        return filteredModels.slice(start, end);
    }, [filteredModels, page, pageSize]);

    const prevPageRef = useRef(page);
    const direction = page > prevPageRef.current ? 1 : page < prevPageRef.current ? -1 : 1;
    useEffect(() => {
        prevPageRef.current = page;
    }, [page]);

    return (
        <AnimatePresence mode="popLayout" initial={false} custom={direction}>
            <motion.div
                key={`model-page-${page}`}
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
                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
                    <AnimatePresence mode="popLayout">
                        {pagedModels.map((model, index) => (
                            <motion.div
                                key={"model-" + model.name}
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
                                <ModelItem model={model} />
                            </motion.div>
                        ))}
                    </AnimatePresence>
                </div>
            </motion.div>
        </AnimatePresence>
    );
}
