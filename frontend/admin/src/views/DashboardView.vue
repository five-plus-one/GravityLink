<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { ArrowRight, Globe2, LayoutTemplate, Link2, LoaderCircle } from '@lucide/vue';
import { listDomains, listLandingPages, listLinks, type DomainItem, type LandingPageItem, type LinkItem } from '../api';

const links = ref<LinkItem[]>([]);
const domains = ref<DomainItem[]>([]);
const pages = ref<LandingPageItem[]>([]);
const loading = ref(true);
const error = ref('');
const activeLinks = computed(() => links.value.filter((item) => item.Status === 'active').length);

onMounted(async () => {
  try {
    const [linkData, domainData, pageData] = await Promise.all([listLinks(), listDomains(), listLandingPages()]);
    links.value = linkData.items;
    domains.value = domainData.items;
    pages.value = pageData.items;
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载概览失败';
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <div class="view">
    <header class="view-header"><div><h1>概览</h1><p>链接服务与关键资源的当前状态。</p></div></header>
    <div v-if="loading" class="inline-state"><LoaderCircle class="spin" :size="20" />正在加载</div>
    <p v-else-if="error" class="error">{{ error }}</p>
    <template v-else>
      <section class="metrics">
        <RouterLink class="metric-block" to="/links"><span><Link2 :size="18" />链接总数</span><strong>{{ links.length }}</strong><small>{{ activeLinks }} 条可访问</small></RouterLink>
        <RouterLink class="metric-block" to="/domains"><span><Globe2 :size="18" />域名</span><strong>{{ domains.length }}</strong><small>入口、落地与中转域名</small></RouterLink>
        <RouterLink class="metric-block" to="/landing-pages"><span><LayoutTemplate :size="18" />落地页</span><strong>{{ pages.length }}</strong><small>公开页面模板</small></RouterLink>
      </section>
      <section class="surface">
        <div class="surface-head"><div><h2>最近链接</h2><p>按当前接口返回顺序显示。</p></div><RouterLink class="text-link" to="/links">查看全部<ArrowRight :size="15" /></RouterLink></div>
        <div class="table-wrap"><table><thead><tr><th>短码</th><th>名称</th><th>类型</th><th>目标</th><th>状态</th></tr></thead>
          <tbody><tr v-for="item in links.slice(0, 6)" :key="item.ID"><td class="code">{{ item.Code }}</td><td>{{ item.Title || '未命名' }}</td><td>{{ item.Type }}</td><td class="truncate">{{ item.TargetURL || '动态路由' }}</td><td><span class="status" :class="item.Status">{{ item.Status }}</span></td></tr>
          <tr v-if="!links.length"><td colspan="5" class="empty">尚未创建链接</td></tr></tbody>
        </table></div>
      </section>
    </template>
  </div>
</template>
