<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { AlertTriangle, ExternalLink, LoaderCircle, RotateCcw, Save, Settings } from '@lucide/vue';
import { getSystemConfigs, resetSystem, updateSystemConfigs, type AuthConfigStatus } from '../api';
import { clearSession, loadCurrentUser, type AuthUser } from '../auth';

const loading = ref(true);
const saving = ref(false);
const error = ref('');
const success = ref('');
const auth = ref<AuthConfigStatus | null>(null);
const user = ref<AuthUser | null>(null);
const resetDialog = ref<HTMLDialogElement | null>(null);
const confirmation = ref('');
const password = ref('');
const resetBusy = ref(false);
const form = reactive({ siteName: 'GravityLink', homeTitle: '链接服务正在运行', homeMessage: '这是短链接访问入口，请使用完整短链接访问目标内容。', notFoundTitle: '链接不存在或已失效', notFoundMessage: '请检查链接是否完整，或联系链接提供方确认当前状态。', goneTitle: '链接已过期', goneMessage: '该链接已超过有效期，无法继续访问。', footer: 'GravityLink' });

onMounted(async () => {
  try {
    const [data, current] = await Promise.all([getSystemConfigs(), loadCurrentUser()]);
    auth.value = data.auth; user.value = current;
    form.siteName = data.configs['public.site_name'] || form.siteName;
    form.homeTitle = data.configs['public.home.title'] || form.homeTitle;
    form.homeMessage = data.configs['public.home.message'] || form.homeMessage;
    form.notFoundTitle = data.configs['public.not_found.title'] || form.notFoundTitle;
    form.notFoundMessage = data.configs['public.not_found.message'] || form.notFoundMessage;
    form.goneTitle = data.configs['public.gone.title'] || form.goneTitle;
    form.goneMessage = data.configs['public.gone.message'] || form.goneMessage;
    form.footer = data.configs['public.footer'] || form.footer;
  } catch (err) { error.value = messageOf(err); } finally { loading.value = false; }
});
async function save() {
  saving.value = true; error.value = ''; success.value = '';
  try {
    await updateSystemConfigs({ 'public.site_name': form.siteName, 'public.home.title': form.homeTitle, 'public.home.message': form.homeMessage, 'public.not_found.title': form.notFoundTitle, 'public.not_found.message': form.notFoundMessage, 'public.gone.title': form.goneTitle, 'public.gone.message': form.goneMessage, 'public.footer': form.footer });
    success.value = '公开页面提示已更新';
  } catch (err) { error.value = messageOf(err); } finally { saving.value = false; }
}
async function resetAll() {
  resetBusy.value = true; error.value = '';
  try {
    await resetSystem(confirmation.value, password.value);
    clearSession();
    window.location.replace('/');
  } catch (err) { error.value = messageOf(err); resetBusy.value = false; resetDialog.value?.close(); }
}
function messageOf(err: unknown) { return err instanceof Error ? err.message : '操作失败'; }
</script>
<template><div class="view">
  <header class="view-header"><div><h1>系统设置</h1><p>公开页面、认证状态和安装生命周期。</p></div><button class="primary icon-text" :disabled="saving" @click="save"><LoaderCircle v-if="saving" class="spin" :size="16" /><Save v-else :size="16" />保存更改</button></header>
  <div v-if="loading" class="inline-state"><LoaderCircle class="spin" :size="20" />正在加载</div><template v-else>
    <p v-if="success" class="success-message banner">{{ success }}</p><p v-if="error" class="error banner">{{ error }}</p>
    <section class="settings-section"><header><div><h2>公开访问提示</h2><p>首页、未知短码和过期链接都会使用这些文案。</p></div><a class="text-link" href="/" target="_blank">预览公开入口<ExternalLink :size="15" /></a></header>
      <div class="settings-fields"><label>站点名称<input v-model="form.siteName" /></label><label>页脚<input v-model="form.footer" /></label><label>首页标题<input v-model="form.homeTitle" /></label><label class="wide">首页说明<textarea v-model="form.homeMessage" /></label><label>链接不存在标题<input v-model="form.notFoundTitle" /></label><label class="wide">链接不存在说明<textarea v-model="form.notFoundMessage" /></label><label>链接过期标题<input v-model="form.goneTitle" /></label><label class="wide">链接过期说明<textarea v-model="form.goneMessage" /></label></div>
    </section>
    <section class="settings-section"><header><div><h2>身份认证</h2><p>认证提供方本身只能通过重新初始化修改。</p></div><Settings :size="20" /></header>
      <dl class="details"><div><dt>模式</dt><dd>{{ user?.auth_source === 'local' ? '本地账号' : user?.auth_source === 'development' ? '开发模式' : 'Logto OIDC' }}</dd></div><div><dt>Issuer</dt><dd>{{ auth?.issuer || '不适用' }}</dd></div><div><dt>Audience</dt><dd>{{ auth?.audience || '不适用' }}</dd></div><div><dt>当前账号</dt><dd>{{ user?.username }} · {{ user?.role }}</dd></div></dl>
    </section>
    <section v-if="user?.role === 'super_admin'" class="settings-section danger-zone"><header><div><h2>危险区</h2><p>清除认证和系统配置，退出登录并返回初始化流程。</p></div><AlertTriangle :size="20" /></header>
      <div class="danger-action"><div><strong>恢复到未初始化状态</strong><p>保留短链、域名、落地页和访问统计；清除账号授权、会话、公开页配置与运行配置文件。</p></div><button class="danger-button icon-text" @click="resetDialog?.showModal()"><RotateCcw :size="16" />清除所有配置</button></div>
    </section>
  </template>
  <dialog ref="resetDialog" class="modal"><form class="modal-body" @submit.prevent="resetAll"><header><div><h2>确认清除所有配置</h2><p>此操作会让管理端立即退出并进入首次配置。</p></div><AlertTriangle class="danger" :size="22" /></header>
    <div class="warning-box">业务数据会保留，但所有用户授权和登录会话将失效。</div><label>输入 <code>RESET GRAVITYLINK</code><input v-model="confirmation" autocomplete="off" /></label><label v-if="user?.auth_source === 'local'">再次输入当前密码<input v-model="password" type="password" autocomplete="current-password" /></label>
    <footer><button type="button" @click="resetDialog?.close()">取消</button><button class="danger-button icon-text" :disabled="resetBusy || confirmation !== 'RESET GRAVITYLINK' || (user?.auth_source === 'local' && !password)"><LoaderCircle v-if="resetBusy" class="spin" :size="16" /><RotateCcw v-else :size="16" />确认清除</button></footer>
  </form></dialog>
</div></template>
