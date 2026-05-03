import type { NotifChannelType, GotifyConfig, EmailConfig } from '@/api/endpoints/alert';

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
    type: NotifChannelType;
    url: string;
    secret: string;
    gotify: GotifyConfig;
    email: EmailConfig;
}

export interface AlertChannelEditable {
    id: number;
    type: string;
    name: string;
    url: string;
    secret?: string;
    headers?: string;
    config?: string;
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

function parseGotifyConfig(config?: string): GotifyConfig {
    if (!config) return { server_url: '', token: '' };
    try {
        return JSON.parse(config);
    } catch {
        return { server_url: '', token: '' };
    }
}

function parseEmailConfig(config?: string): EmailConfig {
    if (!config) return { smtp_host: '', smtp_port: 587, username: '', password: '', from: '', to: '', use_tls: true };
    try {
        return { smtp_host: '', smtp_port: 587, username: '', password: '', from: '', to: '', use_tls: true, ...JSON.parse(config) };
    } catch {
        return { smtp_host: '', smtp_port: 587, username: '', password: '', from: '', to: '', use_tls: true };
    }
}

export function createAlertChannelDraft<T extends Partial<AlertChannelEditable>>(channel: T = {} as T): AlertChannelDraft {
    const chType = (channel.type || 'webhook') as NotifChannelType;
    return {
        name: channel.name ?? '',
        type: chType,
        url: channel.url ?? '',
        secret: channel.secret ?? '',
        gotify: chType === 'gotify' ? parseGotifyConfig(channel.config) : { server_url: '', token: '' },
        email: chType === 'email' ? parseEmailConfig(channel.config) : { smtp_host: '', smtp_port: 587, username: '', password: '', from: '', to: '', use_tls: true },
    };
}

export function applyAlertChannelDraft<T extends AlertChannelEditable>(channel: T, draft: AlertChannelDraft): T {
    const result: T = {
        ...channel,
        name: draft.name,
        type: draft.type,
        url: draft.url,
        secret: draft.secret,
    };

    // Build config JSON based on channel type
    switch (draft.type) {
        case 'gotify': {
            const config: GotifyConfig = {
                server_url: draft.gotify.server_url,
                token: draft.gotify.token,
                priority: draft.gotify.priority,
            };
            result.config = JSON.stringify(config);
            // Also set url and secret from gotify config for backward compat
            if (!draft.url && config.server_url) {
                result.url = config.server_url;
            }
            if (!draft.secret && config.token) {
                result.secret = config.token;
            }
            break;
        }
        case 'email': {
            const config: EmailConfig = {
                smtp_host: draft.email.smtp_host,
                smtp_port: draft.email.smtp_port || 587,
                username: draft.email.username,
                password: draft.email.password,
                from: draft.email.from,
                to: draft.email.to,
                use_tls: draft.email.use_tls,
            };
            result.config = JSON.stringify(config);
            break;
        }
        default:
            // webhook: no config needed
            result.config = '';
    }

    return result;
}
