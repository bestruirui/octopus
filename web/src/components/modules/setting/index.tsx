'use client';

import { PageWrapper } from '@/components/common/PageWrapper';
import { SettingAppearance } from './Appearance';
import { SettingSystem } from './System';
import { SettingAPIKey } from './APIKey';
import { SettingLLMPrice } from './LLMPrice';
import { SettingAccount } from './Account';
import { SettingInfo } from './Info';
import { SettingLLMSync } from './LLMSync';
import { SettingLog } from './Log';
import { SettingBackup } from './Backup';
import { SettingCircuitBreaker } from './CircuitBreaker';

export function Setting() {
    return (
        <div className="h-full min-h-0 overflow-y-auto overscroll-contain rounded-t-3xl">
            <PageWrapper className="columns-1 gap-4 pb-24 md:columns-2 md:pb-4 *:mb-4 *:break-inside-avoid">
                <SettingInfo key="setting-info" />
                <SettingAppearance key="setting-appearance" />
                <SettingAccount key="setting-account" />
                <SettingSystem key="setting-system" />
                <SettingLog key="setting-log" />
                <SettingLLMPrice key="setting-llmprice" />
                <SettingAPIKey key="setting-apikey" />
                <SettingLLMSync key="setting-llmsync" />
                <SettingCircuitBreaker key="setting-circuit-breaker" />
                <SettingBackup key="setting-backup" />
            </PageWrapper>
        </div>
    );
}
