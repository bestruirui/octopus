'use client';

import { CONTENT_MAP } from './config';

export function ContentLoader({ activeRoute }: { activeRoute: string }) {
    const Component = CONTENT_MAP[activeRoute];

    if (!Component) {
        return (
            <div className="flex items-center justify-center h-64">
                <p className="text-muted-foreground">Route not found: {activeRoute}</p>
            </div>
        );
    }

    return <Component />;
}
