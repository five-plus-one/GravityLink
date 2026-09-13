# 分支与自动发布

## 分支
- `dev`：日常开发集成；推送后执行检查并发布开发容器。
- `preview`：预发布验收；由 dev 通过 PR 晋级，发布测试容器和 GitHub prerelease。
- `stable`：默认分支与正式版本；由 preview 通过 PR 晋级，版本号来自 VERSION。
- `main`：保留已有历史，后续通过 stable 发布。

## 镜像与版本
镜像仓库为 `5plus1/gravitylink`。正式发布生成 `latest`、`stable`、完整版本标签；开发/预览发布生成分支标签及包含版本、运行编号的唯一标签。测试容器不更新 latest。
正式版本为 `v<VERSION>`，本次 `v0.2.0`。同一版本只能对应同一提交；改变代码必须更新 VERSION。发布成功后自动生成 GitHub Release 并附带镜像摘要。镜像支持 linux/amd64 与 linux/arm64。

## 流程
1. 功能分支 PR 合入 dev；CI 包含 Go 测试和构建、前端测试和构建。
2. 将 dev 的成果通过 PR 合入 preview；自动发布预发布容器。
3. 更新 VERSION，通过 PR 合入 stable；检查通过后发布正式镜像并创建 Release。
4. 首次初始化从当前验收版本创建 dev/preview/stable，运行各通道发布并验证镜像标签。

## 权限
Docker Hub 令牌存于 GitHub Actions Secret `DOCKERHUB_TOKEN`，用户名为 Secret `DOCKERHUB_USERNAME`。使用当前已登录的 Docker Hub PAT 配置。工作流仅在受信任分支的 push 上发布，PR 只运行检查，不使用发布密钥。正式分支要求 PR、检查通过、讨论解决，禁止删除和强推。个人仓库不强制另一位审核者，以免无法合并自己的 PR。

## 重试与回滚
发布任务按分支串行运行，正式版本防止重用。发布任务失败可在 Actions 重新运行；容器发布成功后才创建 Release。部署回滚使用明确的旧版本镜像并恢复匹配的数据备份；数据库升级程序随镜像保留，启动时自动执行兼容迁移。
