'use client';

import { Activity } from './activity';
import { Total } from './total';
import { StatsChart } from './chart';
import { Rank } from './rank';
import { PageWrapper } from '@/components/common/PageWrapper';

export function Home() {
    return (
        <PageWrapper>
            <Total />
            <Activity />
            <StatsChart />
            <Rank />
        </PageWrapper>
    );
}
