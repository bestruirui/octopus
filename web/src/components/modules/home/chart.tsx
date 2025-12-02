'use client';

import { useStatsDaily, useStatsHourly } from '@/api/endpoints/stats';
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart';
import { useMemo, useState } from 'react';
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from 'recharts';
import { useTranslations } from 'next-intl';
import { formatCount, formatMoney } from '@/lib/utils';
import dayjs from 'dayjs';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
const PERIODS = ['1', '7', '30'] as const;

export function StatsChart() {
    const { data: statsDaily } = useStatsDaily();
    const { data: statsHourly } = useStatsHourly();
    const [period, setPeriod] = useState<typeof PERIODS[number]>('1');
    const t = useTranslations('home.chart');

    const chartData = useMemo(() => {
        if (period === '1') {
            if (!statsHourly) return [];
            return statsHourly.map((stat) => {
                return {
                    date: `${stat.hour}:00`,
                    total_cost: stat.total_cost.raw,
                };
            });
        } else {
            if (!statsDaily) return [];
            const days = parseInt(period);
            return statsDaily.slice(-days).map((stat) => {
                return {
                    date: dayjs(stat.date).format('MM/DD'),
                    total_cost: stat.total_cost.raw,
                };
            });
        }
    }, [statsDaily, statsHourly, period]);

    const totals = useMemo(() => {
        if (period === '1') {
            if (!statsHourly) return { requests: 0, cost: 0 };
            return {
                requests: statsHourly.reduce((acc, stat) => acc + stat.request_count.raw, 0),
                cost: statsHourly.reduce((acc, stat) => acc + stat.total_cost.raw, 0),
            };
        } else {
            if (!statsDaily) return { requests: 0, cost: 0 };
            const days = parseInt(period);
            const recentStats = statsDaily.slice(-days);
            return {
                requests: recentStats.reduce((acc, stat) => acc + stat.request_success.raw + stat.request_failed.raw, 0),
                cost: recentStats.reduce((acc, stat) => acc + stat.total_cost.raw, 0),
            };
        }
    }, [statsDaily, statsHourly, period]);

    const chartConfig = {
        total_cost: { label: t('totalCost') },
    };

    const getPeriodLabel = (p: typeof period) => {
        const labels = {
            '1': t('period.today'),
            '7': t('period.last7Days'),
            '30': t('period.last30Days'),
        };
        return labels[p];
    };

    const handlePeriodClick = () => {
        const currentIndex = PERIODS.indexOf(period);
        const nextIndex = (currentIndex + 1) % PERIODS.length;
        setPeriod(PERIODS[nextIndex]);
    };

    return (
        <div className="rounded-3xl bg-card border-card-border border pt-2 pb-0 text-card-foreground custom-shadow">
            <div className="flex justify-between items-start px-4 pb-2">
                <div className="flex gap-2 text-sm">
                    <div>
                        <div className="text-xs text-muted-foreground">{t('totalRequests')}</div>
                        <div className="text-xl font-semibold">
                            <AnimatedNumber value={formatCount(totals.requests).formatted.value} />
                            <span className="ml-0.5 text-sm text-muted-foreground">{formatCount(totals.requests).formatted.unit}</span>
                        </div>
                    </div>
                    <div className="w-px bg-border self-stretch"></div>
                    <div>
                        <div className="text-xs text-muted-foreground">{t('totalCost')}</div>
                        <div className="text-xl font-semibold">
                            <AnimatedNumber value={formatMoney(totals.cost).formatted.value} />
                            <span className="ml-0.5 text-sm text-muted-foreground">{formatMoney(totals.cost).formatted.unit}</span>
                        </div>
                    </div>
                </div>
                <div
                    className="flex gap-2 text-sm cursor-pointer hover:opacity-80 transition-opacity"
                    onClick={handlePeriodClick}
                >
                    <div>
                        <div className="text-xs text-muted-foreground">{t('timePeriod')}</div>
                        <div className="text-xl font-semibold">{getPeriodLabel(period)}</div>
                    </div>
                </div>
            </div>
            <ChartContainer config={chartConfig} className="h-40 w-full" >
                <AreaChart accessibilityLayer data={chartData}>
                    <defs>
                        <linearGradient id="fillTotalCost" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-1)" stopOpacity={1.0} />
                            <stop offset="95%" stopColor="var(--chart-1)" stopOpacity={0.1} />
                        </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} />
                    <XAxis dataKey="date" tickLine={false} axisLine={false} />
                    <YAxis
                        tickLine={false}
                        axisLine={false}
                        tickFormatter={(value) => {
                            const formatted = formatMoney(value);
                            return `${formatted.formatted.value}${formatted.formatted.unit}`;
                        }}
                    />
                    <ChartTooltip cursor={false} content={<ChartTooltipContent indicator="line" />} />
                    <Area type="monotone" dataKey="total_cost" stroke="var(--chart-1)" fill="url(#fillTotalCost)" />
                </AreaChart>
            </ChartContainer>
        </div>
    );
}
