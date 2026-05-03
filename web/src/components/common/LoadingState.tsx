'use client';

import { motion } from 'motion/react';
import { Loader2 } from 'lucide-react';
import { useTranslations } from 'next-intl';

interface LoadingStateProps {
    message?: string;
}

export function LoadingState({ message }: LoadingStateProps) {
    const t = useTranslations('common.loadingState');

    return (
        <div className="flex h-full min-h-[18rem] items-center justify-center">
            <motion.div
                initial={{ opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
                className="flex flex-col items-center gap-3"
            >
                <div className="grid size-14 place-items-center rounded-[1.45rem] border border-border/35 bg-background/56 shadow-waterhouse-soft">
                    <Loader2 className="size-6 animate-spin text-primary" />
                </div>
                <span className="text-sm text-muted-foreground">{message ?? t('message')}</span>
            </motion.div>
        </div>
    );
}
