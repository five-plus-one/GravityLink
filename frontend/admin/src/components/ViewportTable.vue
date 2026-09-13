<script setup lang="ts">
import {ref,onMounted,onBeforeUnmount,onUpdated,nextTick,useAttrs} from 'vue';
import {NDataTable} from 'naive-ui';
defineOptions({inheritAttrs:false});
const attrs=useAttrs();
const root=ref<HTMLElement>();const bodyHeight=ref(400);
let observer:ResizeObserver|undefined;let frame=0;
function measure(){cancelAnimationFrame(frame);frame=requestAnimationFrame(()=>{if(!root.value)return;const top=root.value.getBoundingClientRect().top;bodyHeight.value=Math.max(80,Math.floor(window.innerHeight-top-132));})}
onMounted(()=>{if(typeof ResizeObserver!=='undefined')observer=new ResizeObserver(measure);let el=root.value?.parentElement;for(let i=0;el&&i<4;i++,el=el.parentElement)observer?.observe(el);window.addEventListener('resize',measure);window.addEventListener('scroll',measure,true);nextTick(measure)});
onUpdated(measure);
onBeforeUnmount(()=>{observer?.disconnect();cancelAnimationFrame(frame);window.removeEventListener('resize',measure);window.removeEventListener('scroll',measure,true)});
</script>
<template><div ref="root" :class="attrs.class" :style="attrs.style"><NDataTable v-bind="attrs" :class="undefined" :style="undefined" :max-height="bodyHeight"><template v-for="(_,name) in $slots" #[name]="scope"><slot :name="name" v-bind="scope||{}"/></template></NDataTable></div></template>

