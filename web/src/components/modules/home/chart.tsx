'use client';

import { useStatsDaily, useStatsHourly } from '@/api/endpoints/stats';
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart';
import { useMemo } from 'react';
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from 'recharts';
import { useTranslations } from 'next-intl';
import { formatCount, formatMoney } from '@/lib/utils';
import dayjs from 'dayjs';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { Tabs, TabsList, TabsTrigger } from '@/components/animate-ui/components/animate/tabs';
import { useHomeViewStore, type ChartMetricType, type ChartPeriod } from '@/components/modules/home/store';

export function StatsChart() {
    const PERIODS: readonly ChartPeriod[] = ['1', '7', '30'];
    const { data: statsDaily } = useStatsDaily();
    const { data: statsHourly } = useStatsHourly();
    const t = useTranslations('home.chart');

    const chartMetricType = useHomeViewStore((state) => state.chartMetricType);
    const setChartMetricType = useHomeViewStore((state) => state.setChartMetricType);
    const period = useHomeViewStore((state) => state.chartPeriod);
    const setChartPeriod = useHomeViewStore((state) => state.setChartPeriod);

    const sortedDaily = useMemo(() => {
        if (!statsDaily) return [];
        return [...statsDaily].sort((a, b) => a.date.localeCompare(b.date));
    }, [statsDaily]);

    const getChartDataKey = (type: ChartMetricType) => {
        return type === 'cost' ? 'total_cost' : type === 'count' ? 'request_count' : 'total_token';
    };

    const chartData = useMemo(() => {
        const dataKey = getChartDataKey(chartMetricType);
        if (period === '1') {
            if (!statsHourly) return [];
            return statsHourly.map((stat) => ({
                date: `${stat.hour}:00`,
                [dataKey]: chartMetricType === 'cost'
                    ? stat.total_cost.raw
                    : chartMetricType === 'count'
                        ? stat.request_count.raw
                        : (stat.input_token.raw + stat.output_token.raw),
            }));
        } else {
            const days = Number(period);
            return sortedDaily.slice(-days).map((stat) => ({
                date: dayjs(stat.date).format('MM/DD'),
                [dataKey]: chartMetricType === 'cost'
                    ? stat.total_cost.raw
                    : chartMetricType === 'count'
                        ? (stat.request_success.raw + stat.request_failed.raw)
                        : (stat.input_token.raw + stat.output_token.raw),
            }));
        }
    }, [sortedDaily, statsHourly, period, chartMetricType]);

    const totals = useMemo(() => {
        if (period === '1') {
            if (!statsHourly) return { requests: 0, cost: 0, tokens: 0 };
            const requests = statsHourly.reduce((acc, stat) => acc + stat.request_count.raw, 0);
            const cost = statsHourly.reduce((acc, stat) => acc + stat.total_cost.raw, 0);
            const tokens = statsHourly.reduce((acc, stat) => acc + stat.input_token.raw + stat.output_token.raw, 0);
            return {
                requests,
                cost,
                tokens,
            };
        } else {
            const days = Number(period);
            const recentStats = sortedDaily.slice(-days);
            const requests = recentStats.reduce((acc, stat) => acc + stat.request_success.raw + stat.request_failed.raw, 0);
            const cost = recentStats.reduce((acc, stat) => acc + stat.total_cost.raw, 0);
            const tokens = recentStats.reduce((acc, stat) => acc + stat.input_token.raw + stat.output_token.raw, 0);
            return {
                requests,
                cost,
                tokens,
            };
        }
    }, [sortedDaily, statsHourly, period]);

    const chartConfig = useMemo(() => {
        const dataKey = getChartDataKey(chartMetricType);
        const labels = {
            'total_cost': t('totalCost'),
            'request_count': t('totalRequests'),
            'total_token': t('totalTokens'),
        };
        return {
            [dataKey]: { label: labels[dataKey] },
        };
    }, [chartMetricType, t]);

    const getPeriodLabel = (p: ChartPeriod) => {
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
        setChartPeriod(PERIODS[nextIndex]);
    };


    const getChartStroke = (type: ChartMetricType) => {
        if (type === 'cost') return 'var(--chart-1)';
        if (type === 'count') return 'var(--chart-2)';
        return 'var(--chart-3)';
    };

    const getChartFill = (type: ChartMetricType) => {
        if (type === 'cost') return 'url(#fillMetric1)';
        if (type === 'count') return 'url(#fillMetric2)';
        return 'url(#fillMetric3)';
    };

    return (
        <div className="rounded-3xl bg-card border-card-border border pt-4 pb-0 text-card-foreground custom-shadow">
            <div className="px-4 pb-2 space-y-2">
                <div className="flex justify-between items-center">
                    <h3 className="font-semibold text-base">{t('title')}</h3>
                    <Tabs value={chartMetricType} onValueChange={(value) => setChartMetricType(value as ChartMetricType)}>
                        <TabsList>
                            <TabsTrigger value="cost">{t('metricType.cost')}</TabsTrigger>
                            <TabsTrigger value="count">{t('metricType.count')}</TabsTrigger>
                            <TabsTrigger value="tokens">{t('metricType.tokens')}</TabsTrigger>
                        </TabsList>
                    </Tabs>
                </div>

                {/* 第二行：汇总统计 + 周期选择 */}
                <div className="flex justify-between items-start">
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
                        <div className="w-px bg-border self-stretch"></div>
                        <div>
                            <div className="text-xs text-muted-foreground">{t('totalTokens')}</div>
                            <div className="text-xl font-semibold">
                                <AnimatedNumber value={formatCount(totals.tokens).formatted.value} />
                                <span className="ml-0.5 text-sm text-muted-foreground">{formatCount(totals.tokens).formatted.unit}</span>
                            </div>
                        </div>
                    </div>
                    <div
                        className="flex gap-2 text-sm cursor-pointer hover:opacity-80 transition-opacity"
                        onClick={handlePeriodClick}
                    >
                        <div>
                            <div className="text-xs text-muted-foreground">{t('timePeriod')}</div>
                            <div className="text-base font-semibold">{getPeriodLabel(period)}</div>
                        </div>
                    </div>
                </div>
            </div>
            <ChartContainer config={chartConfig} className="h-40 w-full" >
                <AreaChart accessibilityLayer data={chartData}>
                    <defs>
                        <linearGradient id="fillMetric1" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-1)" stopOpacity={1.0} />
                            <stop offset="95%" stopColor="var(--chart-1)" stopOpacity={0.1} />
                        </linearGradient>
                        <linearGradient id="fillMetric2" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-2)" stopOpacity={1.0} />
                            <stop offset="95%" stopColor="var(--chart-2)" stopOpacity={0.1} />
                        </linearGradient>
                        <linearGradient id="fillMetric3" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-3)" stopOpacity={1.0} />
                            <stop offset="95%" stopColor="var(--chart-3)" stopOpacity={0.1} />
                        </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} />
                    <XAxis dataKey="date" tickLine={false} axisLine={false} />
                    <YAxis
                        tickLine={false}
                        axisLine={false}
                        tickFormatter={(value) => {
                            if (chartMetricType === 'cost') {
                                const formatted = formatMoney(value);
                                return `${formatted.formatted.value}${formatted.formatted.unit}`;
                            } else if (chartMetricType === 'count' || chartMetricType === 'tokens') {
                                const formatted = formatCount(value);
                                return `${formatted.formatted.value}${formatted.formatted.unit}`;
                            }
                            return value.toString();
                        }}
                    />
                    <ChartTooltip cursor={false} content={<ChartTooltipContent indicator="line" />} />
                    <Area
                        type="monotone"
                        dataKey={getChartDataKey(chartMetricType)}
                        stroke={getChartStroke(chartMetricType)}
                        fill={getChartFill(chartMetricType)}
                    />
                </AreaChart>
            </ChartContainer>
        </div>
    );
}
