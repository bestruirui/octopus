'use client';

import { useMemo, useState } from 'react';
import { Bot } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useGenerateAIRoute } from '@/api/endpoints/group';
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
};

export function AIRouteButton({ variant = 'ghost', className }: AIRouteButtonProps) {
    const t = useTranslations('group');
    const { data: settings } = useSettingList();
    const generateAIRoute = useGenerateAIRoute();
    const [open, setOpen] = useState(false);

    const groupID = useMemo(() => {
        const raw = settings?.find((item) => item.key === SettingKey.AIRouteGroupID)?.value?.trim() ?? '0';
        const parsed = Number(raw);
        return Number.isFinite(parsed) ? parsed : 0;
    }, [settings]);

    const handleOpen = () => {
        if (groupID <= 0) {
            toast.error(t('toast.aiRouteMissingGroup'));
            return;
        }
        setOpen(true);
    };

    const handleSubmit = () => {
        generateAIRoute.mutate(
            { group_id: groupID },
            {
                onSuccess: (result) => {
                    setOpen(false);
                    toast.success(
                        t('toast.aiRouteSuccess', {
                            routes: result.route_count,
                            items: result.item_count,
                        }),
                    );
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
                    {t('actions.aiRoute')}
                </Button>
            ) : (
                <button
                    type="button"
                    className={buttonClassName}
                    onClick={handleOpen}
                    disabled={generateAIRoute.isPending}
                >
                    <Bot className="size-4" />
                    <span>{t('actions.aiRoute')}</span>
                </button>
            )}

            <AlertDialog open={open} onOpenChange={setOpen}>
                <AlertDialogContent className="rounded-2xl">
                    <AlertDialogHeader>
                        <AlertDialogTitle>{t('aiRoute.confirmTitle')}</AlertDialogTitle>
                        <AlertDialogDescription className="whitespace-pre-line">
                            {t('aiRoute.confirmDescription')}
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
                            {generateAIRoute.isPending ? t('aiRoute.submitting') : t('aiRoute.submit')}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </>
    );
}
