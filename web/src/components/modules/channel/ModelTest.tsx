import { FlaskConical, ListChecks, Loader2, RotateCw } from 'lucide-react';
import { useTranslations } from 'use-intl';
import { Protocol } from '@/api/channel';
import { IconButton } from '@/components/common/IconButton';
import type { ModelTestState } from './test';

// protocolLabel 把协议位还原成表格列上的名字，与实际走的协议一致。
function protocolLabel(protocol: number) {
    switch (protocol) {
        case Protocol.OpenAIChatCompletion: return 'chat';
        case Protocol.OpenAIResponse: return 'response';
        case Protocol.AnthropicMessage: return 'message';
        default: return String(protocol);
    }
}

// ModelTestButton 模型行里的测试按钮：请求中转圈并禁用，防止同一模型重复测试。
// disabled 另有来源（批量在跑）：那时本行的模型还没轮到，只压灰不转圈，不假装它正在请求。
// 走 IconButton 的 type="button"，不会连带提交外层的渠道表单。
export function ModelTestButton({ pending, disabled, onClick }: {
    pending: boolean;
    disabled?: boolean;
    onClick: () => void;
}) {
    const t = useTranslations('channel.form');
    return (
        <IconButton
            onClick={onClick}
            disabled={pending || disabled}
            className="size-7"
            tip={pending ? t('modelTestPending') : disabled ? t('modelTestBatchPending') : t('modelTest')}
            aria-label={t('modelTest')}
        >
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : <FlaskConical className="size-3.5" />}
        </IconButton>
    );
}

// ModelTestBatchButton 表头里的批量测试按钮：按当前凭据顺序测一遍已勾选协议的模型。
// 执行中转圈并禁用；一个都没勾选，或单测、批量已在跑时也禁用。
// aria-label 与提示都写明是批量测试，与模型行里的单测按钮区分开。
export function ModelTestBatchButton({ pending, disabled, onClick }: {
    pending: boolean;
    disabled?: boolean;
    onClick: () => void;
}) {
    const t = useTranslations('channel.form');
    return (
        <IconButton
            onClick={onClick}
            disabled={pending || disabled}
            className="size-7"
            tip={pending ? t('modelTestBatchPending') : t('modelTestBatch')}
            aria-label={t('modelTestBatch')}
        >
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : <ListChecks className="size-3.5" />}
        </IconButton>
    );
}

// ModelTestPanel 展示该模型的测试结果或无法测试的原因，就贴在模型行下方。
// 正文按纯文本渲染，长响应在固定高度内滚动，不截断也不自动消失；重测按钮就在结果旁边。
export function ModelTestPanel({ modelName, keyName, state, onRetest, disabled }: {
    modelName: string;
    keyName: string;
    state: ModelTestState;
    onRetest: () => void;
    disabled?: boolean; // 批量在跑：重测不插入，同一模型的重测按钮随之压灰。
}) {
    const t = useTranslations('channel.form');
    const { pending, outcome, notice } = state;

    if (notice) {
        return (
            <div className="mx-3 mb-2 rounded-lg border border-border bg-muted/20 px-3 py-2 text-xs">
                <p className="text-destructive">{notice === 'noKey' ? t('modelTestNoKey') : t('modelTestNoGrant')}</p>
            </div>
        );
    }
    if (!outcome) return null;

    return (
        <div className="mx-3 mb-2 rounded-lg border border-border bg-muted/20 px-3 py-2 text-xs">
            <div className="flex items-center gap-2">
                <span className={outcome.failed ? 'font-medium text-destructive' : 'font-medium text-green-600'}>
                    {outcome.failed ? t('modelTestFailed') : t('modelTestSuccess')}
                </span>
                <span className="ml-auto tabular-nums text-muted-foreground">{outcome.latencyMs} ms</span>
                <IconButton
                    onClick={onRetest}
                    disabled={pending || disabled}
                    className="size-6"
                    tip={disabled && !pending ? t('modelTestBatchPending') : t('modelTestRetest')}
                    aria-label={t('modelTestRetest')}
                >
                    {pending ? <Loader2 className="size-3 animate-spin" /> : <RotateCw className="size-3" />}
                </IconButton>
            </div>
            <p className="mt-1 text-muted-foreground">
                {modelName} · {t('modelTestKey')} {keyName} · {t('modelTestProtocol')} {protocolLabel(outcome.protocol)}
            </p>
            {/* 上游正文原样作为文本节点渲染，不解析 HTML；失败时给的是后端错误消息。 */}
            <div className="mt-1 max-h-40 overflow-y-auto overscroll-contain whitespace-pre-wrap wrap-break-word rounded bg-background/60 p-2 font-mono text-[11px]">
                {outcome.failed ? outcome.message : (outcome.content || t('modelTestEmpty'))}
            </div>
        </div>
    );
}
