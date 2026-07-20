'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { Activity, AlertTriangle, CheckCircle2, HelpCircle, Timer, XCircle } from 'lucide-react';
import { useRealtimeDashboard, type ChannelHealthStatus } from '@/api/endpoints/stats';
import { useSettingList, SettingKey } from '@/api/endpoints/setting';
import { formatMoney, formatTime } from '@/lib/utils';
import { cn } from '@/lib/utils';

const statusStyle: Record<ChannelHealthStatus, { icon: typeof CheckCircle2; color: string; bg: string }> = {
    healthy: { icon: CheckCircle2, color: 'text-emerald-500', bg: 'bg-emerald-500/10' },
    degraded: { icon: AlertTriangle, color: 'text-amber-500', bg: 'bg-amber-500/10' },
    unhealthy: { icon: XCircle, color: 'text-destructive', bg: 'bg-destructive/10' },
    unknown: { icon: HelpCircle, color: 'text-muted-foreground', bg: 'bg-muted/40' },
};

export function HealthBoard() {
    const { data: settings } = useSettingList();
    const refreshSec = useMemo(() => {
        const raw = settings?.find((s) => s.key === SettingKey.HealthDashboardRefresh)?.value;
        const n = raw ? parseInt(raw, 10) : 15;
        return Number.isFinite(n) && n >= 3 ? n : 15;
    }, [settings]);

    const { data, isLoading } = useRealtimeDashboard(refreshSec);
    const t = useTranslations('home.health');

    const channels = data?.channels ?? [];
    const summary = data?.summary;

    const sorted = useMemo(() => {
        const order: Record<ChannelHealthStatus, number> = {
            unhealthy: 0,
            degraded: 1,
            unknown: 2,
            healthy: 3,
        };
        return [...channels].sort((a, b) => {
            const sa = order[a.health.status] ?? 9;
            const sb = order[b.health.status] ?? 9;
            if (sa !== sb) return sa - sb;
            return b.total_cost - a.total_cost;
        });
    }, [channels]);

    return (
        <div className="rounded-3xl bg-card text-card-foreground border-card-border border p-4 space-y-4">
            <div className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                    <Activity className="w-4 h-4 text-primary" />
                    <h3 className="font-semibold text-base">{t('title')}</h3>
                    <span className="text-[10px] text-muted-foreground">/{refreshSec}s</span>
                </div>
                {summary && (
                    <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                        <span className="inline-flex items-center gap-1">
                            <span className="w-2 h-2 rounded-full bg-emerald-500" />
                            {summary.healthy_count}
                        </span>
                        <span className="inline-flex items-center gap-1">
                            <span className="w-2 h-2 rounded-full bg-amber-500" />
                            {summary.degraded_count}
                        </span>
                        <span className="inline-flex items-center gap-1">
                            <span className="w-2 h-2 rounded-full bg-destructive" />
                            {summary.unhealthy_count}
                        </span>
                        <span className="inline-flex items-center gap-1 tabular-nums">
                            {t('successRate')}: {summary.success_rate.toFixed(1)}%
                        </span>
                        <span className="inline-flex items-center gap-1 tabular-nums">
                            <Timer className="w-3 h-3" />
                            {formatTime(summary.avg_latency_ms).formatted.value}
                            {formatTime(summary.avg_latency_ms).formatted.unit}
                        </span>
                    </div>
                )}
            </div>

            {isLoading && !data ? (
                <div className="py-8 text-center text-sm text-muted-foreground">{t('loading')}</div>
            ) : sorted.length === 0 ? (
                <div className="py-8 text-center text-sm text-muted-foreground">{t('noData')}</div>
            ) : (
                <div className="space-y-2 max-h-[360px] overflow-y-auto">
                    {sorted.map((ch) => {
                        const st = statusStyle[ch.health.status] ?? statusStyle.unknown;
                        const Icon = st.icon;
                        const cost = formatMoney(ch.total_cost);
                        const latency = formatTime(ch.avg_latency_ms);
                        return (
                            <div
                                key={ch.channel_id}
                                className="flex items-center gap-3 p-3 rounded-2xl hover:bg-accent/5 transition-colors"
                            >
                                <div className={cn('w-9 h-9 rounded-xl flex items-center justify-center shrink-0', st.bg, st.color)}>
                                    <Icon className="w-4 h-4" />
                                </div>
                                <div className="flex-1 min-w-0">
                                    <div className="flex items-center gap-2">
                                        <p className="font-medium text-sm truncate">{ch.channel_name}</p>
                                        {!ch.enabled && (
                                            <span className="text-[10px] px-1.5 py-0.5 rounded-md bg-muted text-muted-foreground">
                                                {t('disabled')}
                                            </span>
                                        )}
                                        {ch.health.circuit_open && (
                                            <span className="text-[10px] px-1.5 py-0.5 rounded-md bg-destructive/10 text-destructive">
                                                {t('circuit')}
                                                {ch.health.circuit_remain_sec
                                                    ? ` ${ch.health.circuit_remain_sec}s`
                                                    : ''}
                                            </span>
                                        )}
                                    </div>
                                    <div className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-muted-foreground mt-1">
                                        <span>
                                            {t('status')}: {t(`status_${ch.health.status}`)}
                                        </span>
                                        {ch.health.last_probe_delay > 0 && (
                                            <span>
                                                {t('probe')}: {ch.health.last_probe_delay}ms
                                            </span>
                                        )}
                                        {ch.health.last_probe_error && (
                                            <span className="truncate max-w-[220px]" title={ch.health.last_probe_error}>
                                                {ch.health.last_probe_error}
                                            </span>
                                        )}
                                    </div>
                                </div>
                                <div className="text-right shrink-0 space-y-0.5">
                                    <div className="text-sm font-semibold tabular-nums">
                                        {ch.success_rate.toFixed(1)}%
                                    </div>
                                    <div className="text-xs text-muted-foreground tabular-nums">
                                        {latency.formatted.value}
                                        {latency.formatted.unit}
                                        {' · '}
                                        {cost.formatted.value}
                                        {cost.formatted.unit}
                                    </div>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
