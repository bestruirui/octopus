'use client';

import { motion } from 'motion/react';
import {
    Activity,
    MessageSquare,
    Clock,
    ArrowDownToLine,
    ChartColumnBig,
    Bot,
    ArrowUpFromLine,
    Rewind,
    DollarSign,
    FastForward
} from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useMemo } from 'react';
import { useStatsTotal, useStatsDaily } from '@/api/endpoints/stats';
import type { StatsTotalFormatted } from '@/api/endpoints/stats';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { EASING } from '@/lib/animations/fluid-transitions';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';
import { useHomeViewStore } from '@/components/modules/home/store';


export function Total() {
    const { data: statsTotalFormatted } = useStatsTotal();
    const { data: statsDaily } = useStatsDaily();
    const dateRange = useHomeViewStore((state) => state.dateRange);
    const t = useTranslations('home.total');

    const displayData: StatsTotalFormatted | undefined = useMemo(() => {
        if (!dateRange.from) return statsTotalFormatted;
        if (!statsDaily) return statsTotalFormatted;

        const filtered = dateRange.to
            ? statsDaily.filter((s) => s.date >= dateRange.from! && s.date <= dateRange.to!)
            : statsDaily.filter((s) => s.date === dateRange.from!);

        if (filtered.length === 0) return statsTotalFormatted;

        const raw = {
            request_success: filtered.reduce((a, s) => a + s.request_success.raw, 0),
            request_failed: filtered.reduce((a, s) => a + s.request_failed.raw, 0),
            input_token: filtered.reduce((a, s) => a + s.input_token.raw, 0),
            output_token: filtered.reduce((a, s) => a + s.output_token.raw, 0),
            input_cost: filtered.reduce((a, s) => a + s.input_cost.raw, 0),
            output_cost: filtered.reduce((a, s) => a + s.output_cost.raw, 0),
            wait_time: filtered.reduce((a, s) => a + s.wait_time.raw, 0),
        };

        return {
            request_count: formatCount(raw.request_success + raw.request_failed),
            request_success: formatCount(raw.request_success),
            request_failed: formatCount(raw.request_failed),
            total_token: formatCount(raw.input_token + raw.output_token),
            input_token: formatCount(raw.input_token),
            output_token: formatCount(raw.output_token),
            total_cost: formatMoney(raw.input_cost + raw.output_cost),
            input_cost: formatMoney(raw.input_cost),
            output_cost: formatMoney(raw.output_cost),
            wait_time: formatTime(raw.wait_time),
        };
    }, [dateRange, statsDaily, statsTotalFormatted]);

    const cards = [
        {
            title: t('requestStats'),
            headerIcon: Activity,
            items: [
                {
                    label: t('requestCount'),
                    value: displayData?.request_count.formatted.value,
                    icon: MessageSquare,
                    color: 'text-primary',
                    bgColor: 'bg-primary/10',
                    unit: displayData?.request_count.formatted.unit
                },
                {
                    label: t('timeConsumed'),
                    value: displayData?.wait_time.formatted.value,
                    icon: Clock,
                    color: 'text-primary',
                    bgColor: 'bg-accent/10',
                    unit: displayData?.wait_time.formatted.unit
                }
            ]
        },
        {
            title: t('totalStats'),
            headerIcon: ChartColumnBig,
            items: [
                {
                    label: t('totalToken'),
                    value: displayData?.total_token.formatted.value,
                    icon: Bot,
                    color: 'text-primary',
                    bgColor: 'bg-chart-1/10',
                    unit: displayData?.total_token.formatted.unit
                },
                {
                    label: t('totalCost'),
                    value: displayData?.total_cost.formatted.value,
                    icon: DollarSign,
                    color: 'text-primary',
                    bgColor: 'bg-chart-2/10',
                    unit: displayData?.total_cost.formatted.unit
                }
            ]
        },
        {
            title: t('inputStats'),
            headerIcon: ArrowDownToLine,
            items: [
                {
                    label: t('inputTokens'),
                    value: displayData?.input_token.formatted.value,
                    icon: Rewind,
                    color: 'text-primary',
                    bgColor: 'bg-chart-3/10',
                    unit: displayData?.input_token.formatted.unit
                },
                {
                    label: t('inputCost'),
                    value: displayData?.input_cost.formatted.value,
                    icon: DollarSign,
                    color: 'text-primary',
                    bgColor: 'bg-chart-3/10',
                    unit: displayData?.input_cost.formatted.unit
                }
            ]
        },
        {
            title: t('outputStats'),
            headerIcon: ArrowUpFromLine,
            items: [
                {
                    label: t('outputTokens'),
                    value: displayData?.output_token.formatted.value,
                    icon: FastForward,
                    color: 'text-primary',
                    bgColor: 'bg-chart-4/10',
                    unit: displayData?.output_token.formatted.unit
                },
                {
                    label: t('outputCost'),
                    value: displayData?.output_cost.formatted.value,
                    icon: DollarSign,
                    color: 'text-primary',
                    bgColor: 'bg-chart-4/10',
                    unit: displayData?.output_cost.formatted.unit
                }
            ]
        }
    ];

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {cards.map((card, index) => (
                <motion.section
                    key={index}
                    className="rounded-3xl bg-card border-card-border border p-5 text-card-foreground flex flex-row items-center gap-4"
                    initial={{ opacity: 0, y: 20, filter: 'blur(8px)' }}
                    animate={{ opacity: 1, y: 0, filter: 'blur(0px)' }}
                    transition={{
                        duration: 0.5,
                        ease: EASING.easeOutExpo,
                        delay: index * 0.08,
                    }}
                >
                    <div className="flex flex-col items-center justify-center gap-3 border-r border-border/50 pr-4 py-1 self-stretch">
                        <card.headerIcon className="w-4 h-4" />
                        <h3 className="font-medium text-sm [writing-mode:vertical-lr]">{card.title}</h3>
                    </div>

                    <div className="flex flex-col gap-4 flex-1 min-w-0">
                        {card.items.map((item, idx) => (
                            <div key={idx} className="flex items-center gap-3">
                                <div className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${item.bgColor} ${item.color}`}>
                                    <item.icon className="w-5 h-5" />
                                </div>
                                <div className="flex flex-col min-w-0">
                                    <span className="text-xs text-muted-foreground">{item.label}</span>
                                    <div className="flex items-baseline gap-1">
                                        <span className="text-xl">
                                            <AnimatedNumber value={item.value} />
                                        </span>
                                        {item.unit && (
                                            <span className="text-sm text-muted-foreground">{item.unit}</span>
                                        )}
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                </motion.section>
            ))}
        </div>
    );
}
