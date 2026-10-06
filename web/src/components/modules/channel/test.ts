import { useRef, useState } from 'react';
import { Protocol, useTestModel } from '@/api/channel';
import { grantKey, toChannelConfig, type ChannelFormState } from './state';

// ModelTestOutcome 是一次测试的结果，连同请求时的凭据与协议一起留档，重测即覆盖。
export type ModelTestOutcome = {
    keyName: string; // 测试用的凭据名称。
    protocol: number; // 实际使用的协议，失败时为请求用的那一个。
    latencyMs: number; // 成功取上游端到端耗时，失败取本地计时。
    failed: boolean;
    content: string; // 上游回复正文，失败为空串。
    message: string; // 失败原因，成功为空串。
};

// ModelTestNotice 是无法发起测试的原因：凭据没填 Key，或该凭据没给这个模型授权协议。
export type ModelTestNotice = 'noKey' | 'noGrant';

// ModelTestState 是某个模型在当前凭据下的测试状态，按钮与结果区都只读它。
export type ModelTestState = {
    pending: boolean;
    outcome?: ModelTestOutcome;
    notice?: ModelTestNotice;
};

// chooseProtocol 从授权掩码里挑一个协议，优先级 Response > Message > Chat。
// 一次测试只走一个协议；掩码为 0 表示该凭据对这个模型没有任何授权。
export function chooseProtocol(protocols: number) {
    if (protocols & Protocol.OpenAIResponse) return Protocol.OpenAIResponse;
    if (protocols & Protocol.AnthropicMessage) return Protocol.AnthropicMessage;
    if (protocols & Protocol.OpenAIChatCompletion) return Protocol.OpenAIChatCompletion;
    return 0;
}

// useModelTest 管理模型页的测试：单测一个模型，或按当前凭据批量顺序测试一批模型。
// 都用当前选中的凭据与它对该模型的授权协议请求上游，一次只走一个协议。
// 结果按 (凭据, 模型) 归档，提示也记下所属凭据，切换凭据或模型都不会把旧结果挂到新组合上。
// 缺 Key 或没有授权时只记提示，不改用其他凭据，也不替用户另选协议，更不自动重试。
export function useModelTest(state: ChannelFormState, activeKey: string) {
    const testModel = useTestModel();
    const [pending, setPending] = useState<Set<string>>(new Set()); // 正在测试的模型名。
    const [batchPending, setBatchPending] = useState(false); // 批量测试是否在跑，批量按钮据此转圈。
    const [outcomes, setOutcomes] = useState<Map<string, ModelTestOutcome>>(new Map()); // 键为 grantKey(模型名, 凭据名)。
    const [notices, setNotices] = useState<Map<string, { keyName: string; notice: ModelTestNotice }>>(new Map());
    // 重复触发靠这两个 ref 挡：pending 与 batchPending 要到下一次渲染才更新，
    // 同一 tick 里的第二次点击（连点、双击）读到的还是旧值，只有同步的 ref 拦得住。
    const batchRef = useRef(false); // 批量在跑；期间单测与重测一律不插入，不与批量并发抢上游。
    const runningRef = useRef<Set<string>>(new Set()); // 正在请求的模型名；挡住同一模型在同一 tick 里重复发起。

    // request 按给定的凭据与配置快照请求一次上游并归档结果。
    // 快照由调用方在发起那一刻定下，批量跑到中途用户改勾选、改配置或换凭据都不会测错组合。
    const request = async (modelName: string, keyName: string, snapshot: ChannelFormState) => {
        const channelKey = snapshot.keys.find((k) => k.name === keyName);
        const protocols = snapshot.grants.get(grantKey(modelName, keyName)) ?? 0;
        const protocol = chooseProtocol(protocols);
        const reason: ModelTestNotice | undefined =
            !channelKey || channelKey.key.trim() === '' ? 'noKey' : protocol === 0 ? 'noGrant' : undefined;

        // 先按本次凭据记下提示或清掉旧提示，结果区才不会留着别的凭据留下的说法。
        setNotices((prev) => {
            const next = new Map(prev);
            if (reason) next.set(modelName, { keyName, notice: reason });
            else next.delete(modelName);
            return next;
        });
        if (reason || !channelKey) return;

        setPending((prev) => new Set(prev).add(modelName));
        const startedAt = Date.now();
        try {
            // 配置取发起这一刻的表单内容：地址、路径、代理、Header 与参数覆盖都要与保存后生效的一致。
            const data = await testModel.mutateAsync({
                channel: toChannelConfig(snapshot),
                key: channelKey.key.trim(),
                model: modelName,
                protocol,
            });
            setOutcomes((prev) => new Map(prev).set(grantKey(modelName, keyName), {
                keyName,
                protocol: data.protocol || protocol,
                latencyMs: data.latency_ms,
                failed: false,
                content: data.content,
                message: '',
            }));
        } catch (error) {
            // 上游失败只落到这个模型的结果里，不往上抛：批量还要接着测后面的模型。
            setOutcomes((prev) => new Map(prev).set(grantKey(modelName, keyName), {
                keyName,
                protocol,
                latencyMs: Date.now() - startedAt,
                failed: true,
                content: '',
                message: error instanceof Error ? error.message : String(error),
            }));
        } finally {
            setPending((prev) => {
                const next = new Set(prev);
                next.delete(modelName);
                return next;
            });
        }
    };

    // run 单测指定模型。协议与 Key 都取自当前选中的凭据，无法发起时只给提示，不发请求。
    // 批量在跑时不插入，同一模型已在请求中也不重复发起。
    const run = async (modelName: string) => {
        if (batchRef.current || runningRef.current.has(modelName)) return;
        runningRef.current.add(modelName);
        try {
            await request(modelName, activeKey, state);
        } finally {
            runningRef.current.delete(modelName);
        }
    };

    // runAll 按给定次序逐个测试，一次只打一个上游请求（避免瞬间打爆上游），单个失败继续后面的。
    // 凭据、配置与模型列表都在发起这一刻快照；任一单测在途或批量已在跑都不重复启动，没有模型时不发请求。
    const runAll = async (modelNames: string[]) => {
        if (batchRef.current || runningRef.current.size > 0 || modelNames.length === 0) return;
        const keyName = activeKey;
        const snapshot = state;
        batchRef.current = true;
        setBatchPending(true);
        try {
            for (const modelName of modelNames) await request(modelName, keyName, snapshot);
        } finally {
            batchRef.current = false;
            setBatchPending(false);
        }
    };

    // of 给出某模型在当前凭据下的状态；结果与提示都随凭据归档，换走再换回来仍在。
    const of = (modelName: string): ModelTestState => {
        const notice = notices.get(modelName);
        return {
            pending: pending.has(modelName),
            outcome: outcomes.get(grantKey(modelName, activeKey)),
            notice: notice && notice.keyName === activeKey ? notice.notice : undefined,
        };
    };

    // busy 表示已有测试在途：单测请求中或批量在跑，表头的批量按钮据此禁用。
    const busy = batchPending || pending.size > 0;

    return { run, runAll, of, batchPending, busy };
}
