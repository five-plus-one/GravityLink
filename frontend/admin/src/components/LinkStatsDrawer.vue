<script setup lang="ts">
import {computed,ref,watch} from 'vue';
import {NDrawer,NDrawerContent,NAlert,NStatistic,NButton,NSpin} from 'naive-ui';
import VChart from 'vue-echarts';
import '../echarts';
import {request,type LinkItem} from '../api';
import ResponsiveDateRange from './ResponsiveDateRange.vue';
import {presetRange} from '../timeRange';
const props=defineProps<{show:boolean;link:LinkItem|null}>();
const emit=defineEmits<{'update:show':[boolean]}>();
const range=ref<[number,number]|null>(presetRange('7d')),busy=ref(false),error=ref('');
const data=ref<{pv:number;uv:number;daily:{date:string;pv:number;uv:number}[];device:{device:{label:string;value:number}[]}}|null>(null);
let generation=0;
async function load(){if(!props.show||!props.link)return;const id=++generation;busy.value=true;error.value='';try{const r=range.value||presetRange('7d');const q=new URLSearchParams({link_id:String(props.link.ID),start:new Date(r[0]).toISOString(),end:new Date(r[1]).toISOString()});const result=await request<NonNullable<typeof data.value>>('/api/v1/stats/window?'+q);if(id===generation)data.value=result}catch(e){if(id===generation)error.value=e instanceof Error?e.message:'读取统计失败'}finally{if(id===generation)busy.value=false}}
watch(()=>[props.show,props.link?.ID],()=>{generation++;data.value=null;if(props.show){range.value=presetRange('7d')}});
watch(range,load);
const option=computed(()=>({tooltip:{trigger:'axis'},legend:{data:['PV','UV'],bottom:0},grid:{left:48,right:16,top:24,bottom:55},xAxis:{type:'category',data:data.value?.daily.map(i=>i.date.slice(5))||[]},yAxis:{type:'value',minInterval:1},series:['pv','uv'].map((key,i)=>({name:key.toUpperCase(),type:'line',data:data.value?.daily.map(d=>key==='pv'?d.pv:d.uv)||[],itemStyle:{color:i?'#0f766e':'#1677ff'}}))}));
const labels:Record<string,string>={mobile:'手机',desktop:'电脑',tablet:'平板',bot:'机器人',unknown:'未知'};
</script>
<template><NDrawer :show="show" width="min(760px,100vw)" placement="right" @update:show="v=>emit('update:show',v)"><NDrawerContent :title="`访问统计 · ${link?.Title||link?.Code||''}`" closable>
<div class="controls"><ResponsiveDateRange v-model:value="range" with-time/><NButton :loading="busy" @click="load">刷新</NButton></div>
<NAlert v-if="error" type="error">{{error}}</NAlert><NSpin :show="busy"><div class="totals"><NStatistic label="访问次数 PV" :value="data?.pv||0"/><NStatistic label="访客 UV" :value="data?.uv||0"/></div><VChart :option="option" autoresize style="height:300px"/><h3>设备分布</h3><div class="devices"><span v-for="d in data?.device.device" :key="d.label">{{labels[d.label]||d.label}} <b>{{d.value}}</b></span><span v-if="!data?.device.device?.length">暂无访问记录</span></div></NSpin>
<template #footer><NButton @click="emit('update:show',false)">返回</NButton></template></NDrawerContent></NDrawer></template>
<style scoped>.controls{display:flex;gap:8px;flex-wrap:wrap}.totals{display:grid;grid-template-columns:1fr 1fr;margin:24px 0}.devices{display:flex;gap:16px;flex-wrap:wrap;margin:14px 0}</style>

