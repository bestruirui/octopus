'use client';

import { Check, Loader, Trash2, X } from 'lucide-react';
import { motion } from 'motion/react';
import { useTranslations } from 'next-intl';
import { Input } from '@/components/ui/input';

type EditValues = {
    input: string;
    output: string;
    cache_read: string;
    cache_write: string;
};

type ModelDeleteOverlayProps = {
    layoutId: string;
    isPending: boolean;
    onCancel: () => void;
    onConfirm: () => void;
};

export function ModelDeleteOverlay({
    layoutId,
    isPending,
    onCancel,
    onConfirm,
}: ModelDeleteOverlayProps) {
    const t = useTranslations('model.overlay');
    return (
        <motion.div
            layoutId={layoutId}
            className="absolute inset-0 flex items-center justify-center gap-3 bg-destructive p-4 rounded-2xl"
            transition={{ type: 'spring', stiffness: 400, damping: 30 }}
        >
            <button
                type="button"
                onClick={onCancel}
                className="h-9 px-4 flex items-center justify-center gap-1.5 rounded-xl bg-destructive-foreground/20 text-destructive-foreground text-sm font-medium transition-all hover:bg-destructive-foreground/30 active:scale-[0.98]"
            >
                <X className="size-4" />
                {t('cancel')}
            </button>
            <button
                type="button"
                onClick={onConfirm}
                disabled={isPending}
                className="h-9 px-4 flex items-center justify-center gap-1.5 rounded-xl bg-destructive-foreground text-destructive text-sm font-medium transition-all hover:bg-destructive-foreground/90 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed"
            >
                {isPending ? (
                    <Loader className="size-4 animate-spin" />
                ) : (
                    <Trash2 className="size-4" />
                )}
                {isPending ? t('deleting') : t('confirmDelete')}
            </button>
        </motion.div>
    );
}

type ModelEditOverlayProps = {
    layoutId: string;
    modelName: string;
    brandColor: string;
    editValues: EditValues;
    isPending: boolean;
    onChange: (next: EditValues) => void;
    onCancel: () => void;
    onSave: () => void;
};

export function ModelEditOverlay({
    layoutId,
    modelName,
    brandColor,
    editValues,
    isPending,
    onChange,
    onCancel,
    onSave,
}: ModelEditOverlayProps) {
    const t = useTranslations('model.overlay');
    return (
        <motion.div
            layoutId={layoutId}
            className="absolute inset-x-0 top-0 z-20 flex flex-col bg-card p-5 rounded-3xl border border-border custom-shadow"
            transition={{ type: 'spring', stiffness: 400, damping: 30 }}
        >
            <h3 className="text-sm font-semibold text-card-foreground line-clamp-1 mb-3">
                {modelName}
            </h3>

            <div className="grid grid-cols-2 gap-2">
                <label className="grid gap-1 text-xs text-muted-foreground">
                    {t('input')}
                    <Input
                        type="number"
                        step="any"
                        value={editValues.input}
                        onChange={(e) => onChange({ ...editValues, input: e.target.value })}
                        className="h-9 text-sm rounded-xl"
                    />
                </label>
                <label className="grid gap-1 text-xs text-muted-foreground">
                    {t('output')}
                    <Input
                        type="number"
                        step="any"
                        value={editValues.output}
                        onChange={(e) => onChange({ ...editValues, output: e.target.value })}
                        className="h-9 text-sm rounded-xl"
                    />
                </label>
                <label className="grid gap-1 text-xs text-muted-foreground">
                    {t('cacheRead')}
                    <Input
                        type="number"
                        step="any"
                        value={editValues.cache_read}
                        onChange={(e) => onChange({ ...editValues, cache_read: e.target.value })}
                        className="h-9 text-sm rounded-xl"
                    />
                </label>
                <label className="grid gap-1 text-xs text-muted-foreground">
                    {t('cacheWrite')}
                    <Input
                        type="number"
                        step="any"
                        value={editValues.cache_write}
                        onChange={(e) => onChange({ ...editValues, cache_write: e.target.value })}
                        className="h-9 text-sm rounded-xl"
                    />
                </label>
            </div>

            <div className="flex gap-2 pt-2 mt-3">
                <button
                    type="button"
                    onClick={onCancel}
                    disabled={isPending}
                    className="flex-1 h-9 flex items-center justify-center gap-1.5 rounded-xl bg-muted text-muted-foreground text-sm font-medium transition-all hover:bg-muted/80 active:scale-[0.98] disabled:opacity-50"
                >
                    <X className="size-4" />
                    {t('cancel')}
                </button>
                <button
                    type="button"
                    onClick={onSave}
                    disabled={isPending}
                    className="flex-1 h-9 flex items-center justify-center gap-1.5 rounded-xl text-sm font-medium transition-all active:scale-[0.98] disabled:opacity-50"
                    style={{ backgroundColor: brandColor, color: '#fff' }}
                >
                    {isPending ? <Loader className="size-4 animate-spin" /> : <Check className="size-4" />}
                    {t('save')}
                </button>
            </div>
        </motion.div>
    );
}