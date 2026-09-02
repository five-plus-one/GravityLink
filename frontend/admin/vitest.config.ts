// 测试配置独立成文件：避免 vitest 内嵌的 vite 类型与项目 vite 6
// 在 vite.config.ts 中发生类型冲突（vue-tsc 会检查后者）。
import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
  },
});
