'use client';

import { useMemo, useState } from 'react';
import { Sparkles } from 'lucide-react';
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
                    <Button type="button" className={cn('rounded-xl', className)}>
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
                                className: 'rounded-xl transition-none hover:bg-transparent text-muted-foreground hover:text-foreground px-3',
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
