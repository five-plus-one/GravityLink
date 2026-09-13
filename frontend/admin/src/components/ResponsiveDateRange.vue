<script setup lang="ts">
import {ref,watch} from 'vue';
import {NDatePicker,NButton} from 'naive-ui';
import {timePresets,presetRange,type TimePreset} from '../timeRange';
const props=defineProps<{value:[number,number]|null;withTime?:boolean}>();
const emit=defineEmits<{'update:value':[value:[number,number]|null]}>();
const panel=ref(false);
function selectPreset(key:TimePreset){emit('update:value',presetRange(key));panel.value=false;}
const start=ref<number|null>(null),end=ref<number|null>(null);
watch(()=>props.value,v=>{start.value=v?.[0]??null;end.value=v?.[1]??null},{immediate:true});
function update(which:'start'|'end',value:number|null){
 if(which==='start') start.value=value; else end.value=value;
 if(value===null){emit('update:value',null);return;}
 if(start.value!==null && end.value!==null) emit('update:value',[start.value,end.value]);
}
</script>
<template>
 <div class="responsive-range">
  <NDatePicker class="range-desktop" :value="value" :type="withTime?'datetimerange':'daterange'" v-model:show="panel" clearable @update:value="v=>emit('update:value',v as [number,number]|null)">
   <template #footer><div class="date-presets"><NButton v-for="p in timePresets" :key="p.key" size="tiny" @click="selectPreset(p.key)">{{p.label}}</NButton></div></template>
  </NDatePicker>
  <div class="range-mobile"><div class="date-presets"><NButton v-for="p in timePresets" :key="p.key" size="tiny" @click="selectPreset(p.key)">{{p.label}}</NButton></div>
   <label>开始{{ withTime?'时间':'日期' }}<NDatePicker :value="start" :type="withTime?'datetime':'date'" clearable placeholder="选择开始" @update:value="v=>update('start',v)" /></label>
   <label>结束{{ withTime?'时间':'日期' }}<NDatePicker :value="end" :type="withTime?'datetime':'date'" clearable placeholder="选择结束" @update:value="v=>update('end',v)" /></label>
  </div>
 </div>
</template>
<style scoped>
.date-presets{display:flex;flex-wrap:wrap;gap:6px;max-width:580px}
.responsive-range{min-width:0}.range-mobile{display:none}
@media(max-width:768px){.range-desktop{display:none}.range-mobile{display:grid;gap:10px}.range-mobile label{display:grid;gap:5px;font-size:12px;color:var(--color-text-tertiary)}}
</style>

