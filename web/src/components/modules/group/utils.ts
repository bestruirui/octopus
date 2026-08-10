import type { LLMChannel } from '@/api/endpoints/model';
import { GroupMode } from '@/api/endpoints/group';

export const MODE_LABELS: Record<GroupMode, string> = {
    [GroupMode.RoundRobin]: 'roundRobin',
    [GroupMode.Random]: 'random',
    [GroupMode.Failover]: 'failover',
    [GroupMode.Weighted]: 'weighted',
} as const;

/**
 * 分组列表中模型的可选状态：
 * - ok: 正常可用
 * - channel-disabled: 所属渠道(Provider)未启用
 * - tripped: 被熔断（额度耗尽或连续失败触发）
 * - manual-disabled: 被手动禁用
 */
export type ModelHealthState = 'ok' | 'channel-disabled' | 'tripped' | 'manual-disabled';

export function modelHealth(mc: Pick<LLMChannel, 'enabled' | 'breaker_state' | 'breaker_manual_disabled'>): ModelHealthState {
    if (mc.breaker_manual_disabled) return 'manual-disabled';
    if (mc.enabled === false) return 'channel-disabled';
    if (mc.breaker_state === 1 || mc.breaker_state === 2) return 'tripped';
    return 'ok';
}

export function normalizeKey(value: string) {
    return value.trim().toLowerCase();
}

export function modelChannelKey(channelId: number, modelName: string) {
    return `${channelId}-${modelName}`;
}

export function memberKey(member: Pick<LLMChannel, 'channel_id' | 'name'>) {
    return modelChannelKey(member.channel_id, member.name);
}

export function matchesGroupName(modelName: string, groupKey: string) {
    if (!groupKey) return false;
    return modelName.toLowerCase().includes(groupKey);
}

export function buildChannelNameByModelKey(modelChannels: LLMChannel[]) {
    const map = new Map<string, string>();
    modelChannels.forEach((mc) => {
        map.set(modelChannelKey(mc.channel_id, mc.name), mc.channel_name);
    });
    return map;
}


