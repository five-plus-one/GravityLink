<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { LoaderCircle, Plus, RefreshCw, X } from '@lucide/vue';
import { createLandingPage, listLandingPages, type LandingPageItem } from '../api';

const items = ref<LandingPageItem[]>([]);
const loading = ref(false);
const saving = ref(false);
const error = ref('');
const dialog = ref<HTMLDialogElement | null>(null);
const form = reactive({ title: '', domainId: 1, headline: '', subtext: '', footer: '', color: '#1677ff' });
onMounted(refresh);
async function refresh() { loading.value = true; try { items.value = (await listLandingPages()).items; } catch (err) { error.value = messageOf(err); } finally { loading.value = false; } }
async function submit() {
  saving.value = true;
  try {
    await createLandingPage({ template: 'liveqr', title: form.title, domain_id: form.domainId, content: { headline: form.headline, subtext: form.subtext, footer_text: form.footer, theme_color: form.color } });
    dialog.value?.close(); Object.assign(form, { title: '', domainId: 1, headline: '', subtext: '', footer: '', color: '#1677ff' }); await refresh();
  } catch (err) { error.value = messageOf(err); } finally { saving.value = false; }
}
function messageOf(err: unknown) { return err instanceof Error ? err.message : '操作失败'; }
</script>
<template><div class="view">
  <header class="view-header"><div><h1>落地页</h1><p>配置活码访问时展示的公开页面。</p></div><button class="primary icon-text" @click="dialog?.showModal()"><Plus :size="17" />创建落地页</button></header>
  <section class="surface"><div class="toolbar"><span class="muted">{{ items.length }} 个页面</span><button class="icon-button" title="刷新" @click="refresh"><RefreshCw :class="{ spin: loading }" :size="18" /></button></div>
    <p v-if="error" class="error banner">{{ error }}</p><div class="table-wrap"><table><thead><tr><th>ID</th><th>名称</th><th>模板</th><th>域名 ID</th></tr></thead><tbody>
      <tr v-for="item in items" :key="item.ID"><td class="code">{{ item.ID }}</td><td>{{ item.Title }}</td><td>{{ item.Template }}</td><td>{{ item.DomainID }}</td></tr><tr v-if="!loading && !items.length"><td colspan="4" class="empty">尚未创建落地页</td></tr>
    </tbody></table></div></section>
  <dialog ref="dialog" class="modal"><form class="modal-body" @submit.prevent="submit"><header><div><h2>创建落地页</h2><p>首版使用简洁的活码落地模板。</p></div><button class="icon-button" type="button" title="关闭" @click="dialog?.close()"><X :size="19" /></button></header>
    <div class="form-row"><label>名称<input v-model="form.title" required /></label><label>落地域名 ID<input v-model.number="form.domainId" min="1" type="number" /></label></div>
    <label>主标题<input v-model="form.headline" required /></label><label>说明<textarea v-model="form.subtext" /></label><div class="form-row"><label>页脚<input v-model="form.footer" /></label><label>主题色<input v-model="form.color" type="color" /></label></div>
    <footer><button type="button" @click="dialog?.close()">取消</button><button class="primary icon-text" :disabled="saving"><LoaderCircle v-if="saving" class="spin" :size="16" /><Plus v-else :size="16" />创建</button></footer></form></dialog>
</div></template>
