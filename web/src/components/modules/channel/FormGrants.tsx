import { useState, type ReactNode } from 'react';
import { ChevronRight, ChevronsDownUp, ChevronsUpDown, Eraser, Plus, RefreshCw, Trash2, type LucideIcon } from 'lucide-react';
import { useTranslations } from 'use-intl';
import { Protocol } from '@/api/channel';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { IconButton } from '@/components/common/IconButton';
import { ModelTestBatchButton, ModelTestButton, ModelTestPanel } from './ModelTest';
import { useModelProbe } from './probe';
import { grantKey, type ChannelFormState } from './state';
import { useModelTest } from './test';

// GrantCells 渲染一行右侧固定的五格: chat, response, message 三个协议勾选、测试和删除。
// 表头, 模型行, 凭据子行的差别只是这一行覆盖的 (模型 × 凭据) 范围与删除动作, 勾选,
// 三态和写入是同一套逻辑, 故三级共用此段, 列宽与对齐也因此天然一致。
// 三个协议列固定, 凭据作为模型的子行, 故列数不随凭据数变化。
function GrantCells({ state, setState, models, keyNames, test, remove, icon: Icon, tip }: {
    state: ChannelFormState;
    setState: (next: ChannelFormState) => void;
    models: string[]; // 本行覆盖的模型名。
    keyNames: string[]; // 本行覆盖的凭据名。
    test?: ReactNode; // 测试格的内容, 只有模型行有; 其余行留空占位以保持列对齐。
    remove?: () => void; // 为空表示本行没有删除动作, 末格仍占位以保持列对齐。
    icon: LucideIcon; // 删除格的图标, 表头用橡皮, 模型与凭据行用垃圾桶。
    tip: string; // 删除格的提示, 三级的删除语义不同。
}) {
    // cell 渲染一个协议格: 覆盖范围内全部组合都开为 true, 全关为 false, 其余为半选。
    const cell = (bit: number) => {
        let on = 0;
        for (const modelName of models) {
            for (const keyName of keyNames) {
                if ((state.grants.get(grantKey(modelName, keyName)) ?? 0) & bit) on += 1;
            }
        }
        const total = models.length * keyNames.length;
        const value = total === 0 || on === 0 ? false : on === total ? true : 'indeterminate';
        return (
            <span className="w-7 flex justify-center">
                <Checkbox
                    checked={value}
                    disabled={total === 0}
                    onCheckedChange={() => {
                        // 已全开则整片取消, 否则整片打开; 协议位清空即该条授权不存在, 直接从 Map 移除。
                        const grants = new Map(state.grants);
                        for (const modelName of models) {
                            for (const keyName of keyNames) {
                                const mapKey = grantKey(modelName, keyName);
                                const current = grants.get(mapKey) ?? 0;
                                const protocols = value === true ? current & ~bit : current | bit;
                                if (protocols === 0) grants.delete(mapKey);
                                else grants.set(mapKey, protocols);
                            }
                        }
                        setState({ ...state, grants });
                    }}
                />
            </span>
        );
    };

    return (
        <>
            {cell(Protocol.OpenAIChatCompletion)}
            {cell(Protocol.OpenAIResponse)}
            {cell(Protocol.AnthropicMessage)}
            <span className="w-7 flex justify-center">{test}</span>
            <span className="w-7 flex justify-center">
                {remove && (
                    <IconButton
                        onClick={remove}
                        disabled={models.length === 0}
                        className="size-7 hover:text-destructive"
                        tip={tip}
                    >
                        <Icon className="size-3.5" />
                    </IconButton>
                )}
            </span>
        </>
    );
}

// FormGrants 模型集合与授权矩阵。
// 模型行的复选框一键设置该模型下所有凭据, 凭据子行只改自己那一条授权;
// 协议位全为空即该授权不存在, 提交时被丢弃。
export function FormGrants({ state, setState }: {
    state: ChannelFormState;
    setState: (next: ChannelFormState) => void;
}) {
    const t = useTranslations('channel.form');
    const { probe, pendingKey } = useModelProbe();
    const [expanded, setExpanded] = useState<Set<string>>(new Set()); // 已展开的模型名, 支持同时展开多个。
    const [adding, setAdding] = useState('');
    const [selectedKey, setSelectedKey] = useState('');

    const keyNames = state.keys.map((k) => k.name);
    const activeKey = selectedKey || keyNames[0] || '';
    // 测试只用当前选中的凭据: 结果按 (凭据, 模型) 归档, 换凭据后看到的就是新凭据的那一份。
    const modelTest = useModelTest(state, activeKey);
    const allExpanded = state.models.length > 0 && state.models.every((m) => expanded.has(m));
    // 批量测试只测当前凭据下确实勾了协议的模型：空掩码不是授权，跳过它才能一次点完不报错。
    // 列表在点击这一刻由按钮取走，执行途中再改勾选或换凭据都不影响已在跑的这一批。
    const testableModels = state.models.filter(
        (modelName) => (state.grants.get(grantKey(modelName, activeKey)) ?? 0) !== 0
    );

    // removeGrant 移除该模型在指定凭据上的授权, 模型与凭据本身保留。
    const removeGrant = (modelName: string, keyName: string) => {
        const grants = new Map(state.grants);
        grants.delete(grantKey(modelName, keyName));
        setState({ ...state, grants });
    };

    const removeModel = (modelName: string) => {
        const grants = new Map(state.grants);
        for (const keyName of keyNames) grants.delete(grantKey(modelName, keyName));
        setState({ ...state, models: state.models.filter((m) => m !== modelName), grants });
    };

    // 自定义模型直接授权给当前选中的凭据, 协议位沿用该凭据在其他模型上已有的并集。
    // 并集来自此前刷新探测出的结果, 即该凭据确实讲得通的协议; 该凭据尚无任何授权时退回 Responses,
    // 与探测的默认一致: Chat Completions 已被官方标记弃用, 需要它的渠道由用户手动勾选 Chat 列。
    const addModel = () => {
        const name = adding.trim();
        if (!name || state.models.includes(name)) return;
        let protocols = 0;
        for (const modelName of state.models) {
            protocols |= state.grants.get(grantKey(modelName, activeKey)) ?? 0;
        }
        const grants = new Map(state.grants);
        if (activeKey) grants.set(grantKey(name, activeKey), protocols || Protocol.OpenAIResponse);
        setState({ ...state, models: [...state.models, name], grants });
        setAdding('');
    };

    if (state.keys.length === 0) {
        return <p className="flex h-full items-center justify-center text-sm text-muted-foreground">{t('keysRequiredFirst')}</p>;
    }

    return (
        // 撑满步骤区高度, 模型列表内部滚动, 避免与步骤区形成两层滚动容器。
        <div className="flex flex-col gap-3 h-full min-h-0">
            {/* 凭据选择同时作用于自定义添加与刷新, 使新增模型与它所属的凭据在同一行里对应清楚。 */}
            <div className="flex items-center gap-2 shrink-0">
                <Select value={activeKey} onValueChange={setSelectedKey}>
                    <SelectTrigger className="rounded-xl h-9 w-32"><SelectValue /></SelectTrigger>
                    <SelectContent className="rounded-xl">
                        {state.keys.map((k) => (
                            <SelectItem key={k.name} value={k.name} className="rounded-lg">{k.name}</SelectItem>
                        ))}
                    </SelectContent>
                </Select>
                <Input
                    value={adding}
                    onChange={(e) => setAdding(e.target.value)}
                    onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addModel(); } }}
                    placeholder={t('modelCustomPlaceholder')}
                    className="rounded-xl h-9 flex-1"
                />
                <IconButton
                    onClick={addModel}
                    disabled={!adding.trim() || state.models.includes(adding.trim())}
                    className="size-9"
                    tip={t('modelAdd')}
                >
                    <Plus className="size-4" />
                </IconButton>
                <IconButton
                    onClick={() => probe(state, setState, activeKey)}
                    disabled={pendingKey !== null || !state.base_url.trim()}
                    className="size-9"
                    tip={t('modelRefresh')}
                >
                    <RefreshCw className={`size-4 ${pendingKey !== null ? 'animate-spin' : ''}`} />
                </IconButton>
            </div>

            <div className="flex-1 min-h-0 flex flex-col rounded-xl border border-border overflow-hidden">
                {/* 展开折叠在最左, 与下面模型行的箭头同侧; 协议标签, 三个批量勾选, 批量测试和清空靠右成组。
                    标签用 ml-auto 顶到右侧, 紧挨复选框, 才能读作这三列的表头。 */}
                <div className="flex items-center gap-1 px-3 py-2 border-b border-border bg-muted/30 shrink-0">
                    {/* 全部展开与全部折叠共用一个按钮: 已全展开时折叠, 否则展开全部。 */}
                    <IconButton
                        onClick={() => setExpanded(allExpanded ? new Set() : new Set(state.models))}
                        disabled={state.models.length === 0}
                        className="size-5"
                        tip={allExpanded ? t('grantCollapseAll') : t('grantExpandAll')}
                    >
                        {allExpanded ? <ChevronsDownUp className="size-3.5" /> : <ChevronsUpDown className="size-3.5" />}
                    </IconButton>
                    <span className="ml-auto min-w-0 truncate text-xs text-muted-foreground">
                        chat / response / message
                    </span>
                    {/* 表头覆盖全部模型全部凭据：勾选即批量，测试即把当前凭据下已勾协议勾选的模型一次测完，
                        删除即清空全部模型及其授权。批量按钮就在清空之前。 */}
                    <GrantCells
                        state={state} setState={setState}
                        models={state.models} keyNames={keyNames}
                        test={
                            <ModelTestBatchButton
                                pending={modelTest.batchPending}
                                disabled={testableModels.length === 0 || modelTest.busy}
                                onClick={() => modelTest.runAll(testableModels)}
                            />
                        }
                        remove={() => setState({ ...state, models: [], grants: new Map() })}
                        icon={Eraser}
                        tip={t('grantClearAll')}
                    />
                </div>

                <div className="flex-1 min-h-0 overflow-y-auto overscroll-contain">
                    {state.models.length === 0 ? (
                        <p className="px-3 py-6 text-center text-sm text-muted-foreground">{t('modelNoSelected')}</p>
                    ) : state.models.map((modelName) => {
                        const isOpen = expanded.has(modelName);
                        // 测试按钮与结果区共用同一份状态：结果按 (凭据, 模型) 归档，换凭据即换一份。
                        const test = modelTest.of(modelName);
                        const granted = keyNames.filter(
                            (keyName) => (state.grants.get(grantKey(modelName, keyName)) ?? 0) !== 0
                        ).length;
                        return (
                            <div key={modelName} className="border-b border-border last:border-0">
                                <div className="flex items-center gap-1 px-3 py-2">
                                    <button
                                        type="button"
                                        onClick={() => setExpanded((prev) => {
                                            const next = new Set(prev);
                                            if (!next.delete(modelName)) next.add(modelName);
                                            return next;
                                        })}
                                        className="flex items-center gap-1.5 flex-1 min-w-0 text-left"
                                    >
                                        <ChevronRight className={`size-3.5 shrink-0 text-muted-foreground transition-transform ${isOpen ? 'rotate-90' : ''}`} />
                                        <span className="text-sm truncate">{modelName}</span>
                                        <span className="text-xs text-muted-foreground tabular-nums shrink-0">
                                            {granted}/{keyNames.length}
                                        </span>
                                    </button>
                                    <GrantCells
                                        state={state} setState={setState}
                                        models={[modelName]} keyNames={keyNames}
                                        test={
                                            <ModelTestButton
                                                pending={test.pending}
                                                disabled={modelTest.batchPending}
                                                onClick={() => modelTest.run(modelName)}
                                            />
                                        }
                                        remove={() => removeModel(modelName)}
                                        icon={Trash2}
                                        tip={t('modelRemove')}
                                    />
                                </div>

                                {/* 结果与提示就贴在模型行下方，跟着这个模型走，切换模型不会看错。 */}
                                <ModelTestPanel
                                    modelName={modelName}
                                    keyName={activeKey}
                                    state={test}
                                    onRetest={() => modelTest.run(modelName)}
                                    disabled={modelTest.batchPending}
                                />

                                {/* 未授权的凭据也要列出, 否则没有入口给它打勾; 压暗以区分于已授权的凭据。 */}
                                {isOpen && state.keys.map((channelKey) => {
                                    const protocols = state.grants.get(grantKey(modelName, channelKey.name)) ?? 0;
                                    return (
                                        <div
                                            key={channelKey.name}
                                            className={`flex items-center gap-1 pl-9 pr-3 py-1.5 bg-muted/20 ${protocols === 0 ? 'opacity-45' : ''}`}
                                        >
                                            <span className="flex-1 text-xs text-muted-foreground truncate">{channelKey.name}</span>
                                            <GrantCells
                                                state={state} setState={setState}
                                                models={[modelName]} keyNames={[channelKey.name]}
                                                remove={protocols !== 0 ? () => removeGrant(modelName, channelKey.name) : undefined}
                                                icon={Trash2}
                                                tip={t('grantRemove')}
                                            />
                                        </div>
                                    );
                                })}
                            </div>
                        );
                    })}
                </div>
            </div>
        </div>
    );
}
