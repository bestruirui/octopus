'use client';

import { useState, useEffect } from 'react';
import { ChevronLeft, ChevronRight, Plus, Search, X, LayoutGrid, Rows3 } from 'lucide-react';
import { motion, AnimatePresence } from 'motion/react';
import {
    MorphingDialog,
    MorphingDialogTrigger,
    MorphingDialogContainer,
    MorphingDialogContent,
} from '@/components/ui/morphing-dialog';
import { buttonVariants } from '@/components/ui/button';
import { useNavStore, type NavItem } from '@/components/modules/navbar';
import { CreateDialogContent as ChannelCreateContent } from '@/components/modules/channel/Create';
import { CreateDialogContent as GroupCreateContent } from '@/components/modules/group/Create';
import { CreateDialogContent as ModelCreateContent } from '@/components/modules/model/Create';
import { useSearchStore } from './search-store';
import { usePaginationStore } from './pagination-store';
import { useLayoutStore } from './layout-store';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';
import { useTranslations } from 'next-intl';

const TOOLBAR_PAGES: NavItem[] = ['channel', 'group', 'model'];

function CreateDialogContent({ activeItem }: { activeItem: NavItem }) {
    switch (activeItem) {
        case 'channel':
            return <ChannelCreateContent />;
        case 'group':
            return <GroupCreateContent />;
        case 'model':
            return <ModelCreateContent />;
        default:
            return null;
    }
}

export function Toolbar() {
    const { activeItem } = useNavStore();
    const t = useTranslations('common.layout');
    const searchTerm = useSearchStore((s) => s.searchTerms[activeItem] || '');
    const setSearchTerm = useSearchStore((s) => s.setSearchTerm);
    const page = usePaginationStore((s) => s.getPage(activeItem));
    const totalPages = usePaginationStore((s) => s.getTotalPages(activeItem));
    const prevPage = usePaginationStore((s) => s.prevPage);
    const nextPage = usePaginationStore((s) => s.nextPage);
    const setPage = usePaginationStore((s) => s.setPage);
    const currentLayout = useLayoutStore((s) => s.getLayout(activeItem));
    const setLayout = useLayoutStore((s) => s.setLayout);
    const [searchExpanded, setSearchExpanded] = useState(false);

    useEffect(() => {
        queueMicrotask(() => {
            setSearchExpanded(false);
            setSearchTerm(activeItem, '');
            setPage(activeItem, 1);
        });
    }, [activeItem, setSearchTerm, setPage]);

    const showToolbar = TOOLBAR_PAGES.includes(activeItem);

    return (
        <AnimatePresence mode="wait">
            {showToolbar && (
                <motion.div
                    key="toolbar"
                    initial={{ opacity: 0, scale: 0.9 }}
                    animate={{ opacity: 1, scale: 1 }}
                    exit={{ opacity: 0, scale: 0.9 }}
                    transition={{ duration: 0.2 }}
                    className="flex items-center gap-2"
                >
                    {/* 搜索按钮/展开框 */}
                    <div className="relative h-9 w-9">
                        {!searchExpanded ? (
                            <motion.button
                                layoutId="search-box"
                                onClick={() => setSearchExpanded(true)}
                                className={buttonVariants({ variant: "ghost", size: "icon", className: "absolute inset-0 rounded-xl transition-none hover:bg-transparent text-muted-foreground hover:text-foreground" })}
                            >
                                <motion.span layout="position"><Search className="size-4 transition-colors duration-300" /></motion.span>
                            </motion.button>
                        ) : (
                            <motion.div
                                layoutId="search-box"
                                className="absolute right-0 top-0 flex items-center gap-2 h-9 px-3 rounded-xl border"
                                transition={{ type: 'spring', stiffness: 400, damping: 30 }}
                            >
                                <motion.span layout="position"><Search className="size-4 text-muted-foreground shrink-0" /></motion.span>
                                <input
                                    type="text"
                                    value={searchTerm}
                                    onChange={(e) => setSearchTerm(activeItem, e.target.value)}
                                    autoFocus
                                    className="w-20 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
                                />
                                <button
                                    onClick={() => {
                                        setSearchTerm(activeItem, '');
                                        setSearchExpanded(false);
                                    }}
                                    className="p-0.5 rounded shrink-0 text-muted-foreground hover:text-foreground transition-colors"
                                >
                                    <X className="size-3.5" />
                                </button>
                            </motion.div>
                        )}
                    </div>

                    {/* 布局切换按钮 - 仅在渠道页面显示 */}
                    {activeItem === 'channel' && (
                        <div className="flex items-center h-9 rounded-xl border">
                            <Tooltip>
                                <TooltipTrigger asChild>
                                    <button
                                        type="button"
                                        aria-label="Grid layout"
                                        onClick={() => setLayout(activeItem, 'grid')}
                                        className={`size-8 inline-flex items-center justify-center rounded-lg transition-colors ${
                                            currentLayout === 'grid'
                                                ? 'text-foreground bg-accent'
                                                : 'text-muted-foreground hover:text-foreground'
                                        }`}
                                    >
                                        <LayoutGrid className="size-4" />
                                    </button>
                                </TooltipTrigger>
                                <TooltipContent>{t('grid')}</TooltipContent>
                            </Tooltip>
                            <Tooltip>
                                <TooltipTrigger asChild>
                                    <button
                                        type="button"
                                        aria-label="Single column layout"
                                        onClick={() => setLayout(activeItem, 'single-column')}
                                        className={`size-8 inline-flex items-center justify-center rounded-lg transition-colors ${
                                            currentLayout === 'single-column'
                                                ? 'text-foreground bg-accent'
                                                : 'text-muted-foreground hover:text-foreground'
                                        }`}
                                    >
                                        <Rows3 className="size-4" />
                                    </button>
                                </TooltipTrigger>
                                <TooltipContent>{t('singleColumn')}</TooltipContent>
                            </Tooltip>
                        </div>
                    )}

                    {/* 页码指示器 */}
                    <div className="flex items-center h-9 rounded-xl border">
                        <button
                            type="button"
                            aria-label="Previous page"
                            onClick={() => prevPage(activeItem)}
                            disabled={page <= 1}
                            className="size-8 inline-flex items-center justify-center rounded-lg text-muted-foreground hover:text-foreground disabled:opacity-40 disabled:hover:text-muted-foreground"
                        >
                            <ChevronLeft className="size-4" />
                        </button>
                        <button
                            type="button"
                            onClick={() => setPage(activeItem, 1)}
                            className="px-2 text-sm tabular-nums text-muted-foreground hover:text-foreground"
                            aria-label="Page indicator"
                            title="Click to go to first page"
                        >
                            {page}/{totalPages}
                        </button>
                        <button
                            type="button"
                            aria-label="Next page"
                            onClick={() => nextPage(activeItem)}
                            disabled={page >= totalPages}
                            className="size-8 inline-flex items-center justify-center rounded-lg text-muted-foreground hover:text-foreground disabled:opacity-40 disabled:hover:text-muted-foreground"
                        >
                            <ChevronRight className="size-4" />
                        </button>
                    </div>

                    {/* 创建按钮 */}
                    <MorphingDialog>
                        <MorphingDialogTrigger className={buttonVariants({ variant: "ghost", size: "icon", className: "rounded-xl transition-none hover:bg-transparent text-muted-foreground hover:text-foreground" })}>
                            <Plus className="size-4 transition-colors duration-300" />
                        </MorphingDialogTrigger>

                        <MorphingDialogContainer>
                            <MorphingDialogContent className="w-fit max-w-full bg-card text-card-foreground px-6 py-4 rounded-3xl custom-shadow max-h-[calc(100vh-2rem)] flex flex-col overflow-hidden">
                                <CreateDialogContent activeItem={activeItem} />
                            </MorphingDialogContent>
                        </MorphingDialogContainer>
                    </MorphingDialog>
                </motion.div>
            )}
        </AnimatePresence>
    );
}

export { useSearchStore } from './search-store';
export { usePaginationStore } from './pagination-store';
export { useLayoutStore } from './layout-store';
