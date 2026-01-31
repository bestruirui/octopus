import {
    MorphingDialog,
    MorphingDialogTrigger,
    MorphingDialogContainer,
    MorphingDialogContent,
} from '@/components/ui/morphing-dialog';
import { DollarSign, MessageSquare, Key, Globe, CheckCircle2, XCircle } from 'lucide-react';
import { type StatsMetricsFormatted } from '@/api/endpoints/stats';
import { type Channel, useEnableChannel, ChannelType } from '@/api/endpoints/channel';
import { CardContent } from './CardContent';
import { useTranslations } from 'next-intl';
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/animate-ui/components/animate/tooltip';
import { Switch } from '@/components/ui/switch';
import { toast } from '@/components/common/Toast';
import { Badge } from '@/components/ui/badge';

interface CardProps {
    channel: Channel;
    stats: StatsMetricsFormatted;
    layout?: 'grid' | 'single-column';
}

// 单列布局的内部组件
function SingleColumnCardContent({ channel, stats, handleEnableChange, enableChannel, t }: {
    channel: Channel;
    stats: StatsMetricsFormatted;
    handleEnableChange: (checked: boolean) => void;
    enableChannel: ReturnType<typeof useEnableChannel>;
    t: (key: string) => string;
}) {
    // 根据 ChannelType 枚举获取类型标签文本
    const getChannelTypeLabel = (type: number) => {
        switch (type) {
            case ChannelType.OpenAIChat:
                return 'OpenAI Chat';
            case ChannelType.OpenAIResponse:
                return 'OpenAI Response';
            case ChannelType.Anthropic:
                return 'Anthropic';
            case ChannelType.Gemini:
                return 'Gemini';
            case ChannelType.Volcengine:
                return 'Volcengine';
            case ChannelType.OpenAIEmbedding:
                return 'OpenAI Embedding';
            default:
                return `Type ${type}`;
        }
    };

    // 根据类型获取标签颜色
    const getChannelTypeVariant = (type: number): "default" | "secondary" | "destructive" | "outline" => {
        // 所有类型统一使用 outline 样式
        return 'outline';
    };

    return (
        <article className="relative flex flex-col gap-3 rounded-3xl border border-border bg-card text-card-foreground p-4 custom-shadow transition-all duration-300 hover:scale-[1.01]">
            {/* 头部：名称、类型、开关 */}
            <header className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1.5">
                        <Tooltip side="top" sideOffset={10} align="start">
                            <TooltipTrigger asChild>
                                <h3 className="text-lg font-bold truncate">{channel.name}</h3>
                            </TooltipTrigger>
                            <TooltipContent>{channel.name}</TooltipContent>
                        </Tooltip>
                        <Badge variant={getChannelTypeVariant(channel.type)} className="text-xs shrink-0 h-5">
                            {getChannelTypeLabel(channel.type)}
                        </Badge>
                    </div>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        <Globe className="h-3 w-3" />
                        <Tooltip>
                            <TooltipTrigger asChild>
                                <span className="truncate max-w-[300px]">
                                    {channel.base_urls?.[0]?.url || 'N/A'}
                                </span>
                            </TooltipTrigger>
                            <TooltipContent>{channel.base_urls?.[0]?.url || 'N/A'}</TooltipContent>
                        </Tooltip>
                        {channel.base_urls?.[0]?.delay !== undefined && channel.base_urls[0].delay > 0 && (
                            <Badge
                                variant="secondary"
                                className={
                                    channel.base_urls[0].delay < 300
                                        ? "h-4 px-1.5 text-xs bg-green-500/15 text-green-700 dark:text-green-400"
                                        : channel.base_urls[0].delay < 1000
                                        ? "h-4 px-1.5 text-xs bg-orange-500/15 text-orange-700 dark:text-orange-400"
                                        : "h-4 px-1.5 text-xs bg-red-500/15 text-red-700 dark:text-red-400"
                                }
                            >
                                {channel.base_urls[0].delay}ms
                            </Badge>
                        )}
                    </div>
                </div>
                <Switch
                    checked={channel.enabled}
                    onCheckedChange={handleEnableChange}
                    disabled={enableChannel.isPending}
                    onClick={(e) => e.stopPropagation()}
                />
            </header>

            {/* 统计信息网格 */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                {/* 请求次数 */}
                <div className="flex flex-col gap-0.5 rounded-xl border border-border/70 bg-background/80 p-2">
                    <div className="flex items-center gap-1.5 text-muted-foreground">
                        <MessageSquare className="h-3.5 w-3.5" />
                        <span className="text-xs">{t('requestCount')}</span>
                    </div>
                    <div className="text-base font-semibold">
                        {stats.request_count.formatted.value}
                        <span className="ml-1 text-xs text-muted-foreground font-normal">
                            {stats.request_count.formatted.unit}
                        </span>
                    </div>
                </div>

                {/* 总费用 */}
                <div className="flex flex-col gap-0.5 rounded-xl border border-border/70 bg-background/80 p-2">
                    <div className="flex items-center gap-1.5 text-muted-foreground">
                        <DollarSign className="h-3.5 w-3.5" />
                        <span className="text-xs">{t('totalCost')}</span>
                    </div>
                    <div className="text-base font-semibold">
                        {stats.total_cost.formatted.value}
                        <span className="ml-1 text-xs text-muted-foreground font-normal">
                            {stats.total_cost.formatted.unit}
                        </span>
                    </div>
                </div>

                {/* 成功请求 */}
                <div className="flex flex-col gap-0.5 rounded-xl border border-border/70 bg-background/80 p-2">
                    <div className="flex items-center gap-1.5 text-muted-foreground">
                        <CheckCircle2 className="h-3.5 w-3.5 text-green-600 dark:text-green-400" />
                        <span className="text-xs">{t('successRequests')}</span>
                    </div>
                    <div className="text-base font-semibold text-green-600 dark:text-green-400">
                        {stats.request_success.formatted.value}
                        <span className="ml-1 text-xs text-muted-foreground font-normal">
                            {stats.request_success.formatted.unit}
                        </span>
                    </div>
                </div>

                {/* 失败请求 */}
                <div className="flex flex-col gap-0.5 rounded-xl border border-border/70 bg-background/80 p-2">
                    <div className="flex items-center gap-1.5 text-muted-foreground">
                        <XCircle className="h-3.5 w-3.5 text-red-600 dark:text-red-400" />
                        <span className="text-xs">{t('failedRequests')}</span>
                    </div>
                    <div className="text-base font-semibold text-red-600 dark:text-red-400">
                        {stats.request_failed.formatted.value}
                        <span className="ml-1 text-xs text-muted-foreground font-normal">
                            {stats.request_failed.formatted.unit}
                        </span>
                    </div>
                </div>
            </div>

            {/* 底部：Keys 数量、自动同步、代理状态 */}
            <footer className="flex items-center justify-between text-xs pt-2 border-t border-border/50">
                <div className="flex items-center gap-1.5 text-muted-foreground">
                    <Key className="h-3 w-3" />
                    <span>{channel.keys.length} {t('keysCount')}</span>
                </div>
                <div className="flex items-center gap-1.5 flex-wrap">
                    <Badge
                        variant={channel.auto_sync ? "default" : "secondary"}
                        className="text-xs h-5"
                    >
                        {t('autoSync')}
                    </Badge>
                    <Badge
                        variant={channel.proxy ? "default" : "secondary"}
                        className="text-xs h-5"
                    >
                        {t('proxy')}
                    </Badge>
                </div>
            </footer>
        </article>
    );
}

export function Card({ channel, stats, layout = 'grid' }: CardProps) {
    const t = useTranslations('channel.card');
    const enableChannel = useEnableChannel();

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

    // 单列布局：展示更多详细信息
    if (layout === 'single-column') {
        return (
            <MorphingDialog>
                <MorphingDialogTrigger className="w-full">
                    <SingleColumnCardContent
                        channel={channel}
                        stats={stats}
                        handleEnableChange={handleEnableChange}
                        enableChannel={enableChannel}
                        t={t}
                    />
                </MorphingDialogTrigger>

                <MorphingDialogContainer>
                    <MorphingDialogContent className="w-full md:max-w-xl bg-card text-card-foreground px-4 py-2 custom-shadow rounded-3xl max-h-[90vh] overflow-y-auto">
                        <CardContent channel={channel} stats={stats} />
                    </MorphingDialogContent>
                </MorphingDialogContainer>
            </MorphingDialog>
        );
    }

    // 网格布局：使用原有的紧凑卡片
    return (
        <MorphingDialog>
            <MorphingDialogTrigger className="w-full">
                <article className="relative flex h-54 flex-col justify-between gap-5 rounded-3xl border border-border bg-card text-card-foreground p-4 custom-shadow transition-all duration-300 hover:scale-[1.02]">
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
