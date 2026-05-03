'use client';

import { useEffect, useRef } from 'react';
import { useLogs } from '@/api/endpoints/log';
import { LogCard } from './Item';
import { Loader2 } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { PageWrapper } from '@/components/common/PageWrapper';

/**
 * 日志页面组件
 * - 初始加载 pageSize 条历史日志
 * - SSE 实时推送新日志
 * - 滚动自动加载更多
 */
export function Log() {
    const t = useTranslations('log');
    const { logs, hasMore, isLoading, isLoadingMore, loadMore } = useLogs({ pageSize: 10 });
    const loadMoreRef = useRef<HTMLDivElement>(null);
    const armedRef = useRef(true);

    const canLoadMore = hasMore && !isLoading && !isLoadingMore && logs.length > 0;

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

                if (!armedRef.current || !canLoadMore) return;

                armedRef.current = false;
                void loadMore();
            },
            { rootMargin: '160px' }
        );

        observer.observe(target);
        return () => observer.disconnect();
    }, [canLoadMore, loadMore]);

    return (
        <PageWrapper className="grid grid-cols-1 gap-4">
            {logs.map((log) => (
                <LogCard key={`log-${log.id}`} log={log} />
            ))}

            <div key="log-load-more" ref={loadMoreRef} className="flex justify-center py-4">
                {hasMore && (isLoading || isLoadingMore) && (
                    <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                )}
                {!hasMore && logs.length > 0 && (
                    <span className="text-sm text-muted-foreground">{t('list.noMore')}</span>
                )}
            </div>
        </PageWrapper>
    );
}
