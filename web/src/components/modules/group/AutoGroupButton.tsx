'use client';

import { useMemo, useState } from 'react';
import { Sparkles, Waves } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { useAutoGroupModels } from '@/api/endpoints/group';
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
    AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import { cn } from '@/lib/utils';

type AutoGroupButtonProps = {
    variant?: 'ghost' | 'default';
    className?: string;
};

export function AutoGroupButton({ variant = 'ghost', className }: AutoGroupButtonProps) {
    const t = useTranslations('group');
    const autoGroup = useAutoGroupModels();
    const [open, setOpen] = useState(false);

    const summary = useMemo(() => {
        const result = autoGroup.data;
        if (!result) return '';
        return t('toast.autoGroupSuccess', {
            created: result.created_groups,
            skipped: result.skipped_existing_groups,
        });
    }, [autoGroup.data, t]);

    const details = useMemo(() => {
        const result = autoGroup.data;
        if (!result) return '';
        return t('toast.autoGroupSuccessDescription', {
            models: result.total_models_seen,
            candidates: result.total_candidates,
            created: result.created_groups,
            skippedExisting: result.skipped_existing_groups,
            skippedCovered: result.skipped_covered_models,
        });
    }, [autoGroup.data, t]);

    return (
        <AlertDialog open={open} onOpenChange={setOpen}>
            <AlertDialogTrigger asChild>
                {variant === 'default' ? (
                    <Button type="button" className={cn('rounded-[1.2rem] shadow-waterhouse-soft', className)}>
                        <Sparkles className="size-4" />
                        {t('actions.autoGroup')}
                    </Button>
                ) : (
                    <button
                        type="button"
                        className={cn(
                            buttonVariants({
                                variant: 'ghost',
                                size: 'default',
                                className: 'rounded-[1.2rem] border border-border/25 bg-background/32 px-3 text-muted-foreground shadow-waterhouse-soft transition-[transform,border-color,background-color,box-shadow] duration-300 hover:-translate-y-0.5 hover:bg-background/50 hover:text-foreground',
                            }),
                            className,
                        )}
                    >
                        <Sparkles className="size-4" />
                        <span>{t('actions.autoGroup')}</span>
                    </button>
                )}
            </AlertDialogTrigger>
            <AlertDialogContent className="rounded-2xl">
                <AlertDialogHeader>
                    <div className="inline-flex w-fit items-center gap-2 rounded-full border border-primary/15 bg-background/44 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-primary shadow-waterhouse-soft">
                        <Waves className="size-3.5" />
                        {t('actions.autoGroup')}
                    </div>
                    <AlertDialogTitle>{t('autoGroup.confirmTitle')}</AlertDialogTitle>
                    <AlertDialogDescription className="whitespace-pre-line">
                        {t('autoGroup.confirmDescription')}
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel disabled={autoGroup.isPending}>{t('detail.actions.cancel')}</AlertDialogCancel>
                    <AlertDialogAction
                        disabled={autoGroup.isPending}
                        onClick={(event) => {
                            event.preventDefault();
                            autoGroup.mutate(undefined, {
                                onSuccess: () => {
                                    setOpen(false);
                                    toast.success(summary, { description: details });
                                },
                                onError: (error) => {
                                    toast.error(t('toast.autoGroupFailed'), { description: error.message });
                                },
                            });
                        }}
                    >
                        {autoGroup.isPending ? t('autoGroup.submitting') : t('autoGroup.submit')}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    );
}
