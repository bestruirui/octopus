'use client';

import { useMemo } from 'react';
import { RouteCard } from './Card';
import { useRouteList } from '@/api/endpoints/route';
import { useSearchStore } from '@/components/modules/toolbar';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';

export function Route() {
    const { data: routes } = useRouteList();
    const pageKey = 'route' as const;
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));

    const visibleRoutes = useMemo(() => {
        if (!routes) return [];
        const term = searchTerm.toLowerCase().trim();
        if (!term) return routes;
        return routes.filter((r) => r.name.toLowerCase().includes(term));
    }, [routes, searchTerm]);

    return (
        <VirtualizedGrid
            items={visibleRoutes}
            columns={{ default: 1, md: 2, lg: 3 }}
            estimateItemHeight={400}
            getItemKey={(route, index) => route.id ?? `route-${index}`}
            renderItem={(route) => <RouteCard route={route} />}
        />
    );
}
