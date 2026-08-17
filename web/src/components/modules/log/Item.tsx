import { useEffect, useMemo, useState } from 'react';
import { AlertCircle, ArrowDownToLine, ArrowRight, ArrowUpFromLine, Circle, CircleCheck, Clock, Cpu, Database, DollarSign, Loader2, Square } from 'lucide-react';
import { useTranslations } from 'use-intl';
import { AnimatePresence, motion } from 'motion/react';
import JsonView from '@uiw/react-json-view';
import { githubDarkTheme } from '@uiw/react-json-view/githubDark';
import { githubLightTheme } from '@uiw/react-json-view/githubLight';
import { useTheme } from '@/provider/theme';
import { type RelayLogOverview, useLogDetailStream, useLogRequestBody, useLogResponseBody, useStopAttempt } from '@/api/log';
import { ApiError } from '@/api/client';
import { useGroupList, useUpdateGroupActiveItem } from '@/api/group';
import { useModelChannelList } from '@/api/model';
import { getModelIcon } from '@/lib/model-icons';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';
import { CopyIconButton } from '@/components/common/CopyButton';
import { toast } from 'sonner';
import { buildChannelNameByModelKey, modelChannelKey } from '@/components/modules/group/utils';
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

// formatTime 将后端 RFC3339 时间转换为本地时间。
function formatTime(value: string) {
    const date = new Date(value);
    if (Number.isNaN(date.getTime()) || date.getUTCFullYear() === 1) return '--';
    return date.toLocaleString(undefined, {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
    });
}

// formatMilliseconds 将毫秒转换为紧凑耗时文本。
function formatMilliseconds(value: number) {
    const milliseconds = Math.max(0, value);
    if (milliseconds < 1000) return `${Math.round(milliseconds)}ms`;
    return `${(milliseconds / 1000).toFixed(2)}s`;
}

// formatDuration 将 Go time.Duration 的纳秒值转换为显示文本。
function formatDuration(value: number) {
    return formatMilliseconds(value / 1_000_000);
}

// useOverviewDuration 返回运行中请求的实时耗时或完成请求的固定耗时。
function useOverviewDuration(log: RelayLogOverview) {
    const active = log.state === 'running' || log.state === 'committed';
    const [now, setNow] = useState(Date.now());

    useEffect(() => {
        if (!active) return;
        const timer = window.setInterval(() => setNow(Date.now()), 1000);
        return () => window.clearInterval(timer);
    }, [active]);

    if (!active) return formatDuration(log.duration);
    return formatMilliseconds(now - new Date(log.started_at).getTime());
}

// DeferredJsonContent 在详情弹窗稳定后渲染可能较大的 JSON 正文。
function DeferredJsonContent({ content, fallbackText }: { content: string | object | undefined; fallbackText: string }) {
    const { resolvedTheme } = useTheme();
    const { isOpen } = useMorphingDialog();
    const [shouldRender, setShouldRender] = useState(false);

    const parsed = useMemo(() => {
        if (content === undefined || content === '') return { isJson: false, data: null };
        if (typeof content !== 'string') return { isJson: true, data: content };
        try {
            return { isJson: true, data: JSON.parse(content) };
        } catch {
            return { isJson: false, data: content };
        }
    }, [content]);

    useEffect(() => {
        if (!isOpen) {
            setShouldRender(false);
            return;
        }
        const timer = window.setTimeout(() => setShouldRender(true), 300);
        return () => window.clearTimeout(timer);
    }, [isOpen]);

    if (!isOpen) return null;

    if (content === undefined || content === '') {
        return (
            <pre className="p-4 text-xs text-muted-foreground whitespace-pre-wrap wrap-break-word leading-relaxed">
                {fallbackText}
            </pre>
        );
    }

    return (
        <AnimatePresence mode="wait">
            {!shouldRender ? (
                <motion.div
                    key="loading"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: 0.15 }}
                    className="p-4 flex items-center justify-center h-full"
                >
                    <Loader2 className="h-5 w-5 text-muted-foreground animate-spin" />
                </motion.div>
            ) : parsed.isJson ? (
                <motion.div
                    key="json"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: 0.2 }}
                    className="p-4"
                >
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
                </motion.div>
            ) : (
                <motion.pre
                    key="text"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    exit={{ opacity: 0 }}
                    transition={{ duration: 0.2 }}
                    className="p-4 text-xs text-muted-foreground whitespace-pre-wrap wrap-break-word font-mono leading-relaxed"
                >
                    {parsed.data as string}
                </motion.pre>
            )}
        </AnimatePresence>
    );
}

// LogCardContent 使用新日志流为原卡片和弹窗布局提供数据。
function LogCardContent({ log }: { log: RelayLogOverview }) {
    const t = useTranslations('log.card');
    const statusT = useTranslations('log.status');
    const { isOpen } = useMorphingDialog();
    const [leftTab, setLeftTab] = useState<'request' | 'group'>('group');
    const { attempts, runningAttempt, isCommitted } = useLogDetailStream(log.id, log.state, isOpen);
    const requestBody = useLogRequestBody(log.id, log.started_at, isOpen && leftTab === 'request');
    const responseBody = useLogResponseBody(log.id, log.started_at, isOpen && log.state === 'success');
    const { data: groups = [] } = useGroupList(isOpen);
    const { data: modelChannels = [] } = useModelChannelList(isOpen);
    const updateActiveItem = useUpdateGroupActiveItem();
    const stopAttempt = useStopAttempt();
    const [switchingItemId, setSwitchingItemId] = useState<number | null>(null);
    const duration = useOverviewDuration(log);
    const actualModel = log.actual_model || runningAttempt?.model_name || attempts[attempts.length - 1]?.model_name || log.request_model;
    const activeState = log.state;
    // 请求结束前只展示当前状态，避免尝试阶段提前写入的渠道名称出现在概览中。
    const channelName = activeState === 'running' || activeState === 'committed'
        ? statusT('running')
        : log.final_channel_name || attempts[attempts.length - 1]?.channel_name || '-';
    const errorText = log.error ?? '';
    const requestFailed = activeState === 'failed' || activeState === 'canceled';
    const responseCommitted = isCommitted || activeState === 'committed';
    const isWaiting = activeState === 'running' && !responseCommitted;
    const activeGroup = groups.find((group) => group.name === log.request_model);
    const isWaitingForSelection = isWaiting && activeGroup?.active_item_id === 0; // isWaitingForSelection 表示请求正等待分组选择渠道。
    const channelNameByKey = useMemo(() => buildChannelNameByModelKey(modelChannels), [modelChannels]);
    const { Icon, className: iconClassName, color: brandColor } = useMemo(
        () => getModelIcon(actualModel),
        [actualModel]
    );

    useEffect(() => {
        if (!isOpen) setLeftTab('group');
    }, [isOpen]);

    return (
        <>
            <MorphingDialogTrigger
                className={cn(
                    "w-full overflow-hidden rounded-3xl border bg-card text-left",
                    requestFailed ? "border-destructive/40" : "border-border",
                )}
            >
                <div className={cn("p-4 grid grid-cols-[auto_1fr] gap-4", requestFailed ? "items-start" : "items-center")}>
                    <Icon aria-hidden="true" className={iconClassName} width={40} height={40} />
                    <div className="min-w-0 flex flex-col gap-3">
                        <div className="flex min-w-0 flex-col gap-1 text-sm md:flex-row md:items-center md:gap-2">
                            <div className="flex min-w-0 items-center gap-2 md:contents">
                                <span className="min-w-0 flex-1 truncate font-semibold text-card-foreground" title={log.request_model}>
                                    {log.request_model || t('unknownModel')}
                                </span>
                                <ArrowRight className="size-3.5 shrink-0 text-muted-foreground/50" />
                            </div>
                            <div className="flex min-w-0 items-center gap-2 md:contents">
                                <Badge
                                    variant="secondary"
                                    className="max-w-[45%] min-w-0 shrink-0 px-1.5 py-0 text-xs md:max-w-[30%]"
                                    style={{ backgroundColor: `${brandColor}15`, color: brandColor }}
                                >
                                    <span className="truncate">{channelName}</span>
                                </Badge>
                                <span className="min-w-0 flex-1 truncate text-muted-foreground" title={actualModel}>
                                    {actualModel}
                                </span>
                            </div>
                        </div>
                        <div className="grid grid-cols-2 md:grid-cols-7 gap-x-4 gap-y-2 text-xs tabular-nums text-muted-foreground">
                            <div className="flex items-center gap-1.5">
                                <Clock className="size-3.5 shrink-0" style={{ color: brandColor }} />
                                <span>{formatTime(log.started_at)}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <Cpu className="size-3.5 shrink-0 text-blue-500" />
                                <span>{duration}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <ArrowDownToLine className="size-3.5 shrink-0 text-green-500" />
                                <span>{(log.input_tokens - log.cache_read_tokens).toLocaleString()}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <Database className="size-3.5 shrink-0 text-cyan-500" />
                                <span>{log.cache_read_tokens.toLocaleString()}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <ArrowUpFromLine className="size-3.5 shrink-0 text-purple-500" />
                                <span>{log.output_tokens.toLocaleString()}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <Database className="size-3.5 shrink-0 text-orange-500" />
                                <span>{log.cache_write_tokens.toLocaleString()}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <DollarSign className="size-3.5 shrink-0 text-emerald-500" />
                                <span className="font-medium text-emerald-600 dark:text-emerald-400">
                                    {log.total_cost.toFixed(6)}
                                </span>
                            </div>
                        </div>
                        {requestFailed && errorText && (
                            <div className="p-2.5 rounded-xl bg-destructive/10 border border-destructive/20 overflow-hidden">
                                <p className="line-clamp-2 break-words whitespace-pre-line text-xs text-destructive">{errorText}</p>
                            </div>
                        )}
                    </div>
                </div>
            </MorphingDialogTrigger>

            <MorphingDialogContainer>
                <MorphingDialogContent className="relative w-[calc(100vw-2rem)] md:w-[80vw] bg-card text-card-foreground px-6 py-4 rounded-3xl h-[calc(100vh-2rem)] flex flex-col overflow-hidden">
                    <MorphingDialogClose className="top-4 right-5 text-muted-foreground hover:text-foreground transition-colors" />
                    <MorphingDialogTitle className="flex items-center gap-2 mb-3 text-sm">
                        <Icon aria-hidden="true" className={iconClassName} width={28} height={28} />
                        <span className="font-semibold text-card-foreground">{log.request_model || t('unknownModel')}</span>
                        <ArrowRight className="size-3.5 text-muted-foreground/50" />
                        <Badge
                            variant="secondary"
                            className="text-xs px-1.5 py-0"
                            style={{ backgroundColor: `${brandColor}15`, color: brandColor }}
                        >
                            {channelName}
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
                                                {(log.input_tokens - log.cache_read_tokens).toLocaleString()} {t('tokens')}
                                            </Badge>
                                        )}
                                    </div>
                                    <div className="flex-1 overflow-auto min-h-0">
                                        {leftTab === 'request' ? (
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
                                                <DeferredJsonContent content={requestBody.data} fallbackText={t('noRequestContent')} />
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
                                                {[...activeGroup.items].sort((a, b) => a.priority - b.priority).map((item) => {
                                                    const itemChannelName = channelNameByKey.get(modelChannelKey(item.channel_id, item.model_name)) ?? `#${item.channel_id}`;
                                                    const { Icon: ItemIcon, className: itemIconClassName } = getModelIcon(item.model_name);
                                                    const itemActive = item.id === activeGroup.active_item_id;
                                                    const itemSwitching = item.id === switchingItemId;
                                                    return (
                                                        <button
                                                            key={item.id ?? modelChannelKey(item.channel_id, item.model_name)}
                                                            type="button"
                                                            aria-pressed={itemActive}
                                                            disabled={item.id === undefined || switchingItemId !== null || stopAttempt.isPending}
                                                            onClick={async () => {
                                                                if (!activeGroup.id || item.id === undefined) return;
                                                                setSwitchingItemId(item.id);
                                                                try {
                                                                    await updateActiveItem.mutateAsync({ groupId: activeGroup.id, itemId: itemActive ? 0 : item.id });
                                                                    if (runningAttempt) {
                                                                        try {
                                                                            await stopAttempt.mutateAsync({ requestId: log.id, attemptIndex: runningAttempt.attempt_index });
                                                                        } catch (cause) {
                                                                            if (!(cause instanceof ApiError && cause.status === 409)) throw cause;
                                                                        }
                                                                    }
                                                                    toast.success(itemActive ? t('channelCleared') : t('channelChanged'));
                                                                } catch (cause) {
                                                                    toast.error(t('channelChangeFailed'), { description: cause instanceof Error ? cause.message : undefined });
                                                                } finally {
                                                                    setSwitchingItemId(null);
                                                                }
                                                            }}
                                                            className={cn(
                                                                'flex w-full items-center gap-2.5 px-3 py-2.5 text-left text-xs transition-colors disabled:cursor-default',
                                                                itemActive ? 'bg-primary/5' : 'hover:bg-muted/50 disabled:hover:bg-transparent'
                                                            )}
                                                        >
                                                            <ItemIcon aria-hidden="true" className={itemIconClassName} width={20} height={20} />
                                                            <span className="min-w-0 flex-1">
                                                                <span className="block truncate font-semibold text-foreground">{itemChannelName}</span>
                                                                <span className="block truncate text-[11px] text-muted-foreground">{item.model_name}</span>
                                                            </span>
                                                            {itemSwitching ? (
                                                                <Loader2 className="size-4 shrink-0 animate-spin text-muted-foreground" />
                                                            ) : itemActive ? (
                                                                <CircleCheck className="size-4 shrink-0 text-primary" />
                                                            ) : (
                                                                <Circle className="size-4 shrink-0 text-muted-foreground" />
                                                            )}
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
                                            {isWaitingForSelection ? t('waitingChannelSelection') : isWaiting ? t('retryDetails') : requestFailed ? t('errorInfo') : t('responseContent')}
                                        </span>
                                        {isWaiting && runningAttempt ? (
                                            <button
                                                type="button"
                                                disabled={stopAttempt.isPending}
                                                onClick={async () => {
                                                    try {
                                                        await stopAttempt.mutateAsync({ requestId: log.id, attemptIndex: runningAttempt.attempt_index });
                                                    } catch (cause) {
                                                        if (!(cause instanceof ApiError && cause.status === 409)) {
                                                            toast.error(t('stopFailed'), { description: cause instanceof Error ? cause.message : undefined });
                                                        }
                                                    }
                                                }}
                                                className="ml-auto flex items-center gap-1.5 rounded-md px-2 py-1 text-xs text-destructive transition-colors hover:bg-destructive/10 disabled:opacity-50"
                                            >
                                                {stopAttempt.isPending ? <Loader2 className="size-3.5 animate-spin" /> : <Square className="size-3.5" />}
                                                {t('stopAttempt')}
                                            </button>
                                        ) : requestFailed ? (
                                            errorText && (
                                                <CopyIconButton
                                                    text={errorText}
                                                    className="ml-auto p-1 rounded-md text-destructive/60 hover:text-destructive hover:bg-destructive/10 transition-colors"
                                                    copyIconClassName="size-4"
                                                    checkIconClassName="size-4"
                                                />
                                            )
                                        ) : responseCommitted ? (
                                            <Badge variant="secondary" className="ml-auto text-xs">{statusT('committed')}</Badge>
                                        ) : (
                                            <Badge variant="secondary" className="ml-auto text-xs">
                                                {log.output_tokens.toLocaleString()} {t('tokens')}
                                            </Badge>
                                        )}
                                    </div>
                                    <div className="min-h-0 flex-1 overflow-auto">
                                        {isWaitingForSelection ? (
                                            <div className="flex h-full items-center justify-center gap-2 text-xs text-muted-foreground">
                                                <Loader2 className="size-4 animate-spin" />
                                                {t('waitingChannelSelection')}
                                            </div>
                                        ) : isWaiting ? (
                                            attempts.length ? (
                                                <div className="divide-y divide-border">
                                                    {attempts.slice().reverse().map((attempt) => (
                                                        <div key={attempt.attempt_index} className="flex flex-col gap-1.5 px-3 py-2.5 text-xs">
                                                            <div className="flex items-center gap-2">
                                                                <span className="text-muted-foreground">{t('retryIndex', { index: attempt.attempt_index })}</span>
                                                                <span className="font-semibold text-foreground">{attempt.channel_name}</span>
                                                                {runningAttempt?.attempt_index === attempt.attempt_index && <Loader2 className="ml-auto size-3.5 animate-spin text-muted-foreground" />}
                                                            </div>
                                                            {attempt.error && (
                                                                <div className="text-[11px] leading-relaxed text-destructive/90 whitespace-pre-wrap wrap-break-word">
                                                                    {attempt.error}
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
                                            <DeferredJsonContent content={errorText} fallbackText={t('noResponseContent')} />
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
                                            <DeferredJsonContent content={responseBody.data} fallbackText={t('noResponseContent')} />
                                        )}
                                    </div>
                                </div>
                            </div>
                    </MorphingDialogDescription>

                    <div className="flex flex-wrap items-center gap-3 md:gap-4 pt-4 mt-auto text-xs text-muted-foreground shrink-0">
                        <div className="flex items-center gap-1.5">
                            <Clock className="size-3.5" style={{ color: brandColor }} />
                            <span className="tabular-nums">{formatTime(log.started_at)}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                            <Cpu className="size-3.5 text-blue-500" />
                            <span>{activeState !== 'running' && activeState !== 'committed' ? formatDuration(log.duration) : duration}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                            <ArrowDownToLine className="size-3.5 text-green-500" />
                            <span>{(log.input_tokens - log.cache_read_tokens).toLocaleString()}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                            <Database className="size-3.5 text-cyan-500" />
                            <span>{log.cache_read_tokens.toLocaleString()}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                            <ArrowUpFromLine className="size-3.5 text-purple-500" />
                            <span>{log.output_tokens.toLocaleString()}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                            <Database className="size-3.5 text-orange-500" />
                            <span>{log.cache_write_tokens.toLocaleString()}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                            <DollarSign className="size-3.5 text-emerald-500" />
                            <span className="font-medium text-emerald-600 dark:text-emerald-400">
                                {log.total_cost.toFixed(6)}
                            </span>
                        </div>
                    </div>
                </MorphingDialogContent>
            </MorphingDialogContainer>
        </>
    );
}

// LogCard 展示一条概览，并在弹窗打开时建立唯一详情连接。
export function LogCard({ log }: { log: RelayLogOverview }) {
    return (
        <MorphingDialog>
            <LogCardContent log={log} />
        </MorphingDialog>
    );
}
