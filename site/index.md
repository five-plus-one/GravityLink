---
layout: home

hero:
  name: GravityLink
  text: 把链接拉进你的引力场
  tagline: 自托管短链接 · 渠道码 · 群活码 · 落地页 · 多维统计。Go + Vue 3，Docker 一键起，数据留在你手里。
  actions:
    - theme: brand
      text: 开始部署
      link: /guide/quick-start
    - theme: alt
      text: 看功能
      link: /features/
---

<script setup>
import GravityHero from './.vitepress/theme/components/GravityHero.vue'
</script>

<GravityHero />

<div class="gl-home">

## 三条轨道

从任意方向进入，都能在半小时内跑通。

<div class="gl-orbit-nav">
  <a class="gl-orbit-card" href="/guide/quick-start">
    <span class="label">01 · Deploy</span>
    <h3>服务器部署</h3>
    <p>Docker Compose + 外部 MySQL/Redis，端口、防火墙、健康检查一步到位。</p>
  </a>
  <a class="gl-orbit-card" href="/guide/setup">
    <span class="label">02 · Configure</span>
    <h3>初始化与配置</h3>
    <p>向导式连接数据服务、创建管理员、填写公开访问地址，配置自动落盘。</p>
  </a>
  <a class="gl-orbit-card" href="/features/">
    <span class="label">03 · Use</span>
    <h3>业务能力</h3>
    <p>短链、渠道码、群活码、落地页与统计，同一入口域名统一管理。</p>
  </a>
</div>

## 四个质量体

产品围绕「一条访问如何被路由」展开，功能不是并列清单，而是同一引力场里的节点。

<div class="gl-constellation">
  <div class="gl-star">
    <div class="code">/{code} → 302</div>
    <h3>短链接</h3>
    <p>自定义或 Base62 短码，支持过期、禁用、落地页与中转域名。</p>
  </div>
  <div class="gl-star">
    <div class="code">channel · UTM</div>
    <h3>渠道码</h3>
    <p>同一目标不同渠道参数，投放效果按来源拆开看。</p>
  </div>
  <div class="gl-star">
    <div class="code">rotate · threshold</div>
    <h3>群活码</h3>
    <p>二维码轮换与阈值控制，满员自动切换，降低封控损失。</p>
  </div>
  <div class="gl-star">
    <div class="code">geo · ua · daily</div>
    <h3>统计分析</h3>
    <p>地域、设备、时段多维聚合，内置 ip2region，开箱可用。</p>
  </div>
</div>

## 镜像与边界

- 预构建镜像：`5plus1/gravitylink:latest`
- 端口拆分：短链服务公网可达，管理后台建议仅内网或 HTTPS 反代
- 数据边界：配置与上传落在 Docker 卷，业务表在你的 MySQL

</div>
