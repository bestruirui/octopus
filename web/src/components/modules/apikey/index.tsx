'use client';

import { PageWrapper } from '@/components/common/PageWrapper';
import { APIKeyPagePanel } from '@/components/modules/setting/APIKey';

export function APIKeyPage() {
    return (
        <div className="h-full min-h-0 overflow-y-auto overscroll-contain rounded-t-3xl">
            <PageWrapper className="pb-24 md:pb-6">
                <APIKeyPagePanel />
            </PageWrapper>
        </div>
    );
}
