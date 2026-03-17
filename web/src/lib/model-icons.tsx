import {
    OpenAI,
    Claude,
    Gemini,
    DeepSeek,
    Mistral,
    Qwen,
    Meta,
    Ollama,
    Groq,
    Cohere,
    Perplexity,
    Zhipu,
    Yi,
    Kimi,
    Minimax,
    Doubao,
    Hunyuan,
    Spark,
    Wenxin,
    Nvidia,
    Azure,
    Aws,
    Together,
    Fireworks,
    Replicate,
    HuggingFace,
    Grok,
    Google,
    Cerebras,
    SambaNova,
    Cloudflare,
    OpenRouter,
    Volcengine,
    SiliconCloud,
    Novita,
    InternLM,
    Stepfun,
    Gemma,
    Microsoft,
    KwaiKAT,
} from '@lobehub/icons';

type AvatarComponent = typeof OpenAI.Avatar;

type ModelIconConfig = {
    prefixes: string[];
    Avatar: AvatarComponent;
    color: string;
};

/**
 * Provider configurations with prefixes, Avatar components, and brand colors
 * Similar to Go's Provider array in internal/price/price.go
 */
const MODEL_ICON_PATTERNS: ModelIconConfig[] = [
    // OpenAI - GPT series
    { prefixes: ['gpt-', 'o1', 'o3', 'o4', 'chatgpt', 'text-embedding', 'dall-e', 'openai'], Avatar: OpenAI.Avatar, color: '#10A37F' },
    // Anthropic - Claude series
    { prefixes: ['claude', 'anthropic'], Avatar: Claude.Avatar, color: '#D7765A' },
    // Google - Gemini series
    { prefixes: ['gemini'], Avatar: Gemini.Avatar, color: '#4285F4' },
    { prefixes: ['gemma'], Avatar: Gemma.Avatar, color: '#4285F4' },
    { prefixes: ['palm', 'google'], Avatar: Google.Avatar, color: '#4285F4' },
    // DeepSeek series
    { prefixes: ['deepseek'], Avatar: DeepSeek.Avatar, color: '#4D6BFE' },
    // xAI - Grok series
    { prefixes: ['grok', 'xai'], Avatar: Grok.Avatar, color: '#000000' },
    // Alibaba - Qwen series
    { prefixes: ['qwen', 'qwq', 'alibaba'], Avatar: Qwen.Avatar, color: '#6B4EFF' },
    // Zhipu - GLM series
    { prefixes: ['glm', 'chatglm', 'zhipu', 'z-ai'], Avatar: Zhipu.Avatar, color: '#3C5BFC' },
    // MiniMax series
    { prefixes: ['minimax', 'abab'], Avatar: Minimax.Avatar, color: '#1A1A2E' },
    // Moonshot/Kimi series
    { prefixes: ['moonshot', 'kimi'], Avatar: Kimi.Avatar, color: '#000000' },
    // Mistral series
    { prefixes: ['mistral', 'mixtral', 'codestral', 'pixtral'], Avatar: Mistral.Avatar, color: '#F7D046' },
    // Meta - Llama series
    { prefixes: ['llama', 'meta-llama', 'meta'], Avatar: Meta.Avatar, color: '#0668E1' },
    // ByteDance - Doubao series
    { prefixes: ['doubao', 'skylark', 'bytedance'], Avatar: Doubao.Avatar, color: '#00D6C2' },
    // Yi series
    { prefixes: ['yi-', '01-ai'], Avatar: Yi.Avatar, color: '#1B1464' },
    // Tencent - Hunyuan
    { prefixes: ['hunyuan'], Avatar: Hunyuan.Avatar, color: '#0052D9' },
    // iFlytek - Spark
    { prefixes: ['spark'], Avatar: Spark.Avatar, color: '#0078FF' },
    // Baidu - ERNIE/Wenxin
    { prefixes: ['ernie', 'wenxin', 'baidu'], Avatar: Wenxin.Avatar, color: '#2932E1' },
    // InternLM
    { prefixes: ['internlm'], Avatar: InternLM.Avatar, color: '#2F54EB' },
    // Stepfun
    { prefixes: ['stepfun', 'step-'], Avatar: Stepfun.Avatar, color: '#5B5CFF' },
    // Cloud providers
    { prefixes: ['nvidia', 'nemotron'], Avatar: Nvidia.Avatar, color: '#76B900' },
    { prefixes: ['azure'], Avatar: Azure.Avatar, color: '#0078D4' },
    { prefixes: ['aws', 'amazon', 'bedrock'], Avatar: Aws.Avatar, color: '#FF9900' },
    { prefixes: ['volcengine'], Avatar: Volcengine.Avatar, color: '#3370FF' },
    { prefixes: ['siliconflow'], Avatar: SiliconCloud.Avatar, color: '#7C3AED' },
    // Inference providers
    { prefixes: ['groq'], Avatar: Groq.Avatar, color: '#F55036' },
    { prefixes: ['together'], Avatar: Together.Avatar, color: '#0F6FFF' },
    { prefixes: ['fireworks'], Avatar: Fireworks.Avatar, color: '#FF6B00' },
    { prefixes: ['replicate'], Avatar: Replicate.Avatar, color: '#000000' },
    { prefixes: ['ollama'], Avatar: Ollama.Avatar, color: '#FFFFFF' },
    { prefixes: ['openrouter'], Avatar: OpenRouter.Avatar, color: '#6366F1' },
    { prefixes: ['cloudflare'], Avatar: Cloudflare.Avatar, color: '#F38020' },
    { prefixes: ['cerebras'], Avatar: Cerebras.Avatar, color: '#FF5722' },
    { prefixes: ['sambanova'], Avatar: SambaNova.Avatar, color: '#FF6B00' },
    { prefixes: ['novita'], Avatar: Novita.Avatar, color: '#7C3AED' },
    { prefixes: ['huggingface', 'hf'], Avatar: HuggingFace.Avatar, color: '#FFD21E' },
    // Other models
    { prefixes: ['cohere', 'command'], Avatar: Cohere.Avatar, color: '#39594D' },
    { prefixes: ['perplexity'], Avatar: Perplexity.Avatar, color: '#20B8CD' },
    { prefixes: ['phi-'], Avatar: Microsoft.Avatar, color: '#00BCF2' },
    { prefixes: ['kat'], Avatar: KwaiKAT.Avatar, color: '#1969FC' },
];

// Default configuration
const DEFAULT_CONFIG = { Avatar: OpenAI.Avatar, color: '#10A37F' };

/**
 * Get the Avatar component and color for a given model name
 * @param modelName - The name of the model
 * @returns Object containing Avatar component and brand color
 */
export function getModelIcon(modelName: string): { Avatar: AvatarComponent; color: string } {
    // Extract the part after the first '/' if it exists
    // e.g., "qwen/gpt-5.2" -> "gpt-5.2"
    const nameToMatch = modelName.includes('/') ? modelName.split('/')[1] : modelName;
    const lowerName = nameToMatch.toLowerCase();
    for (const { prefixes, Avatar, color } of MODEL_ICON_PATTERNS) {
        if (prefixes.some(prefix => lowerName.startsWith(prefix))) {
            return { Avatar, color };
        }
    }
    return DEFAULT_CONFIG;
}

