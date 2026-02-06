'use client';

import { type ReactNode, useEffect, useMemo, useRef } from 'react';
import { useLogs } from '@/api/endpoints/log';
import { PageWrapper } from '@/components/common/PageWrapper';
import { LogCard } from './Item';
import { Loader2 } from 'lucide-react';
import { useLocale, useTranslations } from 'next-intl';

const LOG_SEGMENT_GAP_SECONDS = 3 * 60;

function normalizeLocale(locale: string): string {
    if (locale === 'zh_hans') return 'zh-CN';
    if (locale === 'zh_hant') return 'zh-TW';
    return 'en-US';
}

function formatSegmentTime(timestamp: number, locale: string): string {
    const date = new Date(timestamp * 1000);
    return date.toLocaleString(normalizeLocale(locale), {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
    });
}

/**
 * 日志页面组件
 * - 初始加载20条历史日志
 * - SSE 实时推送新日志
 * - 滚动自动加载更多
 */
export function Log() {
    const t = useTranslations('log');
    const locale = useLocale();
    const { logs, hasMore, isLoading, isLoadingMore, loadMore } = useLogs({ pageSize: 10 });
    const loadMoreRef = useRef<HTMLDivElement>(null);
    const armedRef = useRef(true);
    const segmentedLogs = useMemo(() => {
        return logs.map((log, index) => {
            if (index === 0) return { log, showDivider: false };

            const previous = logs[index - 1];
            const showDivider = previous.time - log.time >= LOG_SEGMENT_GAP_SECONDS;
            return { log, showDivider };
        });
    }, [logs]);

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

    const renderedItems = useMemo(() => {
        return segmentedLogs.flatMap(({ log, showDivider }) => {
            const items: ReactNode[] = [];
            if (showDivider) {
                const time = formatSegmentTime(log.time, locale);
                const label = t('list.timeGap', { time });
                const safeLabel = (label === 'log.list.timeGap' || label === 'list.timeGap')
                    ? (locale === 'zh_hant' ? `較早日誌 · ${time}` : locale === 'en' ? `Older logs: ${time}` : `更早日志 · ${time}`)
                    : label;

                items.push(
                    <div key={`divider-${log.id}`} className="flex h-7 items-center gap-3 px-1">
                        <div className="h-px flex-1 bg-border/80" />
                        <span className="text-[11px] leading-none tracking-wide text-muted-foreground bg-muted/60 border border-border/60 rounded-full px-3 py-1.5">
                            {safeLabel}
                        </span>
                        <div className="h-px flex-1 bg-border/80" />
                    </div>
                );
            }

            items.push(<LogCard key={`log-${log.id}`} log={log} />);
            return items;
        });
    }, [locale, segmentedLogs, t]);

    return (
        <PageWrapper className="grid grid-cols-1 gap-4">
            {renderedItems}

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
