'use client';

import { useMemo } from 'react';
import { Activity, CheckCircle2, Gauge, KeyRound, TimerReset, TriangleAlert, Waves } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useStatsHealth, type HealthGrade } from '@/api/endpoints/stats';
import { PageWrapper } from '@/components/common/PageWrapper';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { cn, formatCount, formatTime } from '@/lib/utils';

const gradeStyles: Record<HealthGrade, string> = {
    excellent: 'bg-emerald-500/10 text-emerald-700 border-emerald-500/20',
    good: 'bg-sky-500/10 text-sky-700 border-sky-500/20',
    warning: 'bg-amber-500/10 text-amber-700 border-amber-500/20',
    critical: 'bg-rose-500/10 text-rose-700 border-rose-500/20',
};

function ScoreBadge({ score, grade, t }: { score: number; grade: HealthGrade; t: (key: string) => string }) {
    return (
        <Badge variant="outline" className={cn('rounded-full px-2.5 py-1 text-xs font-semibold', gradeStyles[grade])}>
            {t(`grade.${grade}`)} {score.toFixed(1)}
        </Badge>
    );
}

export function Health() {
    const t = useTranslations('health');
    const { data = [], isLoading } = useStatsHealth(true);

    const summary = useMemo(() => {
        const channelCount = data.length;
        const healthyChannels = data.filter((item) => item.score >= 75).length;
        const totalModels = data.reduce((sum, item) => sum + item.total_models, 0);
        const healthyModels = data.reduce((sum, item) => sum + item.healthy_models, 0);
        const avgScore = channelCount > 0
            ? data.reduce((sum, item) => sum + item.score, 0) / channelCount
            : 0;
        return { channelCount, healthyChannels, totalModels, healthyModels, avgScore };
    }, [data]);

    const summaryCards = [
        {
            key: 'channels',
            label: t('summary.channels'),
            value: summary.channelCount,
            sub: `${summary.healthyChannels}/${summary.channelCount} ${t('summary.healthy')}`,
            icon: Activity,
        },
        {
            key: 'models',
            label: t('summary.models'),
            value: summary.totalModels,
            sub: `${summary.healthyModels}/${summary.totalModels} ${t('summary.healthy')}`,
            icon: CheckCircle2,
        },
        {
            key: 'avg',
            label: t('summary.avgScore'),
            value: summary.avgScore.toFixed(1),
            sub: t('summary.scoreHint'),
            icon: Gauge,
        },
    ];

    return (
        <PageWrapper className="h-full min-h-0 overflow-y-auto overscroll-contain space-y-6 pb-24 md:pb-4 rounded-t-3xl">
            <section className="grid grid-cols-1 gap-4 md:grid-cols-3">
                {summaryCards.map((card) => (
                    <Card key={card.key} className="rounded-3xl border-border/70 bg-card/95 py-0 shadow-sm">
                        <CardHeader className="flex flex-row items-center justify-between px-5 py-5">
                            <div>
                                <CardDescription>{card.label}</CardDescription>
                                <CardTitle className="mt-2 text-3xl font-bold">
                                    <AnimatedNumber value={card.value} />
                                </CardTitle>
                            </div>
                            <div className="grid h-11 w-11 place-items-center rounded-2xl bg-primary/10 text-primary">
                                <card.icon className="h-5 w-5" />
                            </div>
                        </CardHeader>
                        <CardContent className="px-5 pb-5 pt-0 text-sm text-muted-foreground">
                            {card.sub}
                        </CardContent>
                    </Card>
                ))}
            </section>

            <Card className="rounded-3xl border-border/70 bg-card/95 py-0 shadow-sm">
                <CardHeader className="px-5 py-5">
                    <CardTitle>{t('title')}</CardTitle>
                    <CardDescription>{t('description')}</CardDescription>
                </CardHeader>
                <CardContent className="space-y-5 px-5 pb-5 pt-0">
                    {isLoading && data.length === 0 && (
                        <div className="rounded-2xl border border-dashed border-border/70 px-4 py-10 text-center text-sm text-muted-foreground">
                            {t('loading')}
                        </div>
                    )}

                    {!isLoading && data.length === 0 && (
                        <div className="rounded-2xl border border-dashed border-border/70 px-4 py-10 text-center text-sm text-muted-foreground">
                            {t('empty')}
                        </div>
                    )}

                    {data.map((channel) => {
                        const successRate = `${(channel.success_rate * 100).toFixed(1)}%`;
                        const avgWait = formatTime(channel.avg_wait_time).formatted;
                        const requests = formatCount(channel.request_count).formatted;

                        return (
                            <article key={channel.channel_id} className="overflow-hidden rounded-3xl border border-border/70 bg-muted/20">
                                <div className="grid gap-4 border-b border-border/60 px-4 py-4 md:grid-cols-[minmax(0,1fr)_240px] md:px-5">
                                    <div className="space-y-3">
                                        <div className="flex flex-wrap items-center gap-2">
                                            <h3 className="text-lg font-semibold">{channel.channel_name}</h3>
                                            <ScoreBadge score={channel.score} grade={channel.grade} t={t} />
                                            {!channel.enabled && (
                                                <Badge variant="outline" className="rounded-full px-2.5 py-1 text-xs font-semibold text-muted-foreground">
                                                    {t('disabled')}
                                                </Badge>
                                            )}
                                        </div>
                                        <Progress value={channel.score} className="h-2.5 bg-muted" />
                                        <div className="grid grid-cols-2 gap-3 text-sm md:grid-cols-4">
                                            <div className="rounded-2xl bg-background/80 px-3 py-3">
                                                <div className="mb-1 flex items-center gap-2 text-muted-foreground">
                                                    <Waves className="h-4 w-4" />
                                                    {t('columns.successRate')}
                                                </div>
                                                <div className="font-semibold">{successRate}</div>
                                            </div>
                                            <div className="rounded-2xl bg-background/80 px-3 py-3">
                                                <div className="mb-1 flex items-center gap-2 text-muted-foreground">
                                                    <TimerReset className="h-4 w-4" />
                                                    {t('columns.avgWait')}
                                                </div>
                                                <div className="font-semibold">{avgWait.value}{avgWait.unit}</div>
                                            </div>
                                            <div className="rounded-2xl bg-background/80 px-3 py-3">
                                                <div className="mb-1 flex items-center gap-2 text-muted-foreground">
                                                    <KeyRound className="h-4 w-4" />
                                                    {t('columns.keys')}
                                                </div>
                                                <div className="font-semibold">{channel.enabled_keys}/{channel.total_keys}</div>
                                            </div>
                                            <div className="rounded-2xl bg-background/80 px-3 py-3">
                                                <div className="mb-1 flex items-center gap-2 text-muted-foreground">
                                                    <TriangleAlert className="h-4 w-4" />
                                                    {t('columns.requests')}
                                                </div>
                                                <div className="font-semibold">{requests.value}{requests.unit}</div>
                                            </div>
                                        </div>
                                    </div>

                                    <div className="rounded-3xl bg-background/90 p-4">
                                        <div className="text-sm text-muted-foreground">{t('detailTitle')}</div>
                                        <div className="mt-3 space-y-2 text-sm">
                                            <div className="flex items-center justify-between">
                                                <span>{t('columns.baseDelay')}</span>
                                                <span className="font-semibold">{channel.base_url_delay}ms</span>
                                            </div>
                                            <div className="flex items-center justify-between">
                                                <span>{t('columns.models')}</span>
                                                <span className="font-semibold">{channel.healthy_models}/{channel.total_models}</span>
                                            </div>
                                            <div className="flex items-center justify-between">
                                                <span>{t('columns.success')}</span>
                                                <span className="font-semibold">{channel.request_success}</span>
                                            </div>
                                            <div className="flex items-center justify-between">
                                                <span>{t('columns.failed')}</span>
                                                <span className="font-semibold">{channel.request_failed}</span>
                                            </div>
                                        </div>
                                    </div>
                                </div>

                                <div className="px-2 py-2 md:px-4 md:py-4">
                                    <Table>
                                        <TableHeader>
                                            <TableRow>
                                                <TableHead>{t('modelColumns.name')}</TableHead>
                                                <TableHead>{t('modelColumns.score')}</TableHead>
                                                <TableHead>{t('modelColumns.successRate')}</TableHead>
                                                <TableHead>{t('modelColumns.requests')}</TableHead>
                                                <TableHead>{t('modelColumns.avgWait')}</TableHead>
                                            </TableRow>
                                        </TableHeader>
                                        <TableBody>
                                            {(channel.models ?? []).map((model) => {
                                                const modelAvgWait = formatTime(model.avg_wait_time).formatted;
                                                return (
                                                    <TableRow key={`${channel.channel_id}-${model.model_name}`}>
                                                        <TableCell className="font-medium">{model.model_name}</TableCell>
                                                        <TableCell>
                                                            <ScoreBadge score={model.score} grade={model.grade} t={t} />
                                                        </TableCell>
                                                        <TableCell>{(model.success_rate * 100).toFixed(1)}%</TableCell>
                                                        <TableCell>{model.request_count}</TableCell>
                                                        <TableCell>{modelAvgWait.value}{modelAvgWait.unit}</TableCell>
                                                    </TableRow>
                                                );
                                            })}
                                        </TableBody>
                                    </Table>
                                </div>
                            </article>
                        );
                    })}
                </CardContent>
            </Card>
        </PageWrapper>
    );
}
