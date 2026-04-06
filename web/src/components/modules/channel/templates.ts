import { AutoGroupType, ChannelType, type Channel } from '@/api/endpoints/channel';
import type { ChannelFormData } from './Form';

export type ChannelTemplate = {
    key: string;
    name: string;
    description: string;
    apply: (current: ChannelFormData) => ChannelFormData;
};

function ensureKeys(keys: ChannelFormData['keys']): ChannelFormData['keys'] {
    return keys && keys.length > 0 ? keys : [{ enabled: true, channel_key: '', remark: '' }];
}

function ensureHeaders(headers: Channel['custom_header']): Channel['custom_header'] {
    return headers && headers.length > 0 ? headers : [{ header_key: '', header_value: '' }];
}

function createTemplatePatch(current: ChannelFormData, patch: Partial<ChannelFormData>): ChannelFormData {
    return {
        ...current,
        ...patch,
        keys: ensureKeys(patch.keys ?? current.keys),
        custom_header: ensureHeaders(patch.custom_header ?? current.custom_header),
        base_urls: patch.base_urls ?? current.base_urls,
    };
}

export const channelTemplates: ChannelTemplate[] = [
    {
        key: 'openai',
        name: 'OpenAI',
        description: 'OpenAI Chat 官方接口',
        apply: (current) => createTemplatePatch(current, {
            name: current.name || 'OpenAI',
            type: ChannelType.OpenAIChat,
            base_urls: [{ url: 'https://api.openai.com/v1', delay: 0 }],
            custom_header: [],
            channel_proxy: '',
            param_override: '',
            model: '',
            custom_model: '',
            auto_group: AutoGroupType.None,
            match_regex: '',
        }),
    },
    {
        key: 'openai-responses',
        name: 'OpenAI Responses',
        description: 'OpenAI Responses API',
        apply: (current) => createTemplatePatch(current, {
            name: current.name || 'OpenAI Responses',
            type: ChannelType.OpenAIResponse,
            base_urls: [{ url: 'https://api.openai.com/v1', delay: 0 }],
            custom_header: [],
            channel_proxy: '',
            param_override: '',
            model: '',
            custom_model: '',
            auto_group: AutoGroupType.None,
            match_regex: '',
        }),
    },
    {
        key: 'anthropic',
        name: 'Anthropic',
        description: 'Claude 官方接口',
        apply: (current) => createTemplatePatch(current, {
            name: current.name || 'Anthropic',
            type: ChannelType.Anthropic,
            base_urls: [{ url: 'https://api.anthropic.com', delay: 0 }],
            custom_header: [],
            channel_proxy: '',
            param_override: '',
            model: '',
            custom_model: '',
            auto_group: AutoGroupType.None,
            match_regex: '',
        }),
    },
    {
        key: 'gemini',
        name: 'Gemini',
        description: 'Google Gemini 官方接口',
        apply: (current) => createTemplatePatch(current, {
            name: current.name || 'Gemini',
            type: ChannelType.Gemini,
            base_urls: [{ url: 'https://generativelanguage.googleapis.com', delay: 0 }],
            custom_header: [],
            channel_proxy: '',
            param_override: '',
            model: '',
            custom_model: '',
            auto_group: AutoGroupType.None,
            match_regex: '',
        }),
    },
    {
        key: 'deepseek',
        name: 'DeepSeek',
        description: 'DeepSeek OpenAI 兼容接口',
        apply: (current) => createTemplatePatch(current, {
            name: current.name || 'DeepSeek',
            type: ChannelType.OpenAIChat,
            base_urls: [{ url: 'https://api.deepseek.com/v1', delay: 0 }],
            custom_header: [],
            channel_proxy: '',
            param_override: '',
            model: '',
            custom_model: '',
            auto_group: AutoGroupType.None,
            match_regex: '',
        }),
    },
    {
        key: 'openrouter',
        name: 'OpenRouter',
        description: 'OpenRouter 聚合接口',
        apply: (current) => createTemplatePatch(current, {
            name: current.name || 'OpenRouter',
            type: ChannelType.OpenAIChat,
            base_urls: [{ url: 'https://openrouter.ai/api/v1', delay: 0 }],
            custom_header: [],
            channel_proxy: '',
            param_override: '',
            model: '',
            custom_model: '',
            auto_group: AutoGroupType.None,
            match_regex: '',
        }),
    },
    {
        key: 'siliconflow',
        name: 'SiliconFlow',
        description: '硅基流动 OpenAI 兼容接口',
        apply: (current) => createTemplatePatch(current, {
            name: current.name || 'SiliconFlow',
            type: ChannelType.OpenAIChat,
            base_urls: [{ url: 'https://api.siliconflow.cn/v1', delay: 0 }],
            custom_header: [],
            channel_proxy: '',
            param_override: '',
            model: '',
            custom_model: '',
            auto_group: AutoGroupType.None,
            match_regex: '',
        }),
    },
    {
        key: 'volcengine',
        name: 'Volcengine',
        description: '火山引擎 Ark 接口',
        apply: (current) => createTemplatePatch(current, {
            name: current.name || 'Volcengine',
            type: ChannelType.Volcengine,
            base_urls: [{ url: 'https://ark.cn-beijing.volces.com/api/v3', delay: 0 }],
            custom_header: [],
            channel_proxy: '',
            param_override: '',
            model: '',
            custom_model: '',
            auto_group: AutoGroupType.None,
            match_regex: '',
        }),
    },
];
