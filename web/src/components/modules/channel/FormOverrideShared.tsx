import { ChevronDown, HelpCircle } from 'lucide-react';
import { useTranslations } from 'use-intl';
import { IconButton } from '@/components/common/IconButton';
import { Input } from '@/components/ui/input';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';

// 条件行编辑的两个共享件: 参数覆盖与自定义 Header 的行结构完全一致, 交互语言须保持统一。

// ConditionArea 是折叠展开后的条件输入区, 以左侧细线缩进与主行区隔。
export function ConditionArea({ value, placeholder, onChange }: {
    value: string;
    placeholder: string;
    onChange: (value: string) => void;
}) {
    const t = useTranslations('channel.form');
    return (
        <div className="flex items-center gap-2 border-l-2 border-border pl-2">
            <span className="shrink-0 text-xs text-muted-foreground">{t('overrideCondition')}</span>
            <Input
                aria-label={t('overrideCondition')}
                value={value}
                placeholder={placeholder}
                onChange={(e) => onChange(e.target.value)}
                className="rounded-xl flex-1 font-mono text-sm h-8"
            />
        </div>
    );
}

// ConditionToggle 是行尾的条件展开箭头, 展开时高亮并旋转。
export function ConditionToggle({ expanded, onToggle, label }: {
    expanded: boolean;
    onToggle: () => void;
    label: string;
}) {
    return (
        <IconButton
            onClick={onToggle}
            className={cn('size-9', expanded && 'text-foreground')}
            tip={label}
        >
            <ChevronDown className={cn('size-4 transition-transform', expanded && 'rotate-180')} />
        </IconButton>
    );
}

// HelpTip 是区块标题旁的问号提示, 内容支持多行, 用于说明模板变量与操作语义。
export function HelpTip({ text }: { text: string }) {
    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <button
                    type="button"
                    aria-label="help"
                    className="inline-flex size-4 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
                >
                    <HelpCircle className="size-3.5" />
                </button>
            </TooltipTrigger>
            <TooltipContent className="max-w-80 whitespace-pre-line text-left">{text}</TooltipContent>
        </Tooltip>
    );
}
