import {
    MorphingDialog,
    MorphingDialogTrigger,
    MorphingDialogContainer,
    MorphingDialogContent,
} from '@/components/ui/morphing-dialog';
import { CheckCircle2, DollarSign, Key, Layers, MessageSquare, XCircle } from 'lucide-react';
import { type StatsMetricsFormatted } from '@/api/endpoints/stats';
import { type Channel, useEnableChannel } from '@/api/endpoints/channel';
import { CardContent } from './CardContent';
import { useTranslations } from 'next-intl';
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/animate-ui/components/animate/tooltip';
import { Switch } from '@/components/ui/switch';
import { toast } from '@/components/common/Toast';

export function Card({ channel, stats, layout = 'grid' }: { channel: Channel; stats: StatsMetricsFormatted; layout?: 'grid' | 'list' }) {
    const t = useTranslations('channel.card');
    const tForm = useTranslations('channel.form');
    const tSections = useTranslations('channel.detail.sections');
    const tMetrics = useTranslations('channel.detail.metrics');
    const enableChannel = useEnableChannel();
    const isListLayout = layout === 'list';

    const splitModels = (models: string) =>
        models
            .split(',')
            .map((item) => item.trim())
            .filter(Boolean);

    const modelCount = new Set([
        ...splitModels(channel.model),
        ...splitModels(channel.custom_model),
    ]).size;
    const enabledKeyCount = channel.keys.filter((item) => item.enabled).length;

    const handleEnableChange = (checked: boolean) => {
        enableChannel.mutate(
            { id: channel.id, enabled: checked },
            {
                onSuccess: () => {
                    toast.success(checked ? t('toast.enabled') : t('toast.disabled'));
                },
                onError: (error) => {
                    toast.error(error.message);
                },
            }
        );
    };

    return (
        <MorphingDialog>
            <MorphingDialogTrigger className="w-full">
                <article className="flex flex-col gap-4 rounded-3xl border border-border bg-card text-card-foreground p-4 transition-all duration-300">
                    <header className="relative flex items-center justify-between gap-2">
                        <Tooltip side="top" sideOffset={10} align="center">
                            <TooltipTrigger asChild>
                                <h3 className="text-lg font-bold truncate min-w-0">{channel.name}</h3>
                            </TooltipTrigger>
                            <TooltipContent key={channel.name}>{channel.name}</TooltipContent>
                        </Tooltip>
                        <Switch
                            checked={channel.enabled}
                            onCheckedChange={handleEnableChange}
                            disabled={enableChannel.isPending}
                            onClick={(e) => e.stopPropagation()}
                        />
                    </header>

                    {isListLayout ? (
                        <dl className="grid grid-cols-2 gap-2 lg:grid-cols-6">
                            <div className="rounded-2xl border border-border/70 bg-background/80 p-2">
                                <dt className="mb-1 flex items-center gap-1 text-xs text-muted-foreground">
                                    <MessageSquare className="size-3.5 text-primary" />
                                    {t('requestCount')}
                                </dt>
                                <dd className="text-sm font-semibold">
                                    {stats.request_count.formatted.value}
                                    <span className="ml-1 text-xs text-muted-foreground">{stats.request_count.formatted.unit}</span>
                                </dd>
                            </div>
                            <div className="rounded-2xl border border-border/70 bg-background/80 p-2">
                                <dt className="mb-1 flex items-center gap-1 text-xs text-muted-foreground">
                                    <Layers className="size-3.5 text-primary" />
                                    {tForm('model')}
                                </dt>
                                <dd className="text-sm font-semibold">{modelCount}</dd>
                            </div>
                            <div className="rounded-2xl border border-border/70 bg-background/80 p-2">
                                <dt className="mb-1 flex items-center gap-1 text-xs text-muted-foreground">
                                    <Key className="size-3.5 text-primary" />
                                    {tSections('keys')}
                                </dt>
                                <dd className="text-sm font-semibold">{enabledKeyCount}/{channel.keys.length}</dd>
                            </div>
                            <div className="rounded-2xl border border-border/70 bg-background/80 p-2">
                                <dt className="mb-1 flex items-center gap-1 text-xs text-muted-foreground">
                                    <CheckCircle2 className="size-3.5 text-emerald-500" />
                                    {tMetrics('successRequests')}
                                </dt>
                                <dd className="text-sm font-semibold">{stats.request_success.formatted.value}</dd>
                            </div>
                            <div className="rounded-2xl border border-border/70 bg-background/80 p-2">
                                <dt className="mb-1 flex items-center gap-1 text-xs text-muted-foreground">
                                    <XCircle className="size-3.5 text-destructive" />
                                    {tMetrics('failedRequests')}
                                </dt>
                                <dd className="text-sm font-semibold">{stats.request_failed.formatted.value}</dd>
                            </div>
                            <div className="rounded-2xl border border-border/70 bg-background/80 p-2">
                                <dt className="mb-1 flex items-center gap-1 text-xs text-muted-foreground">
                                    <DollarSign className="size-3.5 text-primary" />
                                    {t('totalCost')}
                                </dt>
                                <dd className="text-sm font-semibold">
                                    {stats.total_cost.formatted.value}
                                    <span className="ml-1 text-xs text-muted-foreground">{stats.total_cost.formatted.unit}</span>
                                </dd>
                            </div>
                        </dl>
                    ) : (
                        <dl className="grid grid-cols-1 gap-3">
                            <div className="flex items-center justify-between rounded-2xl border border-border/70 bg-background/80 p-2">
                                <div className="flex items-center gap-3">
                                    <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                                        <MessageSquare className="h-5 w-5" />
                                    </span>
                                    <dt className="text-sm text-muted-foreground">{t('requestCount')}</dt>
                                </div>
                                <dd className="text-base">
                                    {stats.request_count.formatted.value}
                                    <span className="ml-1 text-xs text-muted-foreground">{stats.request_count.formatted.unit}</span>
                                </dd>
                            </div>

                            <div className="flex items-center justify-between rounded-2xl border border-border/70 bg-background/80 p-2">
                                <div className="flex items-center gap-3">
                                    <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                                        <DollarSign className="h-5 w-5" />
                                    </span>
                                    <dt className="text-sm text-muted-foreground">{t('totalCost')}</dt>
                                </div>
                                <dd className="text-base">
                                    {stats.total_cost.formatted.value}
                                    <span className="ml-1 text-xs text-muted-foreground">{stats.total_cost.formatted.unit}</span>
                                </dd>
                            </div>
                        </dl>
                    )}

                </article>
            </MorphingDialogTrigger>

            <MorphingDialogContainer>
                <MorphingDialogContent className="w-full md:max-w-xl bg-card text-card-foreground px-4 py-2 rounded-3xl max-h-[90vh] overflow-y-auto">
                    <CardContent channel={channel} stats={stats} />
                </MorphingDialogContent>
            </MorphingDialogContainer>
        </MorphingDialog>
    );
}
