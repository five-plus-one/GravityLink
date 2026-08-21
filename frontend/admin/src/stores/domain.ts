import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { listDomains, type DomainItem } from '../api';

/**
 * 域名列表缓存：让「创建链接」「创建落地页」等表单可以下拉选域名，
 * 不再要求用户手输数字 ID。
 */
export const useDomainStore = defineStore('domain', () => {
  const items = ref<DomainItem[]>([]);
  const loaded = ref(false);
  const loading = ref(false);

  const entryDomains = computed(() => items.value.filter((d) => d.Type === 'entry'));
  const landingDomains = computed(() => items.value.filter((d) => d.Type === 'landing'));
  const transitDomains = computed(() => items.value.filter((d) => d.Type === 'transit'));

  async function refresh(force = false): Promise<DomainItem[]> {
    if (loaded.value && !force) return items.value;
    loading.value = true;
    try {
      items.value = (await listDomains()).items;
      loaded.value = true;
      return items.value;
    } finally {
      loading.value = false;
    }
  }

  function invalidate(): void {
    loaded.value = false;
  }

  return { items, loaded, loading, entryDomains, landingDomains, transitDomains, refresh, invalidate };
});
