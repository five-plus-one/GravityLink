<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NSwitch,
  NTabPane,
  NTabs,
  useMessage,
} from 'naive-ui';
import { request } from '../api';

type ChannelView = {
  type: string;
  enabled: boolean;
  settings: Record<string, any>;
  secret_configured: boolean;
};

type EventCfg = { enabled: boolean; title: string; body: string };
type EventsCfg = {
  expiring_lead_hours: number;
  scan_near_ratio: number;
  qr_default_expire_days: number;
  events: Record<string, EventCfg>;
};

const message = useMessage();
const router = useRouter();
const loading = ref(true);
const channelsSaving = ref(false);
const eventsSaving = ref(false);
const testing = ref<string | null>(null);
const channels = ref<ChannelView[]>([]);

const smtpPassword = ref('');
const eventsCfg = ref<EventsCfg | null>(null);
const editingEvent = ref<string>('target_expiring_soon');

const eventLabels: Record<string, string> = {
  target_expiring_soon: '即将过期',
  target_expired: '已过期（未满员）',
  target_scan_near: '扫码接近阈值',
  target_scan_exhausted: '已达阈值',
  liveqr_no_available: '整码无可用',
};

const smtp = computed(() => channels.value.find((c) => c.type === 'smtp'));
const wecom = computed(() => channels.value.find((c) => c.type === 'wecom'));
const httpCh = computed(() => channels.value.find((c) => c.type === 'http'));
const anyEnabled = computed(() => channels.value.some((c) => c.enabled));

function ensureSMTPDefaults(ch: ChannelView) {
  if (!ch.settings.host) ch.settings.host = 'smtp.qq.com';
  if (!ch.settings.port) ch.settings.port = 465;
  if (!ch.settings.encryption) ch.settings.encryption = 'ssl';
  if (!ch.settings.from_name) ch.settings.from_name = 'GravityLink';
  if (!Array.isArray(ch.settings.to)) ch.settings.to = [];
}

async function load() {
  loading.value = true;
  try {
    const [chData, evData] = await Promise.all([
      request<{ items: ChannelView[] }>('/api/admin/notify/channels'),
      request<EventsCfg>('/api/admin/notify/events'),
    ]);
    channels.value = chData.items;
    channels.value.forEach(ensureSMTPDefaults);
    eventsCfg.value = evData;
  } catch (e) {
    message.error(e instanceof Error ? e.message : '加载通知配置失败');
  } finally {
    loading.value = false;
  }
}

onMounted(load);

function toRecipients(raw: any): string {
  if (Array.isArray(raw)) return raw.join(', ');
  return String(raw || '');
}

function setRecipients(ch: ChannelView | undefined, text: string) {
  if (!ch) return;
  ch.settings.to = text
    .split(/[,;\s]+/)
    .map((s) => s.trim())
    .filter(Boolean);
}

async function saveChannels() {
  channelsSaving.value = true;
  try {
    const payload = {
      channels: channels.value.map((c) => {
        if (c.type === 'smtp') {
          const settings: Record<string, any> = {
            host: c.settings.host,
            port: Number(c.settings.port) || 465,
            encryption: c.settings.encryption,
            from: c.settings.from,
            from_name: c.settings.from_name || 'GravityLink',
            username: c.settings.username,
            to: Array.isArray(c.settings.to) ? c.settings.to : [],
          };
          if (smtpPassword.value.trim()) {
            settings.password = smtpPassword.value.trim();
          }
          return { type: c.type, enabled: c.enabled, settings };
        }
        if (c.type === 'wecom') {
          return { type: c.type, enabled: c.enabled, settings: { webhook_url: c.settings.webhook_url } };
        }
        return { type: c.type, enabled: c.enabled, settings: { url: c.settings.url } };
      }),
    };
    const res = await request<{ items: ChannelView[] }>('/api/admin/notify/channels', {
      method: 'PUT',
      body: JSON.stringify(payload),
    });
    channels.value = res.items;
    channels.value.forEach(ensureSMTPDefaults);
    smtpPassword.value = '';
    message.success('通知渠道已保存');
  } catch (e) {
    message.error(e instanceof Error ? e.message : '保存失败');
  } finally {
    channelsSaving.value = false;
  }
}

async function testChannel(type: string) {
  testing.value = type;
  try {
    const res = await request<{ message: string }>(`/api/admin/notify/channels/${type}/test`, { method: 'POST' });
    message.success(res.message || '测试通知已发送');
  } catch (e) {
    message.error(e instanceof Error ? e.message : '测试失败');
  } finally {
    testing.value = null;
  }
}

async function saveEvents() {
  if (!eventsCfg.value) return;
  eventsSaving.value = true;
  try {
    eventsCfg.value = await request<EventsCfg>('/api/admin/notify/events', {
      method: 'PUT',
      body: JSON.stringify(eventsCfg.value),
    });
    message.success('事件与模板已保存');
  } catch (e) {
    message.error(e instanceof Error ? e.message : '保存失败');
  } finally {
    eventsSaving.value = false;
  }
}

function useQQPreset() {
  if (!smtp.value) return;
  ensureSMTPDefaults(smtp.value);
  smtp.value.settings.host = 'smtp.qq.com';
  smtp.value.settings.port = 465;
  smtp.value.settings.encryption = 'ssl';
  if (!smtp.value.settings.from_name) smtp.value.settings.from_name = 'GravityLink';
  message.info('已填入 QQ 邮箱 SMTP 默认项，请填写授权码与收件人');
}
</script>

<template>
  <section v-if="loading" class="settings-section">加载中…</section>
  <section v-else class="settings-section">
    <div class="section-head">
      <div>
        <strong>通知渠道</strong>
        <p class="muted">
          活码即将过期、已过期、满额、暂无可用，以及域名被封等事件会自动推送。全部关闭则不通知。
        </p>
      </div>
      <div class="actions">
        <NButton :loading="channelsSaving" type="primary" @click="saveChannels">保存渠道</NButton>
      </div>
    </div>

    <NAlert v-if="!anyEnabled" type="warning" style="margin-bottom: 16px">
      尚未启用任何通知渠道。建议至少开启邮件，便于及时更换群二维码。
      <NButton size="tiny" style="margin-left: 8px" @click="useQQPreset">填入 QQ 邮箱默认</NButton>
      <NButton size="tiny" style="margin-left: 8px" @click="router.push('/links')">去管理活码</NButton>
    </NAlert>

    <NTabs type="line" animated>
      <NTabPane name="smtp" tab="邮件 SMTP">
        <NForm v-if="smtp" label-placement="top">
          <div class="row">
            <NFormItem label="启用邮件通知" class="col">
              <NSwitch v-model:value="smtp.enabled" />
            </NFormItem>
            <NFormItem label="SMTP 服务器" class="col">
              <NInput v-model:value="smtp.settings.host" placeholder="smtp.qq.com" />
            </NFormItem>
            <NFormItem label="端口" class="col">
              <NSelect
                v-model:value="smtp.settings.port"
                :options="[
                  { label: '465（SSL，QQ 推荐）', value: 465 },
                  { label: '587（STARTTLS）', value: 587 },
                  { label: '25（不推荐）', value: 25 },
                ]"
              />
            </NFormItem>
            <NFormItem label="加密" class="col">
              <NSelect
                v-model:value="smtp.settings.encryption"
                :options="[
                  { label: 'SSL/TLS', value: 'ssl' },
                  { label: 'STARTTLS', value: 'starttls' },
                  { label: '无', value: 'none' },
                ]"
              />
            </NFormItem>
          </div>
          <div class="row">
            <NFormItem label="发件人名称（收件箱显示）" class="col">
              <NInput v-model:value="smtp.settings.from_name" placeholder="GravityLink" />
            </NFormItem>
            <NFormItem label="发件邮箱" class="col">
              <NInput v-model:value="smtp.settings.from" placeholder="noreply@notice.example.com" />
            </NFormItem>
            <NFormItem label="登录账号" class="col">
              <NInput v-model:value="smtp.settings.username" placeholder="通常与发件邮箱相同" />
            </NFormItem>
            <NFormItem label="SMTP 授权码" class="col">
              <NInput
                v-model:value="smtpPassword"
                type="password"
                show-password-on="click"
                :placeholder="smtp.secret_configured ? '已保存授权码（留空则不修改）' : 'QQ邮箱：设置→账户→开启SMTP后生成'"
              />
            </NFormItem>
          </div>
          <NFormItem label="收件人（逗号分隔，最多 20 个）">
            <NInput
              :value="toRecipients(smtp.settings.to)"
              placeholder="a@qq.com, b@163.com"
              @update:value="(v: string) => setRecipients(smtp, v)"
            />
          </NFormItem>
          <p class="muted">
            发件效果示例：GravityLink &lt;noreply@notice.example.com&gt;。QQ 邮箱请使用「SMTP 授权码」；服务器默认 smtp.qq.com:465。
            <NButton size="tiny" @click="useQQPreset">恢复 QQ 默认</NButton>
            <NButton size="tiny" style="margin-left: 8px" :loading="testing === 'smtp'" @click="testChannel('smtp')">
              发送测试
            </NButton>
          </p>
        </NForm>
      </NTabPane>

      <NTabPane name="wecom" tab="企业微信">
        <NForm v-if="wecom" label-placement="top">
          <NFormItem label="启用">
            <NSwitch v-model:value="wecom.enabled" />
          </NFormItem>
          <NFormItem label="机器人 Webhook">
            <NInput v-model:value="wecom.settings.webhook_url" placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx" />
          </NFormItem>
          <NButton :loading="testing === 'wecom'" @click="testChannel('wecom')">发送测试</NButton>
        </NForm>
      </NTabPane>

      <NTabPane name="http" tab="自定义 HTTP">
        <NForm v-if="httpCh" label-placement="top">
          <NFormItem label="启用">
            <NSwitch v-model:value="httpCh.enabled" />
          </NFormItem>
          <NFormItem label="通知接口">
            <NInput v-model:value="httpCh.settings.url" placeholder="POST JSON {title, content, time}；兼容 Bark/Server酱" />
          </NFormItem>
          <NButton :loading="testing === 'http'" @click="testChannel('http')">发送测试</NButton>
        </NForm>
      </NTabPane>
    </NTabs>

    <NCard v-if="eventsCfg" title="事件订阅与模板" size="small" style="margin-top: 24px">
      <div class="row">
        <NFormItem label="提前多久提醒即将过期（小时）" class="col">
          <NInputNumber v-model:value="eventsCfg.expiring_lead_hours" :min="1" :max="168" />
        </NFormItem>
        <NFormItem label="接近阈值比例（0–1）" class="col">
          <NInputNumber v-model:value="eventsCfg.scan_near_ratio" :min="0.5" :max="0.99" :step="0.05" />
        </NFormItem>
        <NFormItem label="微信群码默认有效天数" class="col">
          <NInputNumber v-model:value="eventsCfg.qr_default_expire_days" :min="1" :max="30" />
        </NFormItem>
      </div>

      <div class="event-grid">
        <div v-for="(label, key) in eventLabels" :key="key" class="event-item">
          <NCheckbox
            :checked="eventsCfg.events[key]?.enabled"
            @update:checked="
              (v: boolean) => {
                if (!eventsCfg!.events[key]) eventsCfg!.events[key] = { enabled: v, title: '', body: '' };
                else eventsCfg!.events[key].enabled = v;
              }
            "
          >
            {{ label }}
          </NCheckbox>
          <NButton text size="small" @click="editingEvent = key">编辑文案</NButton>
        </div>
      </div>

      <div v-if="eventsCfg.events[editingEvent]" class="template-editor">
        <p class="muted">
          当前事件：{{ eventLabels[editingEvent] || editingEvent }}。可用变量：
          LinkCode、LinkTitle、TargetLabel、ExpireAt、ExpireIn、ScanCount、ScanLimit、AvailableCount、TotalCount、Reason
        </p>
        <NForm label-placement="top">
          <NFormItem label="标题">
            <NInput v-model:value="eventsCfg.events[editingEvent].title" />
          </NFormItem>
          <NFormItem label="正文">
            <NInput v-model:value="eventsCfg.events[editingEvent].body" type="textarea" :rows="5" />
          </NFormItem>
        </NForm>
      </div>

      <NButton type="primary" :loading="eventsSaving" @click="saveEvents">保存事件配置</NButton>
    </NCard>
  </section>
</template>

<style scoped>
.section-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.muted {
  color: #94a3b8;
  font-size: 13px;
  margin: 4px 0 0;
}
.row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}
.event-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 8px;
  margin: 12px 0;
}
.event-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid #e6ecf5;
  border-radius: 10px;
  background: #f8fafc;
}
.template-editor {
  background: #f5f8fd;
  border-radius: 12px;
  padding: 16px;
  margin: 12px 0 16px;
}
.check-row {
  display: flex;
  align-items: center;
  gap: 10px;
}
.actions {
  display: flex;
  gap: 8px;
}
</style>
