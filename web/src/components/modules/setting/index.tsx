'use client';

import { PageWrapper } from '@/components/common/PageWrapper';
import { SettingAppearance } from './Appearance';
import { SettingSystem } from './System';
import { SettingLLMPrice } from './LLMPrice';
import { SettingAccount } from './Account';
import { SettingInfo } from './Info';
import { SettingLLMSync } from './LLMSync';
import { SettingLog } from './Log';
import { SettingBackup } from './Backup';
import { SettingCircuitBreaker } from './CircuitBreaker';
import { SettingRetry } from './Retry';
import { SettingAutoStrategy } from './AutoStrategy';
import { SettingAIRoute } from './AIRoute';
import { SettingSemanticCache } from './SemanticCache';
import { SettingRouteGroupDanger } from './RouteGroupDanger';

export function Setting() {
    return (
        <div className="setting-shadowless h-full min-h-0 overflow-y-auto overscroll-contain rounded-t-3xl">
            <PageWrapper className="pb-24 md:pb-6">
                <div className="grid gap-5 xl:grid-cols-[minmax(0,1.42fr)_minmax(20rem,0.78fr)] xl:items-start">
                    <div className="min-w-0 space-y-5">
                        <div className="grid gap-5 xl:grid-cols-[minmax(0,0.92fr)_minmax(0,1.08fr)] xl:items-start">
                            <SettingAppearance key="setting-appearance" />
                            <SettingAccount key="setting-account" />
                        </div>

                        <SettingAIRoute key="setting-ai-route" />

                        <div className="grid gap-5 xl:grid-cols-[minmax(0,1.1fr)_minmax(18rem,0.9fr)]">
                            <SettingSemanticCache key="setting-semantic-cache" />
                            <div className="flex min-w-0 flex-col gap-5">
                                <SettingRetry key="setting-retry" />
                            </div>
                        </div>

                        <div className="grid gap-5 xl:grid-cols-2">
                            <SettingAutoStrategy key="setting-auto-strategy" />
                            <SettingLog key="setting-log" />
                        </div>
                    </div>

                    <div className="flex min-w-0 flex-col gap-5 xl:sticky xl:top-0">
                        <SettingInfo key="setting-info" />
                        <SettingSystem key="setting-system" />
                        <SettingLLMPrice key="setting-llmprice" />
                        <SettingLLMSync key="setting-llmsync" />
                        <SettingCircuitBreaker key="setting-circuit-breaker" />
                        <SettingBackup key="setting-backup" />
                        <SettingRouteGroupDanger key="setting-route-group-danger" />
                    </div>
                </div>
            </PageWrapper>
        </div>
    );
}
