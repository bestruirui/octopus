'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import { Bot } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useQueryClient } from '@tanstack/react-query';
import { type AIRouteScope, useGenerateAIRoute, useGenerateAIRouteProgress } from '@/api/endpoints/group';
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

type AIRouteButtonProps = {
    variant?: 'ghost' | 'default';
    className?: string;
    scope?: AIRouteScope;
    groupId?: number;
    onSuccess?: () => void;
};

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
    const [open, setOpen] = useState(false);
    const [currentProgressId, setCurrentProgressId] = useState<string | null>(null);
    const aiRouteProgress = useGenerateAIRouteProgress(currentProgressId);
    const handledProgressRef = useRef<string | null>(null);
    const loadingToastRef = useRef<string | number | null>(null);

    const configuredGroupID = useMemo(() => {
        const raw = settings?.find((item) => item.key === SettingKey.AIRouteGroupID)?.value?.trim() ?? '0';
        const parsed = Number(raw);
        return Number.isFinite(parsed) ? parsed : 0;
    }, [settings]);

    const resolvedGroupID = groupId && groupId > 0 ? groupId : configuredGroupID;
    const isGroupScope = scope === 'group';
    const actionLabel = isGroupScope ? t('actions.aiRouteGroup') : t('actions.aiRoute');
    const isRunning = Boolean(currentProgressId) && !aiRouteProgress.data?.done;

    useEffect(() => {
        const progress = aiRouteProgress.data;
        if (!progress?.done || !progress.id || handledProgressRef.current === progress.id) {
            return;
        }

        handledProgressRef.current = progress.id;
        setCurrentProgressId(null);

        if (loadingToastRef.current !== null) {
            toast.dismiss(loadingToastRef.current);
            loadingToastRef.current = null;
        }

        if (progress.message) {
            toast.error(t('toast.aiRouteFailed'), { description: progress.message });
            return;
        }

        const result = progress.result;
        if (!result) {
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
    }, [aiRouteProgress.data, isGroupScope, onSuccess, queryClient, t]);

    useEffect(() => {
        if (!currentProgressId || !aiRouteProgress.error) {
            return;
        }

        setCurrentProgressId(null);
        if (loadingToastRef.current !== null) {
            toast.dismiss(loadingToastRef.current);
            loadingToastRef.current = null;
        }

        const description = (() => {
            const error = aiRouteProgress.error;
            if (error && typeof error === 'object' && 'message' in error && typeof error.message === 'string') {
                return error.message;
            }
            return undefined;
        })();

        toast.error(t('toast.aiRouteFailed'), { description });
    }, [aiRouteProgress.error, currentProgressId, t]);

    const handleOpen = () => {
        if (isGroupScope && resolvedGroupID <= 0) {
            toast.error(t('toast.aiRouteMissingGroup'));
            return;
        }
        setOpen(true);
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
                onSuccess: (progress) => {
                    setOpen(false);
                    handledProgressRef.current = null;
                    setCurrentProgressId(progress.id);
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

    return (
        <>
            {variant === 'default' ? (
                <Button
                    type="button"
                    className={buttonClassName}
                    onClick={handleOpen}
                    disabled={generateAIRoute.isPending || isRunning}
                >
                    <Bot className="size-4" />
                    {actionLabel}
                </Button>
            ) : (
                <button
                    type="button"
                    className={buttonClassName}
                    onClick={handleOpen}
                    disabled={generateAIRoute.isPending || isRunning}
                >
                    <Bot className="size-4" />
                    <span>{actionLabel}</span>
                </button>
            )}

            <AlertDialog open={open} onOpenChange={setOpen}>
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
                        <AlertDialogCancel disabled={generateAIRoute.isPending || isRunning}>
                            {t('detail.actions.cancel')}
                        </AlertDialogCancel>
                        <AlertDialogAction
                            disabled={generateAIRoute.isPending || isRunning}
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
        </>
    );
}
