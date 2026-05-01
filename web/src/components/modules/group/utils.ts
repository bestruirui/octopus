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
    { labelKey: 'form.endpointType.options.all', value: '*' },
    { labelKey: 'form.endpointType.options.chat', value: 'chat' },
    { labelKey: 'form.endpointType.options.deepseek', value: 'deepseek' },
    { labelKey: 'form.endpointType.options.mimo', value: 'mimo' },
    { labelKey: 'form.endpointType.options.embeddings', value: 'embeddings' },
    { labelKey: 'form.endpointType.options.rerank', value: 'rerank' },
    { labelKey: 'form.endpointType.options.moderations', value: 'moderations' },
    { labelKey: 'form.endpointType.options.imageGeneration', value: 'image_generation' },
    { labelKey: 'form.endpointType.options.audioSpeech', value: 'audio_speech' },
    { labelKey: 'form.endpointType.options.audioTranscription', value: 'audio_transcription' },
    { labelKey: 'form.endpointType.options.videoGeneration', value: 'video_generation' },
    { labelKey: 'form.endpointType.options.musicGeneration', value: 'music_generation' },
    { labelKey: 'form.endpointType.options.search', value: 'search' },
] as const;

export function normalizeEndpointType(value?: string | null) {
    const normalized = value?.trim().toLowerCase();
    if (normalized === 'responses' || normalized === 'messages') {
        return 'chat';
    }
    return normalized || '*';
}

export function endpointTypeLabelKey(value?: string | null) {
    const endpointType = normalizeEndpointType(value);
    return ENDPOINT_TYPE_OPTIONS.find((option) => option.value === endpointType)?.labelKey;
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
