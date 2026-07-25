<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Link2, LoaderCircle, Plus, RefreshCw, X } from '@lucide/vue';
import { createLink, listLinks, type LinkItem } from '../api';

const items = ref<LinkItem[]>([]);
const loading = ref(false);
const saving = ref(false);
const error = ref('');
const dialog = ref<HTMLDialogElement | null>(null);
const query = ref('');
const form = reactive({ type: 'short' as 'short' | 'channel' | 'liveqr', title: '', code: '', entryDomainId: 1, targetUrl: '', targets: '' });
const filtered = computed(() => {
  const value = query.value.trim().toLowerCase();
  return value ? items.value.filter((item) => `${item.Code} ${item.Title || ''} ${item.TargetURL || ''}`.toLowerCase().includes(value)) : items.value;
});

onMounted(refresh);
async function refresh() {
  loading.value = true;
  error.value = '';
  try { items.value = (await listLinks()).items; } catch (err) { error.value = messageOf(err); } finally { loading.value = false; }
}
async function submit() {
  saving.value = true;
  error.value = '';
  try {
    await createLink({
      type: form.type, code: form.code || undefined, title: form.title || undefined,
      entry_domain_id: form.entryDomainId, target_url: form.targetUrl,
      strategy: form.type === 'liveqr' ? {
        mode: 'round_robin',
        targets: form.targets.split('\n').map((value) => value.trim()).filter(Boolean).map((target_url) => ({ target_url, weight: 1 })),
      } : undefined,
    });
    dialog.value?.close();
    Object.assign(form, { type: 'short', title: '', code: '', entryDomainId: 1, targetUrl: '', targets: '' });
    await refresh();
  } catch (err) { error.value = messageOf(err); } finally { saving.value = false; }
}
function messageOf(err: unknown) { return err instanceof Error ? err.message : '操作失败'; }
</script>

<template>
  <div class="view">
    <header class="view-header"><div><h1>链接</h1><p>{{ items.length }} 条记录，{{ items.filter((i) => i.Status === 'active').length }} 条可访问。</p></div>
      <button class="primary icon-text" type="button" @click="dialog?.showModal()"><Plus :size="17" />创建链接</button></header>
    <section class="surface">
      <div class="toolbar"><div class="search"><Link2 :size="17" /><input v-model="query" placeholder="搜索短码、名称或目标" /></div>
        <button class="icon-button" title="刷新" type="button" :disabled="loading" @click="refresh"><RefreshCw :class="{ spin: loading }" :size="18" /></button></div>
      <p v-if="error" class="error banner">{{ error }}</p>
      <div class="table-wrap"><table><thead><tr><th>短码</th><th>名称</th><th>类型</th><th>目标</th><th>创建时间</th><th>状态</th></tr></thead>
        <tbody><tr v-for="item in filtered" :key="item.ID"><td class="code">{{ item.Code }}</td><td>{{ item.Title || '未命名' }}</td><td>{{ item.Type }}</td><td class="truncate">{{ item.TargetURL || '动态路由' }}</td><td>{{ new Date(item.CreatedAt).toLocaleString() }}</td><td><span class="status" :class="item.Status">{{ item.Status }}</span></td></tr>
        <tr v-if="!loading && !filtered.length"><td colspan="6" class="empty">没有匹配的链接</td></tr></tbody></table></div>
    </section>

    <dialog ref="dialog" class="modal"><form class="modal-body" @submit.prevent="submit">
      <header><div><h2>创建链接</h2><p>保存后短码立即生效。</p></div><button class="icon-button" title="关闭" type="button" @click="dialog?.close()"><X :size="19" /></button></header>
      <div class="segmented"><button v-for="option in [['short','短链接'],['channel','渠道链接'],['liveqr','活码']]" :key="option[0]" type="button" :class="{ active: form.type === option[0] }" @click="form.type = option[0] as typeof form.type">{{ option[1] }}</button></div>
      <label>名称<input v-model="form.title" placeholder="便于管理端识别" /></label>
      <div class="form-row"><label>短码<input v-model="form.code" placeholder="留空自动生成" /></label><label>入口域名 ID<input v-model.number="form.entryDomainId" min="1" type="number" /></label></div>
      <label v-if="form.type !== 'liveqr'">目标 URL<input v-model="form.targetUrl" required type="url" placeholder="https://example.com/path" /></label>
      <label v-else>轮询目标<textarea v-model="form.targets" required placeholder="每行一个完整 URL" /></label>
      <footer><button type="button" @click="dialog?.close()">取消</button><button class="primary icon-text" type="submit" :disabled="saving"><LoaderCircle v-if="saving" class="spin" :size="16" /><Plus v-else :size="16" />创建</button></footer>
    </form></dialog>
  </div>
</template>
