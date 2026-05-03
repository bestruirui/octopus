'use client';

import { motion } from 'motion/react';
import { AlertTriangle, RotateCcw } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';

interface ErrorStateProps {
    title?: string;
    message?: string;
    onRetry?: () => void;
}

export function ErrorState({
    title,
    message,
    onRetry,
}: ErrorStateProps) {
    const t = useTranslations('common.errorState');

    return (
        <div className="flex h-full min-h-[18rem] items-center justify-center">
            <motion.div
                initial={{ opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
                className="flex flex-col items-center gap-4 text-center"
            >
                <div className="grid size-14 place-items-center rounded-[1.45rem] border border-destructive/20 bg-destructive/10 shadow-waterhouse-soft">
                    <AlertTriangle className="size-6 text-destructive" />
                </div>
                <div className="space-y-1">
                    <p className="text-base font-medium">{title ?? t('title')}</p>
                    <p className="max-w-xs text-sm text-muted-foreground">{message ?? t('message')}</p>
                </div>
                {onRetry && (
                    <Button
                        onClick={onRetry}
                        variant="outline"
                        className="gap-2 rounded-xl"
                    >
                        <RotateCcw className="size-4" />
                        {t('retry')}
                    </Button>
                )}
            </motion.div>
        </div>
    );
}
