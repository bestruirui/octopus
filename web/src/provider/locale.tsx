import { useEffect, useState, type ReactNode } from 'react';
import { IntlProvider } from 'use-intl';
import { useSettingStore, type Locale } from '@/stores/setting';

import zh_hansMessages from '@/locales/zh_hans.json';
import zh_hantMessages from '@/locales/zh_hant.json';
import enMessages from '@/locales/en.json';

const messages: Record<Locale, typeof zh_hansMessages> = { // 各语言对应的客户端消息集合。
    zh_hans: zh_hansMessages,
    zh_hant: zh_hantMessages,
    en: enMessages,
};

// use-intl 的格式化依赖 Intl,locale 必须是 BCP47 格式(连字符),否则带占位符的消息
// 会解析失败并回退为 key 原文(如 "channel.form.balanceTestSuccess")。
const bcp47Locales: Record<Locale, string> = {
    zh_hans: 'zh-Hans',
    zh_hant: 'zh-Hant',
    en: 'en',
};

export function LocaleProvider({ children }: { children: ReactNode }) {
    const { locale } = useSettingStore();
    const [currentLocale, setCurrentLocale] = useState<Locale>('zh_hans');

    useEffect(() => {
        setCurrentLocale(locale);
    }, [locale]);

    return (
        <IntlProvider
            locale={bcp47Locales[currentLocale]}
            messages={messages[currentLocale]}
            timeZone="Asia/Shanghai"
        >
            {children}
        </IntlProvider>
    );
}
