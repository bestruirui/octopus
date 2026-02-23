'use client';

import { useCallback } from 'react';
import { Plus } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import type { ModelInputProps } from './types';

/**
 * ModelInput 组件 - 手动添加模型的输入框
 * 
 * 功能特性：
 * - 输入框用于输入模型名称（需求 4.1）
 * - 添加按钮用于添加模型（需求 4.4）
 * - 支持 Enter 键添加模型（需求 4.3）
 * - 添加后自动清空输入框（需求 4.3, 4.4）
 */
export function ModelInput({
  value,
  onChange,
  onAdd,
  placeholder,
  disabled = false,
}: ModelInputProps) {
  const t = useTranslations('channel.modelSelector');

  /**
   * 处理 Enter 键按下以添加模型
   * 需求 4.3：支持 Enter 键添加模型
   */
  const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      onAdd();
    }
  }, [onAdd]);

  /**
   * 处理按钮点击以添加模型
   * 需求 4.4：支持按钮点击添加模型
   */
  const handleAddClick = useCallback(() => {
    onAdd();
  }, [onAdd]);

  return (
    <div className="flex flex-col gap-2 md:flex-row md:gap-2">
      {/* 模型名称输入框 */}
      {/* 需求 6.2：移动端和桌面端的响应式布局 */}
      <Input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        disabled={disabled}
        className="flex-1 h-11 md:h-10"
        aria-label={placeholder}
      />

      {/* 添加按钮 */}
      {/* 需求 6.3：移动端触摸目标尺寸至少 44x44px */}
      <Button
        type="button"
        onClick={handleAddClick}
        disabled={disabled || !value.trim()}
        size="default"
        variant="outline"
        aria-label={t('add')}
        className="min-h-[44px] md:min-h-[40px] gap-2 w-full md:w-auto"
      >
        <Plus className="h-4 w-4" />
        {t('add')}
      </Button>
    </div>
  );
}
