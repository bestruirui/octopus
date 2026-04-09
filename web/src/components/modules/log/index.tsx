'use client';

import { useCallback, useMemo, useState } from 'react';
import { Activity, Check, ChevronsUpDown, Loader2, RefreshCw, Search } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useLogs, type RelayLog } from '@/api/endpoints/log';
import { LogCard } from './Item';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';
import { cn } from '@/lib/utils';

type LogFilter = 'all' | 'success' | 'failed' | 'retry';
type LogSort = 'time-desc' | 'time-asc' | 'duration-desc' | 'cost-desc';

function ApiFormatFilter({
    value,
    options,
    onChange,
    t,
}: {
    value: string;
    options: string[];
    onChange: (value: string) => void;
    t: (key: string) => string;
}) {
    const [open, setOpen] = useState(false);
    const [keyword, setKeyword] = useState('');

    const visibleOptions = useMemo(() => {
        const term = keyword.trim().toLowerCase();
        const full = [t('toolbar.apiFormat.all'), ...options];
        return full.filter((item) => item.toLowerCase().includes(term));
    }, [keyword, options, t]);

    return (
        <Popover open={open} onOpenChange={setOpen}>
            <PopoverTrigger asChild>
                <button
                    type="button"
                    className="flex h-10 w-full items-center justify-between rounded-2xl border border-border bg-background px-3 text-sm"
                >
                    <span className="truncate">{value || t('toolbar.apiFormat.all')}</span>
                    <ChevronsUpDown className="size-4 text-muted-foreground" />
                </button>
            </PopoverTrigger>
            <PopoverContent className="w-80 rounded-3xl p-3 bg-card">
                <div className="relative mb-2">
                    <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                    <Input
                        value={keyword}
                        onChange={(e) => setKeyword(e.target.value)}
                        placeholder={t('toolbar.apiFormat.searchPlaceholder')}
                        className="rounded-2xl pl-9"
                    />
                </div>
                <div className="max-h-80 overflow-y-auto space-y-1">
                    {visibleOptions.map((option) => {
                        const selected = (option === t('toolbar.apiFormat.all') ? '' : option) === value;
                        return (
                            <button
                                key={option}
                                type="button"
                                onClick={() => {
                                    onChange(option === t('toolbar.apiFormat.all') ? '' : option);
                                    setOpen(false);
                                }}
                                className={cn(
                                    'flex w-full items-center justify-between rounded-2xl px-3 py-2 text-left text-sm transition-colors',
                                    selected ? 'bg-muted font-medium' : 'hover:bg-muted/60'
                                )}
                            >
                                <span className="truncate">{option}</span>
                                {selected && <Check className="size-4" />}
                            </button>
                        );
                    })}
                </div>
            </PopoverContent>
        </Popover>
    );
}

function resolveAPIFormat(log: RelayLog) {
    return `${log.request_api_format || 'Unknown'} -> ${log.actual_api_format || 'Unknown'}`;
}

export function Log() {
    const t = useTranslations('log');
    const [keyword, setKeyword] = useState('');
    const [filter, setFilter] = useState<LogFilter>('all');
    const [sort, setSort] = useState<LogSort>('time-desc');
    const [realtimeEnabled, setRealtimeEnabled] = useState(true);
    const [apiFormat, setApiFormat] = useState('');

    const { logs, hasMore, isLoading, isLoadingMore, loadMore, isConnected, isConnecting } = useLogs({
        pageSize: 10,
        realtimeEnabled,
    });

    const apiFormatOptions = useMemo(() => {
        return Array.from(new Set(logs.map(resolveAPIFormat))).sort((a, b) => a.localeCompare(b));
    }, [logs]);

    const visibleLogs = useMemo(() => {
        const term = keyword.trim().toLowerCase();
        let items = logs.filter((log) => {
            const format = resolveAPIFormat(log);
            const matchesKeyword = !term
                || log.request_model_name.toLowerCase().includes(term)
                || log.actual_model_name.toLowerCase().includes(term)
                || log.channel_name.toLowerCase().includes(term)
                || (log.request_api_key_name ?? '').toLowerCase().includes(term)
                || (log.error ?? '').toLowerCase().includes(term)
                || format.toLowerCase().includes(term);

            if (!matchesKeyword) return false;
            if (apiFormat && format !== apiFormat) return false;

            switch (filter) {
                case 'success':
                    return !log.error;
                case 'failed':
                    return !!log.error;
                case 'retry':
                    return (log.total_attempts ?? log.attempts?.length ?? 0) > 1;
                case 'all':
                default:
                    return true;
            }
        });

        items = [...items].sort((a, b) => {
            switch (sort) {
                case 'time-asc':
                    return a.time - b.time;
                case 'duration-desc':
                    return b.use_time - a.use_time;
                case 'cost-desc':
                    return b.cost - a.cost;
                case 'time-desc':
                default:
                    return b.time - a.time;
            }
        });

        return items;
    }, [logs, keyword, filter, sort, apiFormat]);

    const canLoadMore = hasMore && !isLoading && !isLoadingMore && logs.length > 0;
    const handleReachEnd = useCallback(() => {
        if (!canLoadMore) return;
        void loadMore();
    }, [canLoadMore, loadMore]);

    const footer = useMemo(() => {
        if (hasMore && (isLoading || isLoadingMore)) {
            return (
                <div className="flex justify-center py-4">
                    <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
            );
        }
        if (!hasMore && visibleLogs.length > 0) {
            return (
                <div className="flex justify-center py-4">
                    <span className="text-sm text-muted-foreground">{t('list.noMore')}</span>
                </div>
            );
        }
        return null;
    }, [hasMore, isLoading, isLoadingMore, visibleLogs.length, t]);

    const realtimeLabel = !realtimeEnabled
        ? t('toolbar.disconnected')
        : isConnected
            ? t('toolbar.connected')
            : t('toolbar.connecting');

    return (
        <div className="h-full min-h-0 overflow-y-auto overscroll-contain rounded-t-3xl space-y-4 pb-24 md:pb-4">
            <Card className="rounded-3xl border-border/70 bg-card/95 py-0 shadow-sm">
                <CardHeader className="px-5 py-5">
                    <div className="flex items-center justify-between gap-4">
                        <div>
                            <CardTitle>{t('toolbar.title')}</CardTitle>
                            <CardDescription>{t('toolbar.description')}</CardDescription>
                        </div>
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            <Activity className={`size-4 ${!realtimeEnabled ? 'text-amber-500' : isConnected ? 'text-emerald-500' : 'text-sky-500'}`} />
                            {realtimeLabel}
                        </div>
                    </div>
                </CardHeader>
                <CardContent className="space-y-4 px-5 pb-5 pt-0">
                    <div className="grid gap-3 md:grid-cols-[minmax(0,1.1fr)_220px_180px_180px_auto]">
                        <div className="relative">
                            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                            <Input
                                value={keyword}
                                onChange={(e) => setKeyword(e.target.value)}
                                placeholder={t('toolbar.searchPlaceholder')}
                                className="rounded-2xl pl-9"
                            />
                        </div>

                        <ApiFormatFilter value={apiFormat} options={apiFormatOptions} onChange={setApiFormat} t={t} />

                        <Select value={filter} onValueChange={(value) => setFilter(value as LogFilter)}>
                            <SelectTrigger className="w-full rounded-2xl">
                                <SelectValue placeholder={t('toolbar.filter')} />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="all">{t('toolbar.filters.all')}</SelectItem>
                                <SelectItem value="success">{t('toolbar.filters.success')}</SelectItem>
                                <SelectItem value="failed">{t('toolbar.filters.failed')}</SelectItem>
                                <SelectItem value="retry">{t('toolbar.filters.retry')}</SelectItem>
                            </SelectContent>
                        </Select>

                        <Select value={sort} onValueChange={(value) => setSort(value as LogSort)}>
                            <SelectTrigger className="w-full rounded-2xl">
                                <SelectValue placeholder={t('toolbar.sort')} />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="time-desc">{t('toolbar.sorts.timeDesc')}</SelectItem>
                                <SelectItem value="time-asc">{t('toolbar.sorts.timeAsc')}</SelectItem>
                                <SelectItem value="duration-desc">{t('toolbar.sorts.durationDesc')}</SelectItem>
                                <SelectItem value="cost-desc">{t('toolbar.sorts.costDesc')}</SelectItem>
                            </SelectContent>
                        </Select>

                        <div className="flex items-center justify-between rounded-2xl border border-border/70 px-3 py-2">
                            <div className="flex items-center gap-2 text-sm text-muted-foreground">
                                <RefreshCw className={`size-4 ${realtimeEnabled && isConnecting ? 'animate-spin' : ''}`} />
                                {t('toolbar.realtime')}
                            </div>
                            <Switch checked={realtimeEnabled} onCheckedChange={setRealtimeEnabled} />
                        </div>
                    </div>

                    <div className="grid gap-2 text-sm text-muted-foreground md:grid-cols-[1fr_auto]">
                        <div>{t('toolbar.resultCount')} {visibleLogs.length}</div>
                        <div>{t('toolbar.apiFormat.label')}: {apiFormat || t('toolbar.apiFormat.all')}</div>
                    </div>
                </CardContent>
            </Card>

            <VirtualizedGrid
                items={visibleLogs}
                layout="list"
                columns={{ default: 1 }}
                estimateItemHeight={80}
                overscan={8}
                getItemKey={(log) => `log-${log.id}`}
                renderItem={(log) => <LogCard log={log} />}
                footer={footer}
                onReachEnd={handleReachEnd}
                reachEndEnabled={canLoadMore}
                reachEndOffset={2}
            />
        </div>
    );
}
