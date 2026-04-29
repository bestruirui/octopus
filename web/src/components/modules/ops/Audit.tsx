'use client';

import { useCallback, useMemo } from 'react';
import { Clock3, FileText, Loader2, ShieldAlert, User } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { motion } from 'motion/react';
import { DEFAULT_AUDIT_PAGE_SIZE, type AuditLogEntry, useAuditLogDetail, useAuditLogs } from '@/api/endpoints/ops';
import { QueryState, StatusBadge } from '@/components/modules/analytics/shared';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import {
    MorphingDialog,
    MorphingDialogClose,
    MorphingDialogContainer,
    MorphingDialogContent,
    MorphingDialogDescription,
    MorphingDialogTitle,
    MorphingDialogTrigger,
} from '@/components/ui/morphing-dialog';

function formatAuditTime(timestamp: number) {
    if (!timestamp) {
        return '-';
    }
    return new Date(timestamp * 1000).toLocaleString();
}

function getStatusTone(statusCode: number) {
    if (statusCode >= 500) return 'danger' as const;
    if (statusCode >= 400) return 'warning' as const;
    return 'success' as const;
}

function getMethodClassName(method: string) {
    switch (method.toUpperCase()) {
        case 'DELETE':
            return 'bg-destructive/10 text-destructive';
        case 'PUT':
        case 'PATCH':
            return 'bg-amber-500/10 text-amber-700 dark:text-amber-300';
        default:
            return 'bg-primary/10 text-primary';
    }
}

function formatAction(action: string) {
    return action.replaceAll('.', ' / ');
}

function InfoRow({ label, value }: { label: string; value: string }) {
    return (
        <div className="flex items-start justify-between gap-4 border-b border-border/40 py-3 last:border-b-0 last:pb-0 first:pt-0">
            <span className="text-sm text-muted-foreground">{label}</span>
            <span className="max-w-[65%] break-all text-right text-sm font-medium">{value || '-'}</span>
        </div>
    );
}

function AuditRecordCard({ item }: { item: AuditLogEntry }) {
    const t = useTranslations('ops');
    const { detail, isLoading, fetchDetail, reset } = useAuditLogDetail();
    const statusTone = getStatusTone(item.status_code);

    const handleOpen = useCallback(() => {
        void fetchDetail(item.id);
    }, [fetchDetail, item.id]);

    return (
        <MorphingDialog onOpen={handleOpen} onClose={reset}>
            <MorphingDialogTrigger
                className={cn(
                    'w-full rounded-3xl border bg-card text-left transition-colors',
                    item.status_code >= 400
                        ? 'border-amber-500/30 hover:bg-amber-500/5'
                        : 'border-border hover:bg-muted/20',
                )}
            >
                <div className="flex flex-col gap-3 p-4">
                    <div className="flex items-start justify-between gap-3">
                        <div className="min-w-0">
                            <div className="truncate text-sm font-semibold text-card-foreground" title={item.action}>
                                {formatAction(item.action)}
                            </div>
                            <div className="mt-1 truncate text-xs text-muted-foreground" title={item.target || item.path}>
                                {item.target || item.path}
                            </div>
                        </div>
                        <div className="flex shrink-0 items-center gap-2">
                            <Badge variant="secondary" className={cn('border-0 text-xs', getMethodClassName(item.method))}>
                                {item.method}
                            </Badge>
                            <StatusBadge label={String(item.status_code)} tone={statusTone} />
                        </div>
                    </div>

                    <div className="grid grid-cols-1 gap-2 text-xs text-muted-foreground md:grid-cols-3">
                        <div className="flex items-center gap-1.5">
                            <User className="h-3.5 w-3.5 shrink-0" />
                            <span className="truncate">{item.username}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                            <Clock3 className="h-3.5 w-3.5 shrink-0" />
                            <span>{formatAuditTime(item.created_at)}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                            <FileText className="h-3.5 w-3.5 shrink-0" />
                            <span className="truncate" title={item.path}>{item.path}</span>
                        </div>
                    </div>
                </div>
            </MorphingDialogTrigger>

            <MorphingDialogContainer>
                <MorphingDialogContent className="relative w-[calc(100vw-2rem)] rounded-3xl bg-card px-6 py-4 text-card-foreground md:w-[62rem]">
                    <MorphingDialogClose className="right-5 top-4 text-muted-foreground hover:text-foreground" />
                    <MorphingDialogTitle className="flex items-center gap-2 text-sm">
                        <ShieldAlert className="h-4 w-4 text-primary" />
                        <span>{t('audit.detail.entryTitle', { id: item.id })}</span>
                    </MorphingDialogTitle>
                    <MorphingDialogDescription className="mt-4">
                        {isLoading ? (
                            <div className="flex min-h-60 items-center justify-center">
                                <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                            </div>
                        ) : !detail ? (
                            <div className="flex min-h-40 items-center justify-center text-sm text-muted-foreground">
                                {t('audit.detail.notFound')}
                            </div>
                        ) : (
                            <div className="space-y-4">
                                <div className="grid grid-cols-1 gap-4 md:grid-cols-[1.1fr_0.9fr]">
                                    <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                                        <div className="mb-2 text-sm font-semibold">{t('audit.detail.summary')}</div>
                                        <InfoRow label={t('audit.fields.action')} value={detail.action} />
                                        <InfoRow label={t('audit.fields.target')} value={detail.target || t('audit.detail.emptyTarget')} />
                                        <InfoRow label={t('audit.fields.path')} value={detail.path} />
                                        <InfoRow label={t('audit.fields.createdAt')} value={formatAuditTime(detail.created_at)} />
                                    </article>

                                    <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                                        <div className="mb-2 text-sm font-semibold">{t('audit.detail.context')}</div>
                                        <InfoRow label={t('audit.fields.username')} value={detail.username} />
                                        <InfoRow label={t('audit.fields.userId')} value={String(detail.user_id)} />
                                        <InfoRow label={t('audit.fields.method')} value={detail.method} />
                                        <InfoRow label={t('audit.fields.statusCode')} value={String(detail.status_code)} />
                                    </article>
                                </div>

                                <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                                    <div className="mb-3 text-sm font-semibold">{t('audit.detail.rawRecord')}</div>
                                    <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded-2xl border border-border/40 bg-card p-3 text-xs text-muted-foreground">
                                        {JSON.stringify(detail, null, 2)}
                                    </pre>
                                </article>
                            </div>
                        )}
                    </MorphingDialogDescription>
                </MorphingDialogContent>
            </MorphingDialogContainer>
        </MorphingDialog>
    );
}

export function Audit() {
    const t = useTranslations('ops');
    const { logs, error, hasMore, isLoading, isLoadingMore, loadMore } = useAuditLogs({ pageSize: DEFAULT_AUDIT_PAGE_SIZE });

    const canLoadMore = hasMore && !isLoading && !isLoadingMore && logs.length > 0;
    const handleReachEnd = useCallback(() => {
        if (!canLoadMore) return;
        void loadMore();
    }, [canLoadMore, loadMore]);

    const footer = useMemo(() => {
        if (hasMore && isLoadingMore) {
            return (
                <div className="flex justify-center py-4">
                    <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
            );
        }
        if (!hasMore && logs.length > 0) {
            return (
                <div className="flex justify-center py-4">
                    <span className="text-sm text-muted-foreground">{t('audit.list.noMore')}</span>
                </div>
            );
        }
        return null;
    }, [hasMore, isLoadingMore, logs.length, t]);

    return (
        <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
            <div className="mb-4 flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-1">
                    <h3 className="text-base font-semibold">{t('tabs.audit')}</h3>
                    <p className="text-sm leading-6 text-muted-foreground">{t('audit.description')}</p>
                </div>
                <div className="flex items-center gap-2">
                    <Badge variant="secondary" className="rounded-full border-0 bg-muted px-3 py-1 text-xs text-muted-foreground">
                        {t('audit.writeScope')}
                    </Badge>
                </div>
            </div>

            <QueryState
                loading={isLoading}
                error={error}
                empty={logs.length === 0}
                emptyLabel={t('audit.empty')}
            >
                <motion.div
                    initial={{ opacity: 0, y: 8 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.2 }}
                    className="h-[30rem] min-h-[30rem] md:h-[36rem]"
                >
                    <VirtualizedGrid
                        items={logs}
                        layout="list"
                        columns={{ default: 1 }}
                        estimateItemHeight={112}
                        overscan={6}
                        getItemKey={(item) => `audit-${item.id}`}
                        renderItem={(item) => <AuditRecordCard item={item} />}
                        footer={footer}
                        onReachEnd={handleReachEnd}
                        reachEndEnabled={canLoadMore}
                        reachEndOffset={2}
                    />
                </motion.div>
            </QueryState>
        </section>
    );
}
