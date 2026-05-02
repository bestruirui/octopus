'use client';

import { useMemo } from 'react';
import { Bot, Sparkles, Waves, PlusCircle } from 'lucide-react';
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
            <div className="flex h-full items-center justify-center px-4 py-6">
                <section className="waterhouse-island relative w-full max-w-5xl overflow-hidden rounded-[2.35rem] border border-border/35 bg-card/62 p-5 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:p-7">
                    <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_14%_18%,color-mix(in_oklch,var(--waterhouse-highlight)_28%,transparent)_0%,transparent_32%),radial-gradient(circle_at_85%_16%,color-mix(in_oklch,var(--primary)_18%,transparent)_0%,transparent_28%),linear-gradient(135deg,color-mix(in_oklch,white_18%,transparent),transparent_44%,color-mix(in_oklch,var(--waterhouse-highlight)_10%,transparent))]" />
                    <div className="pointer-events-none absolute -bottom-20 left-1/2 h-48 w-[72%] -translate-x-1/2 rounded-[999px] border border-white/20 bg-[radial-gradient(circle,color-mix(in_oklch,var(--waterhouse-highlight)_18%,transparent),transparent_68%)]" />

                    <div className="relative grid gap-4 xl:grid-cols-[minmax(0,1.45fr)_minmax(20rem,0.95fr)]">
                        <div className="space-y-4">
                            <div className="inline-flex items-center gap-2 rounded-full border border-primary/15 bg-background/44 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.22em] text-primary shadow-waterhouse-soft backdrop-blur-md">
                                <Waves className="size-3.5" />
                                {t('actions.aiRoute')}
                            </div>

                            <div className="waterhouse-pod relative overflow-hidden rounded-[1.95rem] border border-border/30 bg-background/42 p-5 shadow-waterhouse-soft backdrop-blur-md md:p-6">
                                <div className="pointer-events-none absolute inset-y-0 right-0 w-40 bg-[radial-gradient(circle_at_center,color-mix(in_oklch,var(--primary)_12%,transparent)_0%,transparent_68%)]" />
                                <div className="relative space-y-3">
                                    <h2 className="max-w-xl text-3xl font-semibold tracking-tight text-foreground md:text-[2.2rem]">
                                        {t('emptyState.title')}
                                    </h2>
                                    <p className="max-w-2xl text-sm leading-6 text-muted-foreground md:text-[0.95rem]">
                                        {t('emptyState.description')}
                                    </p>
                                    <div className="grid gap-3 pt-2 sm:grid-cols-3">
                                        <div className="waterhouse-pod flex items-center gap-3 rounded-[1.45rem] border border-border/30 bg-background/38 px-4 py-3 shadow-waterhouse-soft">
                                            <span className="grid size-10 shrink-0 place-items-center rounded-[1.1rem] bg-primary/12 text-primary shadow-waterhouse-soft">
                                                <Sparkles className="size-4" />
                                            </span>
                                            <span className="text-sm font-medium text-foreground">{t('actions.autoGroup')}</span>
                                        </div>
                                        <div className="waterhouse-pod flex items-center gap-3 rounded-[1.45rem] border border-border/30 bg-background/38 px-4 py-3 shadow-waterhouse-soft">
                                            <span className="grid size-10 shrink-0 place-items-center rounded-[1.1rem] bg-primary/12 text-primary shadow-waterhouse-soft">
                                                <Bot className="size-4" />
                                            </span>
                                            <span className="text-sm font-medium text-foreground">{t('actions.aiRoute')}</span>
                                        </div>
                                        <div className="waterhouse-pod flex items-center gap-3 rounded-[1.45rem] border border-border/30 bg-background/38 px-4 py-3 shadow-waterhouse-soft">
                                            <span className="grid size-10 shrink-0 place-items-center rounded-[1.1rem] bg-primary/12 text-primary shadow-waterhouse-soft">
                                                <PlusCircle className="size-4" />
                                            </span>
                                            <span className="text-sm font-medium text-foreground">{t('create.submit')}</span>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div className="waterhouse-pod relative flex flex-col justify-between gap-4 overflow-hidden rounded-[2rem] border border-border/30 bg-background/38 p-5 shadow-waterhouse-soft backdrop-blur-md">
                            <div className="pointer-events-none absolute inset-x-8 top-0 h-24 rounded-b-[999px] bg-[radial-gradient(circle,color-mix(in_oklch,var(--waterhouse-highlight)_16%,transparent),transparent_70%)]" />
                            <div className="relative space-y-3">
                                <div className="inline-flex w-fit items-center gap-2 rounded-full border border-border/25 bg-background/42 px-3 py-1 text-xs font-medium text-muted-foreground shadow-waterhouse-soft">
                                    {t('card.empty')}
                                </div>
                                <div className="space-y-2">
                                    <div className="text-base font-semibold text-foreground">{t('create.title')}</div>
                                    <p className="text-sm leading-6 text-muted-foreground">{t('emptyState.description')}</p>
                                </div>
                            </div>

                            <div className="relative flex flex-col gap-3">
                                <AutoGroupButton variant="default" className="h-12 rounded-[1.35rem] justify-start px-4" />
                                <AIRouteButton variant="default" className="h-12 rounded-[1.35rem] justify-start px-4" />
                                <MorphingDialog>
                                    <MorphingDialogTrigger className={buttonVariants({ variant: 'outline', className: 'h-12 min-w-36 justify-start rounded-[1.35rem] border-border/35 bg-background/55 px-4 shadow-waterhouse-soft backdrop-blur-md hover:bg-background/72' })}>
                                        {t('create.submit')}
                                    </MorphingDialogTrigger>
                                    <MorphingDialogContainer>
                                        <MorphingDialogContent className="h-[calc(100dvh-2rem)] w-[min(100vw-2rem,92rem)] max-w-full flex-col overflow-hidden rounded-[2.4rem] border border-border/35 bg-background/80 px-4 py-4 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:h-[calc(100dvh-3rem)] md:px-6 md:py-5">
                                            <CreateDialogContent />
                                        </MorphingDialogContent>
                                    </MorphingDialogContainer>
                                </MorphingDialog>
                            </div>
                        </div>
                    </div>
                </section>
            </div>
        );
    }

    return (
        <div className="flex h-full min-h-0 flex-col gap-4">
            <section className="waterhouse-island relative flex-none overflow-hidden rounded-[2.1rem] border border-border/35 bg-card/58 p-4 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:p-5">
                <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_16%_12%,color-mix(in_oklch,var(--waterhouse-highlight)_18%,transparent)_0%,transparent_28%),linear-gradient(135deg,color-mix(in_oklch,white_12%,transparent),transparent_52%,color-mix(in_oklch,var(--primary)_8%,transparent))]" />
                <div className="relative flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                    <div className="flex flex-wrap items-center gap-2">
                        <span className="inline-flex items-center gap-2 rounded-full border border-primary/12 bg-background/44 px-3 py-1 text-[0.68rem] font-semibold uppercase tracking-[0.2em] text-primary shadow-waterhouse-soft backdrop-blur-md">
                            <Waves className="size-3.5" />
                            {t('actions.aiRoute')}
                        </span>
                        <span className="waterhouse-pod inline-flex items-center gap-2 rounded-full border border-border/25 bg-background/36 px-3 py-1 text-xs text-muted-foreground shadow-waterhouse-soft">
                            {visibleGroups.length}
                        </span>
                    </div>

                    <div className="flex flex-wrap items-center gap-2">
                        <AutoGroupButton variant="default" className="h-10 rounded-[1.2rem] px-4" />
                        <AIRouteButton variant="default" className="h-10 rounded-[1.2rem] px-4" />
                        <MorphingDialog>
                            <MorphingDialogTrigger className={buttonVariants({ variant: 'outline', className: 'h-10 rounded-[1.2rem] border-border/35 bg-background/55 px-4 shadow-waterhouse-soft backdrop-blur-md hover:bg-background/72' })}>
                                {t('create.submit')}
                            </MorphingDialogTrigger>
                            <MorphingDialogContainer>
                                <MorphingDialogContent className="h-[calc(100dvh-2rem)] w-[min(100vw-2rem,92rem)] max-w-full flex-col overflow-hidden rounded-[2.4rem] border border-border/35 bg-background/80 px-4 py-4 text-card-foreground shadow-waterhouse-deep backdrop-blur-[var(--waterhouse-shell-blur)] md:h-[calc(100dvh-3rem)] md:px-6 md:py-5">
                                    <CreateDialogContent />
                                </MorphingDialogContent>
                            </MorphingDialogContainer>
                        </MorphingDialog>
                    </div>
                </div>
            </section>

            <section className="relative min-h-0 flex-1">
                <VirtualizedGrid
                    items={visibleGroups}
                    columns={{ default: 1, md: 2, lg: 3 }}
                    estimateItemHeight={620}
                    getItemKey={(group, index) => group.id ?? `group-${index}`}
                    renderItem={(group) => <GroupCard group={group} />}
                />
            </section>
        </div>
    );
}
