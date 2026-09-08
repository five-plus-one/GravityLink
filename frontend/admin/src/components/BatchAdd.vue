<script setup lang="ts">
import {computed,ref} from 'vue';
import {NButton,NModal,NInput,NSelect,NInputNumber,NAlert,NFormItem} from 'naive-ui';
import {request,createLink,type DomainItem} from '../api';
import {parseURLBatch} from './batch';
const props=defineProps<{linkId?:number;domains?:DomainItem[]}>();
const emit=defineEmits<{saved:[]}>();
const show=ref(false),text=ref(''),busy=ref(false),submitted=ref(false),error=ref('');
const domain=ref<number|null>(null),limit=ref<number|null>(null);
const results=ref<{line:number;result:string}[]>([]);
const rows=computed(()=>parseURLBatch(text.value));
const valid=computed(()=>rows.value.length>0&&rows.value.length<=100&&rows.value.every(r=>!r.error)&&(!!props.linkId||!!domain.value));
function open(){show.value=true;text.value='';submitted.value=false;results.value=[];error.value='';}
async function submit(){
 if(!valid.value||busy.value||submitted.value)return;
 busy.value=true;submitted.value=true;error.value='';
 try{
  if(props.linkId){
   await request(`/api/admin/links/${props.linkId}/targets/batch`,{method:'POST',body:JSON.stringify({urls:rows.value.map(r=>r.url),scan_limit:limit.value})});
   results.value=rows.value.map(r=>({line:r.line,result:'已添加'}));
  }else{
   for(const row of rows.value){
    try{const link=await createLink({type:'short',entry_domain_id:domain.value!,target_url:row.url,title:`批量短链 ${row.line}`});
     const d=props.domains?.find(d=>d.ID===domain.value);results.value.push({line:row.line,result:link.PublicURL||`${d?.Scheme}://${d?.Host}/${link.Code}`});
    }catch(e){results.value.push({line:row.line,result:`未确认成功：${String(e)}`});error.value='已停止后续提交。请先刷新链接列表核对结果，再新建批次提交剩余条目。';break;}
   }
  }
 }catch(e){error.value=`${String(e)}。请刷新列表核对后再提交新批次。`;}
 finally{busy.value=false;emit('saved');}
}
</script>
<template>
 <NButton @click="open">{{linkId?'批量添加二维码':'批量创建短链'}}</NButton>
 <NModal v-model:show="show" preset="card" :title="linkId?'批量添加二维码':'批量创建短链'" style="width:min(760px,94vw)" :mask-closable="false" :closable="!busy" :close-on-esc="!busy">
  <NAlert>每行一个 {{linkId?'二维码图片':'目标网页'}} HTTP(S) 地址，最多 100 条。先检查下方预览，再提交。{{linkId?'添加的是群二维码图片，不是活码入口地址。':'成功结果不会重复提交。'}}</NAlert>
  <NFormItem v-if="!linkId" label="入口域名"><NSelect v-model:value="domain" :disabled="busy||submitted" :options="(domains||[]).filter(d=>d.Type==='entry'&&d.Status==='active').map(d=>({label:`${d.Scheme}://${d.Host}`,value:d.ID}))"/></NFormItem>
  <NFormItem v-else label="统一扫码阈值（留空不限）"><NInputNumber v-model:value="limit" :min="1" :precision="0" :disabled="busy||submitted"/></NFormItem>
  <NInput v-model:value="text" type="textarea" :disabled="busy||submitted" :autosize="{minRows:5,maxRows:10}" placeholder="https://example.com/one&#10;https://example.com/two"/>
  <p>共 {{rows.length}} 条{{rows.length>100?'，超过 100 条上限':''}}</p>
  <div class="preview"><div v-for="r in rows" :key="r.line" :class="{invalid:r.error}">第 {{r.line}} 行：{{r.error||r.url}}</div></div>
  <NAlert v-if="error" type="error">{{error}}</NAlert>
  <div v-if="results.length" class="preview"><div v-for="r in results" :key="r.line">第 {{r.line}} 行：{{r.result}}</div></div>
  <NButton type="primary" :loading="busy" :disabled="!valid||submitted" @click="submit">{{submitted?'本批次已提交':'确认添加'}}</NButton>
  <NButton :disabled="busy" @click="show=false">关闭</NButton>
 </NModal>
</template>
<style scoped>.preview{max-height:230px;overflow:auto;padding:12px;background:#f5f8fd;border-radius:10px;margin:12px 0;overflow-wrap:anywhere}.invalid{color:#c53030}</style>
