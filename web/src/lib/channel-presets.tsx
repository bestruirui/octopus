import type { ComponentType } from 'react';
import type { SvgIconProps } from '@thesvg/react';
import NewAPIIcon from '@thesvg/react/new-api';
import OpenAIIcon from '@thesvg/react/openai-chatgpt';
import AnthropicIcon from '@thesvg/react/anthropic';
import VolcengineIcon from '@thesvg/react/volcengine';
import DeepSeekIcon from '@thesvg/react/deepseek';
import OpenRouterIcon from '@thesvg/react/openrouter';
import GroqIcon from '@thesvg/react/groq';
import QwenIcon from '@thesvg/react/qwen';
import MoonshotIcon from '@thesvg/react/moonshot-ai';
import ZhipuIcon from '@thesvg/react/zhipu';
import XAIIcon from '@thesvg/react/xai-grok';
import SiliconFlowIcon from '@thesvg/react/siliconcloud-siliconflow';
import { QiniuIcon, FennoIcon } from '@/components/icons/sponsors';
import AzureIcon from '@thesvg/react/azure-azure-openai';
import type { Dialect } from '@/api/channel';
import type { Locale } from '@/stores/setting';

// ChannelPreset 是服务商的地址, 路径与方言预填模板。
// 地址与路径只在前端存在, 不落库: 这些服务商对后端没有区别, 只是地址和路径不同。
// 方言随模板给出: 它表达的是该服务商在标准协议之上的差异, 属于服务商固有属性, 由用户另选没有意义。
export type ChannelPreset = {
    id: string;
    label: string;
    Icon: ComponentType<SvgIconProps>;
    iconClassName?: string; // 单色图标在深色主题下需反色。
    description: Record<Locale, string>; // 模板在不同界面语言下的说明。
    dialect: Dialect;
    base_url: string;
    openai_chat_completion_path: string;
    openai_response_path: string;
    anthropic_message_path: string;
};

// 默认协议路径, 与后端 DDL 默认值一致。
const CHAT = '/v1/chat/completions';
const RESP = '/v1/responses';
const ANTH = '/v1/messages';

export const CHANNEL_PRESETS: ChannelPreset[] = [
    {
        id: 'newapi', label: 'New API', Icon: NewAPIIcon,
        description: {
            zh_hans: '兼容多种模型服务商的统一接口', zh_hant: '兼容多種模型服務商的統一介面', en: 'Unified API compatible with multiple model providers',
        },
        dialect: 'generic',
        base_url: '',
        openai_chat_completion_path: CHAT, openai_response_path: RESP, anthropic_message_path: ANTH,
    },
    {
        id: 'openai', label: 'OpenAI', Icon: OpenAIIcon, iconClassName: 'brightness-0 dark:invert',
        description: {
            zh_hans: 'OpenAI 官方 API', zh_hant: 'OpenAI 官方 API', en: 'Official OpenAI API',
        },
        dialect: 'generic',
        base_url: 'https://api.openai.com',
        openai_chat_completion_path: CHAT, openai_response_path: RESP, anthropic_message_path: ANTH,
    },
    {
        id: 'anthropic', label: 'Anthropic', Icon: AnthropicIcon, iconClassName: 'brightness-0 dark:invert',
        description: {
            zh_hans: 'Anthropic 官方 API', zh_hant: 'Anthropic 官方 API', en: 'Official Anthropic API',
        },
        dialect: 'generic',
        base_url: 'https://api.anthropic.com',
        openai_chat_completion_path: CHAT, openai_response_path: RESP, anthropic_message_path: ANTH,
    },
    {
        id: 'volcengine', label: '火山方舟', Icon: VolcengineIcon,
        description: {
            zh_hans: '火山引擎大模型服务平台', zh_hant: '火山引擎大模型服務平台', en: 'Volcengine Ark model service',
        },
        dialect: 'generic',
        base_url: 'https://ark.cn-beijing.volces.com',
        openai_chat_completion_path: '/api/v3/chat/completions', openai_response_path: '/api/v3/responses', anthropic_message_path: '/api/compatible/v1/messages',
    },
    {
        id: 'volcengine-coding-plan', label: '火山方舟 Coding Plan', Icon: VolcengineIcon,
        description: {
            zh_hans: '火山方舟编程套餐 API', zh_hant: '火山方舟編程套餐 API', en: 'Volcengine Ark Coding Plan API',
        },
        dialect: 'generic',
        base_url: 'https://ark.cn-beijing.volces.com',
        openai_chat_completion_path: '/api/coding/v3/chat/completions', openai_response_path: '/api/coding/v3/responses', anthropic_message_path: '/api/coding/v1/messages',
    },
    {
        id: 'deepseek', label: 'DeepSeek', Icon: DeepSeekIcon,
        description: {
            zh_hans: 'DeepSeek 官方 API', zh_hant: 'DeepSeek 官方 API', en: 'Official DeepSeek API',
        },
        dialect: 'generic',
        base_url: 'https://api.deepseek.com',
        openai_chat_completion_path: '/chat/completions', openai_response_path: '/responses', anthropic_message_path: '/anthropic/v1/messages',
    },
    {
        id: 'openrouter', label: 'OpenRouter', Icon: OpenRouterIcon, iconClassName: 'brightness-0 dark:invert',
        description: {
            zh_hans: '统一访问多种模型', zh_hant: '統一存取多種模型', en: 'Unified access to multiple models',
        },
        dialect: 'generic',
        base_url: 'https://openrouter.ai/api',
        openai_chat_completion_path: CHAT, openai_response_path: RESP, anthropic_message_path: ANTH,
    },
    {
        id: 'groq', label: 'Groq', Icon: GroqIcon,
        description: {
            zh_hans: '高速推理 API', zh_hant: '高速推理 API', en: 'High-speed inference API',
        },
        dialect: 'generic',
        base_url: 'https://api.groq.com/openai',
        openai_chat_completion_path: CHAT, openai_response_path: RESP, anthropic_message_path: ANTH,
    },
    {
        id: 'dashscope', label: '通义千问', Icon: QwenIcon, iconClassName: 'brightness-0 dark:invert',
        description: {
            zh_hans: '阿里云通义千问 API', zh_hant: '阿里雲通義千問 API', en: 'Alibaba Cloud Qwen API',
        },
        dialect: 'generic',
        base_url: 'https://dashscope.aliyuncs.com',
        openai_chat_completion_path: '/compatible-mode/v1/chat/completions', openai_response_path: '/compatible-mode/v1/responses', anthropic_message_path: '/apps/anthropic/v1/messages',
    },
    {
        id: 'qwen-token-plan', label: '通义千问 Token Plan', Icon: QwenIcon, iconClassName: 'brightness-0 dark:invert',
        description: {
            zh_hans: '千问 Token Plan 专属 API', zh_hant: '千問 Token Plan 專屬 API', en: 'Qwen Token Plan API',
        },
        dialect: 'generic',
        base_url: 'https://token-plan.maas.qianwenaiapi.com',
        openai_chat_completion_path: '/compatible-mode/v1/chat/completions', openai_response_path: '/compatible-mode/v1/responses', anthropic_message_path: '/apps/anthropic/v1/messages',
    },
    {
        id: 'moonshot', label: 'Moonshot', Icon: MoonshotIcon, iconClassName: 'brightness-0 dark:invert',
        description: {
            zh_hans: '月之暗面 Kimi API', zh_hant: '月之暗面 Kimi API', en: 'Moonshot Kimi API',
        },
        dialect: 'generic',
        base_url: 'https://api.moonshot.cn',
        openai_chat_completion_path: CHAT, openai_response_path: RESP, anthropic_message_path: '/anthropic/v1/messages',
    },
    {
        id: 'kimi-code', label: 'Kimi Code', Icon: MoonshotIcon, iconClassName: 'brightness-0 dark:invert',
        description: {
            zh_hans: 'Kimi Code 会员订阅 API', zh_hant: 'Kimi Code 會員訂閱 API', en: 'Kimi Code subscription API',
        },
        dialect: 'generic',
        base_url: 'https://api.kimi.com',
        openai_chat_completion_path: '/coding/v1/chat/completions', openai_response_path: '/coding/v1/responses', anthropic_message_path: '/coding/v1/messages',
    },
    {
        id: 'zhipu', label: '智谱 GLM', Icon: ZhipuIcon,
        description: {
            zh_hans: '智谱 AI 大模型 API', zh_hant: '智譜 AI 大模型 API', en: 'Zhipu AI GLM API',
        },
        dialect: 'generic',
        base_url: 'https://open.bigmodel.cn',
        openai_chat_completion_path: '/api/paas/v4/chat/completions', openai_response_path: '/api/v1/responses', anthropic_message_path: '/api/anthropic/v1/messages',
    },
    {
        id: 'glm-coding-plan', label: '智谱 GLM Coding Plan', Icon: ZhipuIcon,
        description: {
            zh_hans: '智谱 GLM 编程套餐 API', zh_hant: '智譜 GLM 編程套餐 API', en: 'Zhipu GLM Coding Plan API',
        },
        dialect: 'generic',
        base_url: 'https://open.bigmodel.cn',
        openai_chat_completion_path: '/api/coding/paas/v4/chat/completions', openai_response_path: '/api/v1/responses', anthropic_message_path: '/api/anthropic/v1/messages',
    },
    {
        id: 'xai', label: 'xAI', Icon: XAIIcon, iconClassName: 'brightness-0 dark:invert',
        description: {
            zh_hans: 'xAI Grok API', zh_hant: 'xAI Grok API', en: 'xAI Grok API',
        },
        dialect: 'generic',
        base_url: 'https://api.x.ai',
        openai_chat_completion_path: CHAT, openai_response_path: RESP, anthropic_message_path: ANTH,
    },
    {
        id: 'qiniu', label: '七牛云', Icon: QiniuIcon,
        description: {
            zh_hans: '七牛云大模型推理 API', zh_hant: '七牛雲大模型推理 API', en: 'Qiniu Cloud AI inference API',
        },
        dialect: 'generic',
        base_url: 'https://api.qnaigc.com',
        openai_chat_completion_path: CHAT, openai_response_path: RESP, anthropic_message_path: ANTH,
    },
    {
        id: 'fenno', label: 'Fenno.ai', Icon: FennoIcon,
        description: {
            zh_hans: '面向编程场景的 Codex / Claude Code API 中转', zh_hant: '面向編程場景的 Codex / Claude Code API 中轉', en: 'Coding API gateway for Codex & Claude Code',
        },
        dialect: 'generic',
        base_url: 'https://api.fenno.ai',
        openai_chat_completion_path: CHAT, openai_response_path: RESP, anthropic_message_path: ANTH,
    },
    {
        id: 'siliconflow', label: 'SiliconFlow', Icon: SiliconFlowIcon,
        description: {
            zh_hans: '硅基流动模型 API', zh_hant: '矽基流動模型 API', en: 'SiliconFlow model API',
        },
        dialect: 'generic',
        base_url: 'https://api.siliconflow.cn',
        openai_chat_completion_path: CHAT, openai_response_path: RESP, anthropic_message_path: ANTH,
    },
    {
        id: 'azure', label: 'Azure OpenAI', Icon: AzureIcon,
        description: {
            zh_hans: 'Azure 托管的 OpenAI API', zh_hant: 'Azure 託管的 OpenAI API', en: 'Azure-hosted OpenAI API',
        },
        dialect: 'generic',
        base_url: '',
        openai_chat_completion_path: '/openai/v1/chat/completions', openai_response_path: '/openai/v1/responses', anthropic_message_path: ANTH,
    },
];
