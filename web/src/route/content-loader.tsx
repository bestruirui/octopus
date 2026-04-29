'use client';

'use client';

import { CONTENT_MAP } from './config';
import type { RouteId } from './config';
import { useTranslations } from 'next-intl';

export function ContentLoader({ activeRoute }: { activeRoute: RouteId }) {
    const t = useTranslations('common');
    const Component = CONTENT_MAP[activeRoute];

    if (!Component) {
        return (
            <div className="flex items-center justify-center h-64">
                <p className="text-muted-foreground">{t('routeNotFound', { route: activeRoute })}</p>
            </div>
        );
    }

    return <Component />;
}
