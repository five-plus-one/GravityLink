import { createApp } from 'vue';
import { createPinia } from 'pinia';
import App from './App.vue';
import router from './router';
import { echarts } from './echarts';
import './styles/base.css';

const app = createApp(App);
app.use(createPinia());
app.use(router);
app.provide('echarts', echarts);
app.mount('#app');
