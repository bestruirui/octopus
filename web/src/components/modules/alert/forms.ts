export interface AlertRuleDraft {
    name: string;
    condition_type: string;
    threshold: number;
    notif_channel_id: number;
    cooldown_sec: number;
}

export interface AlertRuleEditable extends AlertRuleDraft {
    id: number;
    enabled: boolean;
    condition_json?: string;
    scope_channel_id?: number;
    scope_api_key_id?: number;
}

export interface AlertChannelDraft {
    name: string;
    url: string;
    secret: string;
}

export interface AlertChannelEditable {
    id: number;
    type: string;
    name: string;
    url: string;
    secret?: string;
    headers?: string;
}

export function createAlertRuleDraft(rule: Partial<AlertRuleDraft> = {}): AlertRuleDraft {
    return {
        name: rule.name ?? '',
        condition_type: rule.condition_type ?? 'error_rate',
        threshold: rule.threshold ?? 10,
        notif_channel_id: rule.notif_channel_id ?? 0,
        cooldown_sec: rule.cooldown_sec ?? 300,
    };
}

export function applyAlertRuleDraft<T extends AlertRuleEditable>(rule: T, draft: AlertRuleDraft): T {
    return {
        ...rule,
        ...draft,
    };
}

export function createAlertChannelDraft<T extends Partial<AlertChannelDraft>>(channel: T = {} as T): AlertChannelDraft {
    return {
        name: channel.name ?? '',
        url: channel.url ?? '',
        secret: channel.secret ?? '',
    };
}

export function applyAlertChannelDraft<T extends AlertChannelEditable>(channel: T, draft: AlertChannelDraft): T {
    return {
        ...channel,
        ...draft,
    };
}
