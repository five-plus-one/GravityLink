<script setup lang="ts">
import { ref } from 'vue';
import { NButton, NModal, NEmpty, useMessage } from 'naive-ui';
import { request } from '../api';
const props = defineProps<{ origin: string }>();
const emit = defineEmits<{ select: [url: string] }>();
const show = ref(false), busy = ref(false);
const items = ref<{ ID: number; Path: string; Name: string }[]>([]);
const message = useMessage();
async function load() { try { items.value = (await request<{items: typeof items.value}>('/api/admin/materials')).items; } catch(e) { message.error(String(e)); } }
async function open() { show.value = true; await load(); }
async function upload(event: Event) {
 const file = (event.target as HTMLInputElement).files?.[0]; if (!file) return;
 if (file.size > 5 * 1024 * 1024) { message.error('图片不能超过 5 MiB'); return; }
 busy.value = true; try { const form = new FormData(); form.append('file',file); await request('/api/admin/materials',{method:'POST',body:form}); await load(); } catch(e) { message.error(String(e)); } finally { busy.value=false; (event.target as HTMLInputElement).value=''; }
}
function choose(path: string) { if (!props.origin) { message.error('请先选择域名'); return; } emit('select',new URL(path,props.origin).href); show.value=false; }
</script>
<template>
 <NButton @click="open">从素材库选择 / 上传</NButton>
 <NModal v-model:show="show" preset="card" title="图片素材" style="width:min(760px,94vw)">
  <p>支持 PNG、JPEG、GIF、WebP，最大 5 MiB。</p><input type="file" accept="image/png,image/jpeg,image/gif,image/webp" :disabled="busy" @change="upload" />
  <div class="materials"><button v-for="item in items" :key="item.ID" @click="choose(item.Path)"><img :src="item.Path" :alt="item.Name" /><span>选择图片</span></button></div>
  <NEmpty v-if="!items.length" description="暂无素材，请上传图片" />
 </NModal>
</template>
<style scoped>.materials{display:grid;grid-template-columns:repeat(auto-fill,minmax(130px,1fr));gap:16px;margin:24px 0}.materials button{border:1px solid #dce6f2;background:white;border-radius:12px;padding:12px;cursor:pointer}.materials img{width:100%;height:120px;object-fit:contain}.materials span{display:block}</style>
