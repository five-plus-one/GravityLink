<script setup lang="ts">
import {computed,ref,watch} from 'vue';
import {NModal,NRadioGroup,NRadioButton,NInput,NButton,NCheckbox,NSelect,NAlert,NPopconfirm,useMessage} from 'naive-ui';
import {request} from '../api';
const props=defineProps<{show:boolean;categories:string[];tags:string[]}>();
const emit=defineEmits<{ 'update:show':[boolean];saved:[] }>();
const kind=ref('category'),selected=ref<string[]>([]),search=ref(''),name=ref(''),target=ref<string|null>(null),busy=ref(false),error=ref('');
const message=useMessage();
const values=computed(()=>kind.value==='category'?props.categories:props.tags);
const filtered=computed(()=>values.value.filter(v=>v.toLowerCase().includes(search.value.toLowerCase())));
const options=computed(()=>values.value.filter(v=>!selected.value.includes(v)).map(v=>({label:v,value:v})));
watch([kind,()=>props.show],()=>{selected.value=[];search.value='';name.value='';target.value=null;error.value=''});
function select(v:string,on:boolean){selected.value=on?[...selected.value,v]:selected.value.filter(x=>x!==v)}
async function apply(action:string){
 busy.value=true;error.value='';
 try{await request('/api/v1/link-labels',{method:'POST',body:JSON.stringify({kind:kind.value,action,names:action==='add'?[name.value]:selected.value,target:target.value||''})});selected.value=[];name.value='';target.value=null;emit('saved');message.success('已保存')}
 catch(e){error.value=e instanceof Error?e.message:'保存失败'}finally{busy.value=false}
}
</script>
<template>
<NModal :show="show" @update:show="v=>emit('update:show',v)" preset="card" title="分类与标签" style="width:min(600px,94vw)" :mask-closable="false">
 <NRadioGroup v-model:value="kind" :disabled="busy"><NRadioButton value="category">分类</NRadioButton><NRadioButton value="tag">标签</NRadioButton></NRadioGroup>
 <div class="row"><NInput v-model:value="name" :maxlength="kind==='category'?80:40" placeholder="输入新名称" @keyup.enter="name.trim()&&apply('add')"/><NButton type="primary" :disabled="!name.trim()" :loading="busy" @click="apply('add')">添加</NButton></div>
 <NInput v-model:value="search" placeholder="查找名称" clearable/>
 <div class="row"><NCheckbox :checked="!!filtered.length&&filtered.every(v=>selected.includes(v))" @update:checked="v=>selected=v?[...filtered]:[]">全选当前结果</NCheckbox><span>已选 {{selected.length}}</span></div>
 <div class="catalog"><label v-for="v in filtered" :key="v"><NCheckbox :checked="selected.includes(v)" @update:checked="on=>select(v,on)">{{v}}</NCheckbox></label><p v-if="!filtered.length">暂无匹配项</p></div>
 <div v-if="selected.length" class="operations">
  <div class="row"><NSelect v-model:value="target" :options="options" filterable tag clearable placeholder="选择或输入新名称"/><NButton :disabled="!target || busy" @click="apply('move')">{{selected.length===1?'重命名 / 转移':'合并转移'}}</NButton></div>
  <NPopconfirm @positive-click="apply('remove')" positive-text="删除" negative-text="取消"><template #trigger><NButton type="error" secondary :disabled="busy">删除所选 {{selected.length}} 项</NButton></template>{{kind==='category'?'所选分类中的链接将归入未分类。':'所选标签将从关联链接中移除。'}}</NPopconfirm>
 </div>
 <NAlert v-if="error" type="error">{{error}}</NAlert>
</NModal>
</template>
<style scoped>
.row{display:flex;align-items:center;gap:10px;margin:12px 0}.row .n-input,.row .n-select{flex:1;min-width:0}.catalog{max-height:30vh;overflow:auto;border:1px solid #e5e7eb;border-radius:6px;padding:10px}.catalog label{display:block;padding:7px;overflow-wrap:anywhere}.operations{border-top:1px solid #e5e7eb;margin-top:12px;padding-top:4px}
</style>

