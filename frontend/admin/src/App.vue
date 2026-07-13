<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  clearToken,
  createDomain,
  createLandingPage,
  createLink,
  deleteDomain,
  getToken,
  listDomains,
  listLandingPages,
  listLinks,
  setToken,
  type DomainItem,
  type LandingPageItem,
  type LinkItem,
} from './api';

const links = ref<LinkItem[]>([]);
const domains = ref<DomainItem[]>([]);
const landingPages = ref<LandingPageItem[]>([]);
const loading = ref(false);
const error = ref('');
const tokenDraft = ref(getToken());
const activeView = ref<'links' | 'domains' | 'landing' | 'stats'>('links');

const form = reactive({
  type: 'short' as 'short' | 'channel' | 'liveqr',
  title: '',
  code: '',
  entryDomainId: 1,
  targetUrl: '',
  utmSource: '',
  utmMedium: '',
  utmCampaign: '',
  utmTerm: '',
  utmContent: '',
  landingDomainId: 1,
  landingPageId: 1,
  targetList: '',
});

const domainForm = reactive({
  host: '',
  type: 'entry' as 'entry' | 'transit' | 'landing',
  scheme: 'https' as 'http' | 'https',
  remark: '',
});

const landingForm = reactive({
  title: '',
  domainId: 1,
  headline: '',
  subtext: '',
  footerText: '',
  themeColor: '#1677ff',
});

const activeLinks = computed(() => links.value.filter((link) => link.Status === 'active').length);
const viewTitle = computed(() => {
  if (activeView.value === 'domains') {
    return '域名管理';
  }
  if (activeView.value === 'landing') {
    return '落地页';
  }
  if (activeView.value === 'stats') {
    return '统计';
  }
  return '短链接';
});

onMounted(() => {
  void refreshLinks();
  void refreshDomains();
  void refreshLandingPages();
});

async function refreshLinks() {
  loading.value = true;
  error.value = '';
  try {
    const data = await listLinks();
    links.value = data.items;
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
}

async function refreshLandingPages() {
  loading.value = true;
  error.value = '';
  try {
    const data = await listLandingPages();
    landingPages.value = data.items;
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
}

async function refreshDomains() {
  loading.value = true;
  error.value = '';
  try {
    const data = await listDomains();
    domains.value = data.items;
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
}

async function submitLink() {
  error.value = '';
  try {
    await createLink({
      type: form.type,
      code: form.code || undefined,
      entry_domain_id: form.entryDomainId,
      target_url: form.targetUrl,
      title: form.title || undefined,
      landing_domain_id: form.type === 'liveqr' ? form.landingDomainId : undefined,
      landing_page_id: form.type === 'liveqr' ? form.landingPageId : undefined,
      channel:
        form.type === 'channel'
          ? {
              utm_source: form.utmSource || undefined,
              utm_medium: form.utmMedium || undefined,
              utm_campaign: form.utmCampaign || undefined,
              utm_term: form.utmTerm || undefined,
              utm_content: form.utmContent || undefined,
            }
          : undefined,
      strategy:
        form.type === 'liveqr'
          ? {
              mode: 'round_robin',
              targets: form.targetList
                .split('\n')
                .map((line) => line.trim())
                .filter(Boolean)
                .map((targetUrl, index) => ({
                  label: `目标 ${index + 1}`,
                  target_url: targetUrl,
                  weight: 1,
                })),
            }
          : undefined,
    });
    form.type = 'short';
    form.title = '';
    form.code = '';
    form.targetUrl = '';
    form.utmSource = '';
    form.utmMedium = '';
    form.utmCampaign = '';
    form.utmTerm = '';
    form.utmContent = '';
    form.targetList = '';
    await refreshLinks();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '创建失败';
  }
}

function saveToken() {
  setToken(tokenDraft.value.trim());
  void refreshLinks();
  void refreshDomains();
  void refreshLandingPages();
}

function logout() {
  clearToken();
  tokenDraft.value = '';
  links.value = [];
  domains.value = [];
  landingPages.value = [];
}

async function submitDomain() {
  error.value = '';
  try {
    await createDomain({
      host: domainForm.host,
      type: domainForm.type,
      scheme: domainForm.scheme,
      remark: domainForm.remark || undefined,
    });
    domainForm.host = '';
    domainForm.remark = '';
    await refreshDomains();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '创建失败';
  }
}

async function submitLandingPage() {
  error.value = '';
  try {
    await createLandingPage({
      template: 'liveqr',
      title: landingForm.title || landingForm.headline || '扫码加入交流群',
      domain_id: landingForm.domainId,
      content: {
        headline: landingForm.headline || '扫码加入交流群',
        subtext: landingForm.subtext,
        footer_text: landingForm.footerText || '长按识别二维码',
        theme_color: landingForm.themeColor || '#1677ff',
      },
    });
    landingForm.title = '';
    landingForm.headline = '';
    landingForm.subtext = '';
    landingForm.footerText = '';
    await refreshLandingPages();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '创建失败';
  }
}

async function removeDomain(id: number) {
  error.value = '';
  try {
    await deleteDomain(id);
    await refreshDomains();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '删除失败';
  }
}
</script>

<template>
  <main class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <span class="brand-mark">G</span>
        <div>
          <strong>GravityLink</strong>
          <small>Admin</small>
        </div>
      </div>
      <nav class="nav">
        <button class="nav-item" :class="{ active: activeView === 'links' }" @click="activeView = 'links'">
          链接
        </button>
        <button class="nav-item" :class="{ active: activeView === 'domains' }" @click="activeView = 'domains'">
          域名
        </button>
        <button class="nav-item" :class="{ active: activeView === 'landing' }" @click="activeView = 'landing'">
          落地页
        </button>
        <button class="nav-item" :class="{ active: activeView === 'stats' }" @click="activeView = 'stats'">
          统计
        </button>
      </nav>
    </aside>

    <section class="workspace">
      <header class="topbar">
        <div>
          <h1>{{ viewTitle }}</h1>
          <p v-if="activeView === 'links'">{{ links.length }} 条记录，{{ activeLinks }} 条可访问</p>
          <p v-else-if="activeView === 'domains'">{{ domains.length }} 个域名配置</p>
          <p v-else-if="activeView === 'landing'">{{ landingPages.length }} 个页面</p>
          <p v-else>统计看板将在 Phase 3 补齐</p>
        </div>
        <div class="auth-box">
          <input v-model="tokenDraft" type="password" placeholder="Access token" />
          <button @click="saveToken">保存</button>
          <button class="ghost" @click="logout">清除</button>
        </div>
      </header>

      <section v-if="activeView === 'links'" class="content-grid">
        <form class="panel editor" @submit.prevent="submitLink">
          <h2>新建</h2>
          <label>
            类型
            <select v-model="form.type">
              <option value="short">短链接</option>
              <option value="channel">渠道码</option>
              <option value="liveqr">群活码</option>
            </select>
          </label>
          <label>
            名称
            <input v-model="form.title" placeholder="活动链接" />
          </label>
          <label>
            短码
            <input v-model="form.code" placeholder="留空自动生成" />
          </label>
          <label>
            入口域名 ID
            <input v-model.number="form.entryDomainId" min="1" type="number" />
          </label>
          <label>
            目标 URL
            <input v-model="form.targetUrl" required placeholder="https://example.com/path" />
          </label>
          <div v-if="form.type === 'channel'" class="utm-grid">
            <label>
              utm_source
              <input v-model="form.utmSource" placeholder="wechat" />
            </label>
            <label>
              utm_medium
              <input v-model="form.utmMedium" placeholder="social" />
            </label>
            <label>
              utm_campaign
              <input v-model="form.utmCampaign" placeholder="spring2026" />
            </label>
            <label>
              utm_term
              <input v-model="form.utmTerm" placeholder="keyword" />
            </label>
            <label>
              utm_content
              <input v-model="form.utmContent" placeholder="poster_a" />
            </label>
          </div>
          <div v-if="form.type === 'liveqr'" class="utm-grid">
            <label>
              落地域名 ID
              <input v-model.number="form.landingDomainId" min="1" type="number" />
            </label>
            <label>
              落地页 ID
              <input v-model.number="form.landingPageId" min="1" type="number" />
            </label>
            <label>
              目标列表
              <textarea v-model="form.targetList" placeholder="每行一个二维码图片 URL 或跳转 URL"></textarea>
            </label>
          </div>
          <button class="primary" type="submit">创建链接</button>
          <p v-if="error" class="error">{{ error }}</p>
        </form>

        <section class="panel table-panel">
          <div class="table-head">
            <h2>链接列表</h2>
            <button class="ghost" :disabled="loading" @click="refreshLinks">
              {{ loading ? '加载中' : '刷新' }}
            </button>
          </div>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>短码</th>
                  <th>类型</th>
                  <th>名称</th>
                  <th>目标</th>
                  <th>状态</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="link in links" :key="link.ID">
                  <td class="code">{{ link.Code }}</td>
                  <td>{{ link.Type }}</td>
                  <td>{{ link.Title || '-' }}</td>
                  <td class="target">{{ link.TargetURL || '-' }}</td>
                  <td>
                    <span class="status" :class="link.Status">{{ link.Status }}</span>
                  </td>
                </tr>
                <tr v-if="!links.length && !loading">
                  <td colspan="5" class="empty">暂无数据</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </section>

      <section v-else-if="activeView === 'domains'" class="content-grid">
        <form class="panel editor" @submit.prevent="submitDomain">
          <h2>添加域名</h2>
          <label>
            Host
            <input v-model="domainForm.host" required placeholder="go.example.com" />
          </label>
          <label>
            类型
            <select v-model="domainForm.type">
              <option value="entry">入口</option>
              <option value="transit">中转</option>
              <option value="landing">落地</option>
            </select>
          </label>
          <label>
            协议
            <select v-model="domainForm.scheme">
              <option value="https">https</option>
              <option value="http">http</option>
            </select>
          </label>
          <label>
            备注
            <input v-model="domainForm.remark" placeholder="业务线或用途" />
          </label>
          <button class="primary" type="submit">添加域名</button>
          <p v-if="error" class="error">{{ error }}</p>
        </form>

        <section class="panel table-panel">
          <div class="table-head">
            <h2>域名列表</h2>
            <button class="ghost" :disabled="loading" @click="refreshDomains">
              {{ loading ? '加载中' : '刷新' }}
            </button>
          </div>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Host</th>
                  <th>类型</th>
                  <th>协议</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="domain in domains" :key="domain.ID">
                  <td class="code">{{ domain.Host }}</td>
                  <td>{{ domain.Type }}</td>
                  <td>{{ domain.Scheme }}</td>
                  <td><span class="status" :class="domain.Status">{{ domain.Status }}</span></td>
                  <td><button class="ghost danger" @click="removeDomain(domain.ID)">删除</button></td>
                </tr>
                <tr v-if="!domains.length && !loading">
                  <td colspan="5" class="empty">暂无数据</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </section>

      <section v-else-if="activeView === 'landing'" class="content-grid">
        <form class="panel editor" @submit.prevent="submitLandingPage">
          <h2>新建落地页</h2>
          <label>
            标题
            <input v-model="landingForm.title" placeholder="扫码加入交流群" />
          </label>
          <label>
            落地域名 ID
            <input v-model.number="landingForm.domainId" min="1" type="number" />
          </label>
          <label>
            主标题
            <input v-model="landingForm.headline" placeholder="扫码加入交流群" />
          </label>
          <label>
            副文案
            <input v-model="landingForm.subtext" placeholder="群满自动切换，永久有效" />
          </label>
          <label>
            页脚文案
            <input v-model="landingForm.footerText" placeholder="长按识别二维码" />
          </label>
          <label>
            主题色
            <input v-model="landingForm.themeColor" placeholder="#1677ff" />
          </label>
          <button class="primary" type="submit">创建落地页</button>
          <p v-if="error" class="error">{{ error }}</p>
        </form>

        <section class="panel table-panel">
          <div class="table-head">
            <h2>落地页列表</h2>
            <button class="ghost" :disabled="loading" @click="refreshLandingPages">
              {{ loading ? '加载中' : '刷新' }}
            </button>
          </div>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>ID</th>
                  <th>标题</th>
                  <th>模板</th>
                  <th>域名 ID</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="page in landingPages" :key="page.ID">
                  <td class="code">{{ page.ID }}</td>
                  <td>{{ page.Title }}</td>
                  <td>{{ page.Template }}</td>
                  <td>{{ page.DomainID }}</td>
                </tr>
                <tr v-if="!landingPages.length && !loading">
                  <td colspan="4" class="empty">暂无数据</td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </section>

      <section v-else class="panel placeholder">
        <h2>统计</h2>
        <p>Phase 3 将接入多维度聚合查询与图表。</p>
      </section>
    </section>
  </main>
</template>
