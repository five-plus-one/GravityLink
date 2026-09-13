import {describe,it,expect} from 'vitest';
import {presetRange,timePresets} from './timeRange';
describe('time ranges',()=>{
 it('uses rolling durations and calendar boundaries',()=>{const now=new Date(2026,8,13,22,15);for(const [k,n] of [['1h',1],['24h',24],['7d',168],['30d',720]] as const){const [s,e]=presetRange(k,now);expect(e-s).toBe(n*3600000)}expect(new Date(presetRange('week',now)[0]).getDay()).toBe(1);expect(new Date(presetRange('month',now)[0]).getDate()).toBe(1);expect(new Date(presetRange('today',now)[0]).getHours()).toBe(0);expect(timePresets).toHaveLength(7)});
});
