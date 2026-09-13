import {describe,it,expect,vi,beforeEach} from 'vitest';
import {defineComponent,h} from 'vue';
import {mount,flushPromises} from '@vue/test-utils';
import {NMessageProvider,NDataTable,NDatePicker,NInput} from 'naive-ui';
import VisitorsView from './VisitorsView.vue';
import {listVisitors} from '../api';
vi.mock('vue-router',()=>({useRoute:()=>({query:{}})}));
vi.mock('../api',()=>({listLinks:vi.fn().mockResolvedValue({items:[]}),listVisitors:vi.fn()}));
const api=vi.mocked(listVisitors);
function render(){return mount(defineComponent({setup:()=>()=>h(NMessageProvider,null,{default:()=>h(VisitorsView)})}));}
beforeEach(()=>api.mockReset());
describe('visitor queries',()=>{
 it('loads all records once and uses server total for page two',async()=>{
  api.mockResolvedValue({items:[],total:125});
  const w=render();await flushPromises();
  expect(api).toHaveBeenCalledTimes(1);
  expect(api.mock.calls[0][0]).not.toHaveProperty('start');
  const table=w.findComponent(NDataTable);
  expect(table.props('remote')).toBe(true);
  const pagination=table.props('pagination') as any;
  expect(pagination.itemCount).toBe(125);
  pagination.onUpdatePage(2);await flushPromises();
  expect(api).toHaveBeenLastCalledWith(expect.objectContaining({offset:50,limit:50}));
  w.unmount();
 });
 it('submits exact timestamps and resets pagination on search',async()=>{
  api.mockResolvedValue({items:[],total:125});
  const w=render();await flushPromises();
  const start=Date.parse('2026-09-13T00:30:00+08:00'),end=start+3600000;
  w.findComponent(NDatePicker).vm.$emit('update:value',[start,end]);
  await w.find('input[placeholder="搜索 IP、短码或名称"]').setValue('campaign');
  await flushPromises();
  await w.findAll('button').find(b=>b.text()==='查询')!.trigger('click');await flushPromises();
  expect(api).toHaveBeenLastCalledWith({start:new Date(start).toISOString(),end:new Date(end).toISOString(),keyword:'campaign',limit:50,offset:0});
  w.unmount();
 });
 it('ignores a stale first response after changing the range',async()=>{
  let finish!:(value:any)=>void;
  api.mockImplementationOnce(()=>new Promise(resolve=>finish=resolve));
  api.mockResolvedValue({items:[],total:7});
  const w=render();await flushPromises();
  await w.findAll('button').find(b=>b.text()==='今天')!.trigger('click');await flushPromises();
  finish({items:[],total:99});await flushPromises();
  expect((w.findComponent(NDataTable).props('pagination') as any).itemCount).toBe(7);
  w.unmount();
 });
});
