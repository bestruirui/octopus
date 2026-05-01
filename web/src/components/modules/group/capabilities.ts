export type CapabilityType =
    | 'chat'
    | 'embeddings'
    | 'rerank'
    | 'moderation'
    | 'image_generation'
    | 'audio_speech'
    | 'audio_transcription'
    | 'video_generation'
    | 'music_generation'
    | 'search';

export const ALL_CAPABILITIES: CapabilityType[] = [
    'chat',
    'embeddings',
    'rerank',
    'moderation',
    'image_generation',
    'audio_speech',
    'audio_transcription',
    'video_generation',
    'music_generation',
    'search',
];

const CONVERSATION_ENDPOINT_TYPES = new Set<string>(['chat', 'deepseek', 'mimo', 'responses', 'messages']);

const FILTER_CAPABILITY_MAP = {
    chat: 'chat',
    embeddings: 'embeddings',
    rerank: 'rerank',
    moderations: 'moderation',
    image_generation: 'image_generation',
    audio_speech: 'audio_speech',
    audio_transcription: 'audio_transcription',
    video_generation: 'video_generation',
    music_generation: 'music_generation',
    search: 'search',
} as const satisfies Record<string, CapabilityType>;

export type GroupEndpointFilter = keyof typeof FILTER_CAPABILITY_MAP;
type LegacyGroupEndpointFilter = GroupEndpointFilter | 'deepseek' | 'mimo' | 'responses' | 'messages';

const EMBEDDINGS_PATTERNS = [
    /embedding/,
    /voyage-embedding/,
    /(^|[-_/])bge([-.]|$)/,
    /(^|[-_/])gte([-.]|$)/,
    /(^|[-_/])e5([-.]|$)/,
];

function includesAny(value: string, patterns: RegExp[]) {
    return patterns.some((pattern) => pattern.test(value));
}

function normalizeGroupEndpointType(value?: string | null) {
    const normalized = value?.trim().toLowerCase();
    return normalized || '*';
}

/**
 * Infers capabilities from a model name using common naming conventions.
 * This is a best-effort heuristic — if a model doesn't match any known pattern
 * it defaults to ['chat'].
 */
export function inferCapabilities(modelName: string): CapabilityType[] {
    const m = modelName.toLowerCase();

    const caps: CapabilityType[] = [];

    if (includesAny(m, EMBEDDINGS_PATTERNS)) {
        caps.push('embeddings');
    }

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

    if (
        m.includes('whisper') ||
        m.includes('transcri') ||
        m.includes('audio-transcri') ||
        m.includes('deepgram')
    ) {
        caps.push('audio_transcription');
    }

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

    if (
        m.includes('music') ||
        m.includes('stable-audio') ||
        m.includes('audio-craft')
    ) {
        caps.push('music_generation');
    }

    if (
        m.includes('search') ||
        m.includes('serper') ||
        m.includes('brave-search') ||
        m.includes('exa') ||
        m.includes('tavily')
    ) {
        caps.push('search');
    }

    if (
        m.includes('rerank') ||
        m.includes('re-rank') ||
        m.includes('cohere-rerank')
    ) {
        caps.push('rerank');
    }

    if (
        m.includes('moderation') ||
        m.includes('moderat') ||
        m.includes('omni-moderation')
    ) {
        caps.push('moderation');
    }

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
    return ALL_CAPABILITIES.filter((capability) => set.has(capability));
}

export function matchesGroupEndpointFilter(
    filter: LegacyGroupEndpointFilter,
    endpointType?: string | null,
    modelNames: string[] = [],
) {
    const normalizedFilter = filter === 'responses' || filter === 'messages' ? 'chat' : filter;
    const normalizedEndpointType = normalizeGroupEndpointType(endpointType);

    if (normalizedFilter === 'chat') {
        if (CONVERSATION_ENDPOINT_TYPES.has(normalizedEndpointType)) {
            return true;
        }
        return inferGroupCapabilities(modelNames).includes('chat');
    }

    if (normalizedEndpointType === normalizedFilter) {
        return true;
    }

    if (normalizedFilter === 'deepseek' || normalizedFilter === 'mimo') {
        return false;
    }

    return inferGroupCapabilities(modelNames).includes(FILTER_CAPABILITY_MAP[normalizedFilter]);
}

export const CAPABILITY_LABEL_KEYS: Record<CapabilityType, string> = {
    chat: 'capability.chat',
    embeddings: 'capability.embeddings',
    rerank: 'capability.rerank',
    moderation: 'capability.moderation',
    image_generation: 'capability.imageGeneration',
    audio_speech: 'capability.audioSpeech',
    audio_transcription: 'capability.audioTranscription',
    video_generation: 'capability.videoGeneration',
    music_generation: 'capability.musicGeneration',
    search: 'capability.search',
};

export const CAPABILITY_COLORS: Record<CapabilityType, string> = {
    chat: 'bg-blue-500/15 text-blue-700 dark:text-blue-300',
    embeddings: 'bg-indigo-500/15 text-indigo-700 dark:text-indigo-300',
    rerank: 'bg-purple-500/15 text-purple-700 dark:text-purple-300',
    moderation: 'bg-orange-500/15 text-orange-700 dark:text-orange-300',
    image_generation: 'bg-pink-500/15 text-pink-700 dark:text-pink-300',
    audio_speech: 'bg-teal-500/15 text-teal-700 dark:text-teal-300',
    audio_transcription: 'bg-cyan-500/15 text-cyan-700 dark:text-cyan-300',
    video_generation: 'bg-rose-500/15 text-rose-700 dark:text-rose-300',
    music_generation: 'bg-amber-500/15 text-amber-700 dark:text-amber-300',
    search: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-300',
};
