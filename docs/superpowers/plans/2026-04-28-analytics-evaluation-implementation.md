# Analytics Evaluation 最小接入实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 `Analytics -> Evaluation` 从静态说明页升级为真实入口与运行态摘要页，复用现有 `Group Test` 与 `AI Route` 能力，不新增后端接口、数据库表或持久化归档。

**架构：** 本次改动只触达前端。通过一个极小的 `sessionStorage` 共享 helper 让 `Group` 页面与 `Evaluation` 页面共享最近任务 ID；`Evaluation` 再复用现有 `useGroupList`、`useGenerateAIRouteProgress`、`useGroupTestProgress` 组装三张入口卡和两个运行态摘要块。UI 继续沿用现有 `Analytics` 卡片、按钮和空态视觉语言，不新增独立设计体系。

**技术栈：** Next.js 静态导出前端、React 19、React Query、`next-intl`、Zustand 导航状态、现有 `Button` / 卡片样式 / `lucide-react` 图标体系。

---

## 文件结构

- 创建：`web/src/components/modules/group/task-storage.ts`
- 修改：`web/src/components/modules/group/AIRouteButton.tsx`
- 修改：`web/src/components/modules/group/Card.tsx`
- 修改：`web/src/api/endpoints/analytics.ts`
- 修改：`web/src/components/modules/analytics/Evaluation.tsx`
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/zh_hant.json`
- 修改：`web/public/locale/en.json`

## 实施约束

- 不新增 `internal/server/handlers/*`、`internal/op/*`、`internal/model/*` 后端文件。
- 不新增 benchmark 历史归档表、结果表、后台任务表。
- 不把 `Group` 页完整复制到 `Evaluation`；入口按钮只负责跳转到 `group` 一级页。
- `AI Route` 和 `Group Test` 只展示“当前会话最近任务”的运行态，不设计长期留存。
- 当前工作树已是脏树，执行时不要 `git add .`、不要回滚无关文件；如需提交，只 stage 本计划涉及文件。

### 任务 1：抽出任务存储 helper，并让 Group Test 支持跨页恢复

**文件：**

- 创建：`web/src/components/modules/group/task-storage.ts`
- 修改：`web/src/components/modules/group/AIRouteButton.tsx`
- 修改：`web/src/components/modules/group/Card.tsx`

- [ ] **步骤 1：新增共享任务存储 helper，统一 AI Route / Group Test 的 sessionStorage 读写**

```ts
// web/src/components/modules/group/task-storage.ts
import type { AIRouteScope } from '@/api/endpoints/group';

export type StoredAIRouteTask = {
    id: string;
    scope: AIRouteScope;
    groupId?: number;
};

export type StoredGroupTestTask = {
    id: string;
    groupId: number;
};

const AI_ROUTE_PROGRESS_STORAGE_KEY = 'octopus.ai-route-progress';
const GROUP_TEST_PROGRESS_STORAGE_KEY = 'octopus.group-test-progress';

function readSessionJSON<T>(key: string): T | null {
    if (typeof window === 'undefined') {
        return null;
    }
    const raw = window.sessionStorage.getItem(key);
    if (!raw) {
        return null;
    }
    try {
        return JSON.parse(raw) as T;
    } catch {
        return null;
    }
}

export function readStoredAIRouteTask() {
    return readSessionJSON<StoredAIRouteTask>(AI_ROUTE_PROGRESS_STORAGE_KEY);
}

export function writeStoredAIRouteTask(task: StoredAIRouteTask) {
    if (typeof window !== 'undefined') {
        window.sessionStorage.setItem(AI_ROUTE_PROGRESS_STORAGE_KEY, JSON.stringify(task));
    }
}

export function clearStoredAIRouteTask(id?: string) {
    const current = readStoredAIRouteTask();
    if (!current || (id && current.id !== id) || typeof window === 'undefined') {
        return;
    }
    window.sessionStorage.removeItem(AI_ROUTE_PROGRESS_STORAGE_KEY);
}

export function matchesStoredAIRouteTask(task: StoredAIRouteTask | null, scope: AIRouteScope, groupId: number) {
    if (!task || task.scope !== scope) {
        return false;
    }
    return scope === 'group' ? task.groupId === groupId && groupId > 0 : true;
}

export function readStoredGroupTestTask() {
    return readSessionJSON<StoredGroupTestTask>(GROUP_TEST_PROGRESS_STORAGE_KEY);
}

export function writeStoredGroupTestTask(task: StoredGroupTestTask) {
    if (typeof window !== 'undefined') {
        window.sessionStorage.setItem(GROUP_TEST_PROGRESS_STORAGE_KEY, JSON.stringify(task));
    }
}

export function clearStoredGroupTestTask(id?: string) {
    const current = readStoredGroupTestTask();
    if (!current || (id && current.id !== id) || typeof window === 'undefined') {
        return;
    }
    window.sessionStorage.removeItem(GROUP_TEST_PROGRESS_STORAGE_KEY);
}

export function matchesStoredGroupTestTask(task: StoredGroupTestTask | null, groupId: number) {
    return Boolean(task && task.groupId === groupId && groupId > 0);
}
```

- [ ] **步骤 2：让 AI Route 按钮改用共享 helper，并保留最近终态任务供 Evaluation 摘要读取**

```tsx
// web/src/components/modules/group/AIRouteButton.tsx
import {
    clearStoredAIRouteTask,
    matchesStoredAIRouteTask,
    readStoredAIRouteTask,
    writeStoredAIRouteTask,
} from './task-storage';

const [currentProgressId, setCurrentProgressId] = useState<string | null>(() => {
    const storedTask = readStoredAIRouteTask();
    return matchesStoredAIRouteTask(storedTask, scope, resolvedGroupID) ? storedTask?.id ?? null : null;
});

useEffect(() => {
    if (!progress?.id || handledProgressRef.current === progress.id || !isGenerateAIRouteTerminal(progress)) {
        return;
    }

    handledProgressRef.current = progress.id;

    // 不在终态立刻清理 sessionStorage，保留最近一次任务给 Evaluation 页读取
    if (loadingToastRef.current !== null) {
        toast.dismiss(loadingToastRef.current);
        loadingToastRef.current = null;
    }

    // 其余 success / failed / timeout 提示逻辑保持原样
}, [isGroupScope, onSuccess, progress, queryClient, t]);

if (statusCode === 404) {
    clearStoredAIRouteTask(currentProgressId);
    queueMicrotask(() => setCurrentProgressId(null));
}
```

- [ ] **步骤 3：让 Group 卡片把最近 test task 持久化，并在回到同一分组时恢复进度**

```tsx
// web/src/components/modules/group/Card.tsx
import {
    clearStoredGroupTestTask,
    matchesStoredGroupTestTask,
    readStoredGroupTestTask,
    writeStoredGroupTestTask,
} from './task-storage';

const [currentTestId, setCurrentTestId] = useState<string | null>(() => {
    if (!group.id) {
        return null;
    }
    const storedTask = readStoredGroupTestTask();
    return matchesStoredGroupTestTask(storedTask, group.id) ? storedTask?.id ?? null : null;
});

const handleTestGroup = useCallback(() => {
    if (!group.id) return;
    handledTestCompletionRef.current = null;
    testGroup.mutate(group.id, {
        onSuccess: (progress) => {
            setCurrentTestId(progress.id);
            writeStoredGroupTestTask({ id: progress.id, groupId: group.id });
        },
    });
}, [group.id, t, testGroup]);

useEffect(() => {
    if (!currentTestId || !testProgressQuery.error) {
        return;
    }

    const error = testProgressQuery.error;
    const statusCode = error && typeof error === 'object' && 'code' in error && typeof error.code === 'number'
        ? error.code
        : undefined;

    if (statusCode === 404) {
        clearStoredGroupTestTask(currentTestId);
        queueMicrotask(() => setCurrentTestId(null));
    }
}, [currentTestId, testProgressQuery.error]);
```

- [ ] **步骤 4：运行前端静态检查，确认共享 helper 没引入循环依赖或 unused import**

运行：`cd web && pnpm lint`

预期：PASS；`AIRouteButton.tsx`、`Card.tsx`、`task-storage.ts` 无 ESLint 报错。

### 任务 2：在 analytics endpoint 层封装 Evaluation 运行态摘要 hook

**文件：**

- 修改：`web/src/api/endpoints/analytics.ts`

- [ ] **步骤 1：为 Evaluation 定义前端专用摘要结构，不新增任何后端请求类型**

```ts
// web/src/api/endpoints/analytics.ts
import { useEffect, useState } from 'react';
import {
    type GenerateAIRouteProgress,
    type GroupTestProgress,
    useGenerateAIRouteProgress,
    useGroupList,
    useGroupTestProgress,
} from './group';
import {
    clearStoredAIRouteTask,
    clearStoredGroupTestTask,
    readStoredAIRouteTask,
    readStoredGroupTestTask,
    type StoredAIRouteTask,
    type StoredGroupTestTask,
} from '@/components/modules/group/task-storage';

export interface AnalyticsEvaluationRuntime {
    groupCount: number;
    hasGroups: boolean;
    aiRouteTask: StoredAIRouteTask | null;
    groupTestTask: StoredGroupTestTask | null;
    aiRouteProgress: GenerateAIRouteProgress | null;
    groupTestProgress: GroupTestProgress | null;
    isLoading: boolean;
}
```

- [ ] **步骤 2：实现 `useAnalyticsEvaluationRuntime`，把 group list 与两个现有 progress hook 组装在一起**

```ts
export function useAnalyticsEvaluationRuntime(): AnalyticsEvaluationRuntime {
    const { data: groups = [], isLoading: isGroupsLoading } = useGroupList();
    const [aiRouteTask, setAiRouteTask] = useState<StoredAIRouteTask | null>(null);
    const [groupTestTask, setGroupTestTask] = useState<StoredGroupTestTask | null>(null);

    useEffect(() => {
        const syncFromStorage = () => {
            setAiRouteTask(readStoredAIRouteTask());
            setGroupTestTask(readStoredGroupTestTask());
        };

        syncFromStorage();
        window.addEventListener('focus', syncFromStorage);
        return () => window.removeEventListener('focus', syncFromStorage);
    }, []);

    const aiRouteProgressQuery = useGenerateAIRouteProgress(aiRouteTask?.id ?? null);
    const groupTestProgressQuery = useGroupTestProgress(groupTestTask?.id ?? null);

    useEffect(() => {
        const error = aiRouteProgressQuery.error;
        const statusCode = error && typeof error === 'object' && 'code' in error && typeof error.code === 'number'
            ? error.code
            : undefined;
        if (statusCode === 404 && aiRouteTask?.id) {
            clearStoredAIRouteTask(aiRouteTask.id);
            setAiRouteTask(null);
        }
    }, [aiRouteProgressQuery.error, aiRouteTask?.id]);

    useEffect(() => {
        const error = groupTestProgressQuery.error;
        const statusCode = error && typeof error === 'object' && 'code' in error && typeof error.code === 'number'
            ? error.code
            : undefined;
        if (statusCode === 404 && groupTestTask?.id) {
            clearStoredGroupTestTask(groupTestTask.id);
            setGroupTestTask(null);
        }
    }, [groupTestProgressQuery.error, groupTestTask?.id]);

    return {
        groupCount: groups.length,
        hasGroups: groups.length > 0,
        aiRouteTask,
        groupTestTask,
        aiRouteProgress: aiRouteProgressQuery.data ?? null,
        groupTestProgress: groupTestProgressQuery.data ?? null,
        isLoading: isGroupsLoading || aiRouteProgressQuery.isLoading || groupTestProgressQuery.isLoading,
    };
}
```

- [ ] **步骤 3：运行静态检查，确认 hook 组合没有违反 React hooks 规则**

运行：`cd web && pnpm lint`

预期：PASS；`analytics.ts` 中没有 hooks conditionally called、没有类型未导出错误。

### 任务 3：重写 Evaluation 页面，补齐入口卡、摘要区和多语文案

**文件：**

- 修改：`web/src/components/modules/analytics/Evaluation.tsx`
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/zh_hant.json`
- 修改：`web/public/locale/en.json`

- [ ] **步骤 1：先扩展国际化文案，确保卡片标题、按钮、空态和状态文本都有稳定 key**

```json
// web/public/locale/zh_hans.json
{
  "analytics": {
    "evaluation": {
      "title": "评估",
      "description": "聚合现有分组检测与 AI 路由分析入口，当前版本只展示运行态摘要，不新增 benchmark 历史归档。",
      "actions": {
        "openGroupTest": "前往分组检测",
        "openAIRoute": "前往路由分析"
      },
      "availability": {
        "title": "分组可用性检测",
        "description": "检测动作仍在分组卡片内触发，这里只提供统一入口和最近状态摘要。",
        "hint": "当前共有 {count} 个分组可进入检测流程。",
        "empty": "当前没有可检测分组"
      },
      "aiRoute": {
        "title": "AI 路由分析",
        "description": "复用现有 AI Route 入口和进度能力，在这里聚合最近任务状态。",
        "empty": "当前没有 AI 路由任务"
      },
      "archive": {
        "title": "历史归档",
        "description": "当前版本暂无 benchmark 历史归档，这是有意留白，不是加载失败。"
      },
      "summary": {
        "title": "最近评测状态",
        "aiRoute": "最近 AI 路由任务",
        "groupTest": "最近分组检测任务",
        "empty": "当前没有可展示的运行态任务"
      }
    }
  }
}
```

- [ ] **步骤 2：把 `Evaluation.tsx` 从静态映射页改为真实入口卡 + 运行态摘要**

```tsx
// web/src/components/modules/analytics/Evaluation.tsx
'use client';

import { Activity, Archive, Route, ArrowRight, Clock3 } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Button } from '@/components/ui/button';
import { useNavStore } from '@/components/modules/navbar';
import { useAnalyticsEvaluationRuntime } from '@/api/endpoints/analytics';

export function Evaluation() {
    const t = useTranslations('analytics');
    const { setActiveItem } = useNavStore();
    const runtime = useAnalyticsEvaluationRuntime();
    const aiRoute = runtime.aiRouteProgress;
    const groupTest = runtime.groupTestProgress;
    const passedCount = (groupTest?.results ?? []).filter((item) => item.passed).length;
    const failedCount = (groupTest?.results ?? []).filter((item) => !item.passed).length;

    return (
        <section className="rounded-3xl border border-card-border bg-card p-5 text-card-foreground custom-shadow">
            <div className="mb-4 space-y-1">
                <h3 className="text-base font-semibold">{t('evaluation.title')}</h3>
                <p className="text-sm leading-6 text-muted-foreground">{t('evaluation.description')}</p>
            </div>

            <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
                <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                        <Activity className="h-4 w-4" />
                    </div>
                    <h4 className="mt-4 text-sm font-semibold">{t('evaluation.availability.title')}</h4>
                    <p className="mt-2 text-sm leading-6 text-muted-foreground">{t('evaluation.availability.description')}</p>
                    <p className="mt-2 text-sm text-muted-foreground">
                        {runtime.hasGroups
                            ? t('evaluation.availability.hint', { count: runtime.groupCount })
                            : t('evaluation.availability.empty')}
                    </p>
                    <Button className="mt-4 rounded-xl" onClick={() => setActiveItem('group')}>
                        {t('evaluation.actions.openGroupTest')}
                        <ArrowRight className="size-4" />
                    </Button>
                </article>

                <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                        <Route className="h-4 w-4" />
                    </div>
                    <h4 className="mt-4 text-sm font-semibold">{t('evaluation.aiRoute.title')}</h4>
                    <p className="mt-2 text-sm leading-6 text-muted-foreground">{t('evaluation.aiRoute.description')}</p>
                    <Button className="mt-4 rounded-xl" onClick={() => setActiveItem('group')}>
                        {t('evaluation.actions.openAIRoute')}
                        <ArrowRight className="size-4" />
                    </Button>
                </article>

                <article className="rounded-2xl border border-border/60 bg-background/70 p-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-primary/10 text-primary">
                        <Archive className="h-4 w-4" />
                    </div>
                    <h4 className="mt-4 text-sm font-semibold">{t('evaluation.archive.title')}</h4>
                    <p className="mt-2 text-sm leading-6 text-muted-foreground">{t('evaluation.archive.description')}</p>
                </article>
            </div>

            <div className="mt-4 space-y-4">
                <div className="rounded-2xl border border-dashed border-border bg-background/60 p-4">
                    <p className="text-sm font-semibold">{t('evaluation.summary.title')}</p>
                </div>

                <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
                    <article className="rounded-2xl border border-border/60 bg-background/60 p-4">
                        <div className="flex items-center gap-2 text-sm font-semibold">
                            <Clock3 className="size-4 text-primary" />
                            {t('evaluation.summary.aiRoute')}
                        </div>
                        {aiRoute ? (
                            <div className="mt-3 space-y-2 text-sm text-muted-foreground">
                                <p>{aiRoute.status ?? aiRoute.current_step ?? 'queued'}</p>
                                <p>{aiRoute.completed_batches}/{aiRoute.total_batches}</p>
                                {aiRoute.result_ready && aiRoute.result ? (
                                    <p>{aiRoute.result.group_count} / {aiRoute.result.route_count} / {aiRoute.result.item_count}</p>
                                ) : null}
                            </div>
                        ) : (
                            <p className="mt-3 text-sm text-muted-foreground">{t('evaluation.aiRoute.empty')}</p>
                        )}
                    </article>

                    <article className="rounded-2xl border border-border/60 bg-background/60 p-4">
                        <div className="flex items-center gap-2 text-sm font-semibold">
                            <Clock3 className="size-4 text-primary" />
                            {t('evaluation.summary.groupTest')}
                        </div>
                        {groupTest ? (
                            <div className="mt-3 space-y-2 text-sm text-muted-foreground">
                                <p>{groupTest.completed}/{groupTest.total}</p>
                                <p>{passedCount} / {failedCount}</p>
                            </div>
                        ) : (
                            <p className="mt-3 text-sm text-muted-foreground">{t('evaluation.summary.empty')}</p>
                        )}
                    </article>
                </div>
            </div>
        </section>
    );
}
```

- [ ] **步骤 3：运行前端构建验证，确认新的 Evaluation 页面能通过静态导出**

运行：`cd web && pnpm lint`

预期：PASS

运行：`cd web && $env:NEXT_PUBLIC_APP_VERSION='v1.4.2'; pnpm build`

预期：PASS；`web/out` 成功生成，`Evaluation` 相关组件无类型错误。

### 任务 4：把构建产物嵌入本地服务并完成人工验收

**文件：**

- 修改：`static/out/*`（由构建产物覆盖，不手写）

- [ ] **步骤 1：把静态导出产物复制到嵌入目录**

运行：`Copy-Item "F:\codecil\octopus-dev\web\out\*" "F:\codecil\octopus-dev\static\out\" -Recurse -Force`

预期：`static/out` 中的前端产物被更新，后端重新启动后会加载新的 Evaluation 页面。

- [ ] **步骤 2：启动本地服务用于验收**

运行：`go run main.go start`

预期：服务启动成功，`http://127.0.0.1:8080/` 可访问。

- [ ] **步骤 3：按最小验收路径人工验证 Evaluation**

运行：

1. 打开 `Analytics -> Evaluation`
2. 确认页面展示三张卡：`分组可用性检测`、`AI 路由分析`、`历史归档`
3. 点击 `前往分组检测` 与 `前往路由分析`，都应跳到 `group`
4. 在 `group` 页发起一次 `AI Route`，切回 `Evaluation`，确认摘要区能看到状态、批次进度和终态结果
5. 在任意分组发起一次 `可用性检测`，切回 `Evaluation`，确认摘要区能看到 `completed/total`、`passed/failed`
6. 在没有分组或没有任务的情况下，空态文案明确，不出现报错卡或假按钮

预期：页面布局、间距、按钮圆角和卡片密度与现有 `Analytics` / `Ops` 保持一致；没有新增后端接口依赖。

## 自检

### 规格覆盖度

- “三个功能卡片”由任务 3 步骤 1-2 完成。
- “最近 AI Route / Group Test 摘要”由任务 1 步骤 2-3 与任务 2 步骤 1-2、任务 3 步骤 2 共同完成。
- “不新增后端接口/存储”由文件结构与实施约束锁死，只改前端文件。
- “入口按钮只跳转到 group 页面”由任务 3 步骤 2 中 `setActiveItem('group')` 实现。
- “有任务但 progress 丢失时按无活跃任务处理”由任务 1 步骤 2-3 和任务 2 步骤 2 中的 `404 -> clear storage` 实现。
- “UI 保持一致”由任务 3 步骤 2 与任务 4 步骤 3 的人工验收覆盖。

### 占位符扫描

- 已检查全文，无 `TODO`、`TBD`、`待定`、`后续补充`、`类似任务 N` 之类占位语。
- 每个代码改动步骤都给出了具体文件、代码骨架和验证命令。

### 类型一致性

- 任务存储统一使用 `StoredAIRouteTask`、`StoredGroupTestTask`。
- Evaluation 聚合 hook 统一命名为 `useAnalyticsEvaluationRuntime`。
- 页面摘要统一消费 `aiRouteProgress`、`groupTestProgress`，不再混用局部状态名。
