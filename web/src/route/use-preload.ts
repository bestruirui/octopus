import { useCallback } from 'react';
import { CONTENT_MAP } from './config';
import type { RouteId } from './config';

export function usePreload() {
    const preload = useCallback((routeId: RouteId) => {
        const component = CONTENT_MAP[routeId];
        if (component?.preload) {
            component.preload();
        }
    }, []);

    return { preload };
}
