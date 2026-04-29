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

const ENDPOINT_TYPE_VALUES = [
    '*',
    'chat',
    'deepseek',
    'embeddings',
    'rerank',
    'moderations',
    'image_generation',
    'audio_speech',
    'audio_transcription',
    'video_generation',
    'music_generation',
    'search',
] as const;

const ENDPOINT_TYPE_LABEL_KEYS: Record<(typeof ENDPOINT_TYPE_VALUES)[number], string> = {
    '*': 'endpointType.all',
    chat: 'endpointType.chat',
    deepseek: 'endpointType.deepseek',
    embeddings: 'endpointType.embeddings',
    rerank: 'endpointType.rerank',
    moderations: 'endpointType.moderations',
    image_generation: 'endpointType.imageGeneration',
    audio_speech: 'endpointType.audioSpeech',
    audio_transcription: 'endpointType.audioTranscription',
    video_generation: 'endpointType.videoGeneration',
    music_generation: 'endpointType.musicGeneration',
    search: 'endpointType.search',
};

type GroupTranslation = (key: string) => string;

export function getEndpointTypeOptions(t: GroupTranslation) {
    return ENDPOINT_TYPE_VALUES.map((value) => ({
        label: t(ENDPOINT_TYPE_LABEL_KEYS[value]),
        value,
    }));
}

export function normalizeEndpointType(value?: string | null) {
    const normalized = value?.trim().toLowerCase();
    if (normalized === 'responses' || normalized === 'messages') {
        return 'chat';
    }
    return normalized || '*';
}

export function endpointTypeLabel(t: GroupTranslation, value?: string | null) {
    const endpointType = normalizeEndpointType(value);
    const labelKey = ENDPOINT_TYPE_LABEL_KEYS[endpointType as keyof typeof ENDPOINT_TYPE_LABEL_KEYS];
    return labelKey ? t(labelKey) : endpointType;
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
