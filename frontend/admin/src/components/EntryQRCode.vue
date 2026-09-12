<script setup lang="ts">
import { ref } from 'vue';
import { NAlert,NButton,NIcon,NModal,useMessage } from 'naive-ui';
import { QrCode } from '@lucide/vue';
const props=defineProps<{url:string;name?:string}>();
const show=ref(false),image=ref(''),busy=ref(false),local=ref(false),message=useMessage();
async function open(){
 busy.value=true;
 try{
  const address=new URL(props.url);if(!['http:','https:'].includes(address.protocol))throw Error();
  local.value=['localhost','127.0.0.1','[::1]'].includes(address.hostname);
  const {default:QRCode}=await import('qrcode');
  image.value=await QRCode.toDataURL(props.url,{width:640,margin:4,errorCorrectionLevel:'M'});show.value=true;
 }catch{message.error('无法生成二维码，请检查完整访问地址');}finally{busy.value=false;}
}
</script>
<template>
 <NButton text size="small" title="入口二维码" :disabled="!url" :loading="busy" @click="open">
  <template #icon><NIcon :size="15"><QrCode /></NIcon></template>
 </NButton>
 <NModal v-model:show="show" preset="card" title="入口二维码" style="width:min(420px,94vw)">
  <NAlert v-if="local" type="warning">当前是本机测试地址。手机扫码使用前，请绑定可从手机访问的域名。</NAlert>
  <img :src="image" alt="入口二维码" style="display:block;width:100%;height:auto"/>
  <p style="overflow-wrap:anywhere">{{url}}</p>
  <NButton tag="a" :href="image" :download="`${name||'gravitylink'}-qr.png`" type="primary">下载 PNG</NButton>
 </NModal>
</template>
