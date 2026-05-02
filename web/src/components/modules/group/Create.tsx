'use client';

import type { GroupItem } from '@/api/endpoints/group';
import { Sparkles } from 'lucide-react';
import {
    MorphingDialogClose,
    MorphingDialogTitle,
    MorphingDialogDescription,
    useMorphingDialog,
} from '@/components/ui/morphing-dialog';
import { useCreateGroup } from '@/api/endpoints/group';
import { useTranslations } from 'next-intl';
import { GroupEditor } from './Editor';
import { toast } from '@/components/common/Toast';

function buildCreateItems(members: GroupItem[]) {
    const seen = new Set<string>();
    const items: GroupItem[] = [];

    for (const member of members) {
        const modelName = member.model_name.trim();
        const key = `${member.channel_id}-${modelName}`;
        if (!modelName || seen.has(key)) {
            continue;
        }
        seen.add(key);
        items.push({
            ...member,
            model_name: modelName,
            priority: items.length + 1,
            weight: member.weight > 0 ? member.weight : 1,
        });
    }

    return items;
}

export function CreateDialogContent() {
    const { setIsOpen } = useMorphingDialog();
    const createGroup = useCreateGroup();
    const t = useTranslations('group');

    return (
        <div className="relative flex h-full min-h-0 w-full max-w-full flex-col">
            <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_14%_18%,color-mix(in_oklch,var(--waterhouse-highlight)_18%,transparent)_0%,transparent_32%),linear-gradient(135deg,color-mix(in_oklch,white_14%,transparent),transparent_46%,color-mix(in_oklch,var(--primary)_10%,transparent))]" />
            <MorphingDialogTitle className="shrink-0">
                <header className="relative mb-5 flex items-start justify-between gap-4">
                    <div className="space-y-3">
                        <div className="inline-flex items-center gap-2 rounded-full border border-primary/15 bg-background/44 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-primary shadow-waterhouse-soft backdrop-blur-md">
                            <Sparkles className="size-3.5" />
                            {t('create.submit')}
                        </div>
                        <div className="space-y-1">
                            <h2 className="text-2xl font-bold text-card-foreground">
                                {t('create.title')}
                            </h2>
                            <p className="text-sm text-muted-foreground">{t('emptyState.description')}</p>
                        </div>
                    </div>
                    <MorphingDialogClose
                        className="relative right-0 top-0"
                        variants={{
                            initial: { opacity: 0, scale: 0.8 },
                            animate: { opacity: 1, scale: 1 },
                            exit: { opacity: 0, scale: 0.8 },
                        }}
                    />
                </header>
            </MorphingDialogTitle>
            <MorphingDialogDescription className="relative flex-1 min-h-0 overflow-hidden">
                <GroupEditor
                    submitText={t('create.submit')}
                    submittingText={t('create.submitting')}
                    isSubmitting={createGroup.isPending}
                    onSubmit={({ name, endpoint_type, match_regex, condition, mode, first_token_time_out, session_keep_time, members }) => {
                        const items = buildCreateItems(members.map((member) => ({
                            channel_id: member.channel_id,
                            model_name: member.name,
                            priority: 0,
                            weight: member.weight ?? 1,
                        })));

                        createGroup.mutate(
                            {
                                name,
                                endpoint_type: endpoint_type ?? '*',
                                mode,
                                match_regex: match_regex ?? '',
                                condition: condition.trim(),
                                first_token_time_out: first_token_time_out ?? 0,
                                session_keep_time: session_keep_time ?? 0,
                                items,
                            },
                            {
                                onSuccess: () => setIsOpen(false),
                                onError: (error) => toast.error(t('toast.createFailed'), { description: error.message }),
                            }
                        );
                    }}
                />
            </MorphingDialogDescription>
        </div>
    );
}
