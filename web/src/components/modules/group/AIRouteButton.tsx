'use client';

import { useMemo, useState } from 'react';
import { Bot } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { type AIRouteScope, useGenerateAIRoute } from '@/api/endpoints/group';
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
    const { data: settings } = useSettingList();
    const generateAIRoute = useGenerateAIRoute();
    const [open, setOpen] = useState(false);

    const configuredGroupID = useMemo(() => {
        const raw = settings?.find((item) => item.key === SettingKey.AIRouteGroupID)?.value?.trim() ?? '0';
        const parsed = Number(raw);
        return Number.isFinite(parsed) ? parsed : 0;
    }, [settings]);

    const resolvedGroupID = groupId && groupId > 0 ? groupId : configuredGroupID;
    const isGroupScope = scope === 'group';
    const actionLabel = isGroupScope ? t('actions.aiRouteGroup') : t('actions.aiRoute');

    const handleOpen = () => {
        if (isGroupScope && resolvedGroupID <= 0) {
            toast.error(t('toast.aiRouteMissingGroup'));
            return;
        }
        setOpen(true);
    };

    const handleSubmit = () => {
        generateAIRoute.mutate(
            isGroupScope
                ? { scope: 'group', group_id: resolvedGroupID }
                : { scope: 'table' },
            {
                onSuccess: (result) => {
                    setOpen(false);
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
                },
                onError: (error: Error) => {
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
                    disabled={generateAIRoute.isPending}
                >
                    <Bot className="size-4" />
                    {actionLabel}
                </Button>
            ) : (
                <button
                    type="button"
                    className={buttonClassName}
                    onClick={handleOpen}
                    disabled={generateAIRoute.isPending}
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
        </>
    );
}
