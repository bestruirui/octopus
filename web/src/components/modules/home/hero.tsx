'use client';

import { motion } from 'motion/react';
import { Activity, ArrowRight, DollarSign, Leaf, ShieldCheck, Waves } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useStatsToday } from '@/api/endpoints/stats';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { EASING } from '@/lib/animations/fluid-transitions';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';

export function HomeHero() {
    const t = useTranslations('home.hero');
    const { data: statsToday } = useStatsToday();

    const requestCount = (statsToday?.request_success ?? 0) + (statsToday?.request_failed ?? 0);
    const successCount = statsToday?.request_success ?? 0;
    const totalCost = (statsToday?.input_cost ?? 0) + (statsToday?.output_cost ?? 0);
    const totalWaitTime = statsToday?.wait_time ?? 0;
    const successRate = requestCount > 0 ? (successCount / requestCount) * 100 : 0;
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
            className="waterhouse-island relative overflow-hidden rounded-[2.4rem] border-border/35 bg-background/62 p-5 text-card-foreground shadow-none backdrop-blur-[var(--waterhouse-shell-blur)] md:p-6 xl:p-7"
            initial={{ opacity: 0, y: 24, filter: 'blur(8px)' }}
            animate={{ opacity: 1, y: 0, filter: 'blur(0px)' }}
            transition={{ duration: 0.55, ease: EASING.easeOutExpo }}
        >
            <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_18%_18%,color-mix(in_oklch,var(--waterhouse-highlight)_28%,transparent)_0%,transparent_34%),radial-gradient(circle_at_78%_18%,color-mix(in_oklch,var(--primary)_18%,transparent)_0%,transparent_28%),linear-gradient(135deg,color-mix(in_oklch,white_30%,transparent),transparent_38%,color-mix(in_oklch,var(--waterhouse-highlight)_8%,transparent))]" />
            <div className="pointer-events-none absolute left-[7%] top-[42%] h-36 w-[86%] rounded-[999px] border border-primary/10 bg-primary/6 blur-2xl" />
            <div className="pointer-events-none absolute right-10 top-8 hidden h-28 w-28 rounded-full border border-primary/10 bg-background/20 shadow-[inset_0_0_45px_color-mix(in_oklch,var(--primary)_12%,transparent)] md:block" />

            <div className="relative grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(20rem,0.88fr)]">
                <div className="space-y-5">
                    <div className="space-y-3">
                        <div className="flex items-center gap-4">
                            <div className="waterhouse-pod grid h-16 w-16 shrink-0 place-items-center overflow-hidden rounded-[1.55rem] border-border/35 bg-background/52 text-primary shadow-none">
                                <Waves className="h-7 w-7" />
                            </div>
                            <div className="space-y-1">
                                <h1 className="text-3xl font-semibold tracking-[-0.04em] md:text-5xl">{t('title')}</h1>
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
                            <div key={metric.key} className="waterhouse-pod group overflow-hidden rounded-[1.55rem] border-border/30 bg-background/42 px-4 py-3 shadow-none backdrop-blur-md transition-transform duration-300 hover:-translate-y-0.5">
                                <div className="mb-2 h-1 w-10 rounded-full bg-primary/20 transition-all duration-300 group-hover:w-14 group-hover:bg-primary/35" />
                                <div className="text-xs font-medium text-muted-foreground">{metric.label}</div>
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

                <div className="grid auto-rows-fr gap-3 sm:grid-cols-2 xl:grid-cols-2">
                    {signals.map((signal, index) => (
                        <article
                            key={signal.key}
                            className={`waterhouse-pod group overflow-hidden rounded-[1.85rem] border-border/35 bg-background/50 p-4 shadow-none backdrop-blur-md transition-[transform,border-color] duration-500 hover:-translate-y-1 hover:border-primary/24 ${index === 0 ? 'sm:col-span-2 xl:min-h-[11rem]' : ''}`}
                        >
                            <div className="pointer-events-none absolute -right-8 -top-8 h-24 w-24 rounded-full bg-primary/8 blur-2xl transition-opacity duration-500 group-hover:opacity-80" />
                            <div className="flex items-start justify-between gap-3">
                                <div className={`flex h-11 w-11 items-center justify-center rounded-2xl shadow-sm ${signal.accent}`}>
                                    <signal.icon className="h-5 w-5" />
                                </div>
                                <div className="flex items-center gap-2 text-muted-foreground/50">
                                    {index === 0 ? <Leaf className="h-4 w-4 text-primary/60" /> : null}
                                    <ArrowRight className="h-4 w-4 transition-transform duration-300 group-hover:translate-x-0.5" />
                                </div>
                            </div>
                            <div className="mt-5 text-xs text-muted-foreground">{signal.label}</div>
                            <div className="mt-2 flex items-baseline gap-1">
                                <span className={`${index === 0 ? 'text-4xl md:text-5xl' : 'text-3xl'} font-semibold tracking-tight`}>
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
