# 首页分析概览整合 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 让首页承载分析概览和更完整的视觉总览，同时移除分析中心里的概述标签。

**架构：** 复用现有首页模块和 `/api/v1/analytics/overview` 查询，在首页新增 Hero 与概览区，并通过 `home/store.ts` 增加独立的概览范围状态。分析中心只做 tab 收敛，不改后端接口。

**技术栈：** Next.js App Router、React 19、next-intl、TanStack Query、Zustand、Tailwind

---

### 任务 1：补齐首页概览状态的测试保护

**文件：**
- 创建：`web/src/components/modules/home/store.test.ts`
- 修改：`web/src/components/modules/home/store.ts`

- [ ] **步骤 1：先写失败测试**

```ts
import assert from 'node:assert/strict';
import test from 'node:test';
import { normalizeOverviewRange } from './store.ts';

test('normalizeOverviewRange defaults to 7d when value is invalid', () => {
    assert.equal(normalizeOverviewRange('unexpected'), '7d');
});
```

- [ ] **步骤 2：运行测试验证失败**

运行：`node --test web/src/components/modules/home/store.test.ts`
预期：FAIL，报错 `normalizeOverviewRange` 未导出或不存在。

- [ ] **步骤 3：补最小实现**

```ts
export type OverviewRange = '7d' | '30d' | '90d';
const OVERVIEW_RANGES: readonly OverviewRange[] = ['7d', '30d', '90d'];

export function normalizeOverviewRange(value: string | null | undefined): OverviewRange {
    return OVERVIEW_RANGES.includes(value as OverviewRange) ? (value as OverviewRange) : '7d';
}
```

- [ ] **步骤 4：再次运行测试验证通过**

运行：`node --test web/src/components/modules/home/store.test.ts`
预期：PASS

### 任务 2：实现首页 Hero 与分析概览区

**文件：**
- 创建：`web/src/components/modules/home/hero.tsx`
- 创建：`web/src/components/modules/home/analytics-overview.tsx`
- 修改：`web/src/components/modules/home/index.tsx`
- 修改：`web/src/components/modules/home/activity.tsx`
- 修改：`web/src/components/modules/home/store.ts`

- [ ] **步骤 1：新增首页 Hero**
- [ ] **步骤 2：新增首页分析概览区并接入 overview range**
- [ ] **步骤 3：重排首页模块顺序**
- [ ] **步骤 4：为热力图补标题与说明**

### 任务 3：收敛分析中心 tab

**文件：**
- 修改：`web/src/components/modules/analytics/index.tsx`

- [ ] **步骤 1：移除 `overview` tab 与默认值**
- [ ] **步骤 2：保留范围切换给其余分析模块使用**

### 任务 4：补齐文案

**文件：**
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/zh_hant.json`
- 修改：`web/public/locale/en.json`

- [ ] **步骤 1：新增首页 Hero / 分析概览 / 活跃热力图说明文案**
- [ ] **步骤 2：移除分析中心概述依赖后保持页面其余文案完整**

### 任务 5：验证

**文件：**
- 无代码文件新增

- [ ] **步骤 1：运行首页 store 测试**
运行：`node --test web/src/components/modules/home/store.test.ts`

- [ ] **步骤 2：运行 i18n 校验**
运行：`pnpm test:i18n`

- [ ] **步骤 3：运行前端 lint**
运行：`pnpm lint`

- [ ] **步骤 4：运行前端构建**
运行：`$env:NEXT_PUBLIC_APP_VERSION='dev'; pnpm build`

- [ ] **步骤 5：浏览器回看首页与分析中心**
