'use client';

import { useMemo } from 'react';
import { useModelList } from '@/api/endpoints/model';
import { ModelItem } from './Item';
import { useSearchStore, useToolbarViewOptionsStore } from '@/components/modules/toolbar';
import { PageWrapper } from '@/components/common/PageWrapper';

export function Model() {
    const { data: models } = useModelList();
    const pageKey = 'model' as const;
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const layout = useToolbarViewOptionsStore((s) => s.getLayout(pageKey));
    const sortOrder = useToolbarViewOptionsStore((s) => s.getSortOrder(pageKey));
    const filter = useToolbarViewOptionsStore((s) => s.modelFilter);

    const sortedModels = useMemo(() => {
        if (!models) return [];
        return [...models].sort((a, b) =>
            sortOrder === 'asc' ? a.name.localeCompare(b.name) : b.name.localeCompare(a.name)
        );
    }, [models, sortOrder]);

    const visibleModels = useMemo(() => {
        const term = searchTerm.toLowerCase().trim();
        const byName = !term ? sortedModels : sortedModels.filter((m) => m.name.toLowerCase().includes(term));
        const hasPricing = (model: (typeof byName)[number]) =>
            model.input + model.output + model.cache_read + model.cache_write > 0;

        if (filter === 'priced') {
            return byName.filter(hasPricing);
        }
        if (filter === 'free') {
            return byName.filter((m) => !hasPricing(m));
        }

        return byName;
    }, [sortedModels, searchTerm, filter]);

    const gridClassName = layout === 'list'
        ? 'grid grid-cols-1 gap-4'
        : 'grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3';

    return (
        <PageWrapper className={gridClassName}>
            {visibleModels.map((model) => (
                <ModelItem key={`model-${model.name}`} model={model} layout={layout} />
            ))}
        </PageWrapper>
    );
}
