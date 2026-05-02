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
import { BarChart3, CalendarClock } from 'lucide-react';

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
        if (type === 'cost') return 'total_cost';
        if (type === 'count') return 'request_count';
        if (type === 'success-rate') return 'success_rate';
        return 'total_token';
    };

    const chartData = useMemo(() => {
        const dataKey = getChartDataKey(chartMetricType);
        if (period === '1') {
            if (!statsHourly) return [];
            return statsHourly.map((stat) => ({
                date: `${stat.hour}:00`,
                [dataKey]: chartMetricType === 'cost'
                    ? stat.total_cost.raw
                    : chartMetricType === 'success-rate'
                        ? ((stat.request_success.raw + stat.request_failed.raw) > 0
                            ? (stat.request_success.raw / (stat.request_success.raw + stat.request_failed.raw)) * 100
                            : 0)
                    : chartMetricType === 'count'
                        ? stat.request_count.raw
                        : (stat.input_token.raw + stat.output_token.raw),
            }));
        } else {
            const days = Number(period);
            return sortedDaily.slice(-days).map((stat) => ({
                date: /^\d{8}$/.test(stat.date)
                    ? `${stat.date.slice(4, 6)}/${stat.date.slice(6, 8)}`
                    : dayjs(stat.date).format('MM/DD'),
                [dataKey]: chartMetricType === 'cost'
                    ? stat.total_cost.raw
                    : chartMetricType === 'success-rate'
                        ? ((stat.request_success.raw + stat.request_failed.raw) > 0
                            ? (stat.request_success.raw / (stat.request_success.raw + stat.request_failed.raw)) * 100
                            : 0)
                    : chartMetricType === 'count'
                        ? (stat.request_success.raw + stat.request_failed.raw)
                        : (stat.input_token.raw + stat.output_token.raw),
            }));
        }
    }, [sortedDaily, statsHourly, period, chartMetricType]);

    const totals = useMemo(() => {
        if (period === '1') {
            if (!statsHourly) return { requests: 0, cost: 0, tokens: 0, successRate: 0 };
            const requests = statsHourly.reduce((acc, stat) => acc + stat.request_count.raw, 0);
            const cost = statsHourly.reduce((acc, stat) => acc + stat.total_cost.raw, 0);
            const tokens = statsHourly.reduce((acc, stat) => acc + stat.input_token.raw + stat.output_token.raw, 0);
            const success = statsHourly.reduce((acc, stat) => acc + stat.request_success.raw, 0);
            const failed = statsHourly.reduce((acc, stat) => acc + stat.request_failed.raw, 0);
            return {
                requests,
                cost,
                tokens,
                successRate: success+failed > 0 ? (success / (success + failed)) * 100 : 0,
            };
        } else {
            const days = Number(period);
            const recentStats = sortedDaily.slice(-days);
            const requests = recentStats.reduce((acc, stat) => acc + stat.request_success.raw + stat.request_failed.raw, 0);
            const cost = recentStats.reduce((acc, stat) => acc + stat.total_cost.raw, 0);
            const tokens = recentStats.reduce((acc, stat) => acc + stat.input_token.raw + stat.output_token.raw, 0);
            const success = recentStats.reduce((acc, stat) => acc + stat.request_success.raw, 0);
            const failed = recentStats.reduce((acc, stat) => acc + stat.request_failed.raw, 0);
            return {
                requests,
                cost,
                tokens,
                successRate: success+failed > 0 ? (success / (success + failed)) * 100 : 0,
            };
        }
    }, [sortedDaily, statsHourly, period]);

    const chartConfig = useMemo(() => {
        const dataKey = getChartDataKey(chartMetricType);
        const labels = {
            'total_cost': t('totalCost'),
            'request_count': t('totalRequests'),
            'total_token': t('totalTokens'),
            'success_rate': t('successRate'),
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
        if (type === 'success-rate') return 'var(--chart-4)';
        return 'var(--chart-3)';
    };

    const getChartFill = (type: ChartMetricType) => {
        if (type === 'cost') return 'url(#fillMetric1)';
        if (type === 'count') return 'url(#fillMetric2)';
        if (type === 'success-rate') return 'url(#fillMetric4)';
        return 'url(#fillMetric3)';
    };

    const summaryMetrics = [
        {
            key: 'requests',
            label: t('totalRequests'),
            value: formatCount(totals.requests).formatted.value,
            unit: formatCount(totals.requests).formatted.unit,
        },
        {
            key: 'cost',
            label: t('totalCost'),
            value: formatMoney(totals.cost).formatted.value,
            unit: formatMoney(totals.cost).formatted.unit,
        },
        {
            key: 'tokens',
            label: t('totalTokens'),
            value: formatCount(totals.tokens).formatted.value,
            unit: formatCount(totals.tokens).formatted.unit,
        },
        {
            key: 'successRate',
            label: t('successRate'),
            value: totals.successRate.toFixed(2),
            unit: '%',
        },
    ];

    return (
        <div className="waterhouse-island relative overflow-hidden rounded-[2.15rem] border-border/35 bg-card/58 pt-5 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)]">
            <div className="pointer-events-none absolute -right-16 top-8 h-48 w-48 rounded-full bg-primary/8 blur-3xl" />
            <div className="pointer-events-none absolute inset-x-8 top-0 h-px bg-gradient-to-r from-transparent via-primary/28 to-transparent" />
            <div className="relative space-y-4 px-4 pb-3 md:px-5">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                    <div className="space-y-2">
                        <div className="inline-flex items-center gap-2 rounded-full border border-primary/12 bg-background/44 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.2em] text-primary shadow-waterhouse-soft backdrop-blur-md">
                            <BarChart3 className="h-3.5 w-3.5" />
                            <span>{t('title')}</span>
                        </div>
                        <button
                            type="button"
                            className="waterhouse-pod inline-flex items-center gap-2 rounded-[1.2rem] border-border/30 bg-background/38 px-3 py-2 text-left text-sm shadow-waterhouse-soft backdrop-blur-md transition-[transform,border-color,box-shadow] duration-300 hover:-translate-y-0.5 hover:border-primary/20"
                            onClick={handlePeriodClick}
                        >
                            <CalendarClock className="h-4 w-4 text-primary/70" />
                            <span className="text-xs text-muted-foreground">{t('timePeriod')}</span>
                            <span className="font-semibold">{getPeriodLabel(period)}</span>
                        </button>
                    </div>
                    <Tabs value={chartMetricType} onValueChange={(value) => setChartMetricType(value as ChartMetricType)}>
                        <TabsList className="waterhouse-pod flex w-full flex-wrap rounded-[1.4rem] border-border/30 bg-background/38 p-1 shadow-waterhouse-soft backdrop-blur-md sm:w-max">
                            <TabsTrigger value="cost">{t('metricType.cost')}</TabsTrigger>
                            <TabsTrigger value="count">{t('metricType.count')}</TabsTrigger>
                            <TabsTrigger value="tokens">{t('metricType.tokens')}</TabsTrigger>
                            <TabsTrigger value="success-rate">{t('metricType.successRate')}</TabsTrigger>
                        </TabsList>
                    </Tabs>
                </div>

                <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
                    {summaryMetrics.map((metric) => (
                        <div key={metric.key} className="waterhouse-pod overflow-hidden rounded-[1.35rem] border-border/30 bg-background/38 px-3.5 py-3 shadow-waterhouse-soft backdrop-blur-md">
                            <div className="mb-2 h-1 w-9 rounded-full bg-primary/18" />
                            <div className="text-xs text-muted-foreground">{metric.label}</div>
                            <div className="mt-1 flex items-baseline gap-1">
                                <span className="text-xl font-semibold tracking-tight">
                                    <AnimatedNumber value={metric.value} />
                                </span>
                                <span className="text-sm text-muted-foreground">{metric.unit}</span>
                            </div>
                        </div>
                    ))}
                </div>
            </div>
            <div className="relative mx-3 mb-3 overflow-hidden rounded-[1.7rem] border border-border/30 bg-background/30 pt-3 shadow-waterhouse-soft backdrop-blur-md">
                <div className="pointer-events-none absolute inset-x-6 top-0 h-px bg-gradient-to-r from-transparent via-primary/24 to-transparent" />
                <ChartContainer config={chartConfig} className="h-[20rem] w-full md:h-[24rem]">
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
                        <linearGradient id="fillMetric4" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-4)" stopOpacity={1.0} />
                            <stop offset="95%" stopColor="var(--chart-4)" stopOpacity={0.1} />
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
                            } else if (chartMetricType === 'success-rate') {
                                return `${value.toFixed(0)}%`;
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
        </div>
    );
}
