# Fuploader Color System Update: Color Mapping

This document outlines the current color system using "Sky Blue" as the primary theme.

## Primary Palette (Main Brand Color)

| Token | Old Value (Indigo) | Current Value (Sky Blue) | Usage |
|-------|--------------------|--------------------|-------|
| `--color-primary-50` | `#eef2ff` | `#f0f9ff` | Light backgrounds |
| `--color-primary-100` | `#e0e7ff` | `#e0f2fe` | Hover states |
| `--color-primary-200` | `#c7d2fe` | `#bae6fd` | Borders |
| `--color-primary-300` | `#a5b4fc` | `#7dd3fc` | Light accents |
| `--color-primary-400` | `#818cf8` | `#38bdf8` | Secondary buttons |
| `--color-primary-500` | `#6366f1` | `#0ea5e9` | **Primary Brand Color** |
| `--color-primary-600` | `#4f46e5` | `#0284c7` | Hover states |
| `--color-primary-700` | `#4338ca` | `#0369a1` | Active states |
| `--color-primary-800` | `#3730a3` | `#075985` | Dark accents |
| `--color-primary-900` | `#312e81` | `#0c4a6e` | Dark backgrounds |

## Semantic Colors

### Success (Emerald)
*Retained or adjusted for consistency*
| Token | Old Value | New Value |
|-------|-----------|-----------|
| `--color-success-500` | `#10b981` | `#10b981` |

### Warning (Amber)
*Retained or adjusted for consistency*
| Token | Old Value | New Value |
|-------|-----------|-----------|
| `--color-warning-500` | `#f59e0b` | `#f59e0b` |

### Error (Rose)
*Updated from generic red to Rose*
| Token | Old Value | New Value |
|-------|-----------|-----------|
| `--color-error-500` | `#f43f5e` | `#f43f5e` |

### Info (Sky)
*Used to be Primary, now distinct Info color*
| Token | Old Value (N/A) | New Value |
|-------|-----------------|-----------|
| `--color-info-500` | `#0ea5e9` | `#0ea5e9` |

## Theme Variables (Dark/Light)

| Token | Old Implementation | New Implementation (Slate) |
|-------|-------------------|----------------------------|
| `--bg-primary` (Dark) | `#0f0f0f` | `#0f172a` (Slate-950) |
| `--bg-secondary` (Dark) | `#1a1a1a` | `#1e293b` (Slate-900) |
| `--text-primary` (Dark) | `#ffffff` | `#f8fafc` (Slate-50) |
| `--text-muted` (Dark) | `#707070` | `#64748b` (Slate-500) |
| `--border-color` (Dark) | `#404040` | `#334155` (Slate-700) |

## Animation Parameters

| Token | Value | Description |
|-------|-------|-------------|
| `--transition-fast` | `150ms` | Micro-interactions (hover, click) |
| `--transition-base` | `200ms` | Component states |
| `--transition-slow` | `300ms` | Page transitions, large panels |
| `--ease-out` | `cubic-bezier(0.25, 0.46, 0.45, 0.94)` | Out easing |
| `--ease-in-out` | `cubic-bezier(0.4, 0, 0.2, 1)` | Standard easing |
