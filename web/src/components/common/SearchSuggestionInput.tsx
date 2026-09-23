import { useId, useRef, useState, type ReactNode } from 'react';
import { Search, X } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Popover, PopoverAnchor, PopoverContent } from '@/components/ui/popover';
import { cn } from '@/lib/utils';

export interface SearchSuggestion {
    value: string;
    description?: ReactNode;
}

// 数据源和排序由业务层提供；该组件只处理自由输入、候选选择和键盘焦点。
export function SearchSuggestionInput({
    value,
    onValueChange,
    onSelect,
    suggestions,
    label,
    placeholder,
    heading,
    emptyMessage,
    hint,
    clearLabel,
    disabled = false,
    describedBy,
}: {
    value: string;
    onValueChange: (value: string) => void;
    onSelect: (value: string) => void;
    suggestions: SearchSuggestion[];
    label: string;
    placeholder: string;
    heading: string;
    emptyMessage: string;
    hint: string;
    clearLabel: string;
    disabled?: boolean;
    describedBy?: string;
}) {
    const id = useId();
    const inputRef = useRef<HTMLInputElement>(null);
    const anchorRef = useRef<HTMLDivElement>(null);
    const activeSourceRef = useRef<'pointer' | 'keyboard' | null>(null);
    const [open, setOpen] = useState(false);
    const [activeValue, setActiveValue] = useState<string | null>(null);
    const activeIndex = suggestions.findIndex((suggestion) => suggestion.value === activeValue);

    const handleOpenChange = (nextOpen: boolean) => {
        setOpen(nextOpen);
        if (!nextOpen) setActiveValue(null);
    };

    const selectSuggestion = (next: string) => {
        onSelect(next);
        inputRef.current?.focus();
        handleOpenChange(false);
    };

    return (
        <Popover open={open && !disabled} onOpenChange={handleOpenChange}>
            <PopoverAnchor asChild>
                <div ref={anchorRef} className="relative min-w-0 flex-1">
                    <Search aria-hidden="true" className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                    <Input
                        ref={inputRef}
                        role="combobox"
                        aria-label={label}
                        aria-autocomplete="list"
                        aria-expanded={open && !disabled}
                        aria-controls={open && !disabled ? `${id}-list` : undefined}
                        aria-activedescendant={open && !disabled && activeIndex >= 0 ? `${id}-option-${activeIndex}` : undefined}
                        aria-describedby={describedBy}
                        autoComplete="off"
                        spellCheck={false}
                        value={value}
                        disabled={disabled}
                        placeholder={placeholder}
                        className="h-9 rounded-lg bg-background pl-8 pr-8 text-xs md:text-xs"
                        onFocus={() => {
                            setOpen(true);
                            setActiveValue(null);
                        }}
                        onBlur={() => setActiveValue(null)}
                        onClick={() => {
                            // 选择或按 Escape 后焦点仍在输入框，再次点击也能展开候选。
                            setOpen(true);
                            setActiveValue(null);
                        }}
                        onChange={(event) => {
                            onValueChange(event.target.value);
                            setActiveValue(null);
                            setOpen(true);
                        }}
                        onKeyDown={(event) => {
                            if (event.nativeEvent.isComposing || event.keyCode === 229) return;
                            if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
                                event.preventDefault();
                                setOpen(true);
                                if (!suggestions.length) return;
                                const nextIndex = !open || activeIndex < 0
                                    ? (event.key === 'ArrowDown' ? 0 : suggestions.length - 1)
                                    : (activeIndex + (event.key === 'ArrowDown' ? 1 : -1) + suggestions.length) % suggestions.length;
                                activeSourceRef.current = 'keyboard';
                                setActiveValue(suggestions[nextIndex].value);
                            } else if (event.key === 'Enter') {
                                // 没有高亮项时保留自由搜索，不能隐式提交外层分组表单。
                                event.preventDefault();
                                if (open && activeIndex >= 0) selectSuggestion(suggestions[activeIndex].value);
                            } else if (event.key === 'Escape' && open) {
                                event.preventDefault();
                                event.stopPropagation();
                                handleOpenChange(false);
                            } else if (event.key === 'Tab') {
                                handleOpenChange(false);
                            }
                        }}
                    />
                    {value && (
                        <button
                            type="button"
                            aria-label={clearLabel}
                            disabled={disabled}
                            className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-muted-foreground hover:text-foreground focus-visible:outline-2"
                            onClick={() => {
                                onValueChange('');
                                setActiveValue(null);
                                inputRef.current?.focus();
                                setOpen(true);
                            }}
                        >
                            <X aria-hidden="true" className="size-3.5" />
                        </button>
                    )}
                </div>
            </PopoverAnchor>
            <PopoverContent
                align="start"
                role="presentation"
                className="z-[60] w-[max(var(--radix-popover-trigger-width),20rem)] max-w-[calc(100vw-2rem)] max-h-[var(--radix-popover-content-available-height)] overflow-y-auto rounded-xl p-1.5"
                onOpenAutoFocus={(event) => event.preventDefault()}
                onCloseAutoFocus={(event) => event.preventDefault()}
                onInteractOutside={(event) => {
                    if (event.target instanceof Node && anchorRef.current?.contains(event.target)) event.preventDefault();
                }}
                onEscapeKeyDown={(event) => {
                    // 外层 MorphingDialog 也监听 Escape，此处只关闭候选层。
                    event.preventDefault();
                    event.stopPropagation();
                    handleOpenChange(false);
                }}
            >
                <div id={`${id}-heading`} className="px-2 py-1.5 text-xs font-medium text-muted-foreground">{heading}</div>
                <div id={`${id}-list`} role="listbox" aria-labelledby={`${id}-heading`}>
                    {suggestions.map((suggestion, index) => (
                        <button
                            key={suggestion.value}
                            id={`${id}-option-${index}`}
                            type="button"
                            role="option"
                            aria-selected={index === activeIndex}
                            tabIndex={-1}
                            className={cn(
                                'flex w-full flex-col gap-1 rounded-lg px-2 py-2 text-left',
                                index === activeIndex && 'bg-accent',
                            )}
                            onPointerMove={() => {
                                activeSourceRef.current = 'pointer';
                                setActiveValue(suggestion.value);
                            }}
                            onPointerLeave={() => {
                                // 鼠标移出后取消悬停高亮，但不清除方向键选中的候选。
                                if (activeSourceRef.current === 'pointer') {
                                    setActiveValue((current) => current === suggestion.value ? null : current);
                                }
                            }}
                            onPointerDown={(event) => {
                                // 保留输入焦点，避免失焦关闭候选后丢失点击（包括触屏）。
                                if (event.button === 0) event.preventDefault();
                            }}
                            onClick={() => selectSuggestion(suggestion.value)}
                        >
                            <span className="w-full break-all text-sm font-medium">{suggestion.value}</span>
                            {suggestion.description && <span className="text-xs text-muted-foreground">{suggestion.description}</span>}
                        </button>
                    ))}
                </div>
                {!suggestions.length && <p role="status" className="px-2 py-3 text-xs text-muted-foreground">{emptyMessage}</p>}
                <p className="mt-1 border-t px-2 pt-2 pb-1 text-[11px] text-muted-foreground">{hint}</p>
            </PopoverContent>
        </Popover>
    );
}
