import {
    MorphingDialog,
    MorphingDialogTrigger,
    MorphingDialogContainer,
    MorphingDialogContent,
} from '@/components/ui/morphing-dialog';
import { DollarSign, MessageSquare } from 'lucide-react';
import { type StatsMetricsFormatted } from '@/api/endpoints/stats';
import { type Channel } from '@/api/endpoints/channel';
import { CardContent } from './CardContent';
import { useTranslations } from 'next-intl';
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/animate-ui/components/animate/tooltip';

export function Card({ channel, stats }: { channel: Channel; stats: StatsMetricsFormatted }) {
    const t = useTranslations('channel.card');
    const statusClasses = channel.enabled ? 'bg-accent text-accent-foreground' : 'bg-destructive text-destructive-foreground';

    return (
        <MorphingDialog>
            <MorphingDialogTrigger className="w-full">
                <article className="relative flex h-54 flex-col justify-between gap-5 rounded-3xl border border-border bg-card text-card-foreground p-4 custom-shadow transition-all duration-300 hover:scale-[1.02]">
                    <header className="relative flex items-start justify-between gap-2">
                        <Tooltip side="top" sideOffset={10} align="center">
                            <TooltipTrigger asChild>
                                <h3 className="text-lg font-bold truncate min-w-0">{channel.name}</h3>
                            </TooltipTrigger>
                            <TooltipContent>{channel.name}</TooltipContent>
                        </Tooltip>
                        <p className={`shrink-0 rounded-xl px-3 py-1 text-xs ${statusClasses}`}>
                            {channel.enabled ? t('status.enabled') : t('status.disabled')}
                        </p>
                    </header>

                    <dl className="relative grid grid-cols-1 gap-3">
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
                </article>
            </MorphingDialogTrigger>

            <MorphingDialogContainer>
                <MorphingDialogContent className="w-full md:max-w-xl bg-card text-card-foreground px-4 py-2 custom-shadow rounded-3xl max-h-[90vh] overflow-y-auto">
                    <CardContent channel={channel} stats={stats} />
                </MorphingDialogContent>
            </MorphingDialogContainer>
        </MorphingDialog>
    );
}
