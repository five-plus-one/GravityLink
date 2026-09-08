import { createApp } from 'vue';
import { createPinia } from 'pinia';
import App from './App.vue';
import router from './router';
import { installStartupRecovery, isAssetLoadError, showStartupRecovery } from './startupRecovery';
import './styles/base.css';

installStartupRecovery();
router.onError((error) => {
  if (isAssetLoadError(error)) showStartupRecovery();
});

const app = createApp(App);
app.use(createPinia());
app.use(router);
app.mount('#app');
