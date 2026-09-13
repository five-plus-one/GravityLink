export type TimePreset='1h'|'today'|'24h'|'week'|'7d'|'month'|'30d';
export const timePresets=[{key:'1h',label:'最近1小时'},{key:'today',label:'今天'},{key:'24h',label:'最近24小时'},{key:'week',label:'本周'},{key:'7d',label:'最近7天'},{key:'month',label:'本月'},{key:'30d',label:'最近30天'}] as const;
export function presetRange(key:TimePreset,now=new Date()):[number,number]{
 const start=new Date(now);const end=now.getTime();
 if(key==='today'||key==='week'||key==='month')start.setHours(0,0,0,0);
 if(key==='week')start.setDate(start.getDate()-(start.getDay()+6)%7);
 if(key==='month')start.setDate(1);
 if(key==='1h')return[end-3600000,end];
 if(key==='24h')return[end-86400000,end];
 if(key==='7d')return[end-7*86400000,end];
 if(key==='30d')return[end-30*86400000,end];
 return[start.getTime(),end];
}
export function hourLabel(p:{start?:string;hour:number}){if(!p.start)return `${p.hour}:00`;const d=new Date(p.start);return `${d.getMonth()+1}/${d.getDate()} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`}
