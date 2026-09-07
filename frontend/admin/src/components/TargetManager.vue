<script setup lang="ts">
import { ref } from 'vue';
import { NButton,NModal,NInput,NInputNumber,NSelect,NDatePicker,NFormItem,NAlert,useMessage } from 'naive-ui';
import { request } from '../api';
import MaterialPicker from './MaterialPicker.vue';
const props=defineProps<{linkId:number;origin:string}>();
type Target={ID:number;Label:string;TargetURL:string;Weight:number;ScanLimit:number|null;ScanCount:number;Priority:number;Status:string;ExpireAt:string|null;Owner:string};
const show=ref(false),busy=ref(false),mode=ref('round_robin');
const items=ref<Target[]>([]),editing=ref<Target|null>(null),expiry=ref<number|null>(null);
const message=useMessage();
async function load(){const r=await request<{mode:string;items:Target[]}>(`/api/admin/links/${props.linkId}/targets`);items.value=r.items;mode.value=r.mode;}
async function open(){try{await load();show.value=true;editing.value=null;}catch(e){message.error(String(e));}}
function edit(t?:Target){editing.value=t?{...t}:{ID:0,Label:'',TargetURL:'',Weight:1,ScanLimit:null,ScanCount:0,Priority:0,Status:'active',ExpireAt:null,Owner:''};expiry.value=t?.ExpireAt?Date.parse(t.ExpireAt):null;}
async function save(){if(!editing.value)return;busy.value=true;try{await request(`/api/admin/links/${props.linkId}/targets`,{method:'POST',body:JSON.stringify({...editing.value,ExpireAt:expiry.value?new Date(expiry.value).toISOString():null})});await load();editing.value=null;message.success('二维码配置已保存');}catch(e){message.error(String(e));}finally{busy.value=false;}}
async function saveMode(){try{await request(`/api/admin/links/${props.linkId}/strategy`,{method:'PUT',body:JSON.stringify({mode:mode.value})});message.success('展示模式已保存');}catch(e){message.error(String(e));}}
</script>
<template>
 <NButton text size="small" @click="open">二维码配置</NButton>
 <NModal v-model:show="show" preset="card" title="活码二维码配置" style="width:min(1000px,96vw)" :mask-closable="false">
  <div class="toolbar"><NSelect v-model:value="mode" :options="[{label:'顺序阈值',value:'round_robin'},{label:'随机权重',value:'weighted'}]" style="width:180px"/><NButton @click="saveMode">保存模式</NButton><NButton type="primary" @click="edit()">添加二维码</NButton></div>
  <NAlert>优先级越高越先展示；顺序模式达到阈值后切换。停用和到期目标不参与分发。</NAlert>
  <div style="overflow:auto"><table><thead><tr><th>名称</th><th>访问 / 阈值</th><th>群主</th><th>到期</th><th>状态</th><th></th></tr></thead><tbody><tr v-for="t in items" :key="t.ID"><td>{{t.Label||'未命名'}}</td><td>{{t.ScanCount}} / {{t.ScanLimit??'无限制'}}</td><td>{{t.Owner||'未设置'}}</td><td>{{t.ExpireAt?new Date(t.ExpireAt).toLocaleString():'不限'}}</td><td>{{t.Status==='active'?'启用':'停用/耗尽'}}</td><td><NButton @click="edit(t)">编辑</NButton></td></tr></tbody></table></div>
  <section v-if="editing">
   <NFormItem label="名称"><NInput v-model:value="editing.Label"/></NFormItem>
   <NFormItem label="二维码图片 URL"><NInput v-model:value="editing.TargetURL"/></NFormItem><MaterialPicker :origin="origin" @select="editing!.TargetURL=$event"/>
   <div class="fields"><NFormItem label="扫码阈值"><NInputNumber v-model:value="editing.ScanLimit" :min="1" clearable/></NFormItem><NFormItem label="权重"><NInputNumber v-model:value="editing.Weight" :min="1"/></NFormItem><NFormItem label="优先级"><NInputNumber v-model:value="editing.Priority"/></NFormItem><NFormItem label="群主"><NInput v-model:value="editing.Owner"/></NFormItem><NFormItem label="到期时间"><NDatePicker v-model:value="expiry" type="datetime" clearable/></NFormItem><NFormItem label="状态"><NSelect v-model:value="editing.Status" :options="[{label:'启用',value:'active'},{label:'停用',value:'disabled'}]"/></NFormItem></div>
   <NButton type="primary" :loading="busy" @click="save">保存二维码</NButton><NButton @click="editing=null">取消编辑</NButton>
  </section>
 </NModal>
</template>
<style scoped>.toolbar{display:flex;gap:12px;margin-bottom:20px}table{width:100%;border-collapse:collapse;margin:20px 0}td,th{text-align:left;padding:14px;border-bottom:1px solid #e6ecf5;white-space:nowrap}.fields{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}section{background:#f5f8fd;padding:20px;border-radius:16px}</style>
