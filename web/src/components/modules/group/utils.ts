import type { LLMChannel } from '@/api/endpoints/model';
import { GroupMode } from '@/api/endpoints/group';

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

// ─── Capability Type System ─────────────────────────────────────────────────

/**
 * API capability types — mirrors the endpoint modules from the relay layer.
 */
export type CapabilityType =
    | 'chat'
    | 'rerank'
    | 'moderation'
    | 'image_generation'
    | 'audio_speech'
    | 'audio_transcription'
    | 'video_generation'
    | 'music_generation'
    | 'search';

/**
 * All capability keys for iteration.
 */
export const ALL_CAPABILITIES: CapabilityType[] = [
    'chat',
    'rerank',
    'moderation',
    'image_generation',
    'audio_speech',
    'audio_transcription',
    'video_generation',
    'music_generation',
    'search',
];

/**
 * Infers capabilities from a model name using common naming conventions.
 * This is a best-effort heuristic — if a model doesn't match any known pattern
 * it defaults to ['chat'].
 */
export function inferCapabilities(modelName: string): CapabilityType[] {
    const m = modelName.toLowerCase();

    const caps: CapabilityType[] = [];

    // Image generation
    if (
        m.includes('dall-e') ||
        m.includes('dalle') ||
        m.includes('flux') ||
        m.includes('stable-diffusion') ||
        m.includes('sd3') ||
        m.includes('imagen') ||
        m.includes('image') ||
        m.includes('gpt-image') ||
        m.includes('mini-max-image') ||
        m.includes('ideogram') ||
        m.includes('playground')
    ) {
        caps.push('image_generation');
    }

    // Audio speech (TTS)
    if (
        m.includes('tts') ||
        m.includes('speech') ||
        m.includes('audio-speech') ||
        m.includes('playht') ||
        m.includes('elevenlabs') ||
        m.includes('cartesia')
    ) {
        caps.push('audio_speech');
    }

    // Audio transcription (STT)
    if (
        m.includes('whisper') ||
        m.includes('transcri') ||
        m.includes('audio-transcri') ||
        m.includes('deepgram')
    ) {
        caps.push('audio_transcription');
    }

    // Video generation
    if (
        m.includes('video') ||
        m.includes('animate') ||
        m.includes('svd') ||
        m.includes('sora') ||
        m.includes('kling') ||
        m.includes('luma') ||
        m.includes('runway')
    ) {
        caps.push('video_generation');
    }

    // Music generation
    if (
        m.includes('music') ||
        m.includes('stable-audio') ||
        m.includes('audio-craft')
    ) {
        caps.push('music_generation');
    }

    // Search
    if (
        m.includes('search') ||
        m.includes('serper') ||
        m.includes('brave-search') ||
        m.includes('exa') ||
        m.includes('tavily')
    ) {
        caps.push('search');
    }

    // Rerank
    if (
        m.includes('rerank') ||
        m.includes('re-rank') ||
        m.includes('cohere-rerank')
    ) {
        caps.push('rerank');
    }

    // Moderation
    if (
        m.includes('moderation') ||
        m.includes('moderat') ||
        m.includes('omni-moderation')
    ) {
        caps.push('moderation');
    }

    // If no specific capability matched, default to chat
    if (caps.length === 0) {
        caps.push('chat');
    }

    return caps;
}

/**
 * Infers the combined set of capabilities for a group based on all its model names.
 */
export function inferGroupCapabilities(modelNames: string[]): CapabilityType[] {
    const set = new Set<CapabilityType>();
    for (const name of modelNames) {
        for (const cap of inferCapabilities(name)) {
            set.add(cap);
        }
    }
    // Sort for deterministic order: chat first, then alphabetical
    return ALL_CAPABILITIES.filter((c) => set.has(c));
}

/**
 * Display label key for a capability type — used with useTranslations('group').
 */
export const CAPABILITY_LABEL_KEYS: Record<CapabilityType, string> = {
    chat: 'capability.chat',
    rerank: 'capability.rerank',
    moderation: 'capability.moderation',
    image_generation: 'capability.imageGeneration',
    audio_speech: 'capability.audioSpeech',
    audio_transcription: 'capability.audioTranscription',
    video_generation: 'capability.videoGeneration',
    music_generation: 'capability.musicGeneration',
    search: 'capability.search',
};

/**
 * Color mapping for capability badges.
 */
export const CAPABILITY_COLORS: Record<CapabilityType, string> = {
    chat: 'bg-blue-500/15 text-blue-700 dark:text-blue-300',
    rerank: 'bg-purple-500/15 text-purple-700 dark:text-purple-300',
    moderation: 'bg-orange-500/15 text-orange-700 dark:text-orange-300',
    image_generation: 'bg-pink-500/15 text-pink-700 dark:text-pink-300',
    audio_speech: 'bg-teal-500/15 text-teal-700 dark:text-teal-300',
    audio_transcription: 'bg-cyan-500/15 text-cyan-700 dark:text-cyan-300',
    video_generation: 'bg-rose-500/15 text-rose-700 dark:text-rose-300',
    music_generation: 'bg-amber-500/15 text-amber-700 dark:text-amber-300',
    search: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
};
