export function normalizeKey(value: string) {
    return value.trim().toLowerCase();
}

export function memberKey(member: { channel_grant_id: number }) {
    return String(member.channel_grant_id);
}

// 模型搜索使用授权的展示字段，不需要渠道配置或凭据明文。
export interface ModelPickerMember {
    channel_grant_id: number;
    channel_id: number;
    channel_name: string;
    name: string;
    enabled: boolean;
}

export interface ModelSearch {
    keyword: string;
    exactModel: string | null;
}

export interface ModelSuggestion {
    name: string;
    inGroup: boolean;
    channelCount: number;
    channelNames: string[];
    pendingCount: number;
    newChannelCount: number;
    disabledCount: number;
}

// 精确候选只匹配原始模型名；自由输入仍支持渠道名和模型名的包含搜索。
// 左侧列表和批量添加共用此规则，避免看到的结果与实际添加的成员不同。
export function filterModelMembers<T extends ModelPickerMember>(members: T[], search: ModelSearch, onlyEnabled: boolean): T[] {
    const keyword = normalizeKey(search.keyword);
    return members.filter((member) => {
        if (onlyEnabled && !member.enabled) return false;
        if (search.exactModel !== null) return member.name === search.exactModel;
        return !keyword || member.name.toLowerCase().includes(keyword) || member.channel_name.toLowerCase().includes(keyword);
    });
}

// 空搜索允许浏览全部候选，但绝不能因此批量加入全部授权。
export function getAutoAddMembers<T extends ModelPickerMember>(members: T[], selected: { channel_grant_id: number }[], search: ModelSearch, onlyEnabled: boolean): T[] {
    if (!normalizeKey(search.keyword) && search.exactModel === null) return [];
    const existing = new Set(selected.map(memberKey));
    return filterModelMembers(members, search, onlyEnabled).filter((member) => {
        const key = memberKey(member);
        if (existing.has(key)) return false;
        existing.add(key);
        return true;
    });
}

// 以授权 ID 追加，保留已选成员及其顺序；更新状态时再次去重，兼容快速连续添加。
export function appendUniqueMembers<T extends { channel_grant_id: number }>(current: T[], incoming: T[]): T[] {
    const existing = new Set(current.map(memberKey));
    const added = incoming.filter((member) => {
        const key = memberKey(member);
        if (existing.has(key)) return false;
        existing.add(key);
        return true;
    });
    return added.length ? [...current, ...added] : current;
}

// 相同模型在同一渠道的多个 Key 只增加授权数，不增加渠道覆盖数。
export function buildModelSuggestions(members: ModelPickerMember[], selected: Pick<ModelPickerMember, 'channel_grant_id' | 'channel_id' | 'name'>[], onlyEnabled: boolean): ModelSuggestion[] {
    const selectedGrants = new Set(selected.map(memberKey));
    const selectedModels = new Set(selected.map((member) => member.name));
    const selectedChannels = new Set(selected.map((member) => member.channel_id));
    const seenGrants = new Set<string>();
    const models = new Map<string, {
        channels: Set<number>;
        channelNames: Set<string>;
        newChannels: Set<number>;
        pendingCount: number;
        disabledCount: number;
    }>();

    for (const member of members) {
        const key = memberKey(member);
        if ((onlyEnabled && !member.enabled) || seenGrants.has(key)) continue;
        seenGrants.add(key);
        let model = models.get(member.name);
        if (!model) {
            model = { channels: new Set(), channelNames: new Set(), newChannels: new Set(), pendingCount: 0, disabledCount: 0 };
            models.set(member.name, model);
        }
        model.channels.add(member.channel_id);
        model.channelNames.add(member.channel_name);
        if (!member.enabled) model.disabledCount += 1;
        if (!selectedGrants.has(key)) {
            model.pendingCount += 1;
            if (!selectedChannels.has(member.channel_id)) model.newChannels.add(member.channel_id);
        }
    }

    return Array.from(models, ([name, model]) => ({
        name,
        inGroup: selectedModels.has(name),
        channelCount: model.channels.size,
        channelNames: [...model.channelNames],
        pendingCount: model.pendingCount,
        newChannelCount: model.newChannels.size,
        disabledCount: model.disabledCount,
    }));
}

// 先从完整索引中搜索，再取前三；空输入优先本分组已有模型，再按渠道覆盖数排序。
// 输入后相关性仍优先，同等匹配下优先已有模型，避免相似型号挤掉精确命中。
export function rankModelSuggestions(suggestions: ModelSuggestion[], keyword: string, limit = 3): ModelSuggestion[] {
    const normalized = normalizeKey(keyword);
    const relevance = (suggestion: ModelSuggestion) => {
        const name = suggestion.name.toLowerCase();
        if (!normalized || name === normalized) return 0;
        if (name.startsWith(normalized)) return 1;
        if (name.includes(normalized)) return 2;
        if (suggestion.channelNames.some((channel) => channel.toLowerCase().includes(normalized))) return 3;
        return 4;
    };
    return suggestions
        .map((suggestion) => ({ suggestion, relevance: relevance(suggestion) }))
        .filter((item) => item.relevance < 4)
        .sort((a, b) => a.relevance - b.relevance
            || Number(b.suggestion.inGroup) - Number(a.suggestion.inGroup)
            || b.suggestion.channelCount - a.suggestion.channelCount
            || (a.suggestion.name < b.suggestion.name ? -1 : a.suggestion.name > b.suggestion.name ? 1 : 0))
        .slice(0, Math.max(0, limit))
        .map((item) => item.suggestion);
}
