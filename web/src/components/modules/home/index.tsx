'use client';

import { Activity } from './activity';
import { HomeHero } from './hero';
import { HomeAnalyticsOverview } from './analytics-overview';
import { StatsChart } from './chart';
import { Rank } from './rank';
import { PageWrapper } from '@/components/common/PageWrapper';

export function Home() {
    return (
        <PageWrapper className="h-full min-h-0 overflow-y-auto overscroll-contain space-y-6 pb-24 md:pb-4 rounded-t-3xl">
            <HomeHero />
            <HomeAnalyticsOverview />
            <StatsChart />
            <div className="grid gap-6 xl:grid-cols-[minmax(0,1.08fr)_minmax(320px,0.92fr)]">
                <Activity />
                <Rank />
            </div>
        </PageWrapper>
    );
}
