'use client';

import { useCallback } from 'react';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Copy } from 'lucide-react';
import { toast } from '@/components/common/Toast';
import { useTranslations } from 'next-intl';
import type { ModelCheckboxItemProps } from './types';

/**
 * ModelCheckboxItem 组件 - 模型复选框项
 * 
 * 显示单个模型的复选框、名称、来源标签和复制按钮
 */
export function ModelCheckboxItem({
  model,
  onToggle,
  onCopy,
}: ModelCheckboxItemProps) {
  const t = useTranslations('channel.modelSelector');

  // 处理复制模型名称
  const handleCopy = useCallback(async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(model.name);
      toast.success(t('copied', { model: model.name }));
    } catch (error) {
      toast.error('Failed to copy');
    }
  }, [model.name, t]);

  return (
    <div className="flex items-center gap-3 px-4 py-2 hover:bg-muted/50 rounded-lg transition-colors">
      {/* 复选框 */}
      <Checkbox
        checked={model.isSelected}
        onCheckedChange={() => onToggle(model.name, model.source)}
        id={`model-${model.name}`}
      />

      {/* 模型名称 */}
      <label
        htmlFor={`model-${model.name}`}
        className="flex-1 cursor-pointer text-sm select-none"
      >
        {model.name}
      </label>

      {/* 来源标签 */}
      <Badge
        variant={model.source === 'api' ? 'default' : 'secondary'}
        className="text-xs shrink-0"
      >
        {model.source === 'api' ? t('apiModel') : t('customModel')}
      </Badge>

      {/* 复制按钮 */}
      {onCopy && (
        <Button
          variant="ghost"
          size="icon"
          className="h-6 w-6 shrink-0"
          onClick={handleCopy}
          type="button"
        >
          <Copy className="h-3 w-3" />
        </Button>
      )}
    </div>
  );
}
