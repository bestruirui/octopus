'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { Check, ChevronDown, Loader2 } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useLogs } from '@/api/endpoints/log';
import { useGroupList } from '@/api/endpoints/group';
import { useChannelList } from '@/api/endpoints/channel';
import { useModelList } from '@/api/endpoints/model';
import { PageWrapper } from '@/components/common/PageWrapper';
import { LogCard } from './Item';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { cn } from '@/lib/utils';

type FilterOption = {
    value: string;
    label: string;
};

function SearchableFilterSelect({
    value,
    onValueChange,
    options,
    allLabel,
    placeholder,
    searchPlaceholder,
    emptyText,
}: {
    value: string;
    onValueChange: (value: string) => void;
    options: FilterOption[];
    allLabel: string;
    placeholder: string;
    searchPlaceholder: string;
    emptyText: string;
}) {
    const [open, setOpen] = useState(false);
    const [keyword, setKeyword] = useState('');

    const filteredOptions = useMemo(() => {
        const term = keyword.trim().toLowerCase();
        if (!term) return options;
        return options.filter((option) => option.label.toLowerCase().includes(term));
    }, [keyword, options]);

    const selectedLabel = useMemo(() => {
        if (value === 'all') return allLabel;
        return options.find((option) => option.value === value)?.label ?? value;
    }, [allLabel, options, value]);

    return (
        <Popover
            open={open}
            onOpenChange={(nextOpen) => {
                setOpen(nextOpen);
                if (!nextOpen) {
                    setKeyword('');
                }
            }}
        >
            <PopoverTrigger asChild>
                <Button variant="outline" className="w-full justify-between font-normal">
                    <span className="truncate text-left">{selectedLabel || placeholder}</span>
                    <ChevronDown className="size-4 text-muted-foreground" />
                </Button>
            </PopoverTrigger>
            <PopoverContent align="start" className="w-[var(--radix-popover-trigger-width)] p-2">
                <Input
                    value={keyword}
                    onChange={(e) => setKeyword(e.target.value)}
                    placeholder={searchPlaceholder}
                    className="h-8"
                />
                <div className="mt-2 max-h-56 overflow-auto space-y-1">
                    <button
                        type="button"
                        onClick={() => {
                            onValueChange('all');
                            setOpen(false);
                        }}
                        className={cn(
                            'flex w-full items-center justify-between rounded-md px-2 py-1.5 text-sm hover:bg-accent',
                            value === 'all' && 'bg-accent'
                        )}
                    >
                        <span className="truncate">{allLabel}</span>
                        {value === 'all' && <Check className="size-4 text-primary" />}
                    </button>
                    {filteredOptions.map((option) => (
                        <button
                            key={option.value}
                            type="button"
                            onClick={() => {
                                onValueChange(option.value);
                                setOpen(false);
                            }}
                            className={cn(
                                'flex w-full items-center justify-between rounded-md px-2 py-1.5 text-sm hover:bg-accent',
                                value === option.value && 'bg-accent'
                            )}
                        >
                            <span className="truncate">{option.label}</span>
                            {value === option.value && <Check className="size-4 text-primary" />}
                        </button>
                    ))}
                    {filteredOptions.length === 0 && (
                        <div className="px-2 py-2 text-xs text-muted-foreground">{emptyText}</div>
                    )}
                </div>
            </PopoverContent>
        </Popover>
    );
}

/**
 * 日志页面组件
 * - 初始加载20条历史日志
 * - SSE 实时推送新日志
 * - 滚动自动加载更多
 */
export function Log() {
    const t = useTranslations('log');
    const [selectedGroup, setSelectedGroup] = useState('all');
    const [selectedModel, setSelectedModel] = useState('all');
    const [selectedRetried, setSelectedRetried] = useState<'all' | 'yes' | 'no'>('all');
    const [selectedChannel, setSelectedChannel] = useState('all');

    const { logs, hasMore, isLoading, isLoadingMore, loadMore } = useLogs({
        pageSize: 10,
        filters: {
            group: selectedGroup === 'all' ? '' : selectedGroup,
            model: selectedModel === 'all' ? '' : selectedModel,
            retried: selectedRetried,
            channel: selectedChannel === 'all' ? '' : selectedChannel,
        },
    });

    const { data: groups = [] } = useGroupList();
    const { data: channels = [] } = useChannelList();
    const { data: models = [] } = useModelList();

    const loadMoreRef = useRef<HTMLDivElement>(null);
    const armedRef = useRef(true);

    const groupOptions = useMemo(() => {
        const names = groups
            .map((group) => group.name?.trim())
            .filter((name): name is string => !!name);
        if (selectedGroup !== 'all') {
            names.push(selectedGroup);
        }
        return Array.from(new Set(names)).sort((a, b) => a.localeCompare(b));
    }, [groups, selectedGroup]);

    const modelOptions = useMemo(() => {
        const names = models
            .map((model) => model.name?.trim())
            .filter((name): name is string => !!name);
        if (selectedModel !== 'all') {
            names.push(selectedModel);
        }
        return Array.from(new Set(names)).sort((a, b) => a.localeCompare(b));
    }, [models, selectedModel]);

    const channelOptions = useMemo(() => {
        const names = channels
            .map((channel) => channel.raw.name?.trim())
            .filter((name): name is string => !!name);
        if (selectedChannel !== 'all') {
            names.push(selectedChannel);
        }
        return Array.from(new Set(names)).sort((a, b) => a.localeCompare(b));
    }, [channels, selectedChannel]);

    const retriedOptions = useMemo<FilterOption[]>(() => [
        { value: 'yes', label: t('filter.retriedYes') },
        { value: 'no', label: t('filter.retriedNo') },
    ], [t]);

    useEffect(() => {
        const target = loadMoreRef.current;
        if (!target) return;

        const observer = new IntersectionObserver(
            (entries) => {
                const entry = entries[0];
                if (!entry) return;

                if (!entry.isIntersecting) {
                    armedRef.current = true;
                    return;
                }

                if (!armedRef.current) return;
                if (!hasMore || isLoading || isLoadingMore || logs.length === 0) return;

                armedRef.current = false;
                loadMore();
            },
            { rootMargin: '100px' }
        );

        observer.observe(target);
        return () => observer.disconnect();
    }, [hasMore, isLoading, isLoadingMore, loadMore, logs.length]);

    return (
        <PageWrapper className="grid grid-cols-1 gap-4">
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-2">
                <SearchableFilterSelect
                    value={selectedGroup}
                    onValueChange={setSelectedGroup}
                    options={groupOptions.map((group) => ({ value: group, label: group }))}
                    allLabel={t('filter.allGroups')}
                    placeholder={t('filter.group')}
                    searchPlaceholder={t('filter.searchPlaceholder')}
                    emptyText={t('filter.noResult')}
                />

                <SearchableFilterSelect
                    value={selectedModel}
                    onValueChange={setSelectedModel}
                    options={modelOptions.map((model) => ({ value: model, label: model }))}
                    allLabel={t('filter.allModels')}
                    placeholder={t('filter.model')}
                    searchPlaceholder={t('filter.searchPlaceholder')}
                    emptyText={t('filter.noResult')}
                />

                <SearchableFilterSelect
                    value={selectedRetried}
                    onValueChange={(value) => setSelectedRetried(value as 'all' | 'yes' | 'no')}
                    options={retriedOptions}
                    allLabel={t('filter.allRetries')}
                    placeholder={t('filter.retried')}
                    searchPlaceholder={t('filter.searchPlaceholder')}
                    emptyText={t('filter.noResult')}
                />

                <SearchableFilterSelect
                    value={selectedChannel}
                    onValueChange={setSelectedChannel}
                    options={channelOptions.map((channel) => ({ value: channel, label: channel }))}
                    allLabel={t('filter.allChannels')}
                    placeholder={t('filter.channel')}
                    searchPlaceholder={t('filter.searchPlaceholder')}
                    emptyText={t('filter.noResult')}
                />
            </div>

            {logs.map((log) => (
                <LogCard key={`log-${log.id}`} log={log} />
            ))}

            <div ref={loadMoreRef} className="flex justify-center py-4">
                {hasMore && (isLoadingMore || isLoading) && (
                    <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                )}
                {!hasMore && logs.length > 0 && (
                    <span className="text-sm text-muted-foreground">{t('list.noMore')}</span>
                )}
            </div>
        </PageWrapper>
    );
}
