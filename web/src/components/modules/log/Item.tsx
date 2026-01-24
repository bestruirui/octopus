'use client';

import { useMemo, useState, useEffect } from 'react';
import { Clock, Cpu, Zap, AlertCircle, ArrowDownToLine, ArrowUpFromLine, DollarSign, ArrowRight, Send, MessageSquare, Loader2, RotateCw, ChevronDown, ChevronUp } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { motion, AnimatePresence } from 'motion/react';
import JsonView from '@uiw/react-json-view';
import { githubDarkTheme } from '@uiw/react-json-view/githubDark';
import { githubLightTheme } from '@uiw/react-json-view/githubLight';
import { useTheme } from 'next-themes';
import { type RelayLog, type ChannelAttempt } from '@/api/endpoints/log';
import { getModelIcon } from '@/lib/model-icons';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { CopyIconButton } from '@/components/common/CopyButton';
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
import { Tooltip, TooltipContent, TooltipTrigger, TooltipProvider } from '@/components/animate-ui/components/animate/tooltip';

function formatTime(timestamp: number): string {
    const date = new Date(timestamp * 1000);
    return date.toLocaleString('zh-CN', {
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
    });
}

function formatDuration(ms: number): string {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
}

interface RetryBadgeWithTooltipProps {
    channelName: string;
    brandColor: string;
    attempts: ChannelAttempt[];
}

function RetryBadgeWithTooltip({ channelName, brandColor, attempts }: RetryBadgeWithTooltipProps) {
    const t = useTranslations('log.card');

    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <Badge
                    variant="secondary"
                    className="shrink-0 text-xs px-1.5 py-0 cursor-help border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800/50 dark:bg-amber-900/30 dark:text-amber-300"
                    style={{ borderColor: `${brandColor}40` }}
                >
                    <RotateCw className="size-3 mr-1 opacity-80" />
                    {channelName}
                </Badge>
            </TooltipTrigger>
            <TooltipContent className="border bg-card p-0 min-w-[280px] shadow-sm">
                <div className="flex flex-col">
                    <div className="flex flex-col gap-1 border-b p-3 bg-muted/50">
                        <div className="flex items-center gap-2 text-xs font-bold tracking-wider uppercase text-foreground">
                            <RotateCw className="size-3.5" />
                            {t('retryProcess')}
                        </div>
                    </div>
                    <div className="flex flex-col gap-1 p-2">
                        {attempts.map((attempt, idx) => (
                            <div
                                key={idx}
                                className="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-muted/50 transition-colors"
                            >
                                <Badge
                                    className={cn(
                                        "h-5 shrink-0 px-1.5 text-[10px] font-bold uppercase",
                                        attempt.success
                                            ? "bg-green-100 text-green-800 border-green-200 dark:bg-green-900/20 dark:text-green-300 dark:border-green-800"
                                            : "bg-red-100 text-red-800 border-red-200 dark:bg-red-900/20 dark:text-red-300 dark:border-red-800"
                                    )}
                                >
                                    {attempt.success ? t('success') : t('failed')}
                                </Badge>
                                <div className="flex min-w-0 flex-col flex-1">
                                    <span className="truncate text-xs font-semibold text-foreground">
                                        {attempt.channel_name}
                                    </span>
                                    <span className="text-[10px] text-muted-foreground">
                                        {attempt.model_name} • {formatDuration(attempt.duration)}
                                    </span>
                                </div>
                            </div>
                        ))}
                    </div>
                </div>
            </TooltipContent>
        </Tooltip>
    );
}

function DeferredJsonContent({ content, fallbackText }: { content: string | undefined; fallbackText: string }) {
    const { resolvedTheme } = useTheme();
    const { isOpen } = useMorphingDialog();
    const [shouldRender, setShouldRender] = useState(false);

    const parsed = useMemo(() => {
        if (!content) return { isJson: false, data: null };
        try {
            return { isJson: true, data: JSON.parse(content) };
        } catch {
            return { isJson: false, data: content };
        }
    }, [content]);

    useEffect(() => {
        if (isOpen) {
            const timer = setTimeout(() => setShouldRender(true), 300);
            return () => clearTimeout(timer);
        }
    }, [isOpen]);

    if (!isOpen) {
        if (shouldRender) setShouldRender(false);
        return null;
    }

    if (!content) {
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
                    {content}
                </motion.pre>
            )}
        </AnimatePresence>
    );
}

export function LogCard({ log }: { log: RelayLog }) {
    const t = useTranslations('log.card');
    const { Avatar: ModelAvatar, color: brandColor } = useMemo(
        () => getModelIcon(log.actual_model_name),
        [log.actual_model_name]
    );

    const hasError = !!log.error;
    const hasMultipleAttempts = log.attempts && log.attempts.length > 1;
    const [isRetryDetailsExpanded, setIsRetryDetailsExpanded] = useState(false);

    return (
        <TooltipProvider>
            <MorphingDialog>
            <MorphingDialogTrigger
                className={cn(
                    "rounded-3xl border bg-card custom-shadow w-full text-left",
                    "hover:shadow-md transition-shadow duration-200",
                    hasError ? "border-destructive/40" : "border-border",
                )}
            >
                <div className="p-4 grid grid-cols-[auto_1fr] gap-4">
                    <ModelAvatar size={40} />
                    <div className="min-w-0 space-y-3">
                        <div className="flex items-center gap-2 min-w-0 text-sm">
                            <span className="font-semibold text-card-foreground truncate" title={log.request_model_name}>
                                {log.request_model_name}
                            </span>
                            <ArrowRight className="size-3.5 shrink-0 text-muted-foreground/50" />
                            {hasMultipleAttempts ? (
                                <RetryBadgeWithTooltip
                                    channelName={log.channel_name}
                                    brandColor={brandColor}
                                    attempts={log.attempts!}
                                />
                            ) : (
                                <Badge
                                    variant="secondary"
                                    className="shrink-0 text-xs px-1.5 py-0"
                                    style={{ backgroundColor: `${brandColor}15`, color: brandColor }}
                                >
                                    {log.channel_name}
                                </Badge>
                            )}
                            <span className="text-muted-foreground truncate" title={log.actual_model_name}>
                                {log.actual_model_name}
                            </span>
                        </div>
                        <div className="grid grid-cols-2 md:grid-cols-6 gap-x-4 gap-y-2 text-xs tabular-nums text-muted-foreground">
                            <div className="flex items-center gap-1.5">
                                <Clock className="size-3.5 shrink-0" style={{ color: brandColor }} />
                                <span>{formatTime(log.time)}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <Zap className="size-3.5 shrink-0 text-amber-500" />
                                <span>{t('firstToken')} {formatDuration(log.ftut)}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <Cpu className="size-3.5 shrink-0 text-blue-500" />
                                <span>{t('totalTime')} {formatDuration(log.use_time)}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <ArrowDownToLine className="size-3.5 shrink-0 text-green-500" />
                                <span>{t('input')} {log.input_tokens.toLocaleString()}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <ArrowUpFromLine className="size-3.5 shrink-0 text-purple-500" />
                                <span>{t('output')} {log.output_tokens.toLocaleString()}</span>
                            </div>
                            <div className="flex items-center gap-1.5">
                                <DollarSign className="size-3.5 shrink-0 text-emerald-500" />
                                <span className="font-medium text-emerald-600 dark:text-emerald-400">
                                    {t('cost')} {Number(log.cost).toFixed(6)}
                                </span>
                            </div>
                        </div>
                        {hasError && (
                            <div className="p-2.5 rounded-xl bg-destructive/10 border border-destructive/20 overflow-hidden">
                                <p className="text-xs text-destructive line-clamp-2">{log.error}</p>
                            </div>
                        )}
                    </div>
                </div>
            </MorphingDialogTrigger>

            <MorphingDialogContainer>
                <MorphingDialogContent className="relative w-[calc(100vw-2rem)] md:w-[80vw] bg-card text-card-foreground px-6 py-4 rounded-3xl custom-shadow h-[calc(100vh-2rem)] flex flex-col overflow-hidden">
                    <MorphingDialogClose className="top-4 right-5 text-muted-foreground hover:text-foreground transition-colors" />
                    <MorphingDialogTitle className="flex items-center gap-2 mb-3 text-sm">
                        <ModelAvatar size={28} />
                        <span className="font-semibold text-card-foreground">{log.request_model_name}</span>
                        <ArrowRight className="size-3.5 text-muted-foreground/50" />
                        {hasMultipleAttempts ? (
                            <RetryBadgeWithTooltip
                                channelName={log.channel_name}
                                brandColor={brandColor}
                                attempts={log.attempts!}
                            />
                        ) : (
                            <Badge
                                variant="secondary"
                                className="text-xs px-1.5 py-0"
                                style={{ backgroundColor: `${brandColor}15`, color: brandColor }}
                            >
                                {log.channel_name}
                            </Badge>
                        )}
                        <span className="text-muted-foreground">{log.actual_model_name}</span>
                    </MorphingDialogTitle>

                    <MorphingDialogDescription className="flex-1 min-h-0">
                        <div className="flex flex-col min-h-0 h-full gap-4">
                            {hasError && (
                                <div className="flex-initial max-h-[25%] min-h-0 p-2.5 md:p-3 rounded-xl bg-destructive/10 border border-destructive/20 overflow-auto">
                                    <div className="flex items-center gap-2 mb-1">
                                        <AlertCircle className="size-4 text-destructive shrink-0" />
                                        <span className="text-sm font-medium text-destructive">{t('errorInfo')}</span>
                                        <CopyIconButton
                                            text={log.error ?? ''}
                                            className="ml-auto p-1 rounded-md text-destructive/60 hover:text-destructive hover:bg-destructive/10 transition-colors"
                                            copyIconClassName="size-4"
                                            checkIconClassName="size-4"
                                        />
                                    </div>
                                    <p className="text-sm text-destructive whitespace-pre-wrap wrap-break-word">{log.error}</p>
                                </div>
                            )}

                            {/* 新增: 独立的重试信息区域 - 无论成功失败都显示 */}
                            {hasMultipleAttempts && (
                                <div className="flex-initial max-h-[30%] min-h-0 p-2.5 md:p-3 rounded-xl bg-amber-50/50 dark:bg-amber-900/10 border border-amber-200 dark:border-amber-800/30 overflow-auto">
                                    <div
                                        className="flex items-center gap-2 mb-2 cursor-pointer select-none hover:opacity-80 transition-opacity"
                                        onClick={() => setIsRetryDetailsExpanded(!isRetryDetailsExpanded)}
                                    >
                                        <RotateCw className="size-4 text-amber-600 dark:text-amber-400 shrink-0" />
                                        <span className="text-sm font-medium text-amber-700 dark:text-amber-300">{t('retryDetails')}</span>
                                        <Badge variant="secondary" className="text-xs bg-amber-100 text-amber-700 border-amber-200 dark:bg-amber-900/30 dark:text-amber-300 dark:border-amber-800">
                                            {log.total_attempts || log.attempts!.length} {t('attempts')}
                                        </Badge>
                                        {isRetryDetailsExpanded ? (
                                            <ChevronUp className="size-4 text-amber-600 dark:text-amber-400 ml-auto" />
                                        ) : (
                                            <ChevronDown className="size-4 text-amber-600 dark:text-amber-400 ml-auto" />
                                        )}
                                    </div>
                                    <AnimatePresence initial={false}>
                                        {isRetryDetailsExpanded && (
                                            <motion.div
                                                initial={{ height: 0, opacity: 0 }}
                                                animate={{ height: "auto", opacity: 1 }}
                                                exit={{ height: 0, opacity: 0 }}
                                                transition={{ duration: 0.2, ease: "easeInOut" }}
                                                className="overflow-hidden"
                                            >
                                                <div className="space-y-2">
                                                    {log.attempts!.map((attempt, idx) => (
                                                        <div
                                                            key={idx}
                                                            className={cn(
                                                                "text-xs p-2.5 rounded-lg border transition-colors",
                                                                attempt.success
                                                                    ? "bg-green-50/50 dark:bg-green-900/10 border-green-200 dark:border-green-800/30"
                                                                    : "bg-red-50/50 dark:bg-red-900/10 border-red-200 dark:border-red-800/30"
                                                            )}
                                                        >
                                                            <div className="flex items-center gap-2 mb-1.5">
                                                                <Badge
                                                                    className={cn(
                                                                        "h-5 shrink-0 px-1.5 text-[10px] font-bold uppercase",
                                                                        attempt.success
                                                                            ? "bg-green-100 text-green-800 border-green-200 dark:bg-green-900/30 dark:text-green-300 dark:border-green-800"
                                                                            : "bg-red-100 text-red-800 border-red-200 dark:bg-red-900/30 dark:text-red-300 dark:border-red-800"
                                                                    )}
                                                                >
                                                                    {attempt.success ? t('success') : t('failed')}
                                                                </Badge>
                                                                <span className="font-semibold text-foreground">
                                                                    {attempt.channel_name}
                                                                </span>
                                                                <span className="text-muted-foreground">
                                                                    ({attempt.model_name})
                                                                </span>
                                                                <span className="ml-auto text-muted-foreground tabular-nums">
                                                                    {formatDuration(attempt.duration)}
                                                                </span>
                                                            </div>
                                                            {attempt.error && (
                                                                <div className="text-red-700 dark:text-red-300 break-words mt-1 pl-2 border-l-2 border-red-300 dark:border-red-800">
                                                                    {attempt.error}
                                                                </div>
                                                            )}
                                                        </div>
                                                    ))}
                                                </div>
                                            </motion.div>
                                        )}
                                    </AnimatePresence>
                                </div>
                            )}
                            <div className="flex-1 min-h-0 overflow-hidden">
                                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 h-full min-h-0">
                                    <div className="flex flex-col rounded-2xl border border-border bg-muted/30 overflow-hidden min-h-0">
                                        <div className="flex items-center gap-2 px-3 md:px-4 py-2.5 md:py-3 border-b border-border bg-muted/50 shrink-0">
                                            <Send className="size-4 text-green-500" />
                                            <span className="text-sm font-medium text-card-foreground">{t('requestContent')}</span>
                                            <Badge variant="secondary" className="ml-auto text-xs">
                                                {log.input_tokens.toLocaleString()} {t('tokens')}
                                            </Badge>
                                        </div>
                                        <div className="flex-1 overflow-auto min-h-0">
                                            <DeferredJsonContent content={log.request_content} fallbackText={t('noRequestContent')} />
                                        </div>
                                    </div>
                                    <div className="flex flex-col rounded-2xl border border-border bg-muted/30 overflow-hidden min-h-0">
                                        <div className="flex items-center gap-2 px-3 md:px-4 py-2.5 md:py-3 border-b border-border bg-muted/50 shrink-0">
                                            <MessageSquare className="size-4 text-purple-500" />
                                            <span className="text-sm font-medium text-card-foreground">{t('responseContent')}</span>
                                            <Badge variant="secondary" className="ml-auto text-xs">
                                                {log.output_tokens.toLocaleString()} {t('tokens')}
                                            </Badge>
                                        </div>
                                        <div className="flex-1 overflow-auto min-h-0">
                                            <DeferredJsonContent content={log.response_content} fallbackText={t('noResponseContent')} />
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </MorphingDialogDescription>

                    <div className="flex flex-wrap items-center gap-3 md:gap-4 pt-4 mt-auto text-xs text-muted-foreground shrink-0">
                        <div className="flex items-center gap-1.5">
                            <Clock className="size-3.5" style={{ color: brandColor }} />
                            <span className="tabular-nums">{formatTime(log.time)}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                            <Zap className="size-3.5 text-amber-500" />
                            <span>{t('firstTokenTime')}: {formatDuration(log.ftut)}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                            <Cpu className="size-3.5 text-blue-500" />
                            <span>{t('totalTime')}: {formatDuration(log.use_time)}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                            <DollarSign className="size-3.5 text-emerald-500" />
                            <span className="font-medium text-emerald-600 dark:text-emerald-400">
                                {t('cost')}: {Number(log.cost).toFixed(6)}
                            </span>
                        </div>
                    </div>
                </MorphingDialogContent>
            </MorphingDialogContainer>
        </MorphingDialog>
        </TooltipProvider>
    );
}
