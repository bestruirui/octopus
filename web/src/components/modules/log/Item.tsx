import { memo, useEffect, useMemo, useState, type CSSProperties } from 'react';
import { AlertCircle, ArrowDownToLine, ArrowRight, ArrowUpFromLine, Clock, Cpu, Database, DollarSign, Gauge, Loader2, Square, Timer } from 'lucide-react';
import { useTranslations } from 'use-intl';
import JsonView from '@uiw/react-json-view';
import { githubDarkTheme } from '@uiw/react-json-view/githubDark';
import { githubLightTheme } from '@uiw/react-json-view/githubLight';
import { useTheme } from '@/provider/theme';
import { type RelayLogOverview, useLogRequestBody, useLogResponseBody, useStopRound } from '@/api/log';
import { useGroupList, useUpdateGroupActiveItem } from '@/api/group';
import { useChannelList } from '@/api/channel';
import { getModelIcon } from '@/lib/model-icons';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';
import { CopyIconButton } from '@/components/common/CopyButton';
import { toast } from 'sonner';
import { MemberStatus } from '@/components/modules/group/MemberStatus';
import {
    MorphingDialog,
    MorphingDialogTrigger,
    MorphingDialogContainer,
    MorphingDialogContent,
    MorphingDialogClose,
    MorphingDialogTitle,
    MorphingDialogDescription,
    useMorphingDialog,
} from '@/components/ui/morphing-dialog';

// formatTime 将后端 RFC3339 时间转换为本地时分秒。
function formatTime(value: string) {
    const date = new Date(value);
    if (Number.isNaN(date.getTime()) || date.getUTCFullYear() === 1) return '--';
    return date.toLocaleTimeString(undefined, {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
    });
}

// formatMilliseconds 将毫秒转换为紧凑耗时文本。
function formatMilliseconds(value: number) {
    const milliseconds = Math.max(0, value);
    if (milliseconds < 1000) return `${Math.round(milliseconds)}ms`;
    return `${(milliseconds / 1000).toFixed(2)}s`;
}

// LogMetrics 渲染耗时, 费用和 Token 指标; card 变体用于卡片栅格, footer 变体用于弹窗底部。
function LogMetrics({ log, now, brandColor, variant }: { log: RelayLogOverview; now: number; brandColor: string; variant: 'card' | 'footer' }) {
    const t = useTranslations('log.card');
    const cachedTokens = log.usage.prompt_tokens_details?.cached_tokens ?? 0;
    // 进行中的请求按共享时钟推算耗时, 结束后改用后端记录的最终耗时。
    const duration = log.status === 'running' || log.status === 'committed'
        ? formatMilliseconds(now - new Date(log.started_at).getTime())
        : formatMilliseconds(log.duration / 1_000_000);
    const firstTextDuration = log.first_text_duration > 0
        ? formatMilliseconds(log.first_text_duration / 1_000_000)
        : '--';
    const generationSeconds = (log.duration - log.first_text_duration) / 1_000_000_000;
    const outputSpeed = log.first_text_duration > 0 && log.usage.completion_tokens > 0 && generationSeconds > 0
        ? `${(log.usage.completion_tokens / generationSeconds).toFixed(1)} t/s`
        : '--';
    const metrics = [
        { key: 'time', Icon: Clock, iconClassName: 'size-3.5 shrink-0', iconStyle: { color: brandColor } as CSSProperties, value: formatTime(log.started_at), valueClassName: 'tabular-nums', cellClassName: 'col-span-4 whitespace-nowrap md:col-span-1' },
        { key: 'duration', Icon: Cpu, iconClassName: 'size-3.5 shrink-0 text-blue-500', value: duration, cellClassName: 'col-span-4 md:col-span-1' },
        { key: 'firstText', Icon: Timer, iconClassName: 'size-3.5 shrink-0 text-amber-500', value: firstTextDuration, cellClassName: 'col-span-4 md:col-span-1' },
        { key: 'outputSpeed', Icon: Gauge, iconClassName: 'size-3.5 shrink-0 text-violet-500', value: outputSpeed, cellClassName: 'col-span-4 md:col-span-1' },
        { key: 'cost', Icon: DollarSign, iconClassName: 'size-3.5 shrink-0 text-emerald-500', value: log.cost.toFixed(6), valueClassName: 'font-medium text-emerald-600 dark:text-emerald-400', cellClassName: 'col-span-4 md:col-span-1' },
        { key: 'prompt', Icon: ArrowDownToLine, iconClassName: 'size-3.5 shrink-0 text-green-500', value: (log.usage.prompt_tokens - cachedTokens).toLocaleString(), cellClassName: 'col-span-3 md:col-span-1' },
        { key: 'cached', Icon: Database, iconClassName: 'size-3.5 shrink-0 text-cyan-500', value: cachedTokens.toLocaleString(), cellClassName: 'col-span-3 md:col-span-1' },
        { key: 'completion', Icon: ArrowUpFromLine, iconClassName: 'size-3.5 shrink-0 text-purple-500', value: log.usage.completion_tokens.toLocaleString(), cellClassName: 'col-span-3 md:col-span-1' },
        { key: 'cacheWrite', Icon: Database, iconClassName: 'size-3.5 shrink-0 text-orange-500', value: (log.usage.prompt_tokens_details?.write_cached_tokens ?? 0).toLocaleString(), cellClassName: 'col-span-3 md:col-span-1' },
    ];

    return metrics.map((metric) => (
        <div key={metric.key} title={metric.key === 'firstText' ? t('firstText') : metric.key === 'outputSpeed' ? t('outputSpeed') : undefined} className={cn('flex items-center gap-1.5', variant === 'card' && metric.cellClassName)}>
            <metric.Icon className={metric.iconClassName} style={metric.iconStyle} />
            <span className={metric.valueClassName}>{metric.value}</span>
        </div>
    ));
}

// ObservedRound 保存弹窗打开期间观察到的一轮上游请求状态。
interface ObservedRound {
    round: number; // 当前请求内递增的轮次序号。
    channel: string; // 本轮实际请求的渠道名称。
    error: string; // 本轮最近一次上游错误。
    sending: boolean; // 本轮是否仍在等待上游响应。
}

// JsonContent 渲染请求或响应正文, 能解析为 JSON 时使用折叠视图, 否则按纯文本展示。
function JsonContent({ content, fallbackText }: { content: string | object | undefined; fallbackText: string }) {
    const { resolvedTheme } = useTheme();

    const parsed = useMemo(() => {
        if (content === undefined || content === '') return null;
        if (typeof content !== 'string') return { isJson: true, data: content };
        try {
            return { isJson: true, data: JSON.parse(content) as object };
        } catch {
            return { isJson: false, data: content };
        }
    }, [content]);

    if (!parsed) {
        return (
            <pre className="p-4 text-xs text-muted-foreground whitespace-pre-wrap wrap-break-word leading-relaxed">
                {fallbackText}
            </pre>
        );
    }

    if (!parsed.isJson) {
        return (
            <pre className="p-4 text-xs text-muted-foreground whitespace-pre-wrap wrap-break-word font-mono leading-relaxed animate-in fade-in duration-200">
                {parsed.data as string}
            </pre>
        );
    }

    return (
        <div className="p-4 animate-in fade-in duration-200">
            <JsonView
                value={parsed.data as object}
                style={{
                    ...(resolvedTheme === 'dark' ? githubDarkTheme : githubLightTheme),
                    fontSize: '12px',
                    fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace',
                    backgroundColor: 'transparent',
                }}
                displayDataTypes={false}
                displayObjectSize={false}
                collapsed={false}
            />
        </div>
    );
}

// LogDetail 渲染日志详情弹窗内容, 仅在弹窗打开期间挂载, 由此避免列表中的卡片持有详情查询和状态。
function LogDetail({ log, now }: { log: RelayLogOverview; now: number }) {
    const t = useTranslations('log.card');
    const statusT = useTranslations('log.status');
    const [leftTab, setLeftTab] = useState<'request' | 'group'>('group');
    const [rounds, setRounds] = useState<ObservedRound[]>([]);
    const [observedRoundKey, setObservedRoundKey] = useState(''); // observedRoundKey 是已记入 rounds 的最近一次日志快照, 用于跳过重复渲染。
    const [detailReady, setDetailReady] = useState(false); // 展开动画结束后才允许加载详情数据。
    const [switchingItemId, setSwitchingItemId] = useState<number | null>(null);
    const requestBody = useLogRequestBody(log.id, log.started_at, detailReady && leftTab === 'request');
    const responseBody = useLogResponseBody(log.id, log.started_at, detailReady && log.status === 'success');
    const { data: groups = [] } = useGroupList(detailReady, detailReady);
    const { data: channels = [] } = useChannelList(detailReady);
    const updateActiveItem = useUpdateGroupActiveItem();
    const stopRound = useStopRound();
    const channelNameByModelID = useMemo(() => {
        const map = new Map<number, string>();
        channels.forEach(({ raw: channel }) => {
            channel.models.forEach((channelModel) => map.set(channelModel.id, channel.name));
        });
        return map;
    }, [channels]);
    const actualModel = log.target_model || log.model;
    const { Icon, className: iconClassName, color: brandColor } = getModelIcon(actualModel);
    const errorText = log.error ?? '';
    const requestFailed = log.status === 'failed' || log.status === 'canceled';
    const responseCommitted = log.status === 'committed';
    const showRounds = log.status === 'running' || (requestFailed && rounds.length > 0);
    const activeGroup = groups.find((group) => group.name === log.model);
    const isWaitingForSelection = log.status === 'running' && !log.sending && activeGroup?.mode === 'manual' && activeGroup.active_item_id === 0; // isWaitingForSelection 表示手动模式请求正等待选择渠道。

    // 让弹窗先完成展开动画, 避免详情请求及其状态更新占用动画起步帧。
    useEffect(() => {
        const timer = window.setTimeout(() => setDetailReady(true), 600);
        return () => window.clearTimeout(timer);
    }, []);

    // 按轮次记录本次打开期间观察到的上游请求状态, 最新一轮排在最前。
    // 轮次来自逐次推送的日志, 需在渲染期比对已记录的快照累积, 不能仅由当前 log 推导。
    const roundKey = log.round === 0 ? '' : `${log.round}:${log.target_channel}:${log.sending}:${errorText}`;
    if (roundKey !== '' && roundKey !== observedRoundKey) {
        setObservedRoundKey(roundKey);
        setRounds((current) => {
            if (!log.sending && current.every((item) => item.round !== log.round)) return current;
            return [
                { round: log.round, channel: log.target_channel, error: errorText, sending: log.sending },
                ...current.filter((item) => item.round !== log.round),
            ];
        });
    }

    return (
        <MorphingDialogContent className="relative w-[calc(100vw-2rem)] md:w-[80vw] bg-card text-card-foreground px-6 py-4 rounded-3xl h-[calc(100vh-2rem)] flex flex-col overflow-hidden">
            <MorphingDialogClose className="top-4 right-5 text-muted-foreground hover:text-foreground transition-colors" />
            <MorphingDialogTitle className="flex items-center gap-2 mb-3 text-sm">
                <Icon aria-hidden="true" className={iconClassName} width={28} height={28} />
                <span className="font-semibold text-card-foreground">{log.model || t('unknownModel')}</span>
                {log.status === 'running' || responseCommitted
                    ? <Loader2 className="size-3.5 animate-spin text-muted-foreground/50" />
                    : <ArrowRight className="size-3.5 text-muted-foreground/50" />}
                <Badge
                    variant="secondary"
                    className="text-xs px-1.5 py-0"
                    style={{ backgroundColor: `${brandColor}15`, color: brandColor }}
                >
                    {log.target_channel || '-'}
                </Badge>
                <span className="text-muted-foreground">{actualModel}</span>
            </MorphingDialogTitle>

            <MorphingDialogDescription className="flex-1 min-h-0">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 h-full min-h-0">
                    <div className="flex flex-col rounded-2xl border border-border bg-muted/30 overflow-hidden min-h-0">
                        <div className="flex h-10 shrink-0 items-center gap-2 border-b border-border bg-muted/50 pl-1 pr-3 md:pr-4">
                            <Tabs value={leftTab} onValueChange={(value) => setLeftTab(value as 'request' | 'group')}>
                                <TabsList variant="text" className="p-0">
                                    <TabsTrigger value="group" className="pr-0">
                                        {t('group')}
                                    </TabsTrigger>
                                    <span aria-hidden="true" className="mx-1 inline-flex h-full -translate-y-px items-center text-sm font-medium leading-none text-muted-foreground/50">/</span>
                                    <TabsTrigger value="request" className="pl-0">
                                        {t('requestContent')}
                                    </TabsTrigger>
                                </TabsList>
                            </Tabs>
                            {leftTab === 'request' && (
                                <Badge variant="secondary" className="ml-auto text-xs">
                                    {(log.usage.prompt_tokens - (log.usage.prompt_tokens_details?.cached_tokens ?? 0)).toLocaleString()} {t('tokens')}
                                </Badge>
                            )}
                        </div>
                        <div className="flex-1 overflow-auto min-h-0">
                            {!detailReady ? (
                                <div className="flex h-full items-center justify-center">
                                    <Loader2 className="size-5 animate-spin text-muted-foreground" />
                                </div>
                            ) : leftTab === 'request' ? (
                                requestBody.isLoading ? (
                                    <div className="flex h-full items-center justify-center">
                                        <Loader2 className="size-5 animate-spin text-muted-foreground" />
                                    </div>
                                ) : requestBody.error ? (
                                    <div className="flex h-full flex-col items-center justify-center gap-2 px-4 text-xs text-destructive">
                                        <AlertCircle className="size-5" />
                                        <span>{t('detailUnavailable')}</span>
                                    </div>
                                ) : (
                                    <JsonContent content={requestBody.data} fallbackText={t('noRequestContent')} />
                                )
                            ) : !activeGroup ? (
                                <div className="flex h-full items-center justify-center px-4 text-xs text-muted-foreground">
                                    {t('groupUnavailable')}
                                </div>
                            ) : !activeGroup.items?.length ? (
                                <div className="flex h-full items-center justify-center px-4 text-xs text-muted-foreground">
                                    {t('noGroupItems')}
                                </div>
                            ) : (
                                <div className="divide-y divide-border">
                                    {activeGroup.items.map((item) => {
                                        const modelName = item.channel_model?.name ?? '';
                                        const channelName = item.channel_model
                                            ? channelNameByModelID.get(item.channel_model.id) ?? `#${item.channel_model.channel_id}`
                                            : '-';
                                        const { Icon: ItemIcon, className: itemIconClassName } = getModelIcon(modelName);
                                        const itemActive = item.id === activeGroup.active_item_id;
                                        const itemSwitching = item.id === switchingItemId;
                                        const itemCurrent = switchingItemId !== null
                                            ? itemSwitching
                                            : activeGroup.mode === 'failover'
                                                ? activeGroup.runtime?.current_item_id === item.id
                                                : itemActive;
                                        return (
                                            <button
                                                key={item.id ?? item.channel_model_id}
                                                type="button"
                                                aria-pressed={itemCurrent}
                                                disabled={item.id === undefined || activeGroup.mode === 'failover' || switchingItemId !== null || stopRound.isPending}
                                                onClick={async () => {
                                                    if (!activeGroup.id || item.id === undefined || activeGroup.mode === 'failover') return;
                                                    setSwitchingItemId(item.id);
                                                    try {
                                                        await updateActiveItem.mutateAsync({ groupId: activeGroup.id, itemId: itemActive ? 0 : item.id });
                                                        if (log.sending) {
                                                            await stopRound.mutateAsync({ requestId: log.id, round: log.round });
                                                        }
                                                        toast.success(itemActive ? t('channelCleared') : t('channelChanged'));
                                                    } catch (cause) {
                                                        toast.error(t('channelChangeFailed'), { description: cause instanceof Error ? cause.message : undefined });
                                                    } finally {
                                                        setSwitchingItemId(null);
                                                    }
                                                }}
                                                className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2.5 text-left text-xs transition-colors hover:bg-muted/50 disabled:cursor-default disabled:hover:bg-transparent"
                                            >
                                                <ItemIcon aria-hidden="true" className={itemIconClassName} width={20} height={20} />
                                                <span className="min-w-0 flex-1">
                                                    <span className="block truncate font-semibold text-foreground">
                                                        {channelName}
                                                    </span>
                                                    <span className="block truncate text-[11px] text-muted-foreground">{modelName}</span>
                                                </span>
                                                <MemberStatus group={activeGroup} itemId={item.id} now={now} active={itemCurrent} />
                                                {itemSwitching && <Loader2 className="size-4 shrink-0 animate-spin text-muted-foreground" />}
                                            </button>
                                        );
                                    })}
                                </div>
                            )}
                        </div>
                    </div>

                    <div className="flex flex-col rounded-2xl border border-border bg-muted/30 overflow-hidden min-h-0">
                        <div className="flex h-10 shrink-0 items-center gap-2 border-b border-border bg-muted/50 px-3 md:px-4">
                            <span className="text-sm font-medium text-card-foreground">
                                {isWaitingForSelection ? t('waitingChannelSelection') : showRounds ? t('retryDetails') : requestFailed ? t('errorInfo') : t('responseContent')}
                            </span>
                            {log.status === 'running' && log.sending && activeGroup?.mode === 'manual' ? (
                                <button
                                    type="button"
                                    disabled={stopRound.isPending}
                                    onClick={async () => {
                                        try {
                                            await stopRound.mutateAsync({ requestId: log.id, round: log.round });
                                        } catch (cause) {
                                            toast.error(t('stopFailed'), { description: cause instanceof Error ? cause.message : undefined });
                                        }
                                    }}
                                    className="ml-auto flex items-center gap-1.5 rounded-md px-2 py-1 text-xs text-destructive transition-colors hover:bg-destructive/10 disabled:opacity-50"
                                >
                                    {stopRound.isPending ? <Loader2 className="size-3.5 animate-spin" /> : <Square className="size-3.5" />}
                                    {t('stopRound')}
                                </button>
                            ) : !requestFailed && (
                                <Badge variant="secondary" className="ml-auto text-xs">
                                    {responseCommitted
                                        ? statusT('committed')
                                        : `${log.usage.completion_tokens.toLocaleString()} ${t('tokens')}`}
                                </Badge>
                            )}
                        </div>
                        <div className="min-h-0 flex-1 overflow-auto">
                            {!detailReady ? (
                                <div className="flex h-full items-center justify-center">
                                    <Loader2 className="size-5 animate-spin text-muted-foreground" />
                                </div>
                            ) : isWaitingForSelection ? (
                                <div className="flex h-full items-center justify-center gap-2 text-xs text-muted-foreground">
                                    <Loader2 className="size-4 animate-spin" />
                                    {t('waitingChannelSelection')}
                                </div>
                            ) : showRounds ? (
                                rounds.length ? (
                                    <div className="divide-y divide-border">
                                        {rounds.map((round) => (
                                            <div key={round.round} className="flex flex-col gap-1.5 px-3 py-2.5 text-xs">
                                                <div className="flex items-center gap-2">
                                                    <span className="text-muted-foreground">{t('retryIndex', { index: round.round })}</span>
                                                    <span className="font-semibold text-foreground">{round.channel || '-'}</span>
                                                    {round.sending ? (
                                                        <Loader2 className="ml-auto size-3.5 animate-spin text-muted-foreground" />
                                                    ) : round.error ? (
                                                        <CopyIconButton
                                                            text={round.error}
                                                            className="ml-auto p-1 rounded-md text-destructive/60 hover:text-destructive hover:bg-destructive/10 transition-colors"
                                                            copyIconClassName="size-3.5"
                                                            checkIconClassName="size-3.5"
                                                        />
                                                    ) : null}
                                                </div>
                                                {round.error && (
                                                    <div className="text-[11px] leading-relaxed text-destructive/90 whitespace-pre-wrap wrap-break-word">
                                                        {round.error}
                                                    </div>
                                                )}
                                            </div>
                                        ))}
                                    </div>
                                ) : (
                                    <div className="flex h-full items-center justify-center gap-2 text-xs text-muted-foreground">
                                        <Loader2 className="size-4 animate-spin" />
                                        {t('waitingResponse')}
                                    </div>
                                )
                            ) : responseCommitted ? (
                                <div className="flex h-full items-center justify-center gap-2 text-xs text-muted-foreground">
                                    <Loader2 className="size-4 animate-spin" />
                                    {t('responseStreaming')}
                                </div>
                            ) : requestFailed ? (
                                <JsonContent content={errorText} fallbackText={t('noResponseContent')} />
                            ) : responseBody.isLoading ? (
                                <div className="flex h-full items-center justify-center">
                                    <Loader2 className="size-5 animate-spin text-muted-foreground" />
                                </div>
                            ) : responseBody.error ? (
                                <div className="flex h-full flex-col items-center justify-center gap-2 px-4 text-xs text-destructive">
                                    <AlertCircle className="size-5" />
                                    <span>{t('detailUnavailable')}</span>
                                </div>
                            ) : (
                                <JsonContent content={responseBody.data} fallbackText={t('noResponseContent')} />
                            )}
                        </div>
                    </div>
                </div>
            </MorphingDialogDescription>

            <div className="flex w-full shrink-0 flex-wrap items-center gap-3 pt-4 mt-auto text-xs text-muted-foreground md:gap-4">
                <LogMetrics log={log} now={now} brandColor={brandColor} variant="footer" />
            </div>
        </MorphingDialogContent>
    );
}

// LogCardBody 渲染日志概览卡片, 并在弹窗打开时挂载详情面板。
function LogCardBody({ log }: { log: RelayLogOverview }) {
    const t = useTranslations('log.card');
    const { isOpen } = useMorphingDialog();
    const [now, setNow] = useState(() => Date.now());
    const actualModel = log.target_model || log.model;
    const { Icon, className: iconClassName, color: brandColor } = getModelIcon(actualModel);
    const requestRunning = log.status === 'running' || log.status === 'committed';
    const requestFailed = log.status === 'failed' || log.status === 'canceled';
    const errorText = log.error ?? '';

    // 仅在请求进行中或弹窗打开时走秒级刷新, 避免已完成日志持续触发重渲染。
    useEffect(() => {
        if (!requestRunning && !isOpen) return;
        const timer = window.setInterval(() => setNow(Date.now()), 1000);
        return () => window.clearInterval(timer);
    }, [isOpen, requestRunning]);

    return (
        <>
            <MorphingDialogTrigger
                className={cn(
                    "rounded-3xl border bg-card w-full text-left",
                    requestFailed ? "border-destructive/40" : "border-border",
                )}
            >
                <div className={cn("p-4 grid grid-cols-[auto_1fr] gap-4", requestFailed ? "items-start" : "items-center")}>
                    <Icon aria-hidden="true" className={iconClassName} width={40} height={40} />
                    <div className="min-w-0 flex flex-col gap-3">
                        <div className="flex items-center gap-2 min-w-0 text-sm">
                            <span className="font-semibold text-card-foreground truncate" title={log.model}>
                                {log.model || t('unknownModel')}
                            </span>
                            {requestRunning
                                ? <Loader2 className="size-3.5 shrink-0 animate-spin text-muted-foreground/50" />
                                : <ArrowRight className="size-3.5 shrink-0 text-muted-foreground/50" />}
                            <Badge
                                variant="secondary"
                                className="shrink-0 text-xs px-1.5 py-0"
                                style={{ backgroundColor: `${brandColor}15`, color: brandColor }}
                            >
                                {log.target_channel || '-'}
                            </Badge>
                            <span className="text-muted-foreground truncate" title={actualModel}>
                                {actualModel}
                            </span>
                        </div>
                        <div className="grid grid-cols-12 gap-x-4 gap-y-2 text-xs tabular-nums text-muted-foreground md:grid-cols-7">
                            <LogMetrics log={log} now={now} brandColor={brandColor} variant="card" />
                        </div>
                        {requestFailed && errorText && (
                            <div className="p-2.5 rounded-xl bg-destructive/10 border border-destructive/20 overflow-hidden">
                                <p className="text-xs text-destructive line-clamp-2 whitespace-pre-line">{errorText}</p>
                            </div>
                        )}
                    </div>
                </div>
            </MorphingDialogTrigger>

            <MorphingDialogContainer>
                <LogDetail log={log} now={now} />
            </MorphingDialogContainer>
        </>
    );
}

// LogCard 展示一条日志概览, 并在弹窗打开时加载详情。
export const LogCard = memo(function LogCard({ log }: { log: RelayLogOverview }) {
    return (
        <MorphingDialog>
            <LogCardBody log={log} />
        </MorphingDialog>
    );
});
