# Fuploader 颜色系统文档

## 概述

本文档描述了 Fuploader 前端项目的统一颜色系统。所有颜色都通过 CSS 变量管理，支持深色/浅色双主题无缝切换。

## 文件结构

```
src/styles/
├── variables.css   # CSS 变量定义（主题色彩系统）
├── theme.css       # Element Plus 组件主题覆盖
├── global.css      # 全局样式和工具类
└── style.css       # 根样式（字体定义）
```

## 颜色变量分类

### 1. 基础颜色 - 背景色

| 变量名 | 深色主题 | 浅色主题 | 用途 |
|--------|----------|----------|------|
| `--bg-primary` | `#0f0f0f` | `#ffffff` | 主背景色 |
| `--bg-secondary` | `#1a1a1a` | `#ffffff` | 次要背景色 |
| `--bg-tertiary` | `#252525` | `#f5f5f5` | 第三层背景色 |
| `--bg-card` | `#1e1e1e` | `#ffffff` | 卡片背景色 |
| `--bg-hover` | `#2d2d2d` | `#f0f0f0` | 悬停背景色 |
| `--bg-input` | `#2a2a2a` | `#ffffff` | 输入框背景色 |
| `--bg-overlay` | `rgba(0,0,0,0.7)` | `rgba(0,0,0,0.5)` | 遮罩层背景 |

### 2. 基础颜色 - 文字色

| 变量名 | 深色主题 | 浅色主题 | 用途 |
|--------|----------|----------|------|
| `--text-primary` | `#ffffff` | `#000000` | 主要文字 |
| `--text-secondary` | `#e0e0e0` | `#333333` | 次要文字 |
| `--text-tertiary` | `#a0a0a0` | `#666666` | 第三层文字 |
| `--text-muted` | `#707070` | `#999999` | 弱化文字 |
| `--text-inverse` | `#000000` | `#ffffff` | 反色文字 |

### 3. 基础颜色 - 边框色

| 变量名 | 深色主题 | 浅色主题 | 用途 |
|--------|----------|----------|------|
| `--border-color` | `#404040` | `#e0e0e0` | 主要边框 |
| `--border-light` | `#555555` | `#d0d0d0` | 浅色边框 |
| `--border-hover` | `#666666` | `#c0c0c0` | 悬停边框 |

### 4. 主色调 - 品牌色

| 变量名 | 深色主题 | 浅色主题 | 用途 |
|--------|----------|----------|------|
| `--primary-color` | `#0ea5e9` | `#0ea5e9` | 主色调 |
| `--primary-light` | `#38bdf8` | `#38bdf8` | 主色浅色 |
| `--primary-dark` | `#0284c7` | `#0284c7` | 主色深色 |
| `--primary-gradient` | `linear-gradient(...)` | `linear-gradient(...)` | 主色渐变 |

### 5. 选中色

| 变量名 | 深色主题 | 浅色主题 | 用途 |
|--------|----------|----------|------|
| `--selected-bg` | `rgba(14, 165, 233, 0.15)` | `rgba(14, 165, 233, 0.1)` | 选中背景 |
| `--selected-text` | `#38bdf8` | `#0369a1` | 选中文字 |

### 6. 强调色

| 变量名 | 深色主题 | 浅色主题 | 用途 |
|--------|----------|----------|------|
| `--accent-color` | `#666666` | `#666666` | 强调色 |
| `--accent-light` | `#888888` | `#888888` | 强调色浅色 |

### 7. 功能色 - 成功 (Success)

| 变量名 | 值 | 用途 |
|--------|-----|------|
| `--success-color` | `#10b981` | 成功状态 |
| `--success-light` | `#34d399` | 成功浅色 |
| `--success-dark` | `#059669` | 成功深色 |
| `--success-bg` | `rgba(16,185,129,0.1)` | 成功背景 |
| `--success-border` | `rgba(16,185,129,0.3)` | 成功边框 |

### 8. 功能色 - 警告 (Warning)

| 变量名 | 值 | 用途 |
|--------|-----|------|
| `--warning-color` | `#f59e0b` | 警告状态 |
| `--warning-light` | `#fbbf24` | 警告浅色 |
| `--warning-dark` | `#d97706` | 警告深色 |
| `--warning-bg` | `rgba(245,158,11,0.1)` | 警告背景 |
| `--warning-border` | `rgba(245,158,11,0.3)` | 警告边框 |

### 9. 功能色 - 错误/危险 (Error/Danger)

| 变量名 | 值 | 用途 |
|--------|-----|------|
| `--error-color` | `#ef4444` | 错误状态 |
| `--error-light` | `#f87171` | 错误浅色 |
| `--error-dark` | `#dc2626` | 错误深色 |
| `--error-bg` | `rgba(239,68,68,0.1)` | 错误背景 |
| `--error-border` | `rgba(239,68,68,0.3)` | 错误边框 |

### 10. 功能色 - 信息 (Info)

| 变量名 | 值 | 用途 |
|--------|-----|------|
| `--info-color` | `#0ea5e9` | 信息状态 |
| `--info-light` | `#38bdf8` | 信息浅色 |
| `--info-dark` | `#0284c7` | 信息深色 |
| `--info-bg` | `rgba(14, 165, 233, 0.15)` | 信息背景 |
| `--info-border` | `rgba(14, 165, 233, 0.3)` | 信息边框 |

### 11. 平台品牌色

| 变量名 | 值 | 平台 |
|--------|-----|------|
| `--platform-douyin` | `#000000` | 抖音 |
| `--platform-tencent` | `#07C160` | 视频号 |
| `--platform-kuaishou` | `#FF6600` | 快手 |
| `--platform-tiktok` | `#000000` | TikTok |
| `--platform-bilibili` | `#FB7299` | Bilibili |
| `--platform-xiaohongshu` | `#FF2442` | 小红书 |
| `--platform-baijiahao` | `#2932E1` | 百家号 |

### 12. 阴影

| 变量名 | 深色主题 | 浅色主题 | 用途 |
|--------|----------|----------|------|
| `--shadow-color` | `rgba(0,0,0,0.3)` | `rgba(0,0,0,0.08)` | 阴影基础色 |
| `--shadow-sm` | `0 1px 2px ...` | `0 1px 2px ...` | 小阴影 |
| `--shadow-md` | `0 4px 6px ...` | `0 4px 6px ...` | 中阴影 |
| `--shadow-lg` | `0 10px 15px ...` | `0 10px 15px ...` | 大阴影 |
| `--shadow-card` | `0 2px 12px ...` | `0 2px 12px ...` | 卡片阴影 |
| `--shadow-card-hover` | `0 4px 20px ...` | `0 4px 20px ...` | 卡片悬停阴影 |
| `--shadow-glow` | `0 0 20px ...` | `0 0 20px ...` | 发光效果 |

### 13. 间距

| 变量名 | 值 | 用途 |
|--------|-----|------|
| `--spacing-xs` | `4px` | 超小间距 |
| `--spacing-sm` | `8px` | 小间距 |
| `--spacing-md` | `16px` | 中间距 |
| `--spacing-lg` | `24px` | 大间距 |
| `--spacing-xl` | `32px` | 超大间距 |

### 14. 圆角

| 变量名 | 值 | 用途 |
|--------|-----|------|
| `--radius-sm` | `6px` | 小圆角 |
| `--radius-md` | `8px` | 中圆角 |
| `--radius-lg` | `12px` | 大圆角 |
| `--radius-xl` | `16px` | 超大圆角 |
| `--radius-full` | `9999px` | 全圆角 |

### 15. 过渡动画

| 变量名 | 值 | 用途 |
|--------|-----|------|
| `--transition-fast` | `0.15s ease` | 快速过渡 |
| `--transition-base` | `0.2s ease` | 基础过渡 |
| `--transition-slow` | `0.3s ease` | 慢速过渡 |

## 使用指南

### 在 Vue 组件中使用

```vue
<template>
  <div class="my-component">
    <h1 class="title">标题</h1>
    <p class="description">描述文字</p>
  </div>
</template>

<style scoped>
.my-component {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: var(--spacing-md);
}

.title {
  color: var(--text-primary);
  font-size: 18px;
}

.description {
  color: var(--text-secondary);
  font-size: 14px;
}
</style>
```

### 状态样式示例

```css
/* 成功状态 */
.success-message {
  background: var(--success-bg);
  color: var(--success-color);
  border: 1px solid var(--success-border);
}

/* 警告状态 */
.warning-message {
  background: var(--warning-bg);
  color: var(--warning-color);
  border: 1px solid var(--warning-border);
}

/* 错误状态 */
.error-message {
  background: var(--error-bg);
  color: var(--error-color);
  border: 1px solid var(--error-border);
}

/* 信息状态 */
.info-message {
  background: var(--info-bg);
  color: var(--info-color);
  border: 1px solid var(--info-border);
}
```

### 卡片样式示例

```css
.card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-card);
  transition: all var(--transition-slow);
}

.card:hover {
  box-shadow: var(--shadow-card-hover);
  transform: translateY(-2px);
}
```

## 主题切换

主题切换通过 `data-theme` 属性实现：

```javascript
// 切换到浅色主题
document.documentElement.setAttribute('data-theme', 'light');

// 切换到深色主题（默认）
document.documentElement.setAttribute('data-theme', 'dark');
```

项目使用 Pinia 管理主题状态，详见 `src/stores/theme.ts`。

## 最佳实践

1. **始终使用 CSS 变量**：不要直接使用硬编码颜色值
2. **使用语义化变量名**：根据用途选择合适的变量，而不是根据颜色值
3. **功能色使用完整色阶**：使用 `-bg`、`-border` 等变体保持视觉一致性
4. **测试双主题**：修改颜色后务必在深色和浅色主题下都进行测试
5. **保持对比度**：确保文字和背景之间有足够的对比度

## 维护指南

### 添加新颜色

1. 在 `variables.css` 中添加新变量
2. 确保在 `[data-theme="light"]` 中也定义对应的浅色主题值
3. 更新本文档

### 修改现有颜色

1. 在 `variables.css` 中修改变量值
2. 检查 `theme.css` 中是否有硬编码的相同颜色需要同步修改
3. 检查所有 Vue 组件是否使用了该变量
4. 在双主题下测试效果

## 变更日志

### 2024-XX-XX
- 初始建立颜色系统
- 定义完整的 CSS 变量体系
- 替换所有硬编码颜色为变量
- 支持深色/浅色双主题
