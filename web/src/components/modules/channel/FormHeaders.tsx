import { useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { useTranslations } from 'use-intl';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { IconButton } from '@/components/common/IconButton';
import { ConditionArea, ConditionToggle, HelpTip } from './FormOverrideShared';
import type { HeaderOp } from '@/api/channel';
import type { ChannelFormState, HeaderRow } from './state';

// FormHeaders 编辑渠道的自定义 Header: 每条操作一行 (set/delete/rename/copy), 与参数覆盖同一交互。
// delete 只需头名; rename/copy 的两个输入分别是旧头名与新头名; 条件折叠在行尾箭头后。
export function FormHeaders({ state, setState }: {
    state: ChannelFormState;
    setState: (next: ChannelFormState) => void;
}) {
    const t = useTranslations('channel.form');

    const addRow = () => setState({
        ...state,
        custom_header: [...state.custom_header, { op: 'set', key: '', value: '', condition: '' }],
    });

    const updateRow = (index: number, patch: Partial<HeaderRow>) => setState({
        ...state,
        custom_header: state.custom_header.map((row, i) => (i === index ? { ...row, ...patch } : row)),
    });

    const removeRow = (index: number) => setState({
        ...state,
        custom_header: state.custom_header.filter((_, i) => i !== index),
    });

    return (
        <div className="space-y-2">
            <div className="flex items-center gap-1.5">
                <Label>{t('customHeader')}</Label>
                <HelpTip text={t('overrideHelp')} />
                <IconButton onClick={addRow} className="ml-auto size-9" tip={t('customHeaderAdd')}>
                    <Plus className="size-4" />
                </IconButton>
            </div>
            {state.custom_header.map((row, index) => (
                <HeaderRowEditor
                    key={index}
                    row={row}
                    onChange={(patch) => updateRow(index, patch)}
                    onRemove={() => removeRow(index)}
                />
            ))}
        </div>
    );
}

// HeaderRowEditor 单条 Header 操作: 主行是操作类型 + 头名 (+ 值), 条件输入默认折叠。
function HeaderRowEditor({ row, onChange, onRemove }: {
    row: HeaderRow;
    onChange: (patch: Partial<HeaderRow>) => void;
    onRemove: () => void;
}) {
    const t = useTranslations('channel.form');
    const [showCondition, setShowCondition] = useState(row.condition !== '');

    // rename/copy 的两个输入分别是源头名与目标头名, 其余 op 是头名与值。
    const moving = row.op === 'rename' || row.op === 'copy';
    const keyPlaceholder = moving ? t('headerFromPlaceholder') : t('headerKeyPlaceholder');
    const valuePlaceholder = moving ? t('headerToPlaceholder') : t('headerValuePlaceholder');

    return (
        <div className="space-y-1.5">
            <div className="flex items-center gap-2">
                <Select value={row.op} onValueChange={(op) => onChange({ op: op as HeaderOp })}>
                    <SelectTrigger className="w-28 shrink-0 rounded-xl" aria-label={t('customHeader')}>
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="rounded-xl">
                        <SelectItem value="set" className="rounded-xl">{t('overrideOpSet')}</SelectItem>
                        <SelectItem value="delete" className="rounded-xl">{t('overrideOpDelete')}</SelectItem>
                        <SelectItem value="rename" className="rounded-xl">{t('overrideOpRename')}</SelectItem>
                        <SelectItem value="copy" className="rounded-xl">{t('overrideOpCopy')}</SelectItem>
                    </SelectContent>
                </Select>
                <Input
                    value={row.key}
                    placeholder={keyPlaceholder}
                    onChange={(e) => onChange({ key: e.target.value })}
                    className="rounded-xl flex-1 font-mono text-sm"
                />
                {row.op !== 'delete' && (
                    <Input
                        value={row.value}
                        placeholder={valuePlaceholder}
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
