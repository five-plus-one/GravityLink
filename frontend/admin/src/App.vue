<script setup lang="ts">
import { computed } from 'vue';
import {
  darkTheme,
  dateZhCN,
  NConfigProvider,
  NDialogProvider,
  NLoadingBarProvider,
  NMessageProvider,
  NNotificationProvider,
  useOsTheme,
  zhCN,
  type GlobalThemeOverrides,
} from 'naive-ui';

// 当前项目不做暗色模式切换，但保留能力（与设计令牌对齐）。
const osTheme = useOsTheme();
const theme = computed(() => (osTheme.value === 'dark' ? null : null)); // 强制浅色，后续可扩展
void darkTheme;

// 把 Naive UI 主题变量对齐到设计令牌（tokens.css）
const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#0f766e',
    primaryColorHover: '#0d655e',
    primaryColorPressed: '#0b544e',
    primaryColorSuppl: '#0f766e',
    successColor: '#0e7a5f',
    warningColor: '#8a5a00',
    errorColor: '#b3261e',
    infoColor: '#1d4ed8',
    borderRadius: '6px',
    borderRadiusSmall: '4px',
    fontFamily: 'Inter, "Segoe UI", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif',
    fontSize: '14px',
  },
};
</script>

<template>
  <NConfigProvider :theme="theme" :theme-overrides="themeOverrides" :locale="zhCN" :date-locale="dateZhCN">
    <NLoadingBarProvider>
      <NMessageProvider>
        <NNotificationProvider>
          <NDialogProvider>
            <RouterView />
          </NDialogProvider>
        </NNotificationProvider>
      </NMessageProvider>
    </NLoadingBarProvider>
  </NConfigProvider>
</template>
