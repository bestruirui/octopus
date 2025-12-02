'use client';

import { motion } from 'framer-motion';
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
import { useStatsTotal } from '@/api/endpoints/stats';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { GRID_CARD_VARIANTS } from '@/lib/animations/fluid-transitions';


export function Total() {
    const { data: statsTotalFormatted } = useStatsTotal();
    const t = useTranslations('home.total');

    const cards = [
        {
            title: t('requestStats'),
            headerIcon: Activity,
            items: [
                {
                    label: t('requestCount'),
                    value: statsTotalFormatted?.request_count.formatted.value,
                    icon: MessageSquare,
                    color: 'text-primary',
                    bgColor: 'bg-primary/10',
                    unit: statsTotalFormatted?.request_count.formatted.unit
                },
                {
                    label: t('timeConsumed'),
                    value: statsTotalFormatted?.wait_time.formatted.value,
                    icon: Clock,
                    color: 'text-card-foreground',
                    bgColor: 'bg-accent/10',
                    unit: statsTotalFormatted?.wait_time.formatted.unit
                }
            ]
        },
        {
            title: t('totalStats'),
            headerIcon: ChartColumnBig,
            items: [
                {
                    label: t('totalToken'),
                    value: statsTotalFormatted?.total_token.formatted.value,
                    icon: Bot,
                    color: 'text-primary',
                    bgColor: 'bg-chart-1/10',
                    unit: statsTotalFormatted?.total_token.formatted.unit
                },
                {
                    label: t('totalCost'),
                    value: statsTotalFormatted?.total_cost.formatted.value,
                    icon: DollarSign,
                    color: 'text-card-foreground',
                    bgColor: 'bg-chart-2/10',
                    unit: statsTotalFormatted?.total_cost.formatted.unit
                }
            ]
        },
        {
            title: t('inputStats'),
            headerIcon: ArrowDownToLine,
            items: [
                {
                    label: t('inputTokens'),
                    value: statsTotalFormatted?.input_token.formatted.value,
                    icon: Rewind,
                    color: 'text-primary',
                    bgColor: 'bg-chart-3/10',
                    unit: statsTotalFormatted?.input_token.formatted.unit
                },
                {
                    label: t('inputCost'),
                    value: statsTotalFormatted?.input_cost.formatted.value,
                    icon: DollarSign,
                    color: 'text-card-foreground',
                    bgColor: 'bg-chart-3/10',
                    unit: statsTotalFormatted?.input_cost.formatted.unit
                }
            ]
        },
        {
            title: t('outputStats'),
            headerIcon: ArrowUpFromLine,
            items: [
                {
                    label: t('outputTokens'),
                    value: statsTotalFormatted?.output_token.formatted.value,
                    icon: FastForward,
                    color: 'text-primary',
                    bgColor: 'bg-chart-4/10',
                    unit: statsTotalFormatted?.output_token.formatted.unit
                },
                {
                    label: t('outputCost'),
                    value: statsTotalFormatted?.output_cost.formatted.value,
                    icon: DollarSign,
                    color: 'text-card-foreground',
                    bgColor: 'bg-chart-4/10',
                    unit: statsTotalFormatted?.output_cost.formatted.unit
                }
            ]
        }
    ];

    return (
        <motion.div
            className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4"
            variants={GRID_CARD_VARIANTS.container}
            initial="initial"
            animate="animate"
        >
            {cards.map((card, index) => (
                <motion.section
                    key={index}
                    className="rounded-3xl bg-card border-card-border border p-5 text-card-foreground custom-shadow flex flex-row items-center gap-4"
                    variants={GRID_CARD_VARIANTS.item}
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
                                        {item.unit && item.value && (
                                            <span className="text-sm text-muted-foreground">{item.unit}</span>
                                        )}
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                </motion.section>
            ))}
        </motion.div>
    );
}
