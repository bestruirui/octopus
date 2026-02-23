/**
 * Type definitions for the ModelSelector component
 * ModelSelector 组件的类型定义
 */

/**
 * Props for the main ModelSelector component
 * ModelSelector 主组件的属性
 */
export interface ModelSelectorProps {
  /** Auto-fetched models (comma-separated string) | 自动获取的模型（逗号分隔的字符串） */
  autoModels: string;
  /** Manually added models (comma-separated string) | 手动添加的模型（逗号分隔的字符串） */
  customModels: string;
  /** Available models from API (for selection pool) | 从 API 获取的可用模型（用于选择池） */
  availableModels?: string[];
  /** Callback when models change | 模型变更时的回调函数 */
  onModelsChange: (autoModels: string[], customModels: string[]) => void;
  /** Callback to refresh models | 刷新模型的回调函数 */
  onRefresh: () => void;
  /** Whether refresh is in progress | 是否正在刷新 */
  isRefreshing: boolean;
  /** Whether refresh button is disabled | 刷新按钮是否禁用 */
  refreshDisabled: boolean;
  /** Translation function | 翻译函数 */
  t: (key: string, params?: any) => string;
}

/**
 * Model item for display in the list
 * 列表中显示的模型项
 */
export interface ModelItem {
  /** Model name | 模型名称 */
  name: string;
  /** Source of the model | 模型来源 */
  source: 'api' | 'custom';
  /** Whether the model is selected | 模型是否被选中 */
  isSelected: boolean;
}

/**
 * Props for the ModelCheckboxItem component
 * ModelCheckboxItem 组件的属性
 */
export interface ModelCheckboxItemProps {
  /** Model item to display | 要显示的模型项 */
  model: ModelItem;
  /** Callback when checkbox is toggled | 复选框切换时的回调函数 */
  onToggle: (modelName: string, source: 'api' | 'custom') => void;
  /** Optional callback to copy model name | 复制模型名称的可选回调函数 */
  onCopy?: (modelName: string) => void;
}

/**
 * Props for the ModelSelectorHeader component
 * ModelSelectorHeader 组件的属性
 */
export interface ModelSelectorHeaderProps {
  /** Search query string | 搜索查询字符串 */
  searchQuery: string;
  /** Callback when search query changes | 搜索查询变更时的回调函数 */
  onSearchChange: (query: string) => void;
  /** Callback to refresh models | 刷新模型的回调函数 */
  onRefresh: () => void;
  /** Callback to clear all selections | 清除所有选择的回调函数 */
  onClearAll: () => void;
  /** Whether refresh is in progress | 是否正在刷新 */
  isRefreshing: boolean;
  /** Whether refresh button is disabled | 刷新按钮是否禁用 */
  refreshDisabled: boolean;
  /** Number of selected models | 已选择的模型数量 */
  selectedCount: number;
  /** Total number of models | 模型总数 */
  totalCount: number;
  /** Translation function | 翻译函数 */
  t: (key: string, params?: any) => string;
}

/**
 * Props for the ModelInput component
 * ModelInput 组件的属性
 */
export interface ModelInputProps {
  /** Input value | 输入值 */
  value: string;
  /** Callback when value changes | 值变更时的回调函数 */
  onChange: (value: string) => void;
  /** Callback to add model | 添加模型的回调函数 */
  onAdd: () => void;
  /** Placeholder text | 占位符文本 */
  placeholder: string;
  /** Whether input is disabled | 输入框是否禁用 */
  disabled?: boolean;
}

/**
 * Props for the ModelGroup component
 * ModelGroup 组件的属性
 */
export interface ModelGroupProps {
  /** Group title | 组标题 */
  title: string;
  /** List of model names | 模型名称列表 */
  models: string[];
  /** Badge variant | 徽章变体 */
  variant: 'auto' | 'custom';
  /** Callback when model is removed | 移除模型时的回调函数 */
  onRemove: (model: string) => void;
  /** Callback when model name is copied | 复制模型名称时的回调函数 */
  onCopy: (model: string) => void;
  /** Search query for highlighting | 用于高亮的搜索查询 */
  searchQuery: string;
  /** Translation function | 翻译函数 */
  t: (key: string, params?: any) => string;
}
