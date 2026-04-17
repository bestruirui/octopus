'use client';

import { motion } from 'motion/react';
import {
    Activity,
    MessageSquare,
    Clock,
    CircleCheckBig,
    DollarSign,
    Network,
    Percent
} from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useStatsTotal } from '@/api/endpoints/stats';
import { useChannelList } from '@/api/endpoints/channel';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { EASING } from '@/lib/animations/fluid-transitions';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';


export function Total() {
    const { data: statsTotalFormatted } = useStatsTotal();
    const { data: channelData } = useChannelList();
    const t = useTranslations('home.total');

    const requestCount = statsTotalFormatted?.request_count.raw ?? 0;
    const successCount = statsTotalFormatted?.request_success.raw ?? 0;
    const totalCost = statsTotalFormatted?.total_cost.raw ?? 0;
    const totalWaitTime = statsTotalFormatted?.wait_time.raw ?? 0;

    const successRate = requestCount > 0 ? (successCount / requestCount) * 100 : 0;
    const avgWaitTime = requestCount > 0 ? totalWaitTime / requestCount : 0;
    const avgCostPerRequest = requestCount > 0 ? totalCost / requestCount : 0;
    const activeChannelCount = channelData?.filter((channel) => channel.formatted.request_count.raw > 0).length ?? 0;

    const cards = [
        {
            title: t('qualityStats'),
            headerIcon: Activity,
            items: [
                {
                    label: t('successRate'),
                    value: successRate.toFixed(2),
                    icon: CircleCheckBig,
                    color: 'text-primary',
                    bgColor: 'bg-primary/10',
                    unit: '%'
                },
                {
                    label: t('avgWaitTime'),
                    value: formatTime(avgWaitTime).formatted.value,
                    icon: Clock,
                    color: 'text-primary',
                    bgColor: 'bg-accent/10',
                    unit: formatTime(avgWaitTime).formatted.unit
                }
            ]
        },
        {
            title: t('efficiencyStats'),
            headerIcon: DollarSign,
            items: [
                {
                    label: t('avgCostPerRequest'),
                    value: formatMoney(avgCostPerRequest).formatted.value,
                    icon: Percent,
                    color: 'text-primary',
                    bgColor: 'bg-chart-1/10',
                    unit: formatMoney(avgCostPerRequest).formatted.unit
                },
                {
                    label: t('requestCount'),
                    value: statsTotalFormatted?.request_count.formatted.value,
                    icon: MessageSquare,
                    color: 'text-primary',
                    bgColor: 'bg-chart-2/10',
                    unit: statsTotalFormatted?.request_count.formatted.unit
                }
            ]
        },
        {
            title: t('supplyStats'),
            headerIcon: Network,
            items: [
                {
                    label: t('activeChannels'),
                    value: formatCount(activeChannelCount).formatted.value,
                    icon: Network,
                    color: 'text-primary',
                    bgColor: 'bg-chart-3/10',
                    unit: formatCount(activeChannelCount).formatted.unit
                },
                {
                    label: t('totalCost'),
                    value: statsTotalFormatted?.total_cost.formatted.value,
                    icon: DollarSign,
                    color: 'text-primary',
                    bgColor: 'bg-chart-3/10',
                    unit: statsTotalFormatted?.total_cost.formatted.unit
                }
            ]
        }
    ];

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
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
