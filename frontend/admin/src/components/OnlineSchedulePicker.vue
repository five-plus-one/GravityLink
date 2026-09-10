<script setup lang="ts">
import { computed } from 'vue';
import { NButton } from 'naive-ui';

// 每周 7 天 × 3 时段，存 JSON：{"0":[["09:00","12:00"]],...}，key 为周几（0=周日）
type Slot = { start: string; end: string };
type Week = Slot[][]; // [dayIndex 0-6][slotIndex 0-2]

const props = defineProps<{ modelValue: string }>();
const emit = defineEmits<{ 'update:modelValue': [value: string] }>();

const dayLabels = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
const slotLabels = ['时段一', '时段二', '时段三'];

function emptyWeek(): Week {
  return Array.from({ length: 7 }, () => [
    { start: '', end: '' },
    { start: '', end: '' },
    { start: '', end: '' },
  ]);
}

function parseWeek(raw: string): Week {
  const week = emptyWeek();
  if (!raw?.trim()) return week;
  try {
    const data = JSON.parse(raw) as Record<string, [string, string][]>;
    for (let d = 0; d < 7; d++) {
      const slots = data[String(d)];
      if (!Array.isArray(slots)) continue;
      for (let s = 0; s < 3 && s < slots.length; s++) {
        const pair = slots[s];
        if (Array.isArray(pair) && pair.length === 2) {
          week[d][s] = { start: String(pair[0] || ''), end: String(pair[1] || '') };
        }
      }
    }
  } catch {
    /* ignore */
  }
  return week;
}

function serialize(week: Week): string {
  const out: Record<string, [string, string][]> = {};
  for (let d = 0; d < 7; d++) {
    const pairs: [string, string][] = [];
    for (const slot of week[d]) {
      if (slot.start && slot.end && slot.start < slot.end) {
        pairs.push([slot.start, slot.end]);
      }
    }
    if (pairs.length) out[String(d)] = pairs;
  }
  return Object.keys(out).length ? JSON.stringify(out) : '';
}

const week = computed(() => parseWeek(props.modelValue));

function setSlot(day: number, slot: number, field: 'start' | 'end', value: string | null) {
  const next = parseWeek(props.modelValue);
  next[day][slot][field] = value || '';
  emit('update:modelValue', serialize(next));
}

function applyAllDays(day: number) {
  const next = parseWeek(props.modelValue);
  const source = next[day];
  const hasContent = source.some((s) => s.start || s.end);
  if (!hasContent) return;
  for (let d = 0; d < 7; d++) {
    if (d === day) continue;
    next[d] = source.map((s) => ({ ...s }));
  }
  emit('update:modelValue', serialize(next));
}

function clearAll() {
  emit('update:modelValue', '');
}

function quickFillPreset() {
  const next = emptyWeek();
  for (let d = 1; d <= 5; d++) {
    next[d][0] = { start: '09:00', end: '12:00' };
    next[d][1] = { start: '13:30', end: '18:30' };
  }
  emit('update:modelValue', serialize(next));
}

const hasSchedule = computed(() => props.modelValue.trim() !== '');
</script>

<template>
  <div class="schedule-picker">
    <div class="schedule-toolbar">
      <span class="muted">{{ hasSchedule ? '按配置时段判断在线状态' : '未配置 = 全天在线' }}</span>
      <div class="schedule-actions">
        <NButton size="tiny" quaternary type="primary" @click="quickFillPreset">工作日 9-12 / 13:30-18:30</NButton>
        <NButton size="tiny" quaternary @click="clearAll">清空（全天在线）</NButton>
      </div>
    </div>
    <div class="schedule-grid">
      <div v-for="(day, d) in week" :key="d" class="schedule-day" :class="{ active: day.some((s) => s.start || s.end) }">
        <div class="day-head">
          <strong>{{ dayLabels[d] }}</strong>
          <NButton v-if="day.some((s) => s.start || s.end)" text size="tiny" type="primary" @click="applyAllDays(d)">应用到全周</NButton>
        </div>
        <div v-for="(slot, s) in day" :key="s" class="slot-row">
          <span class="slot-label">{{ slotLabels[s] }}</span>
          <input
            type="time"
            :value="slot.start"
            @change="setSlot(d, s, 'start', ($event.target as HTMLInputElement).value)"
          />
          <span class="dash">–</span>
          <input
            type="time"
            :value="slot.end"
            @change="setSlot(d, s, 'end', ($event.target as HTMLInputElement).value)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.schedule-picker {
  width: 100%;
  border: 1px solid var(--color-border, #e3eaed);
  border-radius: 10px;
  background: var(--color-bg-subtle, #f7fafb);
  padding: 12px;
}

.schedule-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 10px;
  flex-wrap: wrap;
}

.schedule-actions {
  display: flex;
  gap: 4px;
}

.muted {
  font-size: 12px;
  color: var(--color-text-secondary, #6b7f88);
}

.schedule-grid {
  display: grid;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  gap: 6px;
}

.schedule-day {
  border: 1px solid var(--color-border, #e3eaed);
  border-radius: 8px;
  padding: 8px 6px;
  background: #fff;
  display: grid;
  gap: 6px;
}

.schedule-day.active {
  border-color: var(--color-primary, #0f766e);
  background: color-mix(in srgb, var(--color-primary, #0f766e) 4%, #fff);
}

.day-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 2px;
  min-height: 22px;
}

.day-head strong {
  font-size: 12px;
}

.slot-row {
  display: grid;
  grid-template-columns: auto 1fr auto 1fr;
  align-items: center;
  gap: 2px;
  font-size: 11px;
}

.slot-label {
  color: var(--color-text-tertiary, #7a8990);
  writing-mode: horizontal-tb;
  white-space: nowrap;
}

.slot-row input[type='time'] {
  width: 100%;
  min-width: 0;
  border: 1px solid var(--color-border, #dbe3e6);
  border-radius: 4px;
  padding: 2px 3px;
  font-size: 11px;
  font-family: inherit;
  background: #fff;
}

.dash {
  color: var(--color-text-tertiary, #7a8990);
}

@media (max-width: 900px) {
  .schedule-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .slot-label {
    display: none;
  }
  .slot-row {
    grid-template-columns: 1fr auto 1fr;
  }
}
</style>
