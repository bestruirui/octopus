'use client';

import { useCallback, useMemo, useState, useEffect } from 'react';
import { useTranslations } from 'next-intl';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { ScrollArea } from '@/components/ui/scroll-area';
import { toast } from '@/components/common/Toast';
import { Package, Search as SearchIcon } from 'lucide-react';
import { cn } from '@/lib/utils';
import { ModelSelectorHeader } from './ModelSelectorHeader';
import { ModelCheckboxItem } from './ModelCheckboxItem';
import { ModelInput } from './ModelInput';
import { parseModels } from './utils';
import type { ModelSelectorProps, ModelItem } from './types';

/**
 * ModelSelector 组件 - 模型选择器（复选框列表模式）
 * 
 * 功能特性：
 * - 单一列表展示所有模型，使用复选框进行选择
 * - 固定高度，内容溢出时可滚动
 * - 搜索和过滤模型
 * - 手动添加自定义模型
 * - 清空所有选择（带确认对话框）
 */
export function ModelSelector({
  autoModels,
  customModels,
  availableModels: availableModelsProp = [],
  onModelsChange,
  onRefresh,
  isRefreshing,
  refreshDisabled,
  t: _t,
}: ModelSelectorProps) {
  const t = useTranslations('channel.modelSelector');

  // 将模型字符串解析为数组
  const selectedAutoModels = useMemo(() => parseModels(autoModels), [autoModels]);
  const selectedCustomModels = useMemo(() => parseModels(customModels), [customModels]);

  // 本地状态
  const [availableModels, setAvailableModels] = useState<string[]>(availableModelsProp);
  const [searchQuery, setSearchQuery] = useState('');
  const [inputValue, setInputValue] = useState('');
  const [showClearDialog, setShowClearDialog] = useState(false);
  // 保存所有曾经出现过的模型（包括取消选中的）
  const [allKnownModels, setAllKnownModels] = useState<Set<string>>(new Set());

  // 当属性变化时更新可用模型列表
  useEffect(() => {
    setAvailableModels(availableModelsProp);
  }, [availableModelsProp]);

  // 更新已知模型集合
  useEffect(() => {
    setAllKnownModels(prev => {
      const updated = new Set(prev);
      availableModels.forEach(m => updated.add(m));
      selectedAutoModels.forEach(m => updated.add(m));
      selectedCustomModels.forEach(m => updated.add(m));
      return updated;
    });
  }, [availableModels, selectedAutoModels, selectedCustomModels]);

  /**
   * 构建显示模型列表
   * 将可用模型和已选择模型合并为单一列表
   * 保留所有曾经出现过的模型，即使取消选中也不会消失
   */
  const displayModels = useMemo((): ModelItem[] => {
    const models = new Map<string, ModelItem>();

    // 添加所有已知的模型（包括曾经选中但现在取消的）
    allKnownModels.forEach(name => {
      // 判断是否为自定义模型
      const isCustom = selectedCustomModels.includes(name);
      const isAutoSelected = selectedAutoModels.includes(name);
      
      models.set(name, {
        name,
        source: isCustom ? 'custom' : 'api',
        isSelected: isAutoSelected || isCustom,
      });
    });

    // 排序：先按选中项，再按名称
    return Array.from(models.values()).sort((a, b) => {
      if (a.isSelected !== b.isSelected) {
        return a.isSelected ? -1 : 1;
      }
      return a.name.localeCompare(b.name);
    });
  }, [allKnownModels, selectedAutoModels, selectedCustomModels]);

  /**
   * 根据搜索查询过滤模型
   */
  const filteredModels = useMemo(() => {
    if (!searchQuery.trim()) return displayModels;

    const lowerQuery = searchQuery.toLowerCase();
    return displayModels.filter(model =>
      model.name.toLowerCase().includes(lowerQuery)
    );
  }, [displayModels, searchQuery]);

  /**
   * 计算模型数量
   */
  const selectedCount = selectedAutoModels.length + selectedCustomModels.length;
  const totalCount = displayModels.length;
  const filteredSelectedCount = filteredModels.filter(m => m.isSelected).length;
  const filteredTotalCount = filteredModels.length;

  /**
   * 处理模型复选框切换
   */
  const handleToggleModel = useCallback((modelName: string, source: 'api' | 'custom') => {
    if (source === 'api') {
      const isSelected = selectedAutoModels.includes(modelName);
      const newAutoModels = isSelected
        ? selectedAutoModels.filter(m => m !== modelName)
        : [...selectedAutoModels, modelName];
      onModelsChange(newAutoModels, selectedCustomModels);
    } else {
      const isSelected = selectedCustomModels.includes(modelName);
      const newCustomModels = isSelected
        ? selectedCustomModels.filter(m => m !== modelName)
        : [...selectedCustomModels, modelName];
      onModelsChange(selectedAutoModels, newCustomModels);
    }
  }, [selectedAutoModels, selectedCustomModels, onModelsChange]);

  /**
   * 处理刷新模型列表
   */
  const handleRefreshModels = useCallback(() => {
    onRefresh();
    // 注意：父组件将调用 API 并通过 props 更新
    // 当收到响应时，我们会更新 availableModels
  }, [onRefresh]);

  /**
   * 处理手动添加新模型
   */
  const handleAddModel = useCallback(() => {
    const trimmedValue = inputValue.trim();

    if (!trimmedValue) return;

    // 检查重复
    const allModels = [...selectedAutoModels, ...selectedCustomModels];
    if (allModels.includes(trimmedValue)) {
      toast.error(t('modelExists', { model: trimmedValue }));
      return;
    }

    // 添加到自定义模型
    const newCustomModels = [...selectedCustomModels, trimmedValue];
    onModelsChange(selectedAutoModels, newCustomModels);

    setInputValue('');
    toast.success(t('modelAdded', { model: trimmedValue }));
  }, [inputValue, selectedAutoModels, selectedCustomModels, onModelsChange, t]);

  /**
   * 处理清空所有模型
   */
  const handleClearAll = useCallback(() => {
    setShowClearDialog(true);
  }, []);

  /**
   * 处理全选所有模型
   * 只选择当前显示的模型（考虑搜索过滤）
   */
  const handleSelectAll = useCallback(() => {
    // 只选择当前过滤后显示的模型
    const apiModels: string[] = [];
    const customModels: string[] = [];
    
    filteredModels.forEach(model => {
      if (model.source === 'api') {
        apiModels.push(model.name);
      } else {
        customModels.push(model.name);
      }
    });
    
    // 合并已有的选中项（不在当前过滤列表中的保持选中）
    const existingApiModels = selectedAutoModels.filter(m => 
      !filteredModels.some(fm => fm.name === m && fm.source === 'api')
    );
    const existingCustomModels = selectedCustomModels.filter(m => 
      !filteredModels.some(fm => fm.name === m && fm.source === 'custom')
    );
    
    const finalApiModels = [...new Set([...existingApiModels, ...apiModels])];
    const finalCustomModels = [...new Set([...existingCustomModels, ...customModels])];
    
    onModelsChange(finalApiModels, finalCustomModels);
    toast.success(t('allModelsSelected', { count: filteredModels.length }));
  }, [filteredModels, selectedAutoModels, selectedCustomModels, onModelsChange, t]);

  /**
   * 确认清空所有模型
   */
  const handleConfirmClearAll = useCallback(() => {
    setShowClearDialog(false);
    onModelsChange([], []);
    toast.success(t('allModelsCleared'));
  }, [onModelsChange, t]);

  /**
   * 取消清空所有模型
   */
  const handleCancelClearAll = useCallback(() => {
    setShowClearDialog(false);
  }, []);

  return (
    <div className="space-y-4">
      {/* 头部：搜索、刷新和清空按钮 */}
      <ModelSelectorHeader
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        onRefresh={handleRefreshModels}
        onClearAll={handleClearAll}
        onSelectAll={handleSelectAll}
        isRefreshing={isRefreshing}
        refreshDisabled={refreshDisabled}
        selectedCount={selectedCount}
        totalCount={totalCount}
        filteredSelectedCount={filteredSelectedCount}
        filteredTotalCount={filteredTotalCount}
        t={t}
      />

      {/* 模型列表，根据内容动态调整高度 */}
      <div className="border rounded-lg">
        <ScrollArea 
          className={cn(
            "w-full",
            // 根据内容动态调整高度，设置最小和最大约束
            filteredModels.length === 0 ? "h-[200px]" : // 空状态
            filteredModels.length <= 5 ? "h-auto max-h-[280px]" : // 少量模型
            "h-96" // 大量模型 - 固定高度并可滚动
          )}
        >
          <div className="p-2 space-y-1">
            {filteredModels.length > 0 ? (
              filteredModels.map(model => (
                <ModelCheckboxItem
                  key={`${model.source}-${model.name}`}
                  model={model}
                  onToggle={handleToggleModel}
                />
              ))
            ) : searchQuery.trim() !== '' ? (
              // 空状态：无搜索结果
              <div className="flex flex-col items-center justify-center h-[180px] text-muted-foreground">
                <SearchIcon className="h-12 w-12 mb-4 opacity-50" />
                <p className="text-sm">{t('noModelsFound', { query: searchQuery })}</p>
              </div>
            ) : (
              // 空状态：完全没有模型
              <div className="flex flex-col items-center justify-center h-[180px] text-muted-foreground">
                <Package className="h-12 w-12 mb-4 opacity-50" />
                <p className="text-sm">{t('noModels')}</p>
              </div>
            )}
          </div>
        </ScrollArea>
      </div>

      {/* 手动添加模型的输入框 */}
      <ModelInput
        value={inputValue}
        onChange={setInputValue}
        onAdd={handleAddModel}
        placeholder={t('inputPlaceholder')}
      />

      {/* 清空所有模型的确认对话框 */}
      <AlertDialog open={showClearDialog} onOpenChange={handleCancelClearAll}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('clearAllConfirmTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('clearAllConfirmDescription', { count: selectedCount })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>
              {t('cancel')}
            </AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmClearAll}>
              {t('confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
