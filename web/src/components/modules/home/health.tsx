'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { Activity, AlertTriangle, CheckCircle2, HelpCircle, Timer, XCircle, Cpu, Network, Zap } from 'lucide-react';
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
                {data?.net_obs && (
                    <NetObsBadge
                        backend={data.net_obs.backend}
                        active={data.net_obs.active}
                        mode={data.net_obs.mode}
                        connectHits={data.net_obs.connect_hits}
                    />
                )}
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
                                        {ch.kernel_status && ch.kernel_status !== 'idle' && (
                                            <KernelPathChip
                                                status={ch.kernel_status}
                                                hint={ch.kernel_hint}
                                                impact={ch.kernel_impact}
                                                rttMs={ch.kernel_rtt_ms}
                                                failRate={ch.kernel_fail_rate}
                                                retransRate={ch.kernel_retrans_rate ?? 0}
                                                rttSource={ch.rtt_source}
                                                rttSamples={ch.rtt_samples}
                                            />
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

// ─── NetObs status badge ───────────────────────────────────────────
// Renders a prominent pill on the health board title row showing the
// active network-observation backend (eBPF / Go / none) with live
// connect-hit counter when eBPF is running.
function NetObsBadge({
    backend,
    active,
    mode,
    connectHits,
}: {
    backend: string;
    active: boolean;
    mode: string;
    connectHits?: number;
}) {
    const isEBPF = backend === 'ebpf' && active;
    const isGo = backend === 'go' && active;

    if (isEBPF) {
        return (
            <span
                className={cn(
                    'inline-flex items-center gap-1.5 text-xs font-semibold',
                    'px-2.5 py-1 rounded-lg border',
                    'bg-violet-500/15 text-violet-600 dark:text-violet-400',
                    'border-violet-500/30',
                    'shadow-sm shadow-violet-500/10',
                )}
                title={`eBPF kernel probe active · mode=${mode} · connect_hits=${connectHits ?? 0}`}
            >
                <Zap className="w-3.5 h-3.5" />
                <span>eBPF</span>
                <span className="inline-flex items-center gap-0.5 text-[10px] font-normal opacity-80">
                    <Network className="w-3 h-3" />
                    {typeof connectHits === 'number'
                        ? connectHits.toLocaleString()
                        : '0'}
                </span>
            </span>
        );
    }

    if (isGo) {
        return (
            <span
                className={cn(
                    'inline-flex items-center gap-1.5 text-xs font-medium',
                    'px-2.5 py-1 rounded-lg border',
                    'bg-muted text-muted-foreground border-border',
                )}
                title={`Go observer · mode=${mode}`}
            >
                <Cpu className="w-3.5 h-3.5" />
                <span>Go</span>
            </span>
        );
    }

    // offline / none
    return (
        <span
            className={cn(
                'inline-flex items-center gap-1.5 text-xs',
                'px-2.5 py-1 rounded-lg border',
                'bg-muted/50 text-muted-foreground/60 border-border/50',
            )}
            title="Network observer offline"
        >
            <Network className="w-3.5 h-3.5" />
            <span>NetObs off</span>
        </span>
    );
}

// ─── KernelPathChip ────────────────────────────────────────────────
// 专业级状态芯片：人话结论 + 悬停展开裸指标 + 选路影响
// 状态 = good(green) / slow(amber) / poor(red) / idle(无数据)
// 参考 GitHub/Grafana 面板设计：状态点 + 一句话 + 次级说明
function KernelPathChip({
    status,
    hint,
    impact,
    rttMs,
    failRate,
    retransRate,
    rttSource,
    rttSamples,
}: {
    status: string;
    hint: string;
    impact: string;
    rttMs: number;
    failRate: number;
    retransRate: number;
    rttSource?: string;
    rttSamples?: number;
}) {
    const config: Record<string, { color: string; bg: string; label: string }> = {
        good: { color: 'text-emerald-600 dark:text-emerald-400', bg: 'bg-emerald-500/10', label: '畅通' },
        slow: { color: 'text-amber-600 dark:text-amber-400', bg: 'bg-amber-500/10', label: '偏慢' },
        poor: { color: 'text-red-600 dark:text-red-400', bg: 'bg-red-500/10', label: '拥堵' },
        idle: { color: 'text-muted-foreground', bg: 'bg-muted/40', label: '待观测' },
    };
    const c = config[status] ?? config.idle;

    // 来源徽标
    const sourceLabel: Record<string, string> = {
        kernel: 'k',
        l7: 'L7',
        probe: 'P',
    };
    const sourceBadge = rttSource && rttSource !== 'none' ? sourceLabel[rttSource] || rttSource : null;

    // 来源中英文名
    const sourceName: Record<string, string> = {
        kernel: '内核 BPF',
        l7: '真实请求 EWMA',
        probe: '健康探测',
    };
    const sourceTitle = rttSource && rttSource !== 'none'
        ? `来源: ${sourceName[rttSource] || rttSource}${rttSamples ? ` (${rttSamples} 样本)` : ''}`
        : '';

    // 裸指标：有数据就显示。RTT=0 表示测不到（非阻塞 connect），显示 n/a。
    // fail = 建连失败（syscall + 异步握手）；rtx = 真·TCP 重传
    const rttLabel = rttMs > 0 ? `kRTT ${rttMs.toFixed(0)}ms` : 'kRTT n/a';
    const metricStr =
        status !== 'idle'
            ? `${rttLabel} · 建连失败 ${(failRate * 100).toFixed(1)}% · 重传 ${(retransRate * 100).toFixed(1)}%`
            : '';

    return (
        <span
            className={cn(
                'inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium',
                'border border-transparent hover:border-current/20 cursor-default',
                c.bg, c.color,
            )}
            title={`${hint}\\n${metricStr ? metricStr + ' · ' : ''}${impact}\\n${sourceTitle ? sourceTitle : ''}`}
        >
            {/* 状态指示点 */}
            <span className={cn('w-1.5 h-1.5 rounded-full', status === 'good' ? 'bg-emerald-500' : status === 'slow' ? 'bg-amber-500' : 'bg-red-500')} />
            <span>{c.label}</span>
            {/* 来源徽标 */}
            {sourceBadge && (
                <span className="text-[10px] font-normal opacity-50 ml-0.5" title={sourceTitle}>
                    [{sourceBadge}]
                </span>
            )}
            {/* 裸指标次级显示 */}
            {metricStr && (
                <span className="text-[10px] font-normal opacity-60 tabular-nums">
                    {metricStr}
                </span>
            )}
        </span>
    );
}
