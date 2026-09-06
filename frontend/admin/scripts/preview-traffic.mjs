// Isolated visual fixture: node scripts/preview-traffic.mjs. No backend writes.
import { createServer } from 'vite';

const html = `<!doctype html><html lang="zh-CN"><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>访问看板 · 测试预览</title><body>
<div style="padding:16px;color:#53647a">界面验收 · 以下均为测试数据，不连接实际后台</div><div id="app" style="padding:24px;max-width:1440px;margin:auto"></div>
<script type="module">
import { createApp, h } from '/node_modules/.vite/deps/vue.js';
import { NConfigProvider } from '/node_modules/.vite/deps/naive-ui.js';
import TrafficOverview from '/src/components/TrafficOverview.vue';
import '/src/styles/base.css';
const points = [20, 52, 8, 0, 34, 16, 5].map((pv, i) => ({ date: '2026-09-' + String(i + 1).padStart(2, '0'), pv, uv: Math.ceil(pv / 3) }));
window.fetch = async (path) => {
  const id = String(path).split('/')[4];
  if (id === '3') return new Response(JSON.stringify({code: 5000, message: '测试：统计服务暂时不可用'}), {status: 503});
  const zero = id === '2';
  const data = String(path).endsWith('/summary') ? { today_pv: zero ? 0 : 5, today_uv: zero ? 0 : 2, total_pv: zero ? 0 : 135, total_uv: zero ? 0 : 48, yesterday_pv: zero ? 0 : 16 }
    : String(path).endsWith('/daily') ? points.map(p => zero ? {...p, pv: 0, uv: 0} : p)
    : Array.from({length: 24}, (_, hour) => ({hour, pv: !zero && hour === 15 ? 5 : 0}));
  return new Response(JSON.stringify({code: 0, data}));
};
const links = ['活动推广', '零访问链接', '失败重试'].map((Title, i) => ({ID: i + 1, Title, Code: 'test-' + (i + 1)}));
createApp({render: () => h(NConfigProvider, {themeOverrides: {common: {primaryColor: '#2787f5', borderRadius: '10px'}}}, {default: () => h(TrafficOverview, {links})})}).mount('#app');
</script></body></html>`;

const server = await createServer({
  server: { host: '127.0.0.1', port: 5175, strictPort: true },
  plugins: [{
    name: 'traffic-visual-fixture',
    configureServer(server) {
      server.middlewares.use('/__traffic-preview', async (_req, res) => {
        res.setHeader('Content-Type', 'text/html; charset=utf-8');
        res.end(await server.transformIndexHtml('/__traffic-preview', html));
      });
    },
  }],
});
await server.listen();
console.log('Visual fixture: http://127.0.0.1:5175/__traffic-preview');
