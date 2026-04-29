'use client';

import dayjs from 'dayjs';
import type { ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';
import { AlertCircle, Loader2 } from 'lucide-react';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { cn } from '@/lib/utils';

export function formatPercent(value: number | undefined) {
    const raw = value ?? 0;
    return {
        raw,
        formatted: {
            value: raw.toFixed(2),
            unit: '%',
        },
    };
}

export function formatUnixTime(value: number | undefined) {
    if (!value) {
        return '';
    }
    return dayjs.unix(value).format('MM/DD HH:mm');
}

export function getErrorMessage(error: unknown) {
    if (error && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
        return error.message;
    }
    return 'Unexpected error';
}

export function QueryState({
    loading,
    error,
    empty,
    emptyLabel,
    children,
}: {
    loading: boolean;
    error: unknown;
    empty: boolean;
    emptyLabel: string;
    children: ReactNode;
}) {
    if (loading) {
        return (
            <div className="flex min-h-40 flex-col items-center justify-center gap-3 text-sm text-muted-foreground">
                <Loader2 className="h-8 w-8 animate-spin opacity-70" />
                <span>{emptyLabel}</span>
            </div>
        );
    }

    if (error) {
        return (
            <div className="flex min-h-40 flex-col items-center justify-center gap-3 text-sm text-destructive">
                <AlertCircle className="h-8 w-8 opacity-80" />
                <span>{getErrorMessage(error)}</span>
            </div>
        );
    }

    if (empty) {
        return (
            <div className="flex min-h-40 items-center justify-center text-sm text-muted-foreground">
                {emptyLabel}
            </div>
        );
    }

    return <>{children}</>;
}

export function MetricCard({
    title,
    value,
    unit,
    icon: Icon,
    helper,
    accentClassName,
}: {
    title: string;
    value: string | number;
    unit?: string;
    icon: LucideIcon;
    helper?: string;
    accentClassName?: string;
}) {
    return (
        <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
            <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                    <div className="text-xs text-muted-foreground">{title}</div>
                    <div className="mt-2 flex items-baseline gap-1">
                        <span className="text-2xl font-semibold">
                            <AnimatedNumber value={value} />
                        </span>
                        {unit ? <span className="text-sm text-muted-foreground">{unit}</span> : null}
                    </div>
                    {helper ? <p className="mt-2 text-xs text-muted-foreground">{helper}</p> : null}
                </div>
                <div
                    className={cn(
                        'flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl bg-primary/10 text-primary',
                        accentClassName,
                    )}
                >
                    <Icon className="h-4 w-4" />
                </div>
            </div>
        </article>
    );
}

export function StatusBadge({
    label,
    tone,
}: {
    label: string;
    tone: 'success' | 'warning' | 'danger' | 'neutral';
}) {
    const toneClassName = {
        success: 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300',
        warning: 'bg-amber-500/10 text-amber-700 dark:text-amber-300',
        danger: 'bg-destructive/10 text-destructive',
        neutral: 'bg-muted text-muted-foreground',
    }[tone];

    return (
        <span className={cn('inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium', toneClassName)}>
            {label}
        </span>
    );
}
