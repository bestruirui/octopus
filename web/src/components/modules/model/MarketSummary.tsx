'use client';

import { Boxes, Clock3, RefreshCw, RadioTower, Rows3 } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';

type MarketSummaryValue = {
    model_count: number;
    coverage_count: number;
    unique_channel_count: number;
    average_latency_ms: number;
    last_update_time: string;
};

function formatLastUpdate(value: string, fallback: string) {
    if (!value) return fallback;

    const date = new Date(value);
    if (Number.isNaN(date.getTime()) || date.getFullYear() <= 1) {
        return fallback;
    }

    return date.toLocaleString();
}

export function ModelMarketSummary({
    summary,
    onRefresh,
    isRefreshing,
}: {
    summary: MarketSummaryValue;
    onRefresh: () => void;
    isRefreshing: boolean;
}) {
    const t = useTranslations('model');

    const metrics = [
        {
            key: 'models',
            icon: Boxes,
            label: t('summary.modelCount'),
            value: summary.model_count.toLocaleString(),
        },
        {
            key: 'coverage',
            icon: Rows3,
            label: t('summary.coverage'),
            value: summary.coverage_count.toLocaleString(),
        },
        {
            key: 'unique',
            icon: RadioTower,
            label: t('summary.uniqueChannels'),
            value: summary.unique_channel_count.toLocaleString(),
        },
        {
            key: 'latency',
            icon: Clock3,
            label: t('summary.averageLatency'),
            value: summary.average_latency_ms > 0 ? `${summary.average_latency_ms}ms` : '—',
        },
    ];

    return (
        <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
            <div className="flex flex-wrap items-start justify-between gap-4">
                <div className="space-y-1">
                    <h2 className="text-xl font-semibold">{t('summary.title')}</h2>
                    <p className="max-w-3xl text-sm leading-6 text-muted-foreground">{t('summary.description')}</p>
                </div>
                <div className="flex items-center gap-3">
                    <span className="text-sm text-muted-foreground">
                        {t('summary.lastUpdate')}: {formatLastUpdate(summary.last_update_time, t('summary.neverUpdated'))}
                    </span>
                    <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={onRefresh}
                        disabled={isRefreshing}
                        className="rounded-xl"
                    >
                        <RefreshCw className={`mr-2 h-4 w-4 ${isRefreshing ? 'animate-spin' : ''}`} />
                        {isRefreshing ? t('summary.refreshing') : t('summary.refresh')}
                    </Button>
                </div>
            </div>
            <div className="mt-5 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
                {metrics.map((metric) => (
                    <div key={metric.key} className="rounded-2xl border border-border/60 bg-muted/20 p-4">
                        <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            <metric.icon className="h-4 w-4" />
                            <span>{metric.label}</span>
                        </div>
                        <div className="mt-2 text-2xl font-semibold tracking-tight">{metric.value}</div>
                    </div>
                ))}
            </div>
        </section>
    );
}
