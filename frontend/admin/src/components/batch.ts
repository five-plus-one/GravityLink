export function parseURLBatch(text: string) {
 const seen = new Set<string>();
 return text.split(/\r?\n/).map((raw,i)=>({line:i+1,url:raw.trim(),error:''})).filter(r=>r.url).map(r=>{
  try { const u=new URL(r.url);if(!['http:','https:'].includes(u.protocol)||u.username||u.password||r.url.length>2048) throw new Error(); }
  catch { r.error='请输入有效的 HTTP(S) 地址'; }
  if(seen.has(r.url))r.error='本批次地址重复';seen.add(r.url);return r;
 });
}
