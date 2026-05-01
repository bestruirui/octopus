'use client';

import { motion } from 'motion/react';
import { Activity, ArrowRight, Coins, DollarSign, RadioTower, ShieldCheck } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useStatsChannel, useStatsToday } from '@/api/endpoints/stats';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { EASING } from '@/lib/animations/fluid-transitions';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';

export function HomeHero() {
    const t = useTranslations('home.hero');
    const { data: statsToday } = useStatsToday();
    const { data: channelData } = useStatsChannel();

    const requestCount = (statsToday?.request_success ?? 0) + (statsToday?.request_failed ?? 0);
    const successCount = statsToday?.request_success ?? 0;
    const totalCost = (statsToday?.input_cost ?? 0) + (statsToday?.output_cost ?? 0);
    const totalWaitTime = statsToday?.wait_time ?? 0;
    const successRate = requestCount > 0 ? (successCount / requestCount) * 100 : 0;
    const enabledChannels = channelData?.filter((channel) => channel.enabled).length ?? 0;
    const avgWait = requestCount > 0 ? totalWaitTime / requestCount : 0;

    const signals = [
        {
            key: 'requests',
            label: t('signals.requests'),
            value: formatCount(requestCount).formatted.value,
            unit: formatCount(requestCount).formatted.unit,
            icon: Activity,
            accent: 'bg-emerald-500/12 text-emerald-700',
        },
        {
            key: 'successRate',
            label: t('signals.successRate'),
            value: successRate.toFixed(2),
            unit: '%',
            icon: ShieldCheck,
            accent: 'bg-primary/12 text-primary',
        },
        {
            key: 'cost',
            label: t('signals.cost'),
            value: formatMoney(totalCost).formatted.value,
            unit: formatMoney(totalCost).formatted.unit,
            icon: DollarSign,
            accent: 'bg-amber-500/12 text-amber-700',
        },
        {
            key: 'channels',
            label: t('signals.channels'),
            value: formatCount(enabledChannels).formatted.value,
            unit: formatCount(enabledChannels).formatted.unit,
            icon: RadioTower,
            accent: 'bg-sky-500/12 text-sky-700',
        },
    ];

    const metrics = [
        {
            key: 'avgWait',
            label: t('metrics.avgWait'),
            value: formatTime(avgWait).formatted.value,
            unit: formatTime(avgWait).formatted.unit,
        },
        {
            key: 'successful',
            label: t('metrics.successful'),
            value: formatCount(successCount).formatted.value,
            unit: formatCount(successCount).formatted.unit,
        },
        {
            key: 'tokens',
            label: t('metrics.tokens'),
            value: formatCount((statsToday?.input_token ?? 0) + (statsToday?.output_token ?? 0)).formatted.value,
            unit: formatCount((statsToday?.input_token ?? 0) + (statsToday?.output_token ?? 0)).formatted.unit,
        },
        {
            key: 'costPerReq',
            label: t('metrics.costPerRequest'),
            value: formatMoney(requestCount > 0 ? totalCost / requestCount : 0).formatted.value,
            unit: formatMoney(requestCount > 0 ? totalCost / requestCount : 0).formatted.unit,
        },
    ];

    return (
        <motion.section
            className="relative overflow-hidden rounded-[2rem] border border-card-border bg-[radial-gradient(circle_at_top_left,rgba(101,163,13,0.18),transparent_30%),radial-gradient(circle_at_80%_20%,rgba(59,130,246,0.10),transparent_28%),linear-gradient(180deg,rgba(255,255,255,0.96),rgba(255,255,255,0.88))] p-5 text-card-foreground custom-shadow md:p-6"
            initial={{ opacity: 0, y: 24, filter: 'blur(8px)' }}
            animate={{ opacity: 1, y: 0, filter: 'blur(0px)' }}
            transition={{ duration: 0.55, ease: EASING.easeOutExpo }}
        >
            <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(135deg,rgba(255,255,255,0.38),transparent_35%,rgba(255,255,255,0.18))]" />

            <div className="relative grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_minmax(0,0.85fr)]">
                <div className="space-y-5">
                    <div className="inline-flex items-center gap-2 rounded-full border border-primary/15 bg-white/70 px-3 py-1 text-xs font-medium text-primary shadow-sm backdrop-blur">
                        <Coins className="h-3.5 w-3.5" />
                        <span>{t('eyebrow')}</span>
                    </div>

                    <div className="space-y-3">
                        <div className="flex items-center gap-3">
                            <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-primary/12 text-primary shadow-sm">
                                <Activity className="h-6 w-6" />
                            </div>
                            <div className="space-y-1">
                                <h1 className="text-3xl font-semibold tracking-tight md:text-4xl">{t('title')}</h1>
                                {t('subtitle') ? (
                                    <p className="text-sm leading-6 text-muted-foreground md:text-base">{t('subtitle')}</p>
                                ) : null}
                            </div>
                        </div>

                        {t('description') ? (
                            <p className="max-w-2xl text-sm leading-7 text-muted-foreground md:text-[15px]">
                                {t('description')}
                            </p>
                        ) : null}
                    </div>

                    <div className="grid gap-3 sm:grid-cols-2">
                        {metrics.map((metric) => (
                            <div key={metric.key} className="rounded-2xl border border-white/70 bg-white/72 px-4 py-3 shadow-sm backdrop-blur">
                                <div className="text-xs text-muted-foreground">{metric.label}</div>
                                <div className="mt-1 flex items-baseline gap-1">
                                    <span className="text-2xl font-semibold">
                                        <AnimatedNumber value={metric.value} />
                                    </span>
                                    {metric.unit ? <span className="text-sm text-muted-foreground">{metric.unit}</span> : null}
                                </div>
                            </div>
                        ))}
                    </div>
                </div>

                <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-2">
                    {signals.map((signal) => (
                        <article key={signal.key} className="rounded-[1.75rem] border border-white/70 bg-white/78 p-4 shadow-sm backdrop-blur">
                            <div className="flex items-start justify-between gap-3">
                                <div className={`flex h-11 w-11 items-center justify-center rounded-2xl ${signal.accent}`}>
                                    <signal.icon className="h-5 w-5" />
                                </div>
                                <ArrowRight className="mt-1 h-4 w-4 text-muted-foreground/50" />
                            </div>
                            <div className="mt-6 text-xs text-muted-foreground">{signal.label}</div>
                            <div className="mt-2 flex items-baseline gap-1">
                                <span className="text-3xl font-semibold tracking-tight">
                                    <AnimatedNumber value={signal.value} />
                                </span>
                                {signal.unit ? <span className="text-sm text-muted-foreground">{signal.unit}</span> : null}
                            </div>
                        </article>
                    ))}
                </div>
            </div>
        </motion.section>
    );
}
