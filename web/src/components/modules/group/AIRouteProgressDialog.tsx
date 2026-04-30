'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import type {
    AIRouteBatchStatus,
    AIRouteChannelStatus,
    AIRouteScope,
    AIRouteTaskStatus,
    AIRouteTaskStep,
    GenerateAIRouteCurrentBatch,
    GenerateAIRouteProgress,
} from '@/api/endpoints/group';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog';
import { Progress } from '@/components/ui/progress';
import { cn } from '@/lib/utils';
import { resolveRuntimeI18nMessage } from '@/lib/i18n-runtime';
import { endpointTypeLabelKey } from './utils';

type AIRouteProgressDialogProps = {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    progress: GenerateAIRouteProgress | null | undefined;
    scope: AIRouteScope;
};

const HEARTBEAT_STALE_MS = 15000;

function formatTime(value?: string) {
    if (!value) {
        return '--';
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
        return '--';
    }

    return new Intl.DateTimeFormat(undefined, {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
    }).format(date);
}

function statusBadgeClass(status?: AIRouteTaskStatus) {
    switch (status) {
        case 'completed':
            return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300';
        case 'failed':
        case 'timeout':
            return 'border-destructive/30 bg-destructive/10 text-destructive';
        case 'queued':
            return 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300';
        default:
            return 'border-primary/30 bg-primary/10 text-primary';
    }
}

function channelBadgeClass(status?: AIRouteChannelStatus) {
    switch (status) {
        case 'completed':
            return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300';
        case 'failed':
            return 'border-destructive/30 bg-destructive/10 text-destructive';
        case 'running':
            return 'border-primary/30 bg-primary/10 text-primary';
        default:
            return 'border-muted-foreground/30 bg-muted text-muted-foreground';
    }
}

function batchBadgeClass(status?: AIRouteBatchStatus | string) {
    switch (status) {
        case 'completed':
            return 'border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300';
        case 'failed':
            return 'border-destructive/30 bg-destructive/10 text-destructive';
        case 'parsing':
            return 'border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300';
        case 'retrying':
            return 'border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300';
        default:
            return 'border-primary/30 bg-primary/10 text-primary';
    }
}

type BatchCardData = Pick<
    GenerateAIRouteCurrentBatch,
    'index' | 'total' | 'endpoint_type' | 'model_count' | 'channel_ids' | 'channel_names' | 'service_name' | 'attempt' | 'status' | 'message' | 'message_key' | 'message_args'
>;

export function AIRouteProgressDialog({
    open,
    onOpenChange,
    progress,
    scope,
}: AIRouteProgressDialogProps) {
    const t = useTranslations('group');
    const [nowTs, setNowTs] = useState(0);

    const status = progress?.status ?? (progress?.done ? 'completed' : 'queued');
    const step = progress?.current_step ?? (progress?.done ? 'completed' : 'queued');
    const summary = progress?.summary;
    const currentBatch = progress?.current_batch;
    const runningBatches = progress?.running_batches ?? [];
    const activeBatches: BatchCardData[] = runningBatches.length > 0
        ? runningBatches
        : currentBatch && !progress?.done && currentBatch.status !== 'completed'
            ? [currentBatch]
            : [];
    const shouldShowRecentBatch = Boolean(
        currentBatch
        && (
            activeBatches.length === 0
            || currentBatch.status === 'completed'
            || currentBatch.status === 'failed'
            || !activeBatches.some((batch) => batch.index === currentBatch.index && batch.status === currentBatch.status)
        ),
    );
    const channels = progress?.channels ?? [];
    const runningChannels = channels.filter((channel) => channel.status === 'running');
    const completedChannels = channels.filter((channel) => channel.status === 'completed');
    const failedChannels = channels.filter((channel) => channel.status === 'failed');
    const pendingChannels = channels.filter((channel) => channel.status !== 'running' && channel.status !== 'completed' && channel.status !== 'failed');

    useEffect(() => {
        if (!open || progress?.done) {
            return;
        }

        const bootstrapTimer = window.setTimeout(() => {
            setNowTs(Date.now());
        }, 0);
        const timer = window.setInterval(() => {
            setNowTs(Date.now());
        }, 5000);

        return () => {
            window.clearTimeout(bootstrapTimer);
            window.clearInterval(timer);
        };
    }, [open, progress?.done]);

    const isHeartbeatStale = (() => {
        if (!progress?.heartbeat_at || progress.done || nowTs <= 0) {
            return false;
        }
        const heartbeat = new Date(progress.heartbeat_at).getTime();
        if (Number.isNaN(heartbeat)) {
            return false;
        }
        return nowTs - heartbeat > HEARTBEAT_STALE_MS;
    })();

    const resultSummary = (() => {
        if (!progress?.done || !progress.result) {
            return null;
        }

        if (scope === 'group') {
            return t('aiRoute.progress.result.groupSummary', {
                routes: progress.result.route_count,
                items: progress.result.item_count,
            });
        }

        return t('aiRoute.progress.result.tableSummary', {
            routes: progress.result.route_count,
            groups: progress.result.group_count,
            items: progress.result.item_count,
        });
    })();

    const descriptionKey = scope === 'group'
        ? 'aiRoute.progress.description.group'
        : 'aiRoute.progress.description.table';

    const resolveMessage = (message?: string, messageKey?: string, messageArgs?: Record<string, unknown>) =>
        resolveRuntimeI18nMessage(messageKey, messageArgs, message) ?? message;

    const renderChannelCards = (items: typeof channels, emptyText: string) => {
        if (items.length === 0) {
            return (
                <div className="rounded-2xl border border-dashed border-border/60 bg-muted/10 px-4 py-3 text-sm text-muted-foreground">
                    {emptyText}
                </div>
            );
        }

        return (
            <div className="space-y-2">
                {items.map((channel) => {
                    const channelPercent = channel.total_models > 0
                        ? Math.min(100, Math.round((channel.processed_models / channel.total_models) * 100))
                        : 0;

                    return (
                        <div
                            key={channel.channel_id}
                            className="rounded-2xl border border-border/60 bg-background px-4 py-3"
                        >
                            <div className="flex flex-wrap items-start justify-between gap-3">
                                <div className="min-w-0">
                                    <div className="truncate text-sm font-medium text-foreground">
                                        {channel.channel_name || t('aiRoute.progress.channelFallbackName', { id: channel.channel_id })}
                                    </div>
                                    <div className="mt-1 text-xs text-muted-foreground">
                                        {channel.provider || '--'}
                                    </div>
                                </div>
                                <Badge variant="outline" className={cn('rounded-full', channelBadgeClass(channel.status))}>
                                    {t(`aiRoute.progress.channelStatus.${channel.status ?? 'pending'}`)}
                                </Badge>
                            </div>
                            <div className="mt-3 flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
                                <span>
                                    {t('aiRoute.progress.channelModels', {
                                        completed: channel.processed_models,
                                        total: channel.total_models,
                                    })}
                                </span>
                                {resolveMessage(channel.message, channel.message_key, channel.message_args) ? (
                                    <span className="max-w-full truncate">{resolveMessage(channel.message, channel.message_key, channel.message_args)}</span>
                                ) : null}
                            </div>
                            <Progress value={channelPercent} className="mt-2 h-1.5" />
                        </div>
                    );
                })}
            </div>
        );
    };

    const renderBatchCard = (batch: BatchCardData, title: string) => (
        <section className="rounded-2xl border border-border/60 bg-background px-4 py-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="text-sm font-medium text-foreground">{title}</div>
                <div className="flex flex-wrap items-center gap-2">
                    {batch.status ? (
                        <Badge variant="outline" className={cn('rounded-full', batchBadgeClass(batch.status))}>
                            {t(`aiRoute.progress.batchStatus.${batch.status}`)}
                        </Badge>
                    ) : null}
                    <Badge variant="outline" className="rounded-full">
                        {t('aiRoute.progress.batchLabel', {
                            index: batch.index,
                            total: batch.total,
                        })}
                    </Badge>
                </div>
            </div>
            <div className="mt-3 grid gap-3 md:grid-cols-5">
                <div>
                    <div className="text-xs text-muted-foreground">{t('aiRoute.progress.batchCapability')}</div>
                    <div className="mt-1 text-sm font-medium text-foreground">
                        {batch.endpoint_type ? t(endpointTypeLabelKey(batch.endpoint_type) ?? 'form.endpointType.options.all') : '--'}
                    </div>
                </div>
                <div>
                    <div className="text-xs text-muted-foreground">{t('aiRoute.progress.batchModels')}</div>
                    <div className="mt-1 text-sm font-medium text-foreground">{batch.model_count}</div>
                </div>
                <div>
                    <div className="text-xs text-muted-foreground">{t('aiRoute.progress.batchChannels')}</div>
                    <div className="mt-1 text-sm font-medium text-foreground">
                        {batch.channel_names?.length || batch.channel_ids?.length || 0}
                    </div>
                </div>
                <div>
                    <div className="text-xs text-muted-foreground">{t('aiRoute.progress.batchService')}</div>
                    <div className="mt-1 text-sm font-medium text-foreground">{batch.service_name || '--'}</div>
                </div>
                <div>
                    <div className="text-xs text-muted-foreground">{t('aiRoute.progress.batchAttempt')}</div>
                    <div className="mt-1 text-sm font-medium text-foreground">{batch.attempt || 1}</div>
                </div>
            </div>
            {resolveMessage(batch.message, batch.message_key, batch.message_args) ? (
                <div className="mt-3 rounded-xl bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                    {resolveMessage(batch.message, batch.message_key, batch.message_args)}
                </div>
            ) : null}
            <div className="mt-3 flex flex-wrap gap-2">
                {(batch.channel_names ?? []).map((channelName, index) => (
                    <span
                        key={`${batch.channel_ids?.[index] ?? index}-${channelName}`}
                        className="inline-flex rounded-full border border-border/60 bg-muted px-2.5 py-1 text-xs text-foreground"
                    >
                        {channelName}
                    </span>
                ))}
            </div>
        </section>
    );

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="rounded-3xl sm:max-w-3xl max-h-[calc(100vh-2rem)] overflow-hidden flex flex-col">
                <DialogHeader className="text-left">
                    <div className="flex flex-wrap items-center gap-2">
                        <DialogTitle>{t('aiRoute.progress.title')}</DialogTitle>
                        <Badge variant="outline" className={cn('rounded-full', statusBadgeClass(status))}>
                            {t(`aiRoute.progress.status.${status}`)}
                        </Badge>
                    </div>
                    <DialogDescription>{t(descriptionKey)}</DialogDescription>
                </DialogHeader>

                <div className="flex-1 min-h-0 overflow-y-auto pr-1 space-y-4">
                    <section className="rounded-2xl border border-border/60 bg-muted/20 p-4">
                        <div className="flex flex-wrap items-center justify-between gap-3">
                            <div>
                                <div className="text-sm font-medium text-foreground">
                                    {t(`aiRoute.progress.steps.${step as AIRouteTaskStep}`)}
                                </div>
                                <p className="mt-1 text-sm text-muted-foreground">
                                    {resolveMessage(progress?.message, progress?.message_key, progress?.message_args) || t('aiRoute.progress.loading')}
                                </p>
                            </div>
                            <div className="text-right">
                                <div className="text-2xl font-semibold text-foreground">
                                    {progress?.progress_percent ?? 0}%
                                </div>
                                {progress?.total_batches ? (
                                    <div className="text-xs text-muted-foreground">
                                        {t('aiRoute.progress.batchProgress', {
                                            completed: progress.completed_batches,
                                            total: progress.total_batches,
                                        })}
                                    </div>
                                ) : null}
                            </div>
                        </div>
                        <Progress value={progress?.progress_percent ?? 0} className="mt-4 h-2.5" />
                        <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                            <span>{t('aiRoute.progress.lastUpdated', { time: formatTime(progress?.updated_at) })}</span>
                            <span>{t('aiRoute.progress.lastHeartbeat', { time: formatTime(progress?.heartbeat_at) })}</span>
                            {isHeartbeatStale ? (
                                <span className="text-amber-600 dark:text-amber-300">
                                    {t('aiRoute.progress.heartbeatStale')}
                                </span>
                            ) : null}
                        </div>
                    </section>

                    {summary ? (
                        <section className="grid grid-cols-2 gap-3 md:grid-cols-3">
                            <div className="rounded-2xl border border-border/60 bg-background px-4 py-3">
                                <div className="text-xs text-muted-foreground">{t('aiRoute.progress.summary.totalChannels')}</div>
                                <div className="mt-1 text-xl font-semibold">{summary.total_channels}</div>
                            </div>
                            <div className="rounded-2xl border border-border/60 bg-background px-4 py-3">
                                <div className="text-xs text-muted-foreground">{t('aiRoute.progress.summary.completedChannels')}</div>
                                <div className="mt-1 text-xl font-semibold">{summary.completed_channels}</div>
                            </div>
                            <div className="rounded-2xl border border-border/60 bg-background px-4 py-3">
                                <div className="text-xs text-muted-foreground">{t('aiRoute.progress.summary.runningChannels')}</div>
                                <div className="mt-1 text-xl font-semibold">{summary.running_channels}</div>
                            </div>
                            <div className="rounded-2xl border border-border/60 bg-background px-4 py-3">
                                <div className="text-xs text-muted-foreground">{t('aiRoute.progress.summary.pendingChannels')}</div>
                                <div className="mt-1 text-xl font-semibold">{summary.pending_channels}</div>
                            </div>
                            <div className="rounded-2xl border border-border/60 bg-background px-4 py-3">
                                <div className="text-xs text-muted-foreground">{t('aiRoute.progress.summary.failedChannels')}</div>
                                <div className="mt-1 text-xl font-semibold">{summary.failed_channels}</div>
                            </div>
                            <div className="rounded-2xl border border-border/60 bg-background px-4 py-3">
                                <div className="text-xs text-muted-foreground">{t('aiRoute.progress.summary.modelsProgress')}</div>
                                <div className="mt-1 text-xl font-semibold">
                                    {summary.completed_models}/{summary.total_models}
                                </div>
                            </div>
                        </section>
                    ) : null}

                    {activeBatches.length > 0 ? (
                        <section className="space-y-3">
                            <div className="flex flex-wrap items-center justify-between gap-2">
                                <div className="text-sm font-medium text-foreground">{t('aiRoute.progress.activeBatches')}</div>
                                <Badge variant="outline" className="rounded-full">
                                    {t('aiRoute.progress.activeBatchCount', { count: activeBatches.length })}
                                </Badge>
                            </div>
                            <div className="grid gap-4 xl:grid-cols-2">
                                {activeBatches.map((batch) => (
                                    <div key={`${batch.index}-${batch.status ?? 'running'}-${batch.attempt ?? 0}`}>
                                        {renderBatchCard(
                                            batch,
                                            t('aiRoute.progress.currentBatch'),
                                        )}
                                    </div>
                                ))}
                            </div>
                        </section>
                    ) : null}

                    {shouldShowRecentBatch && currentBatch ? renderBatchCard(currentBatch, t('aiRoute.progress.recentBatch')) : null}

                    {resultSummary ? (
                        <section className="rounded-2xl border border-emerald-500/30 bg-emerald-500/5 px-4 py-3 text-sm text-emerald-700 dark:text-emerald-300">
                            {resultSummary}
                        </section>
                    ) : null}

                    {channels.length > 0 ? (
                        <section className="space-y-3">
                            <div className="text-sm font-medium text-foreground">{t('aiRoute.progress.channelList')}</div>
                            <div className="grid gap-4 xl:grid-cols-2">
                                <section className="space-y-2">
                                    <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                                        {t('aiRoute.progress.channelSections.running')}
                                    </div>
                                    {renderChannelCards(runningChannels, t('aiRoute.progress.channelSections.empty.running'))}
                                </section>
                                <section className="space-y-2">
                                    <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                                        {t('aiRoute.progress.channelSections.completed')}
                                    </div>
                                    {renderChannelCards(completedChannels, t('aiRoute.progress.channelSections.empty.completed'))}
                                </section>
                                <section className="space-y-2">
                                    <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                                        {t('aiRoute.progress.channelSections.pending')}
                                    </div>
                                    {renderChannelCards(pendingChannels, t('aiRoute.progress.channelSections.empty.pending'))}
                                </section>
                                <section className="space-y-2">
                                    <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                                        {t('aiRoute.progress.channelSections.failed')}
                                    </div>
                                    {renderChannelCards(failedChannels, t('aiRoute.progress.channelSections.empty.failed'))}
                                </section>
                            </div>
                        </section>
                    ) : null}
                </div>

                <DialogFooter>
                    <Button type="button" variant="outline" className="rounded-xl" onClick={() => onOpenChange(false)}>
                        {t('aiRoute.progress.close')}
                    </Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
