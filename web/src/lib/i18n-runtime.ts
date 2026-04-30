import enMessages from '../../public/locale/en.json';
import zhHansMessages from '../../public/locale/zh_hans.json';
import zhHantMessages from '../../public/locale/zh_hant.json';
import { normalizeLocale, useSettingStore, type Locale } from '@/stores/setting';

const LOCALE_MESSAGES: Record<Locale, Record<string, unknown>> = {
    'zh-Hans': zhHansMessages as Record<string, unknown>,
    'zh-Hant': zhHantMessages as Record<string, unknown>,
    en: enMessages as Record<string, unknown>,
};

function getCurrentLocale() {
    return normalizeLocale(useSettingStore.getState().locale);
}

function readNestedMessage(messages: Record<string, unknown>, path: string): unknown {
    return path.split('.').reduce<unknown>((current, segment) => {
        if (!current || typeof current !== 'object') {
            return undefined;
        }
        return (current as Record<string, unknown>)[segment];
    }, messages);
}

function formatMessage(template: string, args?: Record<string, unknown>) {
    if (!args || Object.keys(args).length === 0) {
        return template;
    }

    return template.replace(/\{(\w+)\}/g, (_, key: string) => {
        const value = args[key];
        return value === undefined || value === null ? `{${key}}` : String(value);
    });
}

export function resolveRuntimeI18nMessage(
    messageKey?: string,
    messageArgs?: Record<string, unknown>,
    fallback?: string,
    locale?: Locale,
) {
    if (!messageKey) {
        return fallback;
    }

    const activeLocale = locale ?? getCurrentLocale();
    const messages = LOCALE_MESSAGES[activeLocale];
    const template = readNestedMessage(messages, messageKey);
    if (typeof template !== 'string') {
        return fallback;
    }

    return formatMessage(template, messageArgs);
}
