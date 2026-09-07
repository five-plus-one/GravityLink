<script setup lang="ts">
import { computed,onMounted,reactive,ref } from 'vue';
import { NAlert,NButton,NCard,NModal,NInput,NSelect,NFormItem,NTag,NEmpty,useMessage } from 'naive-ui';
import { request,listDomains,type DomainItem } from '../api';
import MaterialPicker from '../components/MaterialPicker.vue';
import EntryQRCode from '../components/EntryQRCode.vue';
type Card={ID:number;DomainID:number;Title:string;Description:string;ImageURL:string;TargetURL:string;Status:string;Visits:number;PublicURL:string};
const items=ref<Card[]>([]),domains=ref<DomainItem[]>([]),show=ref(false),configShow=ref(false),busy=ref(false),loading=ref(false);
const error=ref(''),message=useMessage();
const form=reactive<Card>({ID:0,DomainID:0,Title:'',Description:'',ImageURL:'',TargetURL:'',Status:'active',Visits:0,PublicURL:''});
const config=reactive({appid:'',secret:'',secret_configured:false});
const origin=computed(()=>{const d=domains.value.find(d=>d.ID===form.DomainID);return d?`${d.Scheme}://${d.Host}`:'';});
async function refresh(){loading.value=true;error.value='';try{items.value=(await request<{items:Card[]}>('/api/admin/share-cards')).items;domains.value=(await listDomains()).items.filter(d=>d.Type==='entry'&&d.Status==='active');}catch(e){error.value=String(e);}finally{loading.value=false;}}
onMounted(refresh);
function edit(card?:Card){Object.assign(form,card||{ID:0,DomainID:0,Title:'',Description:'',ImageURL:'',TargetURL:'',Status:'active',Visits:0,PublicURL:''});show.value=true;}
async function save(){busy.value=true;try{await request('/api/admin/share-cards'+(form.ID?`/${form.ID}`:''),{method:form.ID?'PUT':'POST',body:JSON.stringify(form)});show.value=false;await refresh();message.success('卡片已保存');}catch(e){message.error(String(e));}finally{busy.value=false;}}
async function openConfig(){try{Object.assign(config,await request('/api/admin/wechat-config'),{secret:''});configShow.value=true;}catch(e){message.error(String(e));}}
async function saveConfig(){busy.value=true;try{await request('/api/admin/wechat-config',{method:'PUT',body:JSON.stringify({appid:config.appid,secret:config.secret})});config.secret='';configShow.value=false;message.success('公众号配置已保存');}catch(e){message.error(String(e));}finally{busy.value=false;}}
async function copy(card:Card){try{await navigator.clipboard.writeText(card.PublicURL);message.success('分享地址已复制');}catch{message.error('复制失败，请手动复制下方地址');}}
</script>
<template>
 <NCard title="微信分享卡片" :bordered="false">
  <template #header-extra><div class="actions"><NButton @click="openConfig">公众号配置</NButton><NButton :loading="loading" @click="refresh">刷新</NButton><NButton type="primary" @click="edit()">创建卡片</NButton></div></template>
  <NAlert v-if="error" type="error">{{error}}</NAlert>
  <p class="hint">设置标题、摘要和封面，在微信内打开分享地址，通过右上角菜单发送卡片。</p>
  <div class="cards">
   <article v-for="card in items" :key="card.ID">
    <div class="preview"><div><h3>{{card.Title}}</h3><p>{{card.Description}}</p></div><img :src="card.ImageURL" alt="封面"/></div>
    <div class="meta"><NTag :type="card.Status==='active'?'success':'default'">{{card.Status==='active'?'启用':'停用'}}</NTag><span>{{card.Visits}} 次访问</span></div>
    <p class="url">{{card.PublicURL||'入口域不可用'}}</p>
    <div class="actions" style="flex-wrap:wrap">
     <NButton @click="edit(card)">编辑</NButton>
     <NButton :disabled="!card.PublicURL" @click="copy(card)">复制地址</NButton>
     <EntryQRCode :url="card.PublicURL" :name="`card-${card.ID}`"/>
     <NButton tag="a" :href="card.PublicURL||undefined" :disabled="!card.PublicURL" target="_blank" rel="noopener noreferrer">预览</NButton>
    </div>
   </article>
  </div>
  <NEmpty v-if="!loading&&!error&&!items.length" description="尚未创建分享卡片"/>
 </NCard>
 <NModal v-model:show="show" preset="card" :title="form.ID?'编辑分享卡片':'创建分享卡片'" style="width:min(680px,94vw)" :mask-closable="false" :closable="!busy">
  <NFormItem label="入口域名"><NSelect v-model:value="form.DomainID" :options="domains.map(d=>({label:`${d.Scheme}://${d.Host}`,value:d.ID}))"/></NFormItem>
  <NFormItem label="卡片标题"><NInput v-model:value="form.Title" maxlength="128"/></NFormItem><NFormItem label="摘要"><NInput v-model:value="form.Description" type="textarea" maxlength="500"/></NFormItem>
  <NFormItem label="封面图片 URL"><NInput v-model:value="form.ImageURL"/></NFormItem><MaterialPicker :origin="origin" @select="form.ImageURL=$event"/>
  <NFormItem label="目标 URL"><NInput v-model:value="form.TargetURL" placeholder="https://example.com"/></NFormItem>
  <NFormItem label="状态"><NSelect v-model:value="form.Status" :options="[{label:'启用',value:'active'},{label:'停用',value:'disabled'}]"/></NFormItem>
  <NButton type="primary" :loading="busy" @click="save">保存卡片</NButton>
 </NModal>
 <NModal v-model:show="configShow" preset="card" title="公众号配置" style="width:min(640px,94vw)" :mask-closable="false" :closable="!busy">
  <NAlert type="info">需要公众号具备 JS-SDK 权限，并配置服务器 IP 白名单及 JS 接口安全域名。测试号还需满足测试号的关注和权限要求。</NAlert>
  <NFormItem label="AppID"><NInput v-model:value="config.appid" autocomplete="off"/></NFormItem>
  <NFormItem :label="config.secret_configured?'AppSecret（已配置，留空保留）':'AppSecret'"><NInput v-model:value="config.secret" type="password" autocomplete="new-password"/></NFormItem>
  <NButton type="primary" :loading="busy" @click="saveConfig">保存配置</NButton>
 </NModal>
</template>
<style scoped>.actions,.meta{display:flex;gap:10px;align-items:center}.cards{display:grid;grid-template-columns:repeat(auto-fill,minmax(300px,1fr));gap:20px}.cards article{padding:22px;border:1px solid #e6edf6;border-radius:20px}.preview{display:flex;gap:16px;background:#f4f6f9;padding:16px;border-radius:12px;min-height:130px}.preview div{flex:1}.preview h3{margin:0 0 8px}.preview p,.hint{color:#7b8799}.preview img{width:70px;height:70px;object-fit:cover;border-radius:8px}.meta{margin-top:16px}.url{overflow-wrap:anywhere;font-size:12px;color:#64748b}</style>
