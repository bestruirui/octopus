'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { useLogs } from '@/api/endpoints/log';
import { useGroupList } from '@/api/endpoints/group';
import { PageWrapper } from '@/components/common/PageWrapper';
import { LogCard } from './Item';
import { Loader2 } from 'lucide-react';
import { useTranslations } from 'next-intl';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';

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
    const loadMoreRef = useRef<HTMLDivElement>(null);
    const armedRef = useRef(true);

    const groupOptions = useMemo(() => {
        const names = groups
            .map((group) => group.name?.trim())
            .filter((name): name is string => !!name);
        return Array.from(new Set(names)).sort((a, b) => a.localeCompare(b));
    }, [groups]);

    const modelOptions = useMemo(() => {
        const names = logs
            .map((log) => log.actual_model_name?.trim())
            .filter((name): name is string => !!name);
        return Array.from(new Set(names)).sort((a, b) => a.localeCompare(b));
    }, [logs]);

    const channelOptions = useMemo(() => {
        const names = new Set<string>();
        logs.forEach((log) => {
            if (log.channel_name?.trim()) {
                names.add(log.channel_name.trim());
            }
            log.attempts?.forEach((attempt) => {
                if (attempt.channel_name?.trim()) {
                    names.add(attempt.channel_name.trim());
                }
            });
        });
        return Array.from(names).sort((a, b) => a.localeCompare(b));
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

    return (
        <PageWrapper className="grid grid-cols-1 gap-4">
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-2">
                <Select value={selectedGroup} onValueChange={setSelectedGroup}>
                    <SelectTrigger className="w-full">
                        <SelectValue placeholder={t('filter.group')} />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="all">{t('filter.allGroups')}</SelectItem>
                        {groupOptions.map((group) => (
                            <SelectItem key={group} value={group}>{group}</SelectItem>
                        ))}
                    </SelectContent>
                </Select>

                <Select value={selectedModel} onValueChange={setSelectedModel}>
                    <SelectTrigger className="w-full">
                        <SelectValue placeholder={t('filter.model')} />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="all">{t('filter.allModels')}</SelectItem>
                        {modelOptions.map((model) => (
                            <SelectItem key={model} value={model}>{model}</SelectItem>
                        ))}
                    </SelectContent>
                </Select>

                <Select value={selectedRetried} onValueChange={(value) => setSelectedRetried(value as 'all' | 'yes' | 'no')}>
                    <SelectTrigger className="w-full">
                        <SelectValue placeholder={t('filter.retried')} />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="all">{t('filter.allRetries')}</SelectItem>
                        <SelectItem value="yes">{t('filter.retriedYes')}</SelectItem>
                        <SelectItem value="no">{t('filter.retriedNo')}</SelectItem>
                    </SelectContent>
                </Select>

                <Select value={selectedChannel} onValueChange={setSelectedChannel}>
                    <SelectTrigger className="w-full">
                        <SelectValue placeholder={t('filter.channel')} />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="all">{t('filter.allChannels')}</SelectItem>
                        {channelOptions.map((channel) => (
                            <SelectItem key={channel} value={channel}>{channel}</SelectItem>
                        ))}
                    </SelectContent>
                </Select>
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
