'use client';

import { useStatsDaily } from '@/api/endpoints/stats';
import { ChartContainer, ChartTooltip, ChartLegend, ChartTooltipContent } from '@/components/ui/chart';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { useMemo, useState } from 'react';
import { Area, AreaChart, XAxis } from 'recharts';
import { useTranslations } from 'next-intl';

type Period = '7' | '30' | '60' | '180';

export function StatsChart() {
    const { data: statsDaily } = useStatsDaily();
    const [period, setPeriod] = useState<Period>('7');
    const [hiddenSeries, setHiddenSeries] = useState<Set<string>>(new Set());
    const t = useTranslations('chart');

    const chartData = useMemo(() => {
        if (!statsDaily) return [];
        const days = parseInt(period);
        return statsDaily.raw.slice(-days).map((stat) => {
            return {
                date: new Date(stat.date).toLocaleDateString('en-US', { month: 'numeric', day: 'numeric' }),
                requests: stat.request_count,
                input_tokens: stat.input_token,
                output_tokens: stat.output_token,
                input_cost: stat.input_cost,
                output_cost: stat.output_cost,
            };
        });
    }, [statsDaily, period]);

    const chartConfig = {
        requests: { label: t('requests') },
        input_tokens: { label: t('inputTokens') },
        output_tokens: { label: t('outputTokens') },
        input_cost: { label: t('inputCost') },
        output_cost: { label: t('outputCost') },
    };

    return (
        <div className="rounded-3xl bg-card border-card-border border pt-2 pb-0 text-card-foreground custom-shadow">
            <div className="flex justify-end gap-2 px-6">
                <Select value={period} onValueChange={(value) => setPeriod(value as Period)}>
                    <SelectTrigger size="sm">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                        <SelectItem value="7">{t('period.last7Days')}</SelectItem>
                        <SelectItem value="30">{t('period.last30Days')}</SelectItem>
                        <SelectItem value="60">{t('period.last60Days')}</SelectItem>
                        <SelectItem value="180">{t('period.last6Months')}</SelectItem>
                    </SelectContent>
                </Select>
            </div>
            <ChartContainer config={chartConfig} className="h-64 w-full" >
                <AreaChart accessibilityLayer data={chartData}>
                    <defs>

                        <linearGradient id="fillOutputTokens" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-1)" stopOpacity={1.0} />
                            <stop offset="95%" stopColor="var(--chart-1)" stopOpacity={0.1} />
                        </linearGradient>
                        <linearGradient id="fillInputTokens" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-2)" stopOpacity={1.0} />
                            <stop offset="95%" stopColor="var(--chart-2)" stopOpacity={0.1} />
                        </linearGradient>
                        <linearGradient id="fillInputCost" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-3)" stopOpacity={1.0} />
                            <stop offset="95%" stopColor="var(--chart-3)" stopOpacity={0.1} />
                        </linearGradient>
                        <linearGradient id="fillOutputCost" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-4)" stopOpacity={1.0} />
                            <stop offset="95%" stopColor="var(--chart-4)" stopOpacity={0.1} />
                        </linearGradient>
                        <linearGradient id="fillRequests" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-5)" stopOpacity={1.0} />
                            <stop offset="95%" stopColor="var(--chart-5)" stopOpacity={0.1} />
                        </linearGradient>
                    </defs>
                    <XAxis dataKey="date" tickLine={false} axisLine={false} />
                    <ChartTooltip cursor={false} content={<ChartTooltipContent indicator="line" />} />
                    <Area type="monotone" dataKey="input_tokens" stroke="var(--chart-1)" fill="url(#fillInputTokens)" stackId="a" hide={hiddenSeries.has('input_tokens')} />
                    <Area type="monotone" dataKey="output_tokens" stroke="var(--chart-2)" fill="url(#fillOutputTokens)" stackId="a" hide={hiddenSeries.has('output_tokens')} />
                    <Area type="monotone" dataKey="input_cost" stroke="var(--chart-3)" fill="url(#fillInputCost)" stackId="a" hide={hiddenSeries.has('input_cost')} />
                    <Area type="monotone" dataKey="output_cost" stroke="var(--chart-4)" fill="url(#fillOutputCost)" stackId="a" hide={hiddenSeries.has('output_cost')} />
                    <Area type="monotone" dataKey="requests" stroke="var(--chart-5)" fill="url(#fillRequests)" stackId="a" hide={hiddenSeries.has('requests')} />
                    <ChartLegend content={(props) => (
                        <div className="flex items-center justify-center gap-4 pt-3">
                            {props.payload?.map((item) => (
                                <div
                                    key={item.value}
                                    className="flex items-center gap-1.5 cursor-pointer"
                                    onClick={() => {
                                        const key = item.dataKey as string;
                                        setHiddenSeries(prev => {
                                            const next = new Set(prev);
                                            next.has(key) ? next.delete(key) : next.add(key);
                                            return next;
                                        });
                                    }}
                                >
                                    <div
                                        className="h-2 w-2 shrink-0 rounded-[2px]"
                                        style={{ backgroundColor: item.color, opacity: hiddenSeries.has(item.dataKey as string) ? 0.3 : 1 }}
                                    />
                                    <span style={{ opacity: hiddenSeries.has(item.dataKey as string) ? 0.3 : 1 }}>
                                        {chartConfig[item.dataKey as keyof typeof chartConfig]?.label}
                                    </span>
                                </div>
                            ))}
                        </div>
                    )} />
                </AreaChart>
            </ChartContainer>
        </div>
    );
}
