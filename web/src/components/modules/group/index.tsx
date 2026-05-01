'use client';

import { useMemo } from 'react';
import { useTranslations } from 'next-intl';
import { GroupCard } from './Card';
import { AutoGroupButton } from './AutoGroupButton';
import { AIRouteButton } from './AIRouteButton';
import { useGroupList } from '@/api/endpoints/group';
import { normalizeGroupFilterValue, useSearchStore, useToolbarViewOptionsStore } from '@/components/modules/toolbar';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';
import {
    MorphingDialog,
    MorphingDialogContainer,
    MorphingDialogContent,
    MorphingDialogTrigger,
} from '@/components/ui/morphing-dialog';
import { matchesGroupEndpointFilter } from './utils';
import { CreateDialogContent } from './Create';
import { buttonVariants } from '@/components/ui/button';

export function Group() {
    const t = useTranslations('group');
    const { data: groups } = useGroupList();
    const pageKey = 'group' as const;
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const sortField = useToolbarViewOptionsStore((s) => s.getSortField(pageKey));
    const sortOrder = useToolbarViewOptionsStore((s) => s.getSortOrder(pageKey));
    const filter = useToolbarViewOptionsStore((s) => normalizeGroupFilterValue(s.groupFilter));

    const sortedGroups = useMemo(() => {
        if (!groups) return [];
        return [...groups].sort((a, b) => {
            const diff = sortField === 'name'
                ? a.name.localeCompare(b.name)
                : (a.id || 0) - (b.id || 0);
            return sortOrder === 'asc' ? diff : -diff;
        });
    }, [groups, sortField, sortOrder]);

    const visibleGroups = useMemo(() => {
        const term = searchTerm.toLowerCase().trim();
        const result = !term ? sortedGroups : sortedGroups.filter((g) => g.name.toLowerCase().includes(term));

        if (filter === 'with-members') return result.filter((g) => (g.items?.length || 0) > 0);
        if (filter === 'empty') return result.filter((g) => (g.items?.length || 0) === 0);
        if (filter !== 'all') {
            return result.filter((g) =>
                matchesGroupEndpointFilter(filter, g.endpoint_type, (g.items || []).map((item) => item.model_name)),
            );
        }

        return result;
    }, [sortedGroups, searchTerm, filter]);

    if (groups && groups.length === 0) {
        return (
            <div className="flex h-full items-center justify-center px-4">
                <div className="w-full max-w-xl rounded-3xl border border-border/60 bg-card/70 p-8 text-center shadow-sm">
                    <h2 className="text-2xl font-semibold text-foreground">{t('emptyState.title')}</h2>
                    <p className="mt-3 text-sm text-muted-foreground">{t('emptyState.description')}</p>
                    <div className="mt-6 flex flex-col justify-center gap-3 sm:flex-row">
                        <AutoGroupButton variant="default" className="min-w-36" />
                        <AIRouteButton variant="default" className="min-w-36" />
                        <MorphingDialog>
                            <MorphingDialogTrigger className={buttonVariants({ variant: 'outline', className: 'rounded-xl min-w-36' })}>
                                {t('create.submit')}
                            </MorphingDialogTrigger>
                            <MorphingDialogContainer>
                                <MorphingDialogContent className="w-[min(100vw-1rem,56rem)] max-w-full bg-card text-card-foreground px-4 py-4 rounded-3xl custom-shadow h-[min(44rem,calc(100dvh-1rem))] flex flex-col overflow-hidden md:px-6 md:h-[min(44rem,calc(100dvh-2rem))]">
                                    <CreateDialogContent />
                                </MorphingDialogContent>
                            </MorphingDialogContainer>
                        </MorphingDialog>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <VirtualizedGrid
            items={visibleGroups}
            columns={{ default: 1, md: 2, lg: 3 }}
            estimateItemHeight={520}
            getItemKey={(group, index) => group.id ?? `group-${index}`}
            renderItem={(group) => <GroupCard group={group} />}
        />
    );
}
