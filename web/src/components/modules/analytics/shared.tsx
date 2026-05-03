'use client';

import dayjs from 'dayjs';
import type { ReactNode } from 'react';
import type { LucideIcon } from 'lucide-react';
import { AlertCircle, Loader2, Waves, Orbit } from 'lucide-react';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { cn } from '@/lib/utils';
import { resolveRuntimeI18nMessage } from '@/lib/i18n-runtime';

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
    return resolveRuntimeI18nMessage('errors.unexpectedError', undefined, 'Unexpected error');
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
            <div className="waterhouse-pod flex min-h-44 flex-col items-center justify-center gap-4 rounded-[1.7rem] border border-border/30 bg-background/36 px-4 py-8 text-sm text-muted-foreground shadow-waterhouse-soft backdrop-blur-md">
                <div className="grid size-14 place-items-center rounded-[1.35rem] border border-border/25 bg-background/48 shadow-waterhouse-soft">
                    <Loader2 className="h-6 w-6 animate-spin opacity-70" />
                </div>
                <span>{emptyLabel}</span>
            </div>
        );
    }

    if (error) {
        return (
            <div className="waterhouse-pod flex min-h-44 flex-col items-center justify-center gap-4 rounded-[1.7rem] border border-destructive/20 bg-destructive/6 px-4 py-8 text-sm text-destructive shadow-waterhouse-soft backdrop-blur-md">
                <div className="grid size-14 place-items-center rounded-[1.35rem] border border-destructive/20 bg-destructive/10 shadow-waterhouse-soft">
                    <AlertCircle className="h-6 w-6 opacity-80" />
                </div>
                <span>{getErrorMessage(error)}</span>
            </div>
        );
    }

    if (empty) {
        return (
            <div className="waterhouse-pod flex min-h-44 flex-col items-center justify-center gap-4 rounded-[1.7rem] border border-dashed border-border/30 bg-background/30 px-4 py-8 text-sm text-muted-foreground shadow-waterhouse-soft backdrop-blur-md">
                <div className="grid size-14 place-items-center rounded-[1.35rem] border border-border/25 bg-background/44 shadow-waterhouse-soft">
                    <Waves className="h-6 w-6 opacity-70" />
                </div>
                <span>{emptyLabel}</span>
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
        <article className="waterhouse-pod group relative overflow-hidden rounded-[1.7rem] border border-border/30 bg-background/42 p-4 shadow-waterhouse-soft backdrop-blur-md transition-[transform,border-color,box-shadow] duration-300 hover:-translate-y-0.5 hover:border-primary/18 hover:shadow-[var(--waterhouse-shadow-soft)]">
            <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_85%_18%,color-mix(in_oklch,var(--primary)_10%,transparent)_0%,transparent_28%),linear-gradient(145deg,color-mix(in_oklch,white_8%,transparent),transparent_58%)]" />
            <div className="flex items-start justify-between gap-3">
                <div className="relative min-w-0">
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
                        'relative flex h-10 w-10 shrink-0 items-center justify-center rounded-[1.15rem] bg-primary/10 text-primary shadow-waterhouse-soft',
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
        <span className={cn('inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium shadow-waterhouse-soft', toneClassName)}>
            {label}
        </span>
    );
}

export function ObservatorySection({
    eyebrow,
    title,
    description,
    icon: Icon,
    children,
    actions,
}: {
    eyebrow?: string;
    title: string;
    description?: string;
    icon: LucideIcon;
    children: ReactNode;
    actions?: ReactNode;
}) {
    return (
        <section className="waterhouse-island relative overflow-hidden rounded-[2.15rem] border border-border/35 bg-card/58 p-5 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:p-6">
            <div className="relative space-y-5">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                    <div className="flex min-w-0 items-start gap-4">
                        <div className="waterhouse-pod grid size-12 shrink-0 place-items-center rounded-[1.35rem] border border-border/30 bg-background/46 text-primary shadow-waterhouse-soft">
                            <Icon className="h-5 w-5" />
                        </div>
                        <div className="min-w-0 space-y-2">
                            {eyebrow ? (
                                <div className="inline-flex items-center gap-2 rounded-full border border-primary/12 bg-background/44 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.2em] text-primary shadow-waterhouse-soft">
                                    <Orbit className="h-3.5 w-3.5" />
                                    {eyebrow}
                                </div>
                            ) : null}
                            <div className="space-y-1">
                                <h3 className="text-lg font-semibold tracking-tight">{title}</h3>
                                {description ? (
                                    <p className="max-w-3xl text-sm leading-6 text-muted-foreground">{description}</p>
                                ) : null}
                            </div>
                        </div>
                    </div>
                    {actions ? <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div> : null}
                </div>
                {children}
            </div>
        </section>
    );
}
