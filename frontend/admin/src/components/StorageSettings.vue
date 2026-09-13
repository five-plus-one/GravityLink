<script setup lang="ts">
import {onMounted,onScopeDispose,reactive,ref} from 'vue';
import {NButton,NForm,NFormItem,NInput,NSwitch,NAlert,NProgress} from 'naive-ui';
import {request} from '../api';
const form=reactive({enabled:false,endpoint:'',region:'',bucket:'',prefix:'img',cdn:'',access_key:'',secret_key:'',path_style:false,secret_configured:false});
const testing=ref(false),checkResult=ref<{uploaded:boolean;accessible:boolean;message:string;url:string}|null>(null);
async function testUpload(){testing.value=true;error.value='';checkResult.value=null;try{checkResult.value=await request('/api/admin/storage/check',{method:'POST',body:JSON.stringify(form)})}catch(e){error.value=e instanceof Error?e.message:'检测失败'}finally{testing.value=false}}
const busy=ref(false),migrating=ref(false),error=ref(''),notice=ref(''),total=ref(0),done=ref(0),results=ref<string[]>([]);let stopped=false;
onScopeDispose(()=>{stopped=true});
onMounted(async()=>{try{Object.assign(form,await request('/api/admin/storage'))}catch(e){error.value=e instanceof Error?e.message:'读取失败'}});
async function save(){busy.value=true;error.value='';notice.value='';try{await request('/api/admin/storage',{method:'PUT',body:JSON.stringify(form)});form.secret_key='';Object.assign(form,await request('/api/admin/storage'));notice.value='存储配置已保存'}catch(e){error.value=e instanceof Error?e.message:'保存失败'}finally{busy.value=false}}
async function migrate(){migrating.value=true;error.value='';results.value=[];done.value=0;try{
 const {items}=await request<{items:{ID:number;Name:string;Path:string}[]}>('/api/admin/materials');const local=items.filter(i=>i.Path.startsWith('/uploads/'));total.value=local.length;
 for(const item of local){if(stopped)break;try{await request(`/api/admin/materials/${item.ID}/migrate`,{method:'POST'});results.value.push(`${item.Name}：已上传`)}catch(e){results.value.push(`${item.Name}：${e instanceof Error?e.message:'上传失败'}`)}done.value++}
 if(!stopped)notice.value=local.length?'本批上传结束，失败图片可重试':'全部图片已使用云存储';
 }catch(e){error.value=e instanceof Error?e.message:'读取图片失败'}finally{migrating.value=false}}
</script>
<template>
 <div class="storage-settings">
 <NAlert v-if="error" type="error">{{error}}</NAlert><NAlert v-if="notice" type="success">{{notice}}</NAlert>
 <NForm label-placement="top" :disabled="busy||migrating||testing">
  <NFormItem label="上传至对象存储"><NSwitch v-model:value="form.enabled" /></NFormItem>
  <div class="storage-fields">
   <NFormItem label="存储服务地址（Endpoint）"><NInput v-model:value="form.endpoint" placeholder="https://s3.oss-cn-shanghai.aliyuncs.com" /></NFormItem>
   <NFormItem label="地域（Region）"><NInput v-model:value="form.region" placeholder="cn-shanghai" /></NFormItem>
   <NFormItem label="存储桶名称（Bucket）"><NInput v-model:value="form.bucket" placeholder="存储桶名称" /></NFormItem>
   <NFormItem label="上传目录"><NInput v-model:value="form.prefix" placeholder="img" /></NFormItem>
   <NFormItem label="Access Key ID"><NInput v-model:value="form.access_key" autocomplete="off" /></NFormItem>
   <NFormItem label="Secret Access Key"><NInput v-model:value="form.secret_key" type="password" show-password-on="click" autocomplete="new-password" :placeholder="form.secret_configured?'已配置，留空保留':'输入访问密钥'" /></NFormItem>
   <NFormItem label="CDN 访问域名"><NInput v-model:value="form.cdn" placeholder="https://img.example.com" /></NFormItem>
   <NFormItem label="路径寻址"><NSwitch v-model:value="form.path_style" /></NFormItem>
  </div>
  <NButton type="primary" :loading="busy" @click="save">保存存储配置</NButton><NButton :loading="testing" :disabled="busy||migrating" style="margin-left:10px" @click="testUpload">检测上传</NButton>
 </NForm>
 <NAlert v-if="checkResult" :type="checkResult.accessible?'success':'warning'" style="margin-top:12px">{{checkResult.message}}<a v-if="checkResult.url" :href="checkResult.url" target="_blank" rel="noopener noreferrer"> 查看检测图片</a></NAlert>
 <div class="migration"><h3>本地图片上传</h3><NButton :disabled="!form.enabled||busy||!form.secret_configured" :loading="migrating" @click="migrate">上传全部本地图片 / 重试</NButton>
 <NProgress v-if="total" type="line" :percentage="Math.round(done/total*100)" /><p v-if="total">{{done}} / {{total}}</p>
 <div class="migration-results" role="status"><p v-for="(result,i) in results" :key="i">{{result}}</p></div></div>
 </div>
</template>
<style scoped>.storage-fields{display:grid;grid-template-columns:1fr 1fr;gap:0 20px}.storage-settings{max-width:900px}.migration{margin-top:28px}.migration h3{margin-bottom:12px}.migration-results{max-height:250px;overflow:auto;overflow-wrap:anywhere}@media(max-width:768px){.storage-fields{grid-template-columns:1fr}}</style>
