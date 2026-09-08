<script setup lang="ts">
import { ref } from 'vue';
import { NButton, NModal, NEmpty, useMessage } from 'naive-ui';
import { request } from '../api';
const props = defineProps<{ origin?: string; multiple?:boolean; relative?:boolean }>();
const emit = defineEmits<{ select: [url: string];selectMany:[urls:string[]] }>();
const selected=ref<string[]>([]);
const show = ref(false), busy = ref(false);
const items = ref<{ ID: number; Path: string; Name: string }[]>([]);
const message = useMessage();
const uploadResults=ref<string[]>([]);
async function load() { try { items.value = (await request<{items: typeof items.value}>('/api/admin/materials')).items; } catch(e) { message.error(String(e)); } }
async function open() { selected.value=[];uploadResults.value=[];show.value = true; await load(); }
async function upload(event: Event) {
 const input=event.target as HTMLInputElement,files=Array.from(input.files||[]);if(!files.length||busy.value)return;
 if(files.length>100){message.error('每批最多上传 100 张图片');input.value='';return;}
 busy.value=true;uploadResults.value=[];
 try {for(const file of files){
  if(file.size>5*1024*1024){uploadResults.value.push(`${file.name}：超过 5 MiB，未上传`);continue;}
  try{const form=new FormData();form.append('file',file);await request('/api/admin/materials',{method:'POST',body:form});uploadResults.value.push(`${file.name}：上传成功`);}
  catch(e){uploadResults.value.push(`${file.name}：上传失败 ${String(e)}`);}
 }await load();}finally{busy.value=false;input.value='';}
}
function materialURL(path:string){return props.relative?path:new URL(path,props.origin).href;}
function choose(path: string) { if (!props.relative&&!props.origin) { message.error('请先选择域名'); return; } if(props.multiple){selected.value=selected.value.includes(path)?selected.value.filter(p=>p!==path):[...selected.value,path];return;} emit('select',materialURL(path)); show.value=false; }
function confirmMany(){if((!props.relative&&!props.origin)||!selected.value.length||selected.value.length>100)return;emit('selectMany',selected.value.map(materialURL));show.value=false;}
</script>
<template>
 <NButton @click="open">{{multiple?'从素材库批量添加':'从素材库选择 / 上传'}}</NButton>
 <NModal v-model:show="show" preset="card" title="图片素材" style="width:min(760px,94vw)" :mask-closable="false" :closable="!busy" :close-on-esc="!busy">
  <p>支持多选 PNG、JPEG、GIF、WebP，每张最大 5 MiB，每批最多 100 张。</p><input type="file" multiple accept="image/png,image/jpeg,image/gif,image/webp" :disabled="busy" @change="upload" />
  <div role="status"><p v-for="(result,i) in uploadResults" :key="i">{{result}}</p></div>
  <NButton v-if="multiple" :disabled="busy||!selected.length||selected.length>100" @click="confirmMany">确认添加 {{selected.length}} 张（最多 100 张）</NButton>
  <div class="materials"><button v-for="item in items" :key="item.ID" :aria-pressed="multiple?selected.includes(item.Path):undefined" :disabled="busy" @click="choose(item.Path)"><img :src="item.Path" :alt="item.Name" /><span>{{selected.includes(item.Path)?'已选择':'选择图片'}}</span></button></div>
  <NEmpty v-if="!items.length" description="暂无素材，请上传图片" />
 </NModal>
</template>
<style scoped>.materials{display:grid;grid-template-columns:repeat(auto-fill,minmax(130px,1fr));gap:16px;margin:24px 0}.materials button{border:1px solid #dce6f2;background:white;border-radius:12px;padding:12px;cursor:pointer}.materials img{width:100%;height:120px;object-fit:contain}.materials span{display:block}</style>
