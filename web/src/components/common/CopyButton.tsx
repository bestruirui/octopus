'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { Check, Copy } from 'lucide-react';
import { useCopyToClipboard } from '@uidotdev/usehooks';
import { useTranslations } from 'next-intl';
import { cn } from '@/lib/utils';
import { toast } from '@/components/common/Toast';

export type CopyIconButtonProps = {
    text: string;
    className?: string;
    copyIconClassName?: string;
    checkIconClassName?: string;
};

export function CopyIconButton({
    text,
    className,
    copyIconClassName,
    checkIconClassName,
}: CopyIconButtonProps) {
    const t = useTranslations('common.copy');
    const [, copyToClipboard] = useCopyToClipboard();
    const [copied, setCopied] = useState(false);
    const timerRef = useRef<number | null>(null);

    useEffect(() => {
        return () => {
            if (timerRef.current) window.clearTimeout(timerRef.current);
        };
    }, []);

    const handleClick = useCallback(async () => {
        if (!text) {
            toast.error(t('failed'), { description: t('noContent') });
            return;
        }

        try {
            await copyToClipboard(text);

            setCopied(true);
            toast.success(t('success'));

            if (timerRef.current) window.clearTimeout(timerRef.current);
            timerRef.current = window.setTimeout(() => setCopied(false), 2000);
        } catch (err) {
            const description = err instanceof Error ? err.message : String(err);
            toast.error(t('failed'), { description });
        }
    }, [
        text,
        copyToClipboard,
        t,
    ]);

    return (
        <button
            type="button"
            onClick={handleClick}
            aria-label="Copy"
            className={cn(className)}
        >
            <AnimatePresence mode="wait" initial={false}>
                {copied ? (
                    <motion.div key="check" initial={{ scale: 0 }} animate={{ scale: 1 }} exit={{ scale: 0 }}>
                        <Check className={cn(checkIconClassName)} />
                    </motion.div>
                ) : (
                    <motion.div key="copy" initial={{ scale: 0 }} animate={{ scale: 1 }} exit={{ scale: 0 }}>
                        <Copy className={cn(copyIconClassName)} />
                    </motion.div>
                )}
            </AnimatePresence>
        </button>
    );
}


