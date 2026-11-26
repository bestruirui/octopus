import type { ChannelDetail, HeaderOp } from '@/api/channel';

// OverrideOp 是参数覆盖的操作类型, 取值与后端 model.OverrideOp* 一致。
export type OverrideOp = 'set' | 'set_if_absent' | 'delete';

// OverrideRow 是参数覆盖单条操作的编辑形状; 提交时序列化为 JSON 操作数组。
export type OverrideRow = {
    op: OverrideOp;
    path: string;
    value: string;
    condition: string;
};

// HeaderRow 是自定义 Header 单条操作的编辑形状, HeaderOp 取值与后端一致。
export type HeaderRow = {
    op: HeaderOp;
    key: string;
    value: string;
    condition: string;
};

// ChannelFormState 是渠道表单的全部可编辑内容。
// 全按名称组织而不存主键: 后端凭据与模型都按名称匹配增删改, 而新建渠道和新加模型时主键尚不存在。
export type ChannelFormState = {
    name: string;
    dialect: ChannelDetail['dialect'];
    base_url: string;
    enabled: boolean;
    proxy: boolean;
    openai_chat_completion_path: string;
    openai_response_path: string;
    anthropic_message_path: string;
    keys: { name: string; key: string; enabled: boolean }[];
    models: string[];
    grants: Map<string, number>; // 键为 grantKey(模型名, 凭据名), 值为 Protocol 位掩码。
    custom_header: HeaderRow[]; // 自定义 Header 的行编辑内容。
    channel_proxy: string;
    param_overrides: OverrideRow[]; // 参数覆盖的行编辑内容。
    param_override_legacy: string; // 无法无损转行的旧格式原文; 非空时以 textarea 原样编辑并提交。
    match_regex: string;
};

// grantKey 生成授权在状态里的键; 分隔符取 \0, 模型名与凭据名都不会含它。
export function grantKey(modelName: string, keyName: string) {
    return `${modelName}\0${keyName}`;
}

export const emptyFormState: ChannelFormState = {
    name: '',
    dialect: 'generic',
    base_url: '',
    enabled: true,
    proxy: false,
    openai_chat_completion_path: '/v1/chat/completions',
    openai_response_path: '/v1/responses',
    anthropic_message_path: '/v1/messages',
    keys: [],
    models: [],
    grants: new Map(),
    custom_header: [],
    channel_proxy: '',
    param_overrides: [],
    param_override_legacy: '',
    match_regex: '',
};

// parseParamOverride 把落库的参数覆盖配置还原为编辑状态。
// 操作数组直接转行; 旧平铺对象在键不含点号与冒号时无损转成 set 行(值序列化为紧凑 JSON, 语义不变),
// 否则退回原文由 textarea 原样编辑 —— 键中的点冒号在新格式里是嵌套路径语义, 转行会改变行为。
function parseParamOverride(config: string): { rows: OverrideRow[]; legacy: string } {
    const trimmed = config.trim();
    if (!trimmed) return { rows: [], legacy: '' };
    try {
        if (trimmed.startsWith('[')) {
            const parsed = JSON.parse(trimmed) as { op?: string; path?: string; value?: string; condition?: string }[];
            const rows = parsed.map((item): OverrideRow => ({
                op: item.op === 'set_if_absent' || item.op === 'delete' ? item.op : 'set',
                path: item.path ?? '',
                value: item.value ?? '',
                condition: item.condition ?? '',
            }));
            return { rows, legacy: '' };
        }
        if (trimmed.startsWith('{')) {
            const parsed = JSON.parse(trimmed) as Record<string, unknown>;
            const keys = Object.keys(parsed);
            // model 与 stream 由转发流程管理, 旧配置里即使写了也不生效, 转行时同样丢弃。
            const convertible = keys.every((key) => !key.includes('.') && !key.includes(':') && key !== 'model' && key !== 'stream');
            if (convertible) {
                return {
                    rows: keys.map((key) => ({
                        op: 'set' as const,
                        path: key,
                        value: JSON.stringify(parsed[key]),
                        condition: '',
                    })),
                    legacy: '',
                };
            }
        }
    } catch {
        // 解析失败按原文兜底, 提交时由后端校验兜住。
    }
    return { rows: [], legacy: trimmed };
}

// serializeParamOverride 把编辑状态序列化回落库格式。
// 未填完的行(path 为空, set 类缺值)直接丢弃, 与 custom_header 过滤半成品行的口径一致。
function serializeParamOverride(rows: OverrideRow[], legacy: string): string {
    const raw = legacy.trim();
    if (raw !== '') return raw;
    const ops = rows
        .filter((row) => row.path.trim() !== '' && (row.op === 'delete' || row.value.trim() !== ''))
        .map((row) => {
            const op: Record<string, string> = { op: row.op, path: row.path.trim() };
            if (row.op !== 'delete') op.value = row.value;
            if (row.condition.trim() !== '') op.condition = row.condition.trim();
            return op;
        });
    return ops.length > 0 ? JSON.stringify(ops) : '';
}

// fromChannel 把渠道完整配置还原为表单状态; 授权读写都按名称, 直接建索引即可。
export function fromChannel(channel: ChannelDetail): ChannelFormState {
    const { rows, legacy } = parseParamOverride(channel.param_override);
    return {
        name: channel.name,
        dialect: channel.dialect,
        base_url: channel.base_url,
        enabled: channel.enabled,
        proxy: channel.proxy,
        openai_chat_completion_path: channel.openai_chat_completion_path,
        openai_response_path: channel.openai_response_path,
        anthropic_message_path: channel.anthropic_message_path,
        keys: channel.keys.map(({ name, key, enabled }) => ({ name, key, enabled })),
        models: [...channel.models],
        grants: new Map(channel.grants.map((g) => [grantKey(g.model_name, g.key_name), g.protocols])),
        custom_header: channel.custom_header.map((header) => ({
            op: header.op || 'set', // 历史数据无 op 字段或为空串, 均等价 set。
            key: header.header_key,
            value: header.header_value,
            condition: header.condition ?? '',
        })),
        channel_proxy: channel.channel_proxy,
        param_overrides: rows,
        param_override_legacy: legacy,
        match_regex: channel.match_regex,
    };
}

// toChannelConfig 生成渠道自身的配置字段, 提交与探测共用。
// 探测只用得上其中的地址, 路径, 代理与过滤表达式, 但必须与保存后生效的完全一致, 故由同一处给出。
export function toChannelConfig(state: ChannelFormState) {
    return {
        name: state.name.trim(),
        dialect: state.dialect,
        enabled: state.enabled,
        base_url: state.base_url.trim(),
        openai_chat_completion_path: state.openai_chat_completion_path.trim(),
        openai_response_path: state.openai_response_path.trim(),
        anthropic_message_path: state.anthropic_message_path.trim(),
        proxy: state.proxy,
        // delete 只需头名; 其余 op 需要值(rename/copy 的值是目标头名), 半成品行丢弃。
        custom_header: state.custom_header
            .filter((row) => row.key.trim() !== '')
            .filter((row) => row.op === 'delete' || row.value.trim() !== '')
            .map((row) => ({
                op: row.op,
                header_key: row.key,
                header_value: row.op === 'delete' ? '' : row.value,
                ...(row.condition.trim() !== '' ? { condition: row.condition } : {}),
            })),
        channel_proxy: state.channel_proxy.trim(),
        param_override: serializeParamOverride(state.param_overrides, state.param_override_legacy),
        match_regex: state.match_regex.trim(),
    };
}

// toChannelDetail 把表单状态还原为提交用的完整配置; 创建时 id 取 0, 由后端分配。
// 读写同构, 提交即全量: 无需与原渠道逐字段比对, 表单本就一次给出完整配置。
// 协议位为空的条目不是授权, 在此丢弃。
export function toChannelDetail(state: ChannelFormState, id: number): ChannelDetail {
    return {
        ...toChannelConfig(state),
        id,
        keys: state.keys.map(({ name, key, enabled }) => ({ name: name.trim(), key: key.trim(), enabled })),
        models: [...state.models],
        grants: [...state.grants]
            .filter(([, protocols]) => protocols !== 0)
            .map(([mapKey, protocols]) => {
                const [model_name, key_name] = mapKey.split('\0');
                return { model_name, key_name, protocols };
            }),
    };
}
