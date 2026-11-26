import { useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { useTranslations } from 'use-intl';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { IconButton } from '@/components/common/IconButton';
import { ConditionArea, ConditionToggle, HelpTip } from './FormOverrideShared';
import type { ChannelFormState, OverrideOp, OverrideRow } from './state';

// FormOverrides 编辑渠道的参数覆盖: 每条操作一行, 条件作为行下可展开的附加输入。
// 与自定义 Header 的行编辑同一交互; 无法无损转行的旧格式原文由调用方以 textarea 原样编辑。
export function FormOverrides({ state, setState }: {
    state: ChannelFormState;
    setState: (next: ChannelFormState) => void;
}) {
    const t = useTranslations('channel.form');

    const addRow = () => setState({
        ...state,
        param_overrides: [...state.param_overrides, { op: 'set', path: '', value: '', condition: '' }],
    });

    const updateRow = (index: number, patch: Partial<OverrideRow>) => setState({
        ...state,
        param_overrides: state.param_overrides.map((row, i) => (i === index ? { ...row, ...patch } : row)),
    });

    const removeRow = (index: number) => setState({
        ...state,
        param_overrides: state.param_overrides.filter((_, i) => i !== index),
    });

    if (state.param_override_legacy !== '') {
        return (
            <div className="space-y-2">
                <Label htmlFor="channel-param-override-legacy">{t('paramOverride')}</Label>
                <textarea
                    id="channel-param-override-legacy"
                    value={state.param_override_legacy}
                    onChange={(e) => setState({ ...state, param_override_legacy: e.target.value })}
                    className="min-h-24 w-full rounded-xl border border-border bg-background px-3 py-2 font-mono text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                />
                <p className="text-xs text-muted-foreground">{t('paramOverrideLegacyHint')}</p>
            </div>
        );
    }

    return (
        <div className="space-y-2">
            <div className="flex items-center gap-1.5">
                <Label>{t('paramOverride')}</Label>
                <HelpTip text={t('overrideHelp')} />
                <IconButton onClick={addRow} className="ml-auto size-9" tip={t('overrideAdd')}>
                    <Plus className="size-4" />
                </IconButton>
            </div>
            {state.param_overrides.map((row, index) => (
                <OverrideRowEditor
                    key={index}
                    row={row}
                    onChange={(patch) => updateRow(index, patch)}
                    onRemove={() => removeRow(index)}
                />
            ))}
        </div>
    );
}

// OverrideRowEditor 单条覆盖操作: 主行是操作类型 + 路径 + 值(delete 无值), 条件输入默认折叠,
// 已有条件或展开后显示在行下, 与主行间以细线区隔。
function OverrideRowEditor({ row, onChange, onRemove }: {
    row: OverrideRow;
    onChange: (patch: Partial<OverrideRow>) => void;
    onRemove: () => void;
}) {
    const t = useTranslations('channel.form');
    const [showCondition, setShowCondition] = useState(row.condition !== '');

    return (
        <div className="space-y-1.5">
            <div className="flex items-center gap-2">
                <Select
                    value={row.op}
                    onValueChange={(op) => onChange({ op: op as OverrideOp })}
                >
                    <SelectTrigger className="w-36 shrink-0 rounded-xl" aria-label={t('paramOverride')}>
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="rounded-xl">
                        <SelectItem value="set" className="rounded-xl">{t('overrideOpSet')}</SelectItem>
                        <SelectItem value="set_if_absent" className="rounded-xl">{t('overrideOpSetIfAbsent')}</SelectItem>
                        <SelectItem value="delete" className="rounded-xl">{t('overrideOpDelete')}</SelectItem>
                    </SelectContent>
                </Select>
                <Input
                    value={row.path}
                    placeholder={t('overridePathPlaceholder')}
                    onChange={(e) => onChange({ path: e.target.value })}
                    className="rounded-xl flex-1 font-mono text-sm"
                />
                {row.op !== 'delete' && (
                    <Input
                        value={row.value}
                        placeholder={t('overrideValuePlaceholder')}
                        onChange={(e) => onChange({ value: e.target.value })}
                        className="rounded-xl flex-1 font-mono text-sm"
                    />
                )}
                <ConditionToggle
                    expanded={showCondition}
                    onToggle={() => setShowCondition((v) => !v)}
                    label={t('overrideCondition')}
                />
                <IconButton onClick={onRemove} className="size-9 hover:text-destructive" tip={t('delete')}>
                    <Trash2 className="size-4" />
                </IconButton>
            </div>
            {showCondition && (
                <ConditionArea
                    
                    value={row.condition}
                    placeholder={'{{eq .Model "claude-3-opus-20240229"}}'}
                    onChange={(condition) => onChange({ condition })}
                />
            )}
        </div>
    );
}
