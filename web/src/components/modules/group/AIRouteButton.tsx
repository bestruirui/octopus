'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { Bot } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useQueryClient } from '@tanstack/react-query';
import {
    isGenerateAIRouteTerminal,
    type AIRouteScope,
    useGenerateAIRoute,
    useGenerateAIRouteProgress,
} from '@/api/endpoints/group';
import { SettingKey, useSettingList } from '@/api/endpoints/setting';
import { toast } from '@/components/common/Toast';
import { Button, buttonVariants } from '@/components/ui/button';
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { cn } from '@/lib/utils';
import { AIRouteProgressDialog } from './AIRouteProgressDialog';

type AIRouteButtonProps = {
    variant?: 'ghost' | 'default';
    className?: string;
    scope?: AIRouteScope;
    groupId?: number;
    onSuccess?: () => void;
};

type StoredAIRouteTask = {
    id: string;
    scope: AIRouteScope;
    groupId?: number;
};

const AI_ROUTE_PROGRESS_STORAGE_KEY = 'octopus.ai-route-progress';

function readStoredAIRouteTask(): StoredAIRouteTask | null {
    if (typeof window === 'undefined') {
        return null;
    }

    const raw = window.sessionStorage.getItem(AI_ROUTE_PROGRESS_STORAGE_KEY);
    if (!raw) {
        return null;
    }

    try {
        const parsed = JSON.parse(raw) as StoredAIRouteTask;
        if (!parsed?.id || (parsed.scope !== 'group' && parsed.scope !== 'table')) {
            return null;
        }
        return parsed;
    } catch {
        return null;
    }
}

function writeStoredAIRouteTask(task: StoredAIRouteTask) {
    if (typeof window === 'undefined') {
        return;
    }

    window.sessionStorage.setItem(AI_ROUTE_PROGRESS_STORAGE_KEY, JSON.stringify(task));
}

function clearStoredAIRouteTask(id?: string) {
    if (typeof window === 'undefined') {
        return;
    }

    const current = readStoredAIRouteTask();
    if (!current) {
        return;
    }
    if (id && current.id !== id) {
        return;
    }

    window.sessionStorage.removeItem(AI_ROUTE_PROGRESS_STORAGE_KEY);
}

function matchesStoredAIRouteTask(task: StoredAIRouteTask | null, scope: AIRouteScope, groupId: number) {
    if (!task || task.scope !== scope) {
        return false;
    }
    if (scope === 'group') {
        return task.groupId === groupId && groupId > 0;
    }
    return true;
}

export function AIRouteButton({
    variant = 'ghost',
    className,
    scope = 'table',
    groupId,
    onSuccess,
}: AIRouteButtonProps) {
    const t = useTranslations('group');
    const queryClient = useQueryClient();
    const { data: settings } = useSettingList();
    const generateAIRoute = useGenerateAIRoute();
    const configuredGroupID = useMemo(() => {
        const raw = settings?.find((item) => item.key === SettingKey.AIRouteGroupID)?.value?.trim() ?? '0';
        const parsed = Number(raw);
        return Number.isFinite(parsed) ? parsed : 0;
    }, [settings]);

    const resolvedGroupID = groupId && groupId > 0 ? groupId : configuredGroupID;
    const isGroupScope = scope === 'group';
    const [confirmOpen, setConfirmOpen] = useState(false);
    const [progressOpen, setProgressOpen] = useState(false);
    const [currentProgressId, setCurrentProgressId] = useState<string | null>(() => {
        const storedTask = readStoredAIRouteTask();
        return matchesStoredAIRouteTask(storedTask, scope, resolvedGroupID) ? storedTask?.id ?? null : null;
    });
    const aiRouteProgress = useGenerateAIRouteProgress(currentProgressId);
    const handledProgressRef = useRef<string | null>(null);
    const loadingToastRef = useRef<string | number | null>(null);

    const actionLabel = isGroupScope ? t('actions.aiRouteGroup') : t('actions.aiRoute');
    const progressLabel = t('actions.aiRouteProgress');
    const progress = aiRouteProgress.data ?? null;
    const isRunning = Boolean(currentProgressId) && !isGenerateAIRouteTerminal(progress) && !(aiRouteProgress.error && !progress);

    useEffect(() => {
        if (!progress?.id || handledProgressRef.current === progress.id || !isGenerateAIRouteTerminal(progress)) {
            return;
        }

        handledProgressRef.current = progress.id;
        clearStoredAIRouteTask(progress.id);

        if (loadingToastRef.current !== null) {
            toast.dismiss(loadingToastRef.current);
            loadingToastRef.current = null;
        }

        if (progress.status === 'failed' || progress.status === 'timeout') {
            toast.error(t('toast.aiRouteFailed'), {
                description: progress.error_reason || progress.message,
            });
            return;
        }

        const result = progress.result_ready ? progress.result : undefined;
        if (progress.status !== 'completed' || !result) {
            toast.error(t('toast.aiRouteFailed'), { description: t('toast.aiRouteEmptyResult') });
            return;
        }

        queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
        if (isGroupScope) {
            toast.success(
                t('toast.aiRouteGroupSuccess', {
                    routes: result.route_count,
                    items: result.item_count,
                }),
            );
        } else {
            toast.success(
                t('toast.aiRouteTableSuccess', {
                    routes: result.route_count,
                    groups: result.group_count,
                    items: result.item_count,
                }),
            );
        }
        onSuccess?.();
    }, [isGroupScope, onSuccess, progress, queryClient, t]);

    useEffect(() => {
        if (!currentProgressId || !aiRouteProgress.error) {
            return;
        }
        if (progress) {
            return;
        }

        const error = aiRouteProgress.error;
        const description = error && typeof error === 'object' && 'message' in error && typeof error.message === 'string'
            ? error.message
            : undefined;
        const statusCode = error && typeof error === 'object' && 'code' in error && typeof error.code === 'number'
            ? error.code
            : undefined;

        if (statusCode === 404) {
            clearStoredAIRouteTask(currentProgressId);
            queueMicrotask(() => setCurrentProgressId(null));
            if (loadingToastRef.current !== null) {
                toast.dismiss(loadingToastRef.current);
                loadingToastRef.current = null;
            }
            toast.error(t('toast.aiRouteFailed'), { description });
            return;
        }

        if (loadingToastRef.current !== null) {
            toast.dismiss(loadingToastRef.current);
            loadingToastRef.current = null;
        }
    }, [aiRouteProgress.error, currentProgressId, progress, t]);

    const handleOpen = () => {
        if (isRunning) {
            setProgressOpen(true);
            return;
        }

        if (isGroupScope && resolvedGroupID <= 0) {
            toast.error(t('toast.aiRouteMissingGroup'));
            return;
        }
        setConfirmOpen(true);
    };

    const handleSubmit = () => {
        if (loadingToastRef.current !== null) {
            toast.dismiss(loadingToastRef.current);
        }
        loadingToastRef.current = toast.loading(
            isGroupScope ? t('aiRoute.group.submitting') : t('aiRoute.table.submitting'),
        );

        generateAIRoute.mutate(
            isGroupScope
                ? { scope: 'group', group_id: resolvedGroupID }
                : { scope: 'table' },
            {
                onSuccess: (nextProgress) => {
                    const nextProgressId = nextProgress.id;
                    handledProgressRef.current = null;
                    setConfirmOpen(false);
                    setProgressOpen(true);
                    setCurrentProgressId(nextProgressId);
                    writeStoredAIRouteTask({
                        id: nextProgressId,
                        scope,
                        groupId: isGroupScope ? resolvedGroupID : undefined,
                    });
                    queryClient.setQueryData(['groups', 'ai-route-progress', nextProgressId], nextProgress);
                    if (loadingToastRef.current !== null) {
                        toast.dismiss(loadingToastRef.current);
                        loadingToastRef.current = null;
                    }
                },
                onError: (error: Error) => {
                    if (loadingToastRef.current !== null) {
                        toast.dismiss(loadingToastRef.current);
                        loadingToastRef.current = null;
                    }
                    toast.error(t('toast.aiRouteFailed'), { description: error.message });
                },
            },
        );
    };

    const buttonClassName = cn(
        variant === 'default'
            ? 'rounded-xl'
            : buttonVariants({
                variant: 'ghost',
                size: 'default',
                className: 'rounded-xl transition-none hover:bg-transparent text-muted-foreground hover:text-foreground px-3',
            }),
        className,
    );

    const buttonText = isRunning ? progressLabel : actionLabel;

    return (
        <>
            {variant === 'default' ? (
                <Button
                    type="button"
                    className={buttonClassName}
                    onClick={handleOpen}
                    disabled={generateAIRoute.isPending}
                >
                    <Bot className="size-4" />
                    {buttonText}
                </Button>
            ) : (
                <button
                    type="button"
                    className={buttonClassName}
                    onClick={handleOpen}
                    disabled={generateAIRoute.isPending}
                >
                    <Bot className="size-4" />
                    <span>{buttonText}</span>
                </button>
            )}

            <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
                <AlertDialogContent className="rounded-2xl">
                    <AlertDialogHeader>
                        <AlertDialogTitle>
                            {isGroupScope ? t('aiRoute.group.confirmTitle') : t('aiRoute.table.confirmTitle')}
                        </AlertDialogTitle>
                        <AlertDialogDescription className="whitespace-pre-line">
                            {isGroupScope ? t('aiRoute.group.confirmDescription') : t('aiRoute.table.confirmDescription')}
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel disabled={generateAIRoute.isPending}>
                            {t('detail.actions.cancel')}
                        </AlertDialogCancel>
                        <AlertDialogAction
                            disabled={generateAIRoute.isPending}
                            onClick={(event) => {
                                event.preventDefault();
                                handleSubmit();
                            }}
                        >
                            {generateAIRoute.isPending
                                ? (isGroupScope ? t('aiRoute.group.submitting') : t('aiRoute.table.submitting'))
                                : (isGroupScope ? t('aiRoute.group.submit') : t('aiRoute.table.submit'))}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>

            <AIRouteProgressDialog
                open={progressOpen}
                onOpenChange={setProgressOpen}
                progress={progress}
                scope={scope}
            />
        </>
    );
}
