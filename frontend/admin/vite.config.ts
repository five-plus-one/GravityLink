import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

const apiProxyTarget = process.env.VITE_API_PROXY_TARGET ?? 'http://127.0.0.1:8080';

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: apiProxyTarget,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    // vendor-ui（naive-ui 按需引用组件）与 vendor-charts（echarts）为有意分包，
    // 均在对应路由懒加载时才下载，阈值按实际产物调整以保持构建输出干净
    chunkSizeWarningLimit: 900,
    rollupOptions: {
      output: {
        // 大体积三方库单独分包：业务代码更新时不连带重新下载
        manualChunks: {
          'vendor-vue': ['vue', 'vue-router', 'pinia'],
          'vendor-ui': ['naive-ui'],
          'vendor-charts': ['echarts', 'vue-echarts'],
        },
      },
    },
  },
});
