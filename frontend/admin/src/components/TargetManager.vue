<script setup lang="ts">
import { computed,ref,watch } from 'vue';
import { NButton,NModal,NInput,NInputNumber,NSelect,NDatePicker,NFormItem,NAlert,NPopconfirm,useMessage } from 'naive-ui';
import { request } from '../api';
import MaterialPicker from './MaterialPicker.vue';
import BatchAdd from './BatchAdd.vue';
const props=defineProps<{linkId:number;origin:string}>();
type Target={ID:number;Label:string;TargetURL:string;Weight:number;ScanLimit:number|null;ScanCount:number;Priority:number;Status:string;ExpireAt:string|null;Owner:string};
const show=ref(false),busy=ref(false),mode=ref('round_robin');
const items=ref<Target[]>([]),editing=ref<Target|null>(null),expiry=ref<number|null>(null);
const message=useMessage();
const imageError=ref(false);
function state(t:Target){if(t.Status!=='active')return '已停用';if(t.ExpireAt&&Date.parse(t.ExpireAt)<=Date.now())return '已到期';if(t.ScanLimit!==null&&t.ScanCount>=t.ScanLimit)return '已满额';return '可分发';}
const available=computed(()=>items.value.filter(t=>state(t)==='可分发').length);
async function resetCount(t:Target){if(busy.value)return;busy.value=true;try{await request(`/api/admin/links/${props.linkId}/targets/${t.ID}/reset-count`,{method:'POST',body:JSON.stringify({expected_count:t.ScanCount,confirmation:'RESET TARGET COUNT'})});await load();message.success('分发计数已重置，访问统计保留；二维码仍为停用状态');}catch(e){message.error(String(e));await refresh();}finally{busy.value=false;}}
async function refresh(){try{await load();}catch(e){message.error(String(e));}}
async function addMaterials(urls:string[]){if(busy.value)return;busy.value=true;try{await request(`/api/admin/links/${props.linkId}/targets/batch`,{method:'POST',body:JSON.stringify({urls})});await load();message.success(`已添加 ${urls.length} 张二维码，请按需设置阈值和到期时间`);}catch(e){message.error(String(e));}finally{busy.value=false;}}
async function toggle(t:Target){if(busy.value)return;busy.value=true;try{await request(`/api/admin/links/${props.linkId}/targets`,{method:'POST',body:JSON.stringify({...t,Status:t.Status==='active'?'disabled':'active'})});await load();}catch(e){message.error(String(e));}finally{busy.value=false;}}
watch(()=>editing.value?.TargetURL,()=>{imageError.value=false;});
async function load(){const r=await request<{mode:string;items:Target[]}>(`/api/admin/links/${props.linkId}/targets`);items.value=r.items;mode.value=r.mode;}
async function open(){try{await load();show.value=true;editing.value=null;}catch(e){message.error(String(e));}}
function edit(t?:Target){editing.value=t?{...t}:{ID:0,Label:'',TargetURL:'',Weight:1,ScanLimit:null,ScanCount:0,Priority:0,Status:'active',ExpireAt:null,Owner:''};expiry.value=t?.ExpireAt?Date.parse(t.ExpireAt):null;}
async function save(){if(!editing.value||busy.value)return;busy.value=true;try{await request(`/api/admin/links/${props.linkId}/targets`,{method:'POST',body:JSON.stringify({...editing.value,ExpireAt:expiry.value?new Date(expiry.value).toISOString():null})});await load();editing.value=null;message.success('二维码配置已保存');}catch(e){message.error(String(e));}finally{busy.value=false;}}
async function saveMode(){if(busy.value)return;busy.value=true;try{await request(`/api/admin/links/${props.linkId}/strategy`,{method:'PUT',body:JSON.stringify({mode:mode.value})});message.success('展示模式已保存');}catch(e){message.error(String(e));}finally{busy.value=false;}}
</script>
<template>
 <NButton text size="small" @click="open">二维码配置</NButton>
 <NModal v-model:show="show" preset="card" title="活码二维码配置" style="width:min(1000px,96vw)" :mask-closable="false" :closable="!busy" :close-on-esc="!busy">
  <div class="toolbar"><NSelect v-model:value="mode" :disabled="busy" :options="[{label:'顺序阈值',value:'round_robin'},{label:'随机权重',value:'weighted'}]" style="width:180px"/><NButton :disabled="busy" @click="saveMode">保存模式</NButton><NButton type="primary" :disabled="busy" @click="edit()">添加二维码</NButton><BatchAdd :link-id="linkId" @saved="refresh"/><MaterialPicker multiple relative :origin="origin" @select-many="addMaterials"/></div>
  <NAlert>顺序模式优先级越高越先展示，达到阈值后切换；阈值不限时会持续使用该码。随机模式按权重分配，不按优先级。停用、到期、满额目标不参与分发。</NAlert>
  <p>共 {{items.length}} 个二维码，当前 {{available}} 个可分发。<NButton :disabled="busy" @click="refresh">刷新状态</NButton></p>
  <NAlert v-if="!available" type="warning">暂无可分发二维码，请添加二维码，或检查启停、阈值和到期时间。需要重置阈值计数时请先停用该二维码。</NAlert>
  <div style="overflow:auto"><table><thead><tr><th>名称</th><th>访问 / 阈值</th><th>群主</th><th>到期</th><th>状态</th><th>操作</th></tr></thead><tbody><tr v-for="t in items" :key="t.ID"><td>{{t.Label||'未命名'}}</td><td>{{t.ScanCount}} / {{t.ScanLimit??'无限制'}}</td><td>{{t.Owner||'未设置'}}</td><td>{{t.ExpireAt?new Date(t.ExpireAt).toLocaleString():'不限'}}</td><td>{{state(t)}}</td><td><NButton :disabled="busy" @click="edit(t)">编辑</NButton><NButton :disabled="busy" @click="toggle(t)">{{t.Status==='active'?'停用':'启用'}}</NButton><NPopconfirm @positive-click="resetCount(t)"><template #trigger><NButton :disabled="busy||t.Status!=='disabled'||!t.ScanCount">重置计数</NButton></template>将该二维码分发计数从 {{t.ScanCount}} 归零，历史访问统计保留。确认重置？</NPopconfirm></td></tr></tbody></table></div>
  <section v-if="editing">
   <NFormItem label="名称"><NInput v-model:value="editing.Label"/></NFormItem>
   <NFormItem label="二维码图片 URL"><NInput v-model:value="editing.TargetURL"/></NFormItem><MaterialPicker relative :origin="origin" @select="editing!.TargetURL=$event"/>
   <img v-if="editing.TargetURL" v-show="!imageError" :src="editing.TargetURL" alt="目标二维码预览" style="display:block;width:160px;height:160px;object-fit:contain;margin:16px 0" @error="imageError=true" @load="imageError=false"/>
   <NAlert v-if="imageError" type="error">二维码图片加载失败，请检查地址或从素材库重新上传。</NAlert>
   <div class="fields"><NFormItem label="扫码阈值"><NInputNumber v-model:value="editing.ScanLimit" :min="1" clearable/></NFormItem><NFormItem label="权重"><NInputNumber v-model:value="editing.Weight" :min="1"/></NFormItem><NFormItem label="优先级"><NInputNumber v-model:value="editing.Priority"/></NFormItem><NFormItem label="群主"><NInput v-model:value="editing.Owner"/></NFormItem><NFormItem label="到期时间"><NDatePicker v-model:value="expiry" type="datetime" clearable/></NFormItem><NFormItem label="状态"><NSelect v-model:value="editing.Status" :options="[{label:'启用',value:'active'},{label:'停用',value:'disabled'}]"/></NFormItem></div>
   <NButton type="primary" :loading="busy" @click="save">保存二维码</NButton><NButton @click="editing=null">取消编辑</NButton>
  </section>
 </NModal>
</template>
<style scoped>.toolbar{display:flex;flex-wrap:wrap;gap:12px;margin-bottom:20px}table{width:100%;border-collapse:collapse;margin:20px 0}td,th{text-align:left;padding:14px;border-bottom:1px solid #e6ecf5;white-space:nowrap}.fields{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}section{background:#f5f8fd;padding:20px;border-radius:16px}</style>
