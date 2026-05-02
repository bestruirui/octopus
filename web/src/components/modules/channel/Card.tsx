import {
    MorphingDialog,
    MorphingDialogTrigger,
    MorphingDialogContainer,
    MorphingDialogContent,
} from '@/components/ui/morphing-dialog';
import { CheckCircle2, DollarSign, Globe, Key, Layers, MessageSquare, XCircle } from 'lucide-react';
import { type StatsMetricsFormatted } from '@/api/endpoints/stats';
import { type Channel, useEnableChannel } from '@/api/endpoints/channel';
import { CardContent } from './CardContent';
import { useTranslations } from 'next-intl';
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/animate-ui/components/animate/tooltip';
import { Switch } from '@/components/ui/switch';
import { toast } from '@/components/common/Toast';
import { Badge } from '@/components/ui/badge';

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
    const firstBaseUrl = channel.base_urls?.find((item) => item.url.trim())?.url?.trim() ?? '';

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
            <MorphingDialogTrigger className="w-full text-left">
                <article
                    className={`waterhouse-island group relative flex w-full overflow-hidden rounded-[2.15rem] border border-border/35 bg-card/60 p-4 text-card-foreground shadow-waterhouse-soft transition-[border-color,box-shadow] duration-500 hover:border-primary/22 hover:shadow-waterhouse-deep md:bg-card/58 md:backdrop-blur-[var(--waterhouse-shell-blur)] md:transition-[transform,border-color,box-shadow] md:hover:-translate-y-1 ${isListLayout ? 'min-h-[12rem]' : 'min-h-[18rem]'}`}
                >
                    <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_14%_12%,color-mix(in_oklch,var(--waterhouse-highlight)_28%,transparent)_0%,transparent_26%),radial-gradient(circle_at_86%_16%,color-mix(in_oklch,var(--primary)_14%,transparent)_0%,transparent_24%),linear-gradient(180deg,color-mix(in_oklch,white_22%,transparent),transparent_30%,color-mix(in_oklch,var(--waterhouse-highlight)_10%,transparent))]" />
                    <div
                        className={`pointer-events-none absolute inset-x-3 bottom-3 rounded-[2rem] border border-white/12 bg-[linear-gradient(180deg,color-mix(in_oklch,var(--primary)_8%,transparent),color-mix(in_oklch,var(--waterhouse-highlight)_24%,transparent))] transition-opacity duration-500 ${channel.enabled ? 'h-24 opacity-100' : 'h-16 opacity-55'}`}
                    />

                    <div className="relative flex w-full flex-col gap-4">
                        <header className="flex items-start justify-between gap-3">
                            <div className="min-w-0 space-y-3">
                                <div className="flex items-center gap-2">
                                    <span className={`h-2.5 w-2.5 rounded-full shadow-waterhouse-soft ${channel.enabled ? 'bg-emerald-500' : 'bg-destructive'}`} />
                                    <Badge variant="secondary" className="rounded-full border border-border/25 bg-background/46 px-2.5 py-1 text-[0.68rem] shadow-waterhouse-soft">
                                        #{channel.id}
                                    </Badge>
                                    <Badge variant="secondary" className="rounded-full border border-border/25 bg-background/38 px-2.5 py-1 text-[0.68rem] shadow-waterhouse-soft">
                                        {enabledKeyCount}/{channel.keys.length}
                                    </Badge>
                                </div>
                                <Tooltip side="top" sideOffset={10} align="center">
                                    <TooltipTrigger asChild>
                                        <h3 className="max-w-full truncate text-xl font-semibold tracking-tight">{channel.name}</h3>
                                    </TooltipTrigger>
                                    <TooltipContent key={channel.name}>{channel.name}</TooltipContent>
                                </Tooltip>
                            </div>
                            <Switch
                                checked={channel.enabled}
                                onCheckedChange={handleEnableChange}
                                disabled={enableChannel.isPending}
                                onClick={(e) => e.stopPropagation()}
                            />
                        </header>

                        <div className={`grid gap-3 ${isListLayout ? 'lg:grid-cols-[minmax(0,1.2fr)_minmax(0,1.8fr)]' : 'grid-cols-1'}`}>
                            <div className="waterhouse-pod relative flex min-h-[6.5rem] flex-col justify-between overflow-hidden rounded-[1.7rem] border border-border/28 bg-background/36 p-3.5 shadow-none md:shadow-waterhouse-soft">
                                <div className="flex items-center gap-2 text-[0.72rem] font-semibold uppercase tracking-[0.16em] text-muted-foreground">
                                    <Globe className="size-3.5 text-primary" />
                                    {tSections('baseUrls')}
                                </div>
                                <p className="font-mono text-sm leading-6 text-foreground/90 line-clamp-2 break-all">
                                    {firstBaseUrl || '—'}
                                </p>
                            </div>

                            {isListLayout ? (
                                <dl className="grid grid-cols-2 gap-3 lg:grid-cols-4">
                                    <div className="waterhouse-pod rounded-[1.55rem] border border-border/25 bg-background/38 p-3 shadow-none">
                                        <dt className="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                                            <MessageSquare className="size-3.5 text-primary" />
                                            {t('requestCount')}
                                        </dt>
                                        <dd className="text-base font-semibold">
                                            {stats.request_count.formatted.value}
                                            <span className="ml-1 text-[0.7rem] text-muted-foreground">{stats.request_count.formatted.unit}</span>
                                        </dd>
                                    </div>
                                    <div className="waterhouse-pod rounded-[1.55rem] border border-border/25 bg-background/38 p-3 shadow-none">
                                        <dt className="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                                            <Layers className="size-3.5 text-primary" />
                                            {tForm('model')}
                                        </dt>
                                        <dd className="text-base font-semibold">{modelCount}</dd>
                                    </div>
                                    <div className="waterhouse-pod rounded-[1.55rem] border border-border/25 bg-background/38 p-3 shadow-none">
                                        <dt className="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                                            <CheckCircle2 className="size-3.5 text-emerald-500" />
                                            {tMetrics('successRequests')}
                                        </dt>
                                        <dd className="text-base font-semibold">{stats.request_success.formatted.value}</dd>
                                    </div>
                                    <div className="waterhouse-pod rounded-[1.55rem] border border-border/25 bg-background/38 p-3 shadow-none">
                                        <dt className="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                                            <DollarSign className="size-3.5 text-primary" />
                                            {t('totalCost')}
                                        </dt>
                                        <dd className="text-base font-semibold">
                                            {stats.total_cost.formatted.value}
                                            <span className="ml-1 text-[0.7rem] text-muted-foreground">{stats.total_cost.formatted.unit}</span>
                                        </dd>
                                    </div>
                                </dl>
                            ) : (
                                <dl className="grid grid-cols-2 gap-3">
                                    <div className="waterhouse-pod rounded-[1.55rem] border border-border/25 bg-background/42 p-3 shadow-none">
                                        <dt className="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                                            <MessageSquare className="size-3.5 text-primary" />
                                            {t('requestCount')}
                                        </dt>
                                        <dd className="text-base font-semibold">
                                            {stats.request_count.formatted.value}
                                            <span className="ml-1 text-[0.7rem] text-muted-foreground">{stats.request_count.formatted.unit}</span>
                                        </dd>
                                    </div>
                                    <div className="waterhouse-pod rounded-[1.55rem] border border-border/25 bg-background/42 p-3 shadow-none">
                                        <dt className="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                                            <DollarSign className="size-3.5 text-primary" />
                                            {t('totalCost')}
                                        </dt>
                                        <dd className="text-base font-semibold">
                                            {stats.total_cost.formatted.value}
                                            <span className="ml-1 text-[0.7rem] text-muted-foreground">{stats.total_cost.formatted.unit}</span>
                                        </dd>
                                    </div>
                                    <div className="waterhouse-pod rounded-[1.55rem] border border-border/25 bg-background/34 p-3 shadow-waterhouse-soft">
                                        <dt className="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                                            <Key className="size-3.5 text-primary" />
                                            {tSections('keys')}
                                        </dt>
                                        <dd className="text-base font-semibold">{enabledKeyCount}/{channel.keys.length}</dd>
                                    </div>
                                    <div className="waterhouse-pod rounded-[1.55rem] border border-border/25 bg-background/34 p-3 shadow-waterhouse-soft">
                                        <dt className="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                                            <Layers className="size-3.5 text-primary" />
                                            {tForm('model')}
                                        </dt>
                                        <dd className="text-base font-semibold">{modelCount}</dd>
                                    </div>
                                    <div className="waterhouse-pod rounded-[1.55rem] border border-border/25 bg-background/30 p-3 shadow-waterhouse-soft">
                                        <dt className="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                                            <CheckCircle2 className="size-3.5 text-emerald-500" />
                                            {tMetrics('successRequests')}
                                        </dt>
                                        <dd className="text-base font-semibold">{stats.request_success.formatted.value}</dd>
                                    </div>
                                    <div className="waterhouse-pod rounded-[1.55rem] border border-border/25 bg-background/30 p-3 shadow-waterhouse-soft">
                                        <dt className="mb-1.5 flex items-center gap-1.5 text-xs text-muted-foreground">
                                            <XCircle className="size-3.5 text-destructive" />
                                            {tMetrics('failedRequests')}
                                        </dt>
                                        <dd className="text-base font-semibold">{stats.request_failed.formatted.value}</dd>
                                    </div>
                                </dl>
                            )}
                        </div>
                    </div>
                </article>
            </MorphingDialogTrigger>

            <MorphingDialogContainer>
                <MorphingDialogContent className="waterhouse-island relative flex w-[min(100vw-1rem,56rem)] max-w-full flex-col overflow-hidden rounded-[2.4rem] border border-border/35 bg-background/80 px-4 py-4 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:px-6 md:py-5">
                    <CardContent channel={channel} stats={stats} />
                </MorphingDialogContent>
            </MorphingDialogContainer>
        </MorphingDialog>
    );
}
