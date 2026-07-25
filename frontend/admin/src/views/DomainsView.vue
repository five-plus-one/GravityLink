<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { LoaderCircle, Plus, RefreshCw, Trash2, X } from '@lucide/vue';
import { createDomain, deleteDomain, listDomains, type DomainItem } from '../api';

const items = ref<DomainItem[]>([]);
const loading = ref(false);
const saving = ref(false);
const error = ref('');
const dialog = ref<HTMLDialogElement | null>(null);
const form = reactive({ host: '', type: 'entry' as 'entry' | 'transit' | 'landing', scheme: 'https' as 'http' | 'https', remark: '' });
onMounted(refresh);
async function refresh() { loading.value = true; try { items.value = (await listDomains()).items; } catch (err) { error.value = messageOf(err); } finally { loading.value = false; } }
async function submit() { saving.value = true; try { await createDomain({ ...form, remark: form.remark || undefined }); dialog.value?.close(); Object.assign(form, { host: '', type: 'entry', scheme: 'https', remark: '' }); await refresh(); } catch (err) { error.value = messageOf(err); } finally { saving.value = false; } }
async function remove(item: DomainItem) { if (!confirm(`确认删除域名 ${item.Host}？`)) return; try { await deleteDomain(item.ID); await refresh(); } catch (err) { error.value = messageOf(err); } }
function messageOf(err: unknown) { return err instanceof Error ? err.message : '操作失败'; }
</script>
<template><div class="view">
  <header class="view-header"><div><h1>域名</h1><p>管理入口、中转和落地页域名。</p></div><button class="primary icon-text" @click="dialog?.showModal()"><Plus :size="17" />添加域名</button></header>
  <section class="surface"><div class="toolbar"><span class="muted">{{ items.length }} 个域名配置</span><button class="icon-button" title="刷新" @click="refresh"><RefreshCw :class="{ spin: loading }" :size="18" /></button></div>
    <p v-if="error" class="error banner">{{ error }}</p><div class="table-wrap"><table><thead><tr><th>Host</th><th>用途</th><th>协议</th><th>备注</th><th>状态</th><th></th></tr></thead>
      <tbody><tr v-for="item in items" :key="item.ID"><td class="code">{{ item.Host }}</td><td>{{ item.Type }}</td><td>{{ item.Scheme }}</td><td>{{ item.Remark || '—' }}</td><td><span class="status" :class="item.Status">{{ item.Status }}</span></td><td class="actions"><button class="icon-button danger" title="删除" @click="remove(item)"><Trash2 :size="17" /></button></td></tr><tr v-if="!loading && !items.length"><td colspan="6" class="empty">尚未配置域名</td></tr></tbody>
    </table></div></section>
  <dialog ref="dialog" class="modal"><form class="modal-body" @submit.prevent="submit"><header><div><h2>添加域名</h2><p>DNS 与证书需要在反向代理层完成。</p></div><button class="icon-button" type="button" title="关闭" @click="dialog?.close()"><X :size="19" /></button></header>
    <label>Host<input v-model="form.host" required placeholder="go.example.com" /></label><div class="form-row"><label>用途<select v-model="form.type"><option value="entry">入口</option><option value="transit">中转</option><option value="landing">落地页</option></select></label><label>协议<select v-model="form.scheme"><option value="https">HTTPS</option><option value="http">HTTP</option></select></label></div><label>备注<input v-model="form.remark" /></label>
    <footer><button type="button" @click="dialog?.close()">取消</button><button class="primary icon-text" :disabled="saving"><LoaderCircle v-if="saving" class="spin" :size="16" /><Plus v-else :size="16" />添加</button></footer></form></dialog>
</div></template>
