import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type Locale = 'zh-Hans' | 'zh-Hant' | 'en';

export function normalizeLocale(locale: string | null | undefined): Locale {
    switch (locale) {
        case 'zh-Hans':
        case 'zh_Hans':
        case 'zh-hans':
        case 'zh_hans':
        case 'zh-CN':
        case 'zh_CN':
            return 'zh-Hans';
        case 'zh-Hant':
        case 'zh_Hant':
        case 'zh-hant':
        case 'zh_hant':
        case 'zh-TW':
        case 'zh_TW':
        case 'zh-HK':
        case 'zh_HK':
            return 'zh-Hant';
        case 'en':
        case 'en-US':
        case 'en_US':
            return 'en';
        default:
            return 'zh-Hans';
    }
}

interface SettingState {
    locale: Locale;
    setLocale: (locale: Locale) => void;
}

export const useSettingStore = create<SettingState>()(
    persist(
        (set) => ({
            locale: 'zh-Hans',
            setLocale: (locale) => set({ locale: normalizeLocale(locale) }),
        }),
        {
            name: 'octopus-settings',
            storage: createJSONStorage(() => localStorage),
            merge: (persistedState, currentState) => {
                const typed = (persistedState as Partial<SettingState> | null) ?? null;
                return {
                    ...currentState,
                    ...typed,
                    locale: normalizeLocale(typed?.locale),
                };
            },
        }
    )
);

