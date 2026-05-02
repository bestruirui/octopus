'use client';

import { useState } from 'react';
import { ArrowUpAZ, Clock3, LayoutGrid, List, Plus, Search, SlidersHorizontal, X } from 'lucide-react';
import { motion, AnimatePresence, useReducedMotion } from 'motion/react';
import {
    MorphingDialog,
    MorphingDialogTrigger,
    MorphingDialogContainer,
    MorphingDialogContent,
} from '@/components/ui/morphing-dialog';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { buttonVariants } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { useNavStore, type NavItem } from '@/components/modules/navbar';
import { CreateDialogContent as ChannelCreateContent } from '@/components/modules/channel/Create';
import { CreateDialogContent as GroupCreateContent } from '@/components/modules/group/Create';
import { CreateDialogContent as ModelCreateContent } from '@/components/modules/model/Create';
import { useTranslations } from 'next-intl';
import { useIsMobile } from '@/hooks/use-mobile';
import { useSearchStore } from './search-store';
import {
    useToolbarViewOptionsStore,
    TOOLBAR_PAGES,
    normalizeGroupFilterValue,
    type ToolbarPage,
    type ChannelFilter,
    type GroupFilter,
    type ModelFilter,
    type ModelSortMode,
    type ToolbarSortField,
    type ToolbarSortOrder,
} from './view-options-store';

const CHANNEL_FILTER_OPTIONS: ChannelFilter[] = ['all', 'enabled', 'disabled'];
const GROUP_FILTER_OPTIONS: GroupFilter[] = ['all', 'with-members', 'empty', 'chat', 'deepseek', 'mimo', 'embeddings', 'rerank', 'moderations', 'image_generation', 'audio_speech', 'audio_transcription', 'video_generation', 'music_generation', 'search'];
const MODEL_FILTER_OPTIONS: ModelFilter[] = ['all', 'priced', 'free'];
const MODEL_SORT_OPTIONS: ModelSortMode[] = ['success-rate', 'request-count'];
type CombinedSortOption = {
    value: `${ToolbarSortField}-${ToolbarSortOrder}`;
    field: ToolbarSortField;
    order: ToolbarSortOrder;
    labelKey: string;
};
const COMBINED_SORT_OPTIONS: readonly CombinedSortOption[] = [
    { value: 'name-asc', field: 'name', order: 'asc', labelKey: 'popover.nameAsc' },
    { value: 'name-desc', field: 'name', order: 'desc', labelKey: 'popover.nameDesc' },
    { value: 'created-asc', field: 'created', order: 'asc', labelKey: 'popover.createdAsc' },
    { value: 'created-desc', field: 'created', order: 'desc', labelKey: 'popover.createdDesc' },
] as const;

const COMMAND_CELL_CLASS = 'rounded-[1.25rem] border border-border/35 bg-background/48 shadow-waterhouse-soft backdrop-blur-md transition-[color,background-color,border-color,box-shadow,transform] duration-300 hover:border-primary/24 hover:bg-background/68 hover:shadow-[var(--waterhouse-shadow-soft)] active:scale-[0.97]';
const COMMAND_ICON_BUTTON_CLASS = `${COMMAND_CELL_CLASS} h-11 w-11 text-muted-foreground hover:text-foreground`;
const COMMAND_TEXT_BUTTON_CLASS = `${COMMAND_CELL_CLASS} h-11 px-3.5 text-sm font-medium text-muted-foreground hover:text-foreground`;
const OPTION_BUTTON_CLASS = 'h-9 rounded-[1.05rem] border px-3 text-xs font-medium transition-[color,background-color,border-color,box-shadow,transform] duration-300 active:scale-[0.98]';
const ACTIVE_OPTION_CLASS = 'border-primary/25 bg-primary/82 text-primary-foreground shadow-waterhouse-soft';
const INACTIVE_OPTION_CLASS = 'border-border/45 bg-background/42 text-foreground hover:border-primary/20 hover:bg-muted/32';

function isToolbarPage(item: NavItem): item is ToolbarPage {
    return (TOOLBAR_PAGES as readonly NavItem[]).includes(item);
}

function CreateDialogContent({ activeItem }: { activeItem: ToolbarPage }) {
    switch (activeItem) {
        case 'channel':
            return <ChannelCreateContent />;
        case 'group':
            return <GroupCreateContent />;
        case 'model':
            return <ModelCreateContent />;
    }
}

function getCreateDialogContentClassName(activeItem: ToolbarPage) {
    if (activeItem === 'group') {
        return 'h-[calc(100dvh-2rem)] w-[min(100vw-1rem,44rem)] max-w-full rounded-[2rem] border border-border/35 bg-background/84 px-2 pt-2 pb-[calc(env(safe-area-inset-bottom)+0.75rem)] text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] flex flex-col overflow-hidden md:h-[calc(100dvh-3rem)] md:w-[min(100vw-3rem,52rem)] md:rounded-[2.4rem] md:px-4 md:py-4';
    }

    if (activeItem === 'channel') {
        return 'h-[calc(100dvh-1rem)] w-[min(100vw-1rem,42rem)] max-w-full rounded-[2rem] border border-border/35 bg-background/84 px-2 py-2 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] flex flex-col overflow-hidden md:h-[calc(100dvh-3rem)] md:w-[min(100vw-3rem,50rem)] md:rounded-[2.4rem] md:px-4 md:py-4';
    }

    return 'w-[min(100vw-1rem,34rem)] max-w-full bg-card text-card-foreground px-4 py-4 rounded-3xl custom-shadow max-h-[calc(100dvh-1rem)] flex flex-col overflow-hidden md:px-6 md:max-h-[calc(100dvh-2rem)]';
}

export function Toolbar() {
    const t = useTranslations('toolbar');
    const navT = useTranslations('navbar');
    const isMobile = useIsMobile();
    const reduceMotion = useReducedMotion();
    const lightweightMotion = isMobile || reduceMotion;
    const { activeItem } = useNavStore();
    const toolbarItem = isToolbarPage(activeItem) ? activeItem : null;
    const searchTerm = useSearchStore((s) => (toolbarItem ? s.searchTerms[toolbarItem] || '' : ''));
    const setSearchTerm = useSearchStore((s) => s.setSearchTerm);
    const layout = useToolbarViewOptionsStore((s) => (toolbarItem ? s.getLayout(toolbarItem) : 'grid'));
    const sortField = useToolbarViewOptionsStore((s) =>
        toolbarItem === 'channel' || toolbarItem === 'group' ? s.getSortField(toolbarItem) : 'name'
    );
    const sortOrder = useToolbarViewOptionsStore((s) => (toolbarItem ? s.getSortOrder(toolbarItem) : 'asc'));
    const setLayout = useToolbarViewOptionsStore((s) => s.setLayout);
    const setSortConfig = useToolbarViewOptionsStore((s) => s.setSortConfig);
    const setSortOrder = useToolbarViewOptionsStore((s) => s.setSortOrder);
    const channelFilter = useToolbarViewOptionsStore((s) => s.channelFilter);
    const groupFilter = useToolbarViewOptionsStore((s) => normalizeGroupFilterValue(s.groupFilter));
    const modelFilter = useToolbarViewOptionsStore((s) => s.modelFilter);
    const modelSortMode = useToolbarViewOptionsStore((s) => s.modelSortMode);
    const setChannelFilter = useToolbarViewOptionsStore((s) => s.setChannelFilter);
    const setGroupFilter = useToolbarViewOptionsStore((s) => s.setGroupFilter);
    const setModelFilter = useToolbarViewOptionsStore((s) => s.setModelFilter);
    const setModelSortMode = useToolbarViewOptionsStore((s) => s.setModelSortMode);
    const [expandedSearchItem, setExpandedSearchItem] = useState<ToolbarPage | null>(null);
    const searchExpanded = expandedSearchItem === toolbarItem;

    if (!toolbarItem) return null;
    const showLayoutOptions = toolbarItem !== 'group';
    const showCombinedSortOptions = toolbarItem === 'channel' || toolbarItem === 'group';
    const showSortOptions = toolbarItem !== 'model';

    const channelFilterLabelKeys: Record<ChannelFilter, string> = {
        all: 'popover.filter.channel.all',
        enabled: 'popover.filter.channel.enabled',
        disabled: 'popover.filter.channel.disabled',
    };
    const groupFilterLabelKeys: Record<GroupFilter, string> = {
        all: 'popover.filter.group.all',
        'with-members': 'popover.filter.group.withMembers',
        empty: 'popover.filter.group.empty',
        chat: 'popover.filter.group.chat',
        deepseek: 'popover.filter.group.deepseek',
        mimo: 'popover.filter.group.mimo',
        responses: 'popover.filter.group.chat',
        messages: 'popover.filter.group.chat',
        embeddings: 'popover.filter.group.embeddings',
        rerank: 'popover.filter.group.rerank',
        moderations: 'popover.filter.group.moderations',
        image_generation: 'popover.filter.group.imageGeneration',
        audio_speech: 'popover.filter.group.audioSpeech',
        audio_transcription: 'popover.filter.group.audioTranscription',
        video_generation: 'popover.filter.group.videoGeneration',
        music_generation: 'popover.filter.group.musicGeneration',
        search: 'popover.filter.group.search',
    };
    const modelFilterLabelKeys: Record<ModelFilter, string> = {
        all: 'popover.filter.model.all',
        priced: 'popover.filter.model.priced',
        free: 'popover.filter.model.free',
    };
    const modelSortLabelKeys: Record<ModelSortMode, string> = {
        'success-rate': 'popover.filter.model.sort.successRate',
        'request-count': 'popover.filter.model.sort.requestCount',
    };

    const filterOptions = toolbarItem === 'channel'
        ? CHANNEL_FILTER_OPTIONS.map((value) => ({
            value,
            label: t(channelFilterLabelKeys[value]),
        }))
        : toolbarItem === 'group'
            ? GROUP_FILTER_OPTIONS.map((value) => ({
                value,
                label: t(groupFilterLabelKeys[value]),
            }))
            : MODEL_FILTER_OPTIONS.map((value) => ({
                value,
                label: t(modelFilterLabelKeys[value]),
            }));

    const activeFilter = toolbarItem === 'channel'
        ? channelFilter
        : toolbarItem === 'group'
            ? groupFilter
            : modelFilter;
    const activePageLabel = navT(toolbarItem);
    const searchAriaLabel = `Search ${activePageLabel}`;
    const clearSearchAriaLabel = `Clear ${activePageLabel} search`;
    const createAriaLabel = `Create ${activePageLabel}`;

    const handleFilterChange = (value: string) => {
        switch (toolbarItem) {
            case 'channel':
                setChannelFilter(value as ChannelFilter);
                break;
            case 'group':
                setGroupFilter(normalizeGroupFilterValue(value));
                break;
            case 'model':
                setModelFilter(value as ModelFilter);
                break;
        }
    };

    return (
        <AnimatePresence mode="wait">
            <motion.div
                key="toolbar"
                initial={lightweightMotion ? { opacity: 0 } : { opacity: 0, scale: 0.9 }}
                animate={lightweightMotion ? { opacity: 1 } : { opacity: 1, scale: 1 }}
                exit={lightweightMotion ? { opacity: 0 } : { opacity: 0, scale: 0.9 }}
                transition={{ duration: lightweightMotion ? 0.12 : 0.2 }}
                className="waterhouse-pod flex max-w-full flex-wrap items-center gap-2 rounded-[1.85rem] border-border/35 bg-background/42 p-1.5 shadow-waterhouse-soft backdrop-blur-[var(--waterhouse-shell-blur)]"
            >
                {/* 搜索按钮/展开框 */}
                <div
                    className={cn(
                        'relative h-11 transition-[width] duration-300 ease-out',
                        searchExpanded ? 'w-[min(18rem,calc(100vw-3rem))] sm:w-72' : 'w-11'
                    )}
                >
                    {!searchExpanded ? (
                        <motion.button
                            layoutId="search-box"
                            type="button"
                            aria-label={searchAriaLabel}
                            onClick={() => setExpandedSearchItem(toolbarItem)}
                            className={cn(
                                buttonVariants({ variant: "ghost", size: "icon" }),
                                "absolute inset-0 rounded-[1.25rem] transition-none",
                                COMMAND_ICON_BUTTON_CLASS
                            )}
                        >
                            <motion.span layout="position"><Search className="size-4 transition-colors duration-300" /></motion.span>
                        </motion.button>
                    ) : (
                        <motion.div
                            layoutId="search-box"
                            className={cn(
                                "absolute inset-0 flex items-center gap-2 px-3.5",
                                COMMAND_CELL_CLASS
                            )}
                            transition={lightweightMotion ? { duration: 0.12 } : { type: 'spring', stiffness: 400, damping: 30 }}
                        >
                            <motion.span layout="position"><Search className="size-4 text-muted-foreground shrink-0" /></motion.span>
                            <input
                                type="text"
                                aria-label={searchAriaLabel}
                                value={searchTerm}
                                onChange={(e) => setSearchTerm(toolbarItem, e.target.value)}
                                autoFocus
                                className="min-w-0 flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
                            />
                            <button
                                type="button"
                                aria-label={clearSearchAriaLabel}
                                onClick={() => {
                                    setSearchTerm(toolbarItem, '');
                                    setExpandedSearchItem(null);
                                }}
                                className="grid size-7 shrink-0 place-items-center rounded-full border border-border/30 bg-background/35 text-muted-foreground transition-colors hover:text-foreground"
                            >
                                <X className="size-3.5" />
                            </button>
                        </motion.div>
                    )}
                </div>

                <div className="waterhouse-pod flex h-11 items-center gap-1 rounded-[1.45rem] border-border/30 bg-background/28 p-1">
                    <Popover>
                        <PopoverTrigger asChild>
                            <button
                                type="button"
                                aria-label={t('popover.ariaLabel')}
                                className={cn(
                                    buttonVariants({ variant: 'ghost', size: 'default' }),
                                    "h-9 rounded-[1.15rem] border border-transparent bg-transparent px-3 text-muted-foreground shadow-none transition-[color,background-color,border-color] duration-300 hover:border-border/30 hover:bg-background/45 hover:text-foreground hover:shadow-none"
                                )}
                            >
                                <SlidersHorizontal className="size-4 transition-colors duration-300" />
                                <span className="hidden text-xs font-semibold sm:inline">{t('popover.filter.title')}</span>
                            </button>
                        </PopoverTrigger>
                        <PopoverContent
                            align="end"
                            side="bottom"
                            sideOffset={12}
                            className="w-72 rounded-[1.7rem] border border-border/40 bg-card/84 p-3 shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)]"
                        >
                            <div className="grid gap-3">
                                {showLayoutOptions && (
                                    <div className="grid gap-2 rounded-[1.25rem] border border-border/35 bg-background/34 p-2.5">
                                        <p className="text-xs font-semibold text-muted-foreground">{t('popover.layout')}</p>
                                        <div className="grid grid-cols-2 gap-2">
                                            <button
                                                type="button"
                                                onClick={() => setLayout(toolbarItem, 'grid')}
                                                className={cn(
                                                    OPTION_BUTTON_CLASS,
                                                    'inline-flex items-center justify-center gap-1.5',
                                                    layout === 'grid' ? ACTIVE_OPTION_CLASS : INACTIVE_OPTION_CLASS
                                                )}
                                            >
                                                <LayoutGrid className="size-3.5" />
                                                {t('popover.grid')}
                                            </button>
                                            <button
                                                type="button"
                                                onClick={() => setLayout(toolbarItem, 'list')}
                                                className={cn(
                                                    OPTION_BUTTON_CLASS,
                                                    'inline-flex items-center justify-center gap-1.5',
                                                    layout === 'list' ? ACTIVE_OPTION_CLASS : INACTIVE_OPTION_CLASS
                                                )}
                                            >
                                                <List className="size-3.5" />
                                                {t('popover.list')}
                                            </button>
                                        </div>
                                    </div>
                                )}

                                {showSortOptions && (
                                    <div className="grid gap-2 rounded-[1.25rem] border border-border/35 bg-background/34 p-2.5">
                                        <p className="text-xs font-semibold text-muted-foreground">{t('popover.sort')}</p>
                                        {showCombinedSortOptions ? (
                                            <div className="grid grid-cols-2 gap-2">
                                                {COMBINED_SORT_OPTIONS.map((option) => (
                                                    <button
                                                        key={option.value}
                                                        type="button"
                                                        onClick={() => {
                                                            if (toolbarItem === 'channel' || toolbarItem === 'group') {
                                                                setSortConfig(toolbarItem, option.field, option.order);
                                                            }
                                                        }}
                                                        className={cn(
                                                            OPTION_BUTTON_CLASS,
                                                            'inline-flex items-center justify-center gap-1.5',
                                                            sortField === option.field && sortOrder === option.order
                                                                ? ACTIVE_OPTION_CLASS
                                                                : INACTIVE_OPTION_CLASS
                                                        )}
                                                    >
                                                        {option.field === 'name' ? <ArrowUpAZ className="size-3.5" /> : <Clock3 className="size-3.5" />}
                                                        {t(option.labelKey)}
                                                    </button>
                                                ))}
                                            </div>
                                        ) : (
                                            <div className="grid grid-cols-2 gap-2">
                                                <button
                                                    type="button"
                                                    onClick={() => setSortOrder(toolbarItem, 'asc')}
                                                    className={cn(
                                                        OPTION_BUTTON_CLASS,
                                                        'inline-flex items-center justify-center gap-1.5',
                                                        sortOrder === 'asc' ? ACTIVE_OPTION_CLASS : INACTIVE_OPTION_CLASS
                                                    )}
                                                >
                                                    <ArrowUpAZ className="size-3.5" />
                                                    {t('popover.nameAsc')}
                                                </button>
                                                <button
                                                    type="button"
                                                    onClick={() => setSortOrder(toolbarItem, 'desc')}
                                                    className={cn(
                                                        OPTION_BUTTON_CLASS,
                                                        'inline-flex items-center justify-center gap-1.5',
                                                        sortOrder === 'desc' ? ACTIVE_OPTION_CLASS : INACTIVE_OPTION_CLASS
                                                    )}
                                                >
                                                    <ArrowUpAZ className="size-3.5" />
                                                    {t('popover.nameDesc')}
                                                </button>
                                            </div>
                                        )}
                                    </div>
                                )}

                                <div className="grid gap-2 rounded-[1.25rem] border border-border/35 bg-background/34 p-2.5">
                                    <p className="text-xs font-semibold text-muted-foreground">{t('popover.filter.title')}</p>
                                    <div className="grid max-h-72 gap-2 overflow-y-auto pr-1">
                                        {toolbarItem === 'model' && (
                                            <div className="grid gap-2 rounded-[1.1rem] border border-border/35 bg-muted/14 p-2">
                                                <p className="text-[11px] font-semibold text-muted-foreground">{t('popover.filter.model.sort.title')}</p>
                                                {MODEL_SORT_OPTIONS.map((option) => (
                                                    <button
                                                        key={option}
                                                        type="button"
                                                        onClick={() => setModelSortMode(option)}
                                                        className={cn(
                                                            OPTION_BUTTON_CLASS,
                                                            'text-left',
                                                            modelSortMode === option ? ACTIVE_OPTION_CLASS : INACTIVE_OPTION_CLASS
                                                        )}
                                                    >
                                                        {t(modelSortLabelKeys[option])}
                                                    </button>
                                                ))}
                                            </div>
                                        )}
                                        {filterOptions.map((option) => (
                                            <button
                                                key={option.value}
                                                type="button"
                                                onClick={() => handleFilterChange(option.value)}
                                                className={cn(
                                                    OPTION_BUTTON_CLASS,
                                                    'text-left',
                                                    activeFilter === option.value ? ACTIVE_OPTION_CLASS : INACTIVE_OPTION_CLASS
                                                )}
                                            >
                                                {option.label}
                                            </button>
                                        ))}
                                    </div>
                                </div>
                            </div>
                        </PopoverContent>
                    </Popover>

                    {showLayoutOptions && (
                        <div className="hidden items-center gap-1 rounded-[1.15rem] border border-border/25 bg-background/24 p-0.5 sm:flex">
                            <button
                                type="button"
                                aria-label={t('popover.grid')}
                                onClick={() => setLayout(toolbarItem, 'grid')}
                                className={cn(
                                    'grid size-8 place-items-center rounded-[0.95rem] transition-[color,background-color,box-shadow] duration-300',
                                    layout === 'grid'
                                        ? 'bg-primary/82 text-primary-foreground shadow-waterhouse-soft'
                                        : 'text-muted-foreground hover:bg-background/45 hover:text-foreground'
                                )}
                            >
                                <LayoutGrid className="size-3.5" />
                            </button>
                            <button
                                type="button"
                                aria-label={t('popover.list')}
                                onClick={() => setLayout(toolbarItem, 'list')}
                                className={cn(
                                    'grid size-8 place-items-center rounded-[0.95rem] transition-[color,background-color,box-shadow] duration-300',
                                    layout === 'list'
                                        ? 'bg-primary/82 text-primary-foreground shadow-waterhouse-soft'
                                        : 'text-muted-foreground hover:bg-background/45 hover:text-foreground'
                                )}
                            >
                                <List className="size-3.5" />
                            </button>
                        </div>
                    )}
                </div>

                <div className="flex flex-wrap items-center gap-1.5">
                    {/* 创建按钮 */}
                    <MorphingDialog>
                        <MorphingDialogTrigger
                            ariaLabel={createAriaLabel}
                            className={cn(
                                buttonVariants({ variant: "ghost", size: "default" }),
                                COMMAND_TEXT_BUTTON_CLASS,
                                "bg-primary/82 px-3.5 text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground"
                            )}
                        >
                            <Plus className="size-4 transition-colors duration-300" />
                        </MorphingDialogTrigger>

                        <MorphingDialogContainer>
                            <MorphingDialogContent className={getCreateDialogContentClassName(toolbarItem)}>
                                <CreateDialogContent activeItem={toolbarItem} />
                            </MorphingDialogContent>
                        </MorphingDialogContainer>
                    </MorphingDialog>
                </div>
            </motion.div>
        </AnimatePresence>
    );
}

export { useSearchStore } from './search-store';
export { normalizeGroupFilterValue, useToolbarViewOptionsStore } from './view-options-store';
