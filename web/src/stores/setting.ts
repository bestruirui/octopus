import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type Locale = 'zh' | 'en';

interface SettingState {
    locale: Locale;
    setLocale: (locale: Locale) => void;
}

export const useSettingStore = create<SettingState>()(
    persist(
        (set) => ({
            locale: 'en',
            setLocale: (locale) => set({ locale }),
        }),
        {
            name: 'octopus-settings',
        }
    )
);

