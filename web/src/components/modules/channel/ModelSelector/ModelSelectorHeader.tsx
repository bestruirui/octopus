'use client';

import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { Search, RefreshCw, Trash2 } from 'lucide-react';
import type { ModelSelectorHeaderProps } from './types';

/**
 * ModelSelectorHeader 组件 - 模型选择器头部
 * 
 * 包含搜索框、刷新按钮、清空按钮和计数显示
 */
export function ModelSelectorHeader({
  searchQuery,
  onSearchChange,
  onRefresh,
  onClearAll,
  isRefreshing,
  refreshDisabled,
  selectedCount,
  totalCount,
  t,
}: ModelSelectorHeaderProps) {
  return (
    <div className="space-y-3">
      {/* 搜索和操作按钮行 */}
      <div className="flex flex-col sm:flex-row gap-2">
        {/* 搜索输入框 */}
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            type="text"
            placeholder={t('search')}
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            className="pl-9 rounded-xl"
          />
        </div>

        {/* 操作按钮 */}
        <div className="flex gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onRefresh}
            disabled={refreshDisabled || isRefreshing}
            className="rounded-xl"
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? 'animate-spin' : ''}`} />
            {isRefreshing ? t('refreshing') : t('refresh')}
          </Button>

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onClearAll}
            disabled={selectedCount === 0}
            className="rounded-xl"
          >
            <Trash2 className="h-4 w-4 mr-2" />
            {t('clearAll')}
          </Button>
        </div>
      </div>

      {/* 计数显示 */}
      <div className="text-sm text-muted-foreground text-right">
        {t('selectedCount', { selected: selectedCount, total: totalCount })}
      </div>
    </div>
  );
}
