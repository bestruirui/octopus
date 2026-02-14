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
        <PageWrapper className="columns-1 md:columns-2 gap-4 [&>div]:mb-4 [&>div]:break-inside-avoid">
            <div>
                <SettingInfo key="setting-info" />
            </div>
            <div>
                <SettingAppearance key="setting-appearance" />
            </div>
            <div>
                <SettingAccount key="setting-account" />
            </div>
            <div>
                <SettingSystem key="setting-system" />
            </div>
            <div>
                <SettingLog key="setting-log" />
            </div>
            <div>
                <SettingAPIKey key="setting-apikey" />
            </div>
            <div>
                <SettingLLMPrice key="setting-llmprice" />
            </div>
            <div>
                <SettingLLMSync key="setting-llmsync" />
            </div>
            <div>
                <SettingCircuitBreaker key="setting-circuit-breaker" />
            </div>
            <div>
                <SettingBackup key="setting-backup" />
            </div>
        </PageWrapper>
    );
}
