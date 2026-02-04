import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type Locale = 'zh_hans' | 'zh_hant' | 'en';

interface SettingState {
    locale: Locale;
    setLocale: (locale: Locale) => void;
}

export const useSettingStore = create<SettingState>()(
    persist(
        (set) => ({
            locale: 'zh_hans',
            setLocale: (locale) => set({ locale }),
        }),
        {
            name: 'octopus-settings',
        }
    )
);

