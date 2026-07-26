'use client';

import type { ReactNode } from 'react';
import { useTranslations } from 'next-intl';
import { Cloud, ExternalLink, LoaderCircle, Play, RefreshCw, Square, Terminal, type LucideIcon } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { CopyIconButton } from '@/components/common/CopyButton';
import { toast } from '@/components/common/Toast';
import {
    type TailscalePhase,
    useStartTailscaleFunnel,
    useStopTailscaleFunnel,
    useTailscaleStatus,
} from '@/api/endpoints/tailscale';
import type { ApiError } from '@/api/types';

const PHASE_STYLES: Record<TailscalePhase, string> = {
    unsupported: 'border-border text-muted-foreground',
    not_installed: 'border-border text-muted-foreground',
    needs_login: 'border-amber-500/40 bg-amber-500/10 text-amber-600 dark:text-amber-400',
    stopped: 'border-border text-muted-foreground',
    running: 'border-emerald-500/40 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
    conflict: 'border-amber-500/40 bg-amber-500/10 text-amber-600 dark:text-amber-400',
    credentials_needed: 'border-amber-500/40 bg-amber-500/10 text-amber-600 dark:text-amber-400',
    error: 'border-destructive/40 bg-destructive/10 text-destructive',
};

function TailscaleCard({ children, title, icon: Icon }: { children: ReactNode; title: string; icon: LucideIcon }) {
    return (
        <div className="space-y-5 rounded-3xl border border-border bg-card p-6">
            <h2 className="flex items-center gap-2 text-lg font-bold text-card-foreground">
                <Icon className="h-5 w-5" />
                {title}
            </h2>
            {children}
        </div>
    );
}

function TailscaleRow({ children, label, icon: Icon }: { children: ReactNode; label: string; icon?: LucideIcon }) {
    return (
        <div className="flex items-center justify-between gap-4">
            <div className="flex min-w-0 items-center gap-3">
                {Icon ? <Icon className="h-5 w-5 shrink-0 text-muted-foreground" /> : null}
                <span className="text-sm font-medium">{label}</span>
            </div>
            {children}
        </div>
    );
}

function URLValue({ value }: { value?: string }) {
    if (!value) return <span className="text-sm text-muted-foreground">—</span>;
    return (
        <div className="flex min-w-0 max-w-64 items-center gap-2">
            <a
                href={value}
                target="_blank"
                rel="noreferrer"
                className="truncate font-mono text-xs text-primary hover:underline"
                title={value}
            >
                {value}
            </a>
            <CopyIconButton
                text={value}
                className="shrink-0 text-muted-foreground transition-colors hover:text-foreground"
                copyIconClassName="size-4"
                checkIconClassName="size-4 text-emerald-500"
            />
        </div>
    );
}

export function SettingTailscale() {
    const t = useTranslations('setting.tailscale');
    const statusQuery = useTailscaleStatus();
    const startFunnel = useStartTailscaleFunnel();
    const stopFunnel = useStopTailscaleFunnel();
    const status = statusQuery.data;
    const actionPending = startFunnel.isPending || stopFunnel.isPending;

    const handleToggle = async () => {
        if (!status) return;
        if (!status.running && !window.confirm(t('confirmStart'))) return;

        try {
            if (status.running) {
                await stopFunnel.mutateAsync();
                toast.success(t('stopSuccess'));
            } else {
                await startFunnel.mutateAsync();
                toast.success(t('startSuccess'));
            }
        } catch (error) {
            toast.error(t('actionFailed'), { description: (error as ApiError)?.message });
        }
    };

    const actionDisabled = actionPending
        || !status
        || !status.supported
        || !status.installed
        || status.phase === 'needs_login'
        || status.phase === 'credentials_needed'
        || (status.default_credentials && !status.running)
        || status.phase === 'conflict';

    return (
        <TailscaleCard icon={Cloud} title={t('title')}>
            <div className="space-y-2 text-sm text-muted-foreground">
                <p>{t('description')}</p>
                <p className="text-amber-600 dark:text-amber-400">{t('warning')}</p>
            </div>

            <TailscaleRow label={t('status')}>
                {status ? (
                    <span className={`rounded-full border px-2.5 py-1 text-xs font-medium ${PHASE_STYLES[status.phase]}`}>
                        {t(`phase.${status.phase}`)}
                    </span>
                ) : (
                    <span className="text-sm text-muted-foreground">{statusQuery.isPending ? t('loading') : t('loadFailed')}</span>
                )}
            </TailscaleRow>

            {status ? (
                <>
                    <TailscaleRow label={t('publicUrl')}>
                        <URLValue value={status.public_url} />
                    </TailscaleRow>
                    <TailscaleRow label={t('apiUrl')}>
                        <URLValue value={status.api_url} />
                    </TailscaleRow>
                    <TailscaleRow label={t('target')}>
                        <code className="max-w-64 truncate text-xs" title={status.target_url}>{status.target_url}</code>
                    </TailscaleRow>
                    <TailscaleRow icon={Terminal} label={t('binary')}>
                        <code className="max-w-64 truncate text-xs text-muted-foreground" title={status.binary_path || t('notFound')}>
                            {status.binary_path || t('notFound')}
                        </code>
                    </TailscaleRow>
                </>
            ) : null}

            {status?.phase === 'not_installed' ? (
                <p className="rounded-2xl bg-muted/60 p-3 text-xs text-muted-foreground">
                    {t('installHint')}{' '}
                    <a
                        href="https://tailscale.com/download"
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex items-center gap-1 text-primary hover:underline"
                    >
                        {t('installLink')} <ExternalLink className="size-3" />
                    </a>
                </p>
            ) : null}

            {status?.phase === 'credentials_needed' ? (
                <p className="rounded-2xl bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-300">{t('credentialHint')}</p>
            ) : null}

            {status?.phase === 'needs_login' ? (
                <p className="rounded-2xl bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-300">
                    {t('loginHint')} <code>tailscale login</code>
                </p>
            ) : null}

            {status?.approval_url ? (
                <a
                    href={status.approval_url}
                    target="_blank"
                    rel="noreferrer"
                    className="flex items-center gap-1 rounded-2xl bg-amber-500/10 p-3 text-xs text-amber-700 hover:underline dark:text-amber-300"
                >
                    {t('approvalHint')} <ExternalLink className="size-3" />
                </a>
            ) : null}

            {status?.last_error ? (
                <p className="break-words text-xs text-destructive">{t('lastError')}: {status.last_error}</p>
            ) : null}

            <div className="flex justify-end gap-2 border-t border-border pt-4">
                <Button
                    variant="ghost"
                    size="sm"
                    className="rounded-xl"
                    onClick={() => statusQuery.refetch()}
                    disabled={statusQuery.isFetching || actionPending}
                >
                    <RefreshCw className={`size-4 ${statusQuery.isFetching ? 'animate-spin' : ''}`} />
                    {t('refresh')}
                </Button>
                <Button
                    variant={status?.running ? 'destructive' : 'default'}
                    size="sm"
                    className="rounded-xl"
                    onClick={handleToggle}
                    disabled={actionDisabled}
                >
                    {actionPending ? (
                        <LoaderCircle className="size-4 animate-spin" />
                    ) : status?.running ? (
                        <Square className="size-4" />
                    ) : (
                        <Play className="size-4" />
                    )}
                    {actionPending
                        ? (status?.running ? t('stopping') : t('starting'))
                        : (status?.running ? t('stop') : t('start'))}
                </Button>
            </div>
        </TailscaleCard>
    );
}
