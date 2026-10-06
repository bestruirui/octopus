import { SettingAppearance } from './Appearance';
import { SettingSystem } from './System';
import { SettingTestPrompt } from './TestPrompt';
import { SettingAPIKey } from './APIKey';
import { SettingLLMPrice } from './LLMPrice';
import { SettingAccount } from './Account';
import { SettingInfo } from './Info';
import { SettingLog } from './Log';
import { SettingBackup } from './Backup';

// Setting 渲染设置页面正文。
export function Setting() {
    return (
        <div className="h-full min-h-0 overflow-y-auto overscroll-contain rounded-t-3xl pb-24 md:pb-4">
            <div className="columns-1 gap-4 md:columns-2 *:mb-4 *:break-inside-avoid">
                <SettingInfo />
                <SettingAppearance />
                <SettingAccount />
                <SettingSystem />
                <SettingTestPrompt />
                <SettingLog />
                <SettingLLMPrice />
                <SettingAPIKey />
                <SettingBackup />
            </div>
        </div>
    );
}
