/**
 * Utility functions for the ModelSelector component
 * ModelSelector 组件的工具函数
 */

/**
 * Parse comma-separated model string into an array
 * 将逗号分隔的模型字符串解析为数组
 * @param modelsStr - Comma-separated string of model names | 逗号分隔的模型名称字符串
 * @returns Array of trimmed, non-empty model names | 修剪后的非空模型名称数组
 */
export function parseModels(modelsStr: string): string[] {
  if (!modelsStr) return [];
  return modelsStr
    .split(',')
    .map(m => m.trim())
    .filter(Boolean);
}

/**
 * Convert model array to comma-separated string
 * 将模型数组转换为逗号分隔的字符串
 * @param models - Array of model names | 模型名称数组
 * @returns Comma-separated string | 逗号分隔的字符串
 */
export function stringifyModels(models: string[]): string {
  return models.join(',');
}

/**
 * Filter models by search query (case-insensitive)
 * 根据搜索查询过滤模型（不区分大小写）
 * @param models - Array of model names | 模型名称数组
 * @param query - Search query string | 搜索查询字符串
 * @returns Filtered array of models | 过滤后的模型数组
 */
export function filterModels(models: string[], query: string): string[] {
  if (!query.trim()) return models;
  const lowerQuery = query.toLowerCase();
  return models.filter(m => m.toLowerCase().includes(lowerQuery));
}
