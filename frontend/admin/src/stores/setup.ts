import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { loadSetupStatus, type SetupStatus } from '../setup';

/**
 * 初始化向导状态：决定应用进入「初始化模式」还是「正常运行模式」。
 */
export const useSetupStore = defineStore('setup', () => {
  const status = ref<SetupStatus | null>(null);
  const loaded = ref(false);

  const setupRequired = computed(() => Boolean(status.value?.setup_required));

  async function refresh(): Promise<SetupStatus> {
    status.value = await loadSetupStatus();
    loaded.value = true;
    return status.value;
  }

  function invalidate(): void {
    loaded.value = false;
    status.value = null;
  }

  return { status, loaded, setupRequired, refresh, invalidate };
});
