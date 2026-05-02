# Waterhouse Nature 后台重设计 实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 将 `octopus-dev` 管理后台从“已有 Nature 装饰层的常规后台”升级为统一的 `Waterhouse / 水雾温室` 风格，并在首页、设置、渠道、分组、模型广场、分析中心完成页面级重构。

**架构：** 优先从全局壳层和通用组件层建立新的 Waterhouse 视觉语言，再把该语言映射到重点页面模块。业务数据流、API、路由和虚拟滚动保持不变，重构集中在 `web/src/app/*`、`web/src/components/ui/*`、`web/src/components/nature/*`、`web/src/components/modules/*` 的外观、结构和交互语义上。

**技术栈：** Next.js App Router、React 19、next-intl、TanStack Query、Tailwind CSS、motion/react

---

## 文件结构

### 全局壳层

- 修改：`web/src/app/globals.css`
  - 统一 Waterhouse 设计令牌、背景层、动画关键帧、通用工具类
- 修改：`web/src/app/layout.tsx`
  - 调整全局滤镜、初始加载器和全局环境装饰
- 修改：`web/src/components/app.tsx`
  - 重构主应用壳层、页头与主内容空间关系
- 修改：`web/src/components/common/PageWrapper.tsx`
  - 调整页面段落级入场节奏

### Nature / 通用组件层

- 修改：`web/src/components/nature/BreathingElement.tsx`
- 修改：`web/src/components/nature/MagneticWrapper.tsx`
- 修改：`web/src/components/nature/OrganicBlob.tsx`
- 修改：`web/src/components/nature/ParticleBackground.tsx`
- 修改：`web/src/components/nature/RippleEffect.tsx`
  - 强化 Waterhouse 风格的动效和反馈基础设施
- 修改：`web/src/lib/animations/fluid-transitions.ts`
  - 扩展新的缓动和过渡预设
- 修改：`web/src/components/ui/card.tsx`
- 修改：`web/src/components/ui/button.tsx`
- 修改：`web/src/components/ui/input.tsx`
- 修改：`web/src/components/ui/select.tsx`
- 修改：`web/src/components/ui/switch.tsx`
- 修改：`web/src/components/ui/popover.tsx`
- 修改：`web/src/components/ui/dialog.tsx`
- 修改：`web/src/components/ui/morphing-dialog.tsx`
- 修改：`web/src/components/ui/table.tsx`
- 修改：`web/src/components/ui/accordion.tsx`
- 修改：`web/src/components/ui/alert-dialog.tsx`
- 修改：`web/src/components/ui/badge.tsx`
- 修改：`web/src/components/ui/calendar.tsx`
- 修改：`web/src/components/ui/progress.tsx`
  - 统一通用控件的材质、轮廓和交互反馈

### 导航与工具层

- 修改：`web/src/components/modules/navbar/navbar.tsx`
  - 升级为“导航脊柱”
- 修改：`web/src/components/modules/toolbar/index.tsx`
  - 升级为“雾冠 + 命令舱”编排

### 页面模块层

- 修改：`web/src/components/modules/home/hero.tsx`
- 修改：`web/src/components/modules/home/analytics-overview.tsx`
- 修改：`web/src/components/modules/home/activity.tsx`
- 修改：`web/src/components/modules/home/chart.tsx`
- 修改：`web/src/components/modules/home/rank.tsx`
- 修改：`web/src/components/modules/home/index.tsx`
  - 首页整体重排为“生态仪表盘”
- 修改：`web/src/components/modules/setting/index.tsx`
- 修改：`web/src/components/modules/setting/Appearance.tsx`
- 修改：`web/src/components/modules/setting/Info.tsx`
- 修改：`web/src/components/modules/setting/Account.tsx`
- 修改：`web/src/components/modules/setting/System.tsx`
- 修改：`web/src/components/modules/setting/SemanticCache.tsx`
- 修改：`web/src/components/modules/setting/AIRoute.tsx`
- 修改：`web/src/components/modules/setting/Retry.tsx`
- 修改：`web/src/components/modules/setting/AutoStrategy.tsx`
- 修改：`web/src/components/modules/setting/Log.tsx`
- 修改：`web/src/components/modules/setting/LLMPrice.tsx`
- 修改：`web/src/components/modules/setting/APIKey.tsx`
- 修改：`web/src/components/modules/setting/LLMSync.tsx`
- 修改：`web/src/components/modules/setting/CircuitBreaker.tsx`
- 修改：`web/src/components/modules/setting/Backup.tsx`
- 修改：`web/src/components/modules/setting/RouteGroupDanger.tsx`
  - 设置页整体升级为“生长岛群”
- 修改：`web/src/components/modules/channel/index.tsx`
- 修改：`web/src/components/modules/channel/Card.tsx`
- 修改：`web/src/components/modules/channel/CardContent.tsx`
- 修改：`web/src/components/modules/channel/Create.tsx`
- 修改：`web/src/components/modules/channel/Form.tsx`
  - 渠道页升级为“液态陈列架”
- 修改：`web/src/components/modules/group/index.tsx`
- 修改：`web/src/components/modules/group/Card.tsx`
- 修改：`web/src/components/modules/group/Editor.tsx`
- 修改：`web/src/components/modules/group/ItemList.tsx`
- 修改：`web/src/components/modules/group/Create.tsx`
- 修改：`web/src/components/modules/group/AIRouteButton.tsx`
- 修改：`web/src/components/modules/group/AutoGroupButton.tsx`
  - 分组页升级为“路由温室”
- 修改：`web/src/components/modules/model/index.tsx`
- 修改：`web/src/components/modules/model/Item.tsx`
- 修改：`web/src/components/modules/model/ItemOverlays.tsx`
- 修改：`web/src/components/modules/model/MarketSummary.tsx`
  - 模型广场升级为“雾中标本馆”
- 修改：`web/src/components/modules/analytics/index.tsx`
- 修改：`web/src/components/modules/analytics/Utilization.tsx`
- 修改：`web/src/components/modules/analytics/GroupHealth.tsx`
- 修改：`web/src/components/modules/analytics/Evaluation.tsx`
- 修改：`web/src/components/modules/analytics/shared.tsx`
  - 分析中心升级为“潮汐观测站”

### 文案与测试

- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/zh_hant.json`
- 修改：`web/public/locale/en.json`
  - 补齐可能新增的 Waterhouse 相关文案
- 可选创建：`web/src/components/ui/*.test.ts`
- 可选创建：`web/src/components/modules/*/*.test.ts`
  - 只为新增的纯函数或状态规范补最小测试，不为纯样式重构滥加快照测试

---

### 任务 1：建立 Waterhouse 全局壳层与设计令牌

**文件：**
- 修改：`web/src/app/globals.css`
- 修改：`web/src/app/layout.tsx`
- 修改：`web/src/lib/animations/fluid-transitions.ts`
- 修改：`web/src/components/common/PageWrapper.tsx`

- [ ] **步骤 1：整理现有 Nature 令牌并补充 Waterhouse 变量**

代码目标：

```css
:root {
  --waterhouse-bg-top: oklch(...);
  --waterhouse-bg-bottom: oklch(...);
  --waterhouse-fog: color-mix(in oklch, var(--card) 58%, transparent);
  --waterhouse-fog-strong: color-mix(in oklch, var(--card) 76%, transparent);
  --waterhouse-line: color-mix(in oklch, var(--border) 22%, transparent);
  --waterhouse-highlight: oklch(...);
  --waterhouse-shadow-soft: ...;
  --waterhouse-shadow-deep: ...;
}
```

- [ ] **步骤 2：重写全局背景层和 Waterhouse 工具类**

代码目标：

```css
body {
  background:
    radial-gradient(...),
    radial-gradient(...),
    linear-gradient(180deg, ...);
}

.waterhouse-shell { ... }
.waterhouse-canopy { ... }
.waterhouse-pod { ... }
.waterhouse-island { ... }
.waterhouse-liquid-field { ... }
```

- [ ] **步骤 3：补齐 Waterhouse 过渡与缓动预设**

代码目标：

```ts
export const EASING = {
  ...EASING,
  waterhouseDrift: [0.22, 1, 0.36, 1] as const,
  waterhousePress: [0.34, 1.56, 0.64, 1] as const,
  waterhouseFloat: [0.16, 1, 0.3, 1] as const,
};
```

- [ ] **步骤 4：让 `PageWrapper` 的段落级动画与 Waterhouse 节奏一致**

代码目标：

```ts
transition={{
  duration: 0.65,
  ease: EASING.waterhouseFloat,
  delay: getDiminishingDelay(index),
}}
```

- [ ] **步骤 5：运行前端 lint 确认壳层级改动无语法问题**

运行：`cd web; pnpm lint`
预期：PASS

- [ ] **步骤 6：Commit**

```bash
git add web/src/app/globals.css web/src/app/layout.tsx web/src/lib/animations/fluid-transitions.ts web/src/components/common/PageWrapper.tsx
git commit -m "feat: establish waterhouse global shell"
```

### 任务 2：升级 Nature 基础组件与通用 UI 控件

**文件：**
- 修改：`web/src/components/nature/BreathingElement.tsx`
- 修改：`web/src/components/nature/MagneticWrapper.tsx`
- 修改：`web/src/components/nature/OrganicBlob.tsx`
- 修改：`web/src/components/nature/ParticleBackground.tsx`
- 修改：`web/src/components/nature/RippleEffect.tsx`
- 修改：`web/src/components/ui/card.tsx`
- 修改：`web/src/components/ui/button.tsx`
- 修改：`web/src/components/ui/input.tsx`
- 修改：`web/src/components/ui/select.tsx`
- 修改：`web/src/components/ui/switch.tsx`
- 修改：`web/src/components/ui/popover.tsx`
- 修改：`web/src/components/ui/dialog.tsx`
- 修改：`web/src/components/ui/morphing-dialog.tsx`
- 修改：`web/src/components/ui/table.tsx`
- 修改：`web/src/components/ui/accordion.tsx`
- 修改：`web/src/components/ui/alert-dialog.tsx`
- 修改：`web/src/components/ui/badge.tsx`
- 修改：`web/src/components/ui/calendar.tsx`
- 修改：`web/src/components/ui/progress.tsx`

- [ ] **步骤 1：增强 Nature 基础组件的 Waterhouse 表现力**

代码目标：

```ts
// MagneticWrapper
animate={{
  x: position.x,
  y: position.y,
  scale: isHovered ? scale : 1,
  rotate: isHovered ? position.x * 0.02 : 0,
}}
```

- [ ] **步骤 2：把 `Card` 变成可承载不同岛屿气质的基础容器**

代码目标：

```ts
className={cn(
  "waterhouse-island bg-card/70 ...",
  "rounded-[2rem] border border-border/40 ...",
  className
)}
```

- [ ] **步骤 3：统一按钮、输入框、选择器和开关的液态材质**

代码目标：

```ts
// button/input/select/switch shared visual direction
"bg-background/55 backdrop-blur-md border-border/40 shadow-nature-organic"
"hover:shadow-[var(--waterhouse-shadow-soft)] active:scale-[0.96]"
```

- [ ] **步骤 4：升级弹层和 MorphingDialog 为“水舱弹层”**

代码目标：

```ts
className="rounded-[2.25rem] border border-border/35 bg-background/78 backdrop-blur-xl ..."
```

- [ ] **步骤 5：让表格和次级组件也进入统一语言**

代码目标：

```ts
<div className="waterhouse-pod rounded-[1.75rem] ...">
```

- [ ] **步骤 6：运行前端 lint**

运行：`cd web; pnpm lint`
预期：PASS

- [ ] **步骤 7：Commit**

```bash
git add web/src/components/nature web/src/components/ui web/src/lib/animations/fluid-transitions.ts
git commit -m "feat: upgrade waterhouse nature component system"
```

### 任务 3：重构主应用壳层、导航脊柱与工具冠层

**文件：**
- 修改：`web/src/components/app.tsx`
- 修改：`web/src/components/modules/navbar/navbar.tsx`
- 修改：`web/src/components/modules/toolbar/index.tsx`

- [ ] **步骤 1：把 `AppContainer` 重构为 Waterhouse 壳层**

代码目标：

```tsx
<motion.div className="waterhouse-shell relative mx-auto flex h-dvh max-w-[92rem] ...">
  <NavBar />
  <main className="...">
    <header className="waterhouse-canopy ...">
```

- [ ] **步骤 2：把 `NavBar` 升级为导航脊柱**

代码目标：

```tsx
className="waterhouse-pod fixed ... rounded-[2.5rem] ..."
```

- [ ] **步骤 3：重编排 `Toolbar`，形成雾冠操作带和命令胶囊**

代码目标：

```tsx
className="rounded-[1.4rem] bg-background/42 backdrop-blur-md ..."
```

- [ ] **步骤 4：浏览器确认当前页面导航、页头、工具区已经明显转向 Waterhouse 结构**

运行：使用 in-app browser 打开 `http://localhost:3000/`
预期：能在设置页直接看到导航脊柱、雾冠页头和更柔和的工具操作区

- [ ] **步骤 5：Commit**

```bash
git add web/src/components/app.tsx web/src/components/modules/navbar/navbar.tsx web/src/components/modules/toolbar/index.tsx
git commit -m "feat: rebuild app shell into waterhouse layout"
```

### 任务 4：重构首页为“生态仪表盘”

**文件：**
- 修改：`web/src/components/modules/home/hero.tsx`
- 修改：`web/src/components/modules/home/analytics-overview.tsx`
- 修改：`web/src/components/modules/home/activity.tsx`
- 修改：`web/src/components/modules/home/chart.tsx`
- 修改：`web/src/components/modules/home/rank.tsx`
- 修改：`web/src/components/modules/home/index.tsx`

- [ ] **步骤 1：把 Hero 变成湖面式主视觉**

代码目标：

```tsx
<motion.section className="waterhouse-island rounded-[2.4rem] ...">
```

- [ ] **步骤 2：把分析概览区与次级模块统一到“浮叶 + 观察窗”结构**

代码目标：

```tsx
<div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
```

- [ ] **步骤 3：重排首页整体顺序与节奏**

代码目标：

```tsx
<PageWrapper className="... space-y-7 ...">
```

- [ ] **步骤 4：浏览器验证首页成为全站风格样板页**

运行：在浏览器切到首页
预期：主页比设置页更完整体现 Waterhouse 视觉，且统计信息仍可扫读

- [ ] **步骤 5：Commit**

```bash
git add web/src/components/modules/home
git commit -m "feat: redesign home as waterhouse dashboard"
```

### 任务 5：重构设置页为“生长岛群”

**文件：**
- 修改：`web/src/components/modules/setting/index.tsx`
- 修改：`web/src/components/modules/setting/Appearance.tsx`
- 修改：`web/src/components/modules/setting/Info.tsx`
- 修改：`web/src/components/modules/setting/Account.tsx`
- 修改：`web/src/components/modules/setting/System.tsx`
- 修改：`web/src/components/modules/setting/SemanticCache.tsx`
- 修改：`web/src/components/modules/setting/AIRoute.tsx`
- 修改：`web/src/components/modules/setting/Retry.tsx`
- 修改：`web/src/components/modules/setting/AutoStrategy.tsx`
- 修改：`web/src/components/modules/setting/Log.tsx`
- 修改：`web/src/components/modules/setting/LLMPrice.tsx`
- 修改：`web/src/components/modules/setting/APIKey.tsx`
- 修改：`web/src/components/modules/setting/LLMSync.tsx`
- 修改：`web/src/components/modules/setting/CircuitBreaker.tsx`
- 修改：`web/src/components/modules/setting/Backup.tsx`
- 修改：`web/src/components/modules/setting/RouteGroupDanger.tsx`

- [ ] **步骤 1：重构设置页容器和列节奏**

代码目标：

```tsx
<PageWrapper className="grid gap-5 xl:grid-cols-[1.05fr_0.95fr] ...">
```

- [ ] **步骤 2：将配置卡统一为“生态岛”卡片**

代码目标：

```tsx
className="waterhouse-island rounded-[2rem] ..."
```

- [ ] **步骤 3：把导航顺序、账户设置、AI Route 等高交互块重做为更强的命令培养皿**

代码目标：

```tsx
<div className="rounded-[1.75rem] bg-background/52 ...">
```

- [ ] **步骤 4：浏览器验证设置页的长表单办公感已明显削弱**

运行：浏览器停留在设置页验证
预期：多配置域仍然清晰，但整体不再像传统瀑布白卡后台

- [ ] **步骤 5：Commit**

```bash
git add web/src/components/modules/setting
git commit -m "feat: redesign settings as waterhouse islands"
```

### 任务 6：重构渠道与分组页

**文件：**
- 修改：`web/src/components/modules/channel/index.tsx`
- 修改：`web/src/components/modules/channel/Card.tsx`
- 修改：`web/src/components/modules/channel/CardContent.tsx`
- 修改：`web/src/components/modules/channel/Create.tsx`
- 修改：`web/src/components/modules/channel/Form.tsx`
- 修改：`web/src/components/modules/group/index.tsx`
- 修改：`web/src/components/modules/group/Card.tsx`
- 修改：`web/src/components/modules/group/Editor.tsx`
- 修改：`web/src/components/modules/group/ItemList.tsx`
- 修改：`web/src/components/modules/group/Create.tsx`
- 修改：`web/src/components/modules/group/AIRouteButton.tsx`
- 修改：`web/src/components/modules/group/AutoGroupButton.tsx`

- [ ] **步骤 1：把渠道卡升级为液态陈列架单元**

代码目标：

```tsx
<article className="waterhouse-island rounded-[2rem] ...">
```

- [ ] **步骤 2：把分组卡和编辑器升级为路由温室样式**

代码目标：

```tsx
<section className="waterhouse-pod rounded-[1.8rem] ...">
```

- [ ] **步骤 3：重做渠道/分组创建弹层，使其成为水舱编辑器**

代码目标：

```tsx
className="w-[min(100vw-1rem,56rem)] rounded-[2.4rem] ..."
```

- [ ] **步骤 4：浏览器验证渠道页、分组页、创建弹层都完成语义切换**

运行：在浏览器进入 `渠道`、`分组` 并打开创建弹层
预期：卡片墙、空状态和弹层明显摆脱旧后台观感

- [ ] **步骤 5：Commit**

```bash
git add web/src/components/modules/channel web/src/components/modules/group
git commit -m "feat: redesign channel and group surfaces"
```

### 任务 7：重构模型广场与分析中心

**文件：**
- 修改：`web/src/components/modules/model/index.tsx`
- 修改：`web/src/components/modules/model/Item.tsx`
- 修改：`web/src/components/modules/model/ItemOverlays.tsx`
- 修改：`web/src/components/modules/model/MarketSummary.tsx`
- 修改：`web/src/components/modules/analytics/index.tsx`
- 修改：`web/src/components/modules/analytics/Utilization.tsx`
- 修改：`web/src/components/modules/analytics/GroupHealth.tsx`
- 修改：`web/src/components/modules/analytics/Evaluation.tsx`
- 修改：`web/src/components/modules/analytics/shared.tsx`

- [ ] **步骤 1：把模型广场升级为“雾中标本馆”**

代码目标：

```tsx
className="waterhouse-island rounded-[2rem] ..."
```

- [ ] **步骤 2：把分析中心升级为“潮汐观测站”**

代码目标：

```tsx
<section className="waterhouse-island rounded-[2rem] ...">
```

- [ ] **步骤 3：重做 tabs / range / 图表容器的 Waterhouse 表达**

代码目标：

```tsx
<TabsList className="rounded-full bg-background/48 ...">
```

- [ ] **步骤 4：浏览器验证模型广场和分析中心的可读性仍然成立**

运行：浏览器进入 `模型广场`、`分析中心`
预期：风格明显统一，同时模型卡和图表数据依然可扫读

- [ ] **步骤 5：Commit**

```bash
git add web/src/components/modules/model web/src/components/modules/analytics
git commit -m "feat: redesign model market and analytics"
```

### 任务 8：补齐文案与最终验证

**文件：**
- 修改：`web/public/locale/zh_hans.json`
- 修改：`web/public/locale/zh_hant.json`
- 修改：`web/public/locale/en.json`

- [ ] **步骤 1：补齐 Waterhouse 重构新增文案**

代码目标：

```json
{
  "common": {
    "dialog": {
      "open": "...",
      "close": "..."
    }
  }
}
```

- [ ] **步骤 2：运行前端 lint**

运行：`cd web; pnpm lint`
预期：PASS

- [ ] **步骤 3：运行前端构建**

运行：`cd web; $env:NEXT_PUBLIC_APP_VERSION='dev'; pnpm build`
预期：PASS

- [ ] **步骤 4：在浏览器逐页回看重点页面**

运行：打开并检查
1. `主页`
2. `设置`
3. `渠道`
4. `分组`
5. `模型广场`
6. `分析中心`

预期：
1. 风格统一
2. 动效不过量
3. 操作仍然易懂
4. 页面不再像传统后台模板

- [ ] **步骤 5：Commit**

```bash
git add web/public/locale web/src
git commit -m "feat: finalize waterhouse nature redesign"
```
