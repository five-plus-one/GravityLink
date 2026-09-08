import { describe,it,expect,vi } from 'vitest';
import { defineComponent,h } from 'vue';
import { mount,flushPromises } from '@vue/test-utils';
import { createRouter,createMemoryHistory,RouterView } from 'vue-router';
import { NMessageProvider } from 'naive-ui';
import AdminLayout from './AdminLayout.vue';
vi.mock('../stores/auth',()=>({useAuthStore:()=>({user:{username:'admin',role:'super_admin'},signOut:vi.fn()})}));
describe('admin route transitions',()=>{
 it('renders each destination after leaving a multi-root page',async()=>{
  const multi=defineComponent({setup:()=>()=>[h('div','card contents'),h('div','modal placeholder')]});
  const routes=['domains','landing-pages','stats'].map(name=>({path:name,component:defineComponent({setup:()=>()=>h('section',`${name} contents`)})}));
  const router=createRouter({history:createMemoryHistory(),routes:[{path:'/',component:AdminLayout,children:[{path:'share-cards',component:multi},...routes]}]});
  await router.push('/share-cards');await router.isReady();
  const wrapper=mount(defineComponent({setup:()=>()=>h(NMessageProvider,null,{default:()=>h(RouterView)})}),{global:{plugins:[router],stubs:{transition:false}},attachTo:document.body});
  try{
   await flushPromises();
   for(const path of ['domains','share-cards','landing-pages','share-cards','stats']){
    await router.push('/'+path);await flushPromises();await new Promise(r=>setTimeout(r,100));await flushPromises();
    expect(wrapper.text()).toContain(path==='share-cards'?'card contents':`${path} contents`);
   }
  }finally{wrapper.unmount();}
 });
});
