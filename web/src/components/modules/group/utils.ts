import type { LLMChannel } from '@/api/endpoints/model';
import { GroupMode } from '@/api/endpoints/group';
export {
    ALL_CAPABILITIES,
    CAPABILITY_COLORS,
    CAPABILITY_LABEL_KEYS,
    inferCapabilities,
    inferGroupCapabilities,
    matchesGroupEndpointFilter,
    type CapabilityType,
    type GroupEndpointFilter,
} from './capabilities';

export const MODE_LABELS: Record<GroupMode, string> = {
    [GroupMode.RoundRobin]: 'roundRobin',
    [GroupMode.Random]: 'random',
    [GroupMode.Failover]: 'failover',
    [GroupMode.Weighted]: 'weighted',
    [GroupMode.Auto]: 'auto',
} as const;

export const ENDPOINT_TYPE_OPTIONS = [
    { label: '全部', value: '*' },
    { label: 'Chat', value: 'chat' },
    { label: 'Responses', value: 'responses' },
    { label: 'Messages', value: 'messages' },
    { label: 'Embeddings', value: 'embeddings' },
    { label: 'Rerank', value: 'rerank' },
    { label: 'Moderations', value: 'moderations' },
    { label: '图片生成', value: 'image_generation' },
    { label: '语音合成', value: 'audio_speech' },
    { label: '音频转写', value: 'audio_transcription' },
    { label: '视频生成', value: 'video_generation' },
    { label: '音乐生成', value: 'music_generation' },
    { label: '搜索', value: 'search' },
] as const;

export function normalizeEndpointType(value?: string | null) {
    const normalized = value?.trim().toLowerCase();
    return normalized || '*';
}

export function endpointTypeLabel(value?: string | null) {
    const endpointType = normalizeEndpointType(value);
    return ENDPOINT_TYPE_OPTIONS.find((option) => option.value === endpointType)?.label ?? endpointType;
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
