<script setup lang="ts">
import * as echarts from 'echarts/core';
import { BarChart, LineChart } from 'echarts/charts';
import { GridComponent, TooltipComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';
import { computed, nextTick, onMounted, reactive, ref } from 'vue';
import SetupWizard from './SetupWizard.vue';
import {
  createDomain,
  createLandingPage,
  createLink,
  deleteDomain,
  getDailyStats,
  getHourlyStats,
  getSummaryStats,
  getSystemConfigs,
  listDomains,
  listLandingPages,
  listLinks,
  updateSystemConfigs,
  type AuthConfigStatus,
  type ConfigListData,
  type DomainItem,
  type LandingPageItem,
  type LinkItem,
  type DailyPoint,
  type HourlyPoint,
  type SummaryStats,
} from './api';
import {
  clearSession,
  currentUser,
  handleCallback,
  loadAuthConfig,
  login,
  logout,
  type AuthConfig,
  type AuthUser,
} from './auth';
import { loadSetupStatus, type SetupStatus } from './setup';

echarts.use([BarChart, LineChart, GridComponent, TooltipComponent, CanvasRenderer]);

const links = ref<LinkItem[]>([]);
const domains = ref<DomainItem[]>([]);
const landingPages = ref<LandingPageItem[]>([]);
const summary = ref<SummaryStats | null>(null);
const systemConfigs = ref<ConfigListData | null>(null);
const dailyStats = ref<DailyPoint[]>([]);
const hourlyStats = ref<HourlyPoint[]>([]);
const dailyChartEl = ref<HTMLDivElement | null>(null);
const hourlyChartEl = ref<HTMLDivElement | null>(null);
const loading = ref(false);
const error = ref('');
const authLoading = ref(true);
const authConfig = ref<AuthConfig | null>(null);
const user = ref<AuthUser | null>(null);
const authError = ref('');
const setupStatus = ref<SetupStatus | null>(null);
const activeView = ref<'links' | 'domains' | 'landing' | 'stats' | 'settings'>('links');

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

const statsForm = reactive({
  linkId: 1,
});

const publicSettingsForm = reactive({
  siteName: 'GravityLink',
  homeTitle: '链接服务正在运行',
  homeMessage: '这是短链接访问入口。请使用完整短链接访问目标内容。',
  notFoundTitle: '链接不存在或已失效',
  notFoundMessage: '请检查链接是否完整，或联系链接提供方确认当前链接状态。',
  goneTitle: '链接已过期',
  goneMessage: '该链接已超过有效期，无法继续访问。',
  footer: 'GravityLink',
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
  if (activeView.value === 'settings') {
    return '配置';
  }
  return '短链接';
});

onMounted(async () => {
  await boot();
});

async function boot() {
  authLoading.value = true;
  authError.value = '';
  try {
    const status = await loadSetupStatus();
    setupStatus.value = status;
    if (status.setup_required) {
      return;
    }
    const config = await loadAuthConfig();
    authConfig.value = config;
    await handleCallback(config);
    user.value = currentUser(config);
    if (user.value) {
      await Promise.all([refreshLinks(), refreshDomains(), refreshLandingPages(), refreshSystemConfigs()]);
    }
  } catch (err) {
    authError.value = err instanceof Error ? err.message : '登录初始化失败';
  } finally {
    authLoading.value = false;
  }
}

async function setupCompleted() {
  setupStatus.value = null;
  await boot();
}

async function refreshSystemConfigs() {
  loading.value = true;
  error.value = '';
  try {
    const data = await getSystemConfigs();
    systemConfigs.value = data;
    applyPublicSettings(data.configs);
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
}

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

async function startLogin() {
  if (!authConfig.value) {
    authError.value = '登录配置未加载';
    return;
  }
  try {
    await login(authConfig.value);
  } catch (err) {
    authError.value = err instanceof Error ? err.message : '无法跳转登录';
  }
}

function logoutAdmin() {
  const config = authConfig.value;
  if (config) {
    logout(config);
  } else {
    clearSession();
  }
  user.value = null;
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

async function refreshStats() {
  loading.value = true;
  error.value = '';
  try {
    const [summaryData, dailyData, hourlyData] = await Promise.all([
      getSummaryStats(statsForm.linkId),
      getDailyStats(statsForm.linkId),
      getHourlyStats(statsForm.linkId),
    ]);
    summary.value = summaryData;
    dailyStats.value = dailyData;
    hourlyStats.value = hourlyData;
    await nextTick();
    renderCharts();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
}

async function submitPublicSettings() {
  error.value = '';
  try {
    const data = await updateSystemConfigs({
      'public.site_name': publicSettingsForm.siteName,
      'public.home.title': publicSettingsForm.homeTitle,
      'public.home.message': publicSettingsForm.homeMessage,
      'public.not_found.title': publicSettingsForm.notFoundTitle,
      'public.not_found.message': publicSettingsForm.notFoundMessage,
      'public.gone.title': publicSettingsForm.goneTitle,
      'public.gone.message': publicSettingsForm.goneMessage,
      'public.footer': publicSettingsForm.footer,
    });
    systemConfigs.value = data;
    applyPublicSettings(data.configs);
  } catch (err) {
    error.value = err instanceof Error ? err.message : '保存失败';
  }
}

function applyPublicSettings(configs: Record<string, string>) {
  publicSettingsForm.siteName = configs['public.site_name'] || publicSettingsForm.siteName;
  publicSettingsForm.homeTitle = configs['public.home.title'] || publicSettingsForm.homeTitle;
  publicSettingsForm.homeMessage = configs['public.home.message'] || publicSettingsForm.homeMessage;
  publicSettingsForm.notFoundTitle = configs['public.not_found.title'] || publicSettingsForm.notFoundTitle;
  publicSettingsForm.notFoundMessage = configs['public.not_found.message'] || publicSettingsForm.notFoundMessage;
  publicSettingsForm.goneTitle = configs['public.gone.title'] || publicSettingsForm.goneTitle;
  publicSettingsForm.goneMessage = configs['public.gone.message'] || publicSettingsForm.goneMessage;
  publicSettingsForm.footer = configs['public.footer'] || publicSettingsForm.footer;
}

function authStatusLabel(auth?: AuthConfigStatus): string {
  if (!auth) {
    return '未加载';
  }
  if (auth.auth_disabled) {
    return '开发模式';
  }
  if (auth.issuer && auth.client_id && auth.audience) {
    return '已配置';
  }
  return '未完整配置';
}

function renderCharts() {
  if (dailyChartEl.value) {
    const chart = echarts.init(dailyChartEl.value);
    chart.setOption({
      tooltip: { trigger: 'axis' },
      grid: { left: 36, right: 18, top: 24, bottom: 32 },
      xAxis: { type: 'category', data: dailyStats.value.map((point) => point.date) },
      yAxis: { type: 'value' },
      series: [
        { name: 'PV', type: 'line', smooth: true, data: dailyStats.value.map((point) => point.pv) },
        { name: 'UV', type: 'line', smooth: true, data: dailyStats.value.map((point) => point.uv) },
      ],
    });
  }

  if (hourlyChartEl.value) {
    const chart = echarts.init(hourlyChartEl.value);
    chart.setOption({
      tooltip: { trigger: 'axis' },
      grid: { left: 36, right: 18, top: 24, bottom: 32 },
      xAxis: { type: 'category', data: hourlyStats.value.map((point) => `${point.hour}:00`) },
      yAxis: { type: 'value' },
      series: [{ name: 'PV', type: 'bar', data: hourlyStats.value.map((point) => point.pv) }],
    });
  }
}
</script>

<template>
  <main v-if="authLoading" class="login-shell">
    <section class="login-panel">
      <span class="brand-mark">G</span>
      <h1>GravityLink Admin</h1>
      <p>正在检查登录状态...</p>
    </section>
  </main>

  <SetupWizard
    v-else-if="setupStatus?.setup_required"
    :status="setupStatus"
    @completed="setupCompleted"
  />

  <main v-else-if="!user" class="login-shell">
    <section class="login-panel">
      <span class="brand-mark">G</span>
      <h1>GravityLink Admin</h1>
      <p>请使用授权的 Logto 账号登录后继续。</p>
      <button class="primary login-button" @click="startLogin">使用 Logto 登录</button>
      <p v-if="authError" class="error">{{ authError }}</p>
    </section>
  </main>

  <main v-else class="app-shell">
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
        <button class="nav-item" :class="{ active: activeView === 'settings' }" @click="activeView = 'settings'">
          配置
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
          <p v-else-if="activeView === 'stats'">按链接 ID 查看访问数据</p>
          <p v-else>公开访问面与登录配置</p>
        </div>
        <div class="user-box">
          <div>
            <strong>{{ user.username || user.email || user.subject }}</strong>
            <small>{{ user.role }}</small>
          </div>
          <button class="ghost" @click="logoutAdmin">退出</button>
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

      <section v-else-if="activeView === 'stats'" class="panel placeholder">
        <div class="table-head">
          <h2>统计</h2>
          <div class="stats-query">
            <input v-model.number="statsForm.linkId" min="1" type="number" />
            <button class="ghost" :disabled="loading" @click="refreshStats">
              {{ loading ? '加载中' : '查询' }}
            </button>
          </div>
        </div>
        <div class="stats-body">
          <p v-if="error" class="error">{{ error }}</p>
          <div v-if="summary" class="metric-grid">
            <div class="metric"><span>总 PV</span><strong>{{ summary.total_pv }}</strong></div>
            <div class="metric"><span>总 UV</span><strong>{{ summary.total_uv }}</strong></div>
            <div class="metric"><span>今日 PV</span><strong>{{ summary.today_pv }}</strong></div>
            <div class="metric"><span>今日 UV</span><strong>{{ summary.today_uv }}</strong></div>
          </div>
          <section class="mini-chart">
            <h2>近 30 天 PV</h2>
            <div ref="dailyChartEl" class="chart"></div>
          </section>
          <section class="mini-chart">
            <h2>小时分布</h2>
            <div ref="hourlyChartEl" class="chart"></div>
          </section>
        </div>
      </section>

      <section v-else class="content-grid">
        <form class="panel editor" @submit.prevent="submitPublicSettings">
          <h2>公开提示页</h2>
          <label>
            站点名
            <input v-model="publicSettingsForm.siteName" />
          </label>
          <label>
            首页标题
            <input v-model="publicSettingsForm.homeTitle" />
          </label>
          <label>
            首页提示
            <textarea v-model="publicSettingsForm.homeMessage"></textarea>
          </label>
          <label>
            不存在标题
            <input v-model="publicSettingsForm.notFoundTitle" />
          </label>
          <label>
            不存在提示
            <textarea v-model="publicSettingsForm.notFoundMessage"></textarea>
          </label>
          <label>
            过期标题
            <input v-model="publicSettingsForm.goneTitle" />
          </label>
          <label>
            过期提示
            <textarea v-model="publicSettingsForm.goneMessage"></textarea>
          </label>
          <label>
            页脚
            <input v-model="publicSettingsForm.footer" />
          </label>
          <button class="primary" type="submit">保存公开提示</button>
          <p v-if="error" class="error">{{ error }}</p>
        </form>

        <section class="panel table-panel">
          <div class="table-head">
            <h2>Logto 登录</h2>
            <button class="ghost" :disabled="loading" @click="refreshSystemConfigs">
              {{ loading ? '加载中' : '刷新' }}
            </button>
          </div>
          <div class="settings-list">
            <div class="setting-row">
              <span>状态</span>
              <strong>{{ authStatusLabel(systemConfigs?.auth) }}</strong>
            </div>
            <div class="setting-row">
              <span>Issuer</span>
              <code>{{ systemConfigs?.auth.issuer || '-' }}</code>
            </div>
            <div class="setting-row">
              <span>Client ID</span>
              <code>{{ systemConfigs?.auth.client_id || '-' }}</code>
            </div>
            <div class="setting-row">
              <span>Audience</span>
              <code>{{ systemConfigs?.auth.audience || '-' }}</code>
            </div>
            <div class="setting-row">
              <span>回调地址</span>
              <code>{{ systemConfigs?.auth.redirect_uri || '-' }}</code>
            </div>
            <div class="setting-row">
              <span>允许角色</span>
              <code>{{ systemConfigs?.auth.allowed_roles?.join(', ') || '-' }}</code>
            </div>
          </div>
        </section>
      </section>
    </section>
  </main>
</template>
