import {describe,it,expect} from 'vitest';
import {parseURLBatch} from './batch';
describe('batch input',()=>{
 it('preserves original line numbers and flags duplicates and unsafe URLs',()=>{
  const r=parseURLBatch('\n https://example.com/a \r\nhttps://example.com/a\njavascript:x\nhttps://u:p@example.com');
  expect(r.map(x=>x.line)).toEqual([2,3,4,5]);expect(r[0].error).toBe('');expect(r.slice(1).every(x=>x.error)).toBe(true);
 });
 it('accepts ports and query strings',()=>expect(parseURLBatch('http://localhost:18080/a?b=1')[0].error).toBe(''));
});
