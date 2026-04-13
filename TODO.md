# Weixin Echo Migration TODO

这份清单用于继续把 `packages/openclaw-weixin` 的能力迁移到 Go 版 `golang/weixin-echo`。

## 当前状态

已完成：

- 已支持基础登录流程：
  - `get_bot_qrcode`
  - `get_qrcode_status`
- 已支持长轮询收消息：
  - `getupdates`
  - `get_updates_buf` 持久化
- 已支持基础文本回复：
  - `sendmessage` 文本消息
- 已补齐部分 Go 侧接口封装：
  - `getuploadurl`
  - `getconfig`
  - `sendtyping`
  - 图片上传到微信 CDN
  - 图片消息发送
- 已支持特殊指令：
  - 用户发送文本 `二维码` 时，自动生成二维码 PNG 并通过图片消息回复
- 已改造模型输入提示：
  - 文本、语音、图片、视频、文件会进入统一的入站消息理解流程
  - 语音优先使用接口返回的转写文本
  - 图片/视频/文件在当前无法解析内容时，会显式告知模型不要臆测

## 已落地代码位置

- 消息轮询与入口分流：
  - `golang/weixin-echo/internal/app/polling.go`
- 入站消息抽取与提示拼装：
  - `golang/weixin-echo/internal/app/helpers.go`
- 二维码图片回复：
  - `golang/weixin-echo/internal/app/media_reply.go`
- 系统提示词构造：
  - `golang/weixin-echo/internal/ai/prompt_builder.go`
- iLink API 封装：
  - `golang/weixin-echo/internal/ilinkapi/api.go`
  - `golang/weixin-echo/internal/ilinkapi/types.go`
  - `golang/weixin-echo/internal/ilinkapi/media.go`

## 下一步高优先级

### 1. 迁移完整的媒体入站能力

当前问题：

- Go 版现在只是“识别到有图片/语音/文件/视频”，但还没有像 TypeScript 版那样真正下载、解密、转码这些媒体。
- 图片还没有做视觉理解接入。
- 语音还没有下载 SILK 并转 WAV。

要做的事：

- 对照 `packages/openclaw-weixin/src/media/media-download.ts` 迁移以下能力：
  - CDN 下载 URL 构造
  - AES-128-ECB 解密
  - 图片原图下载
  - 文件附件下载
  - 视频下载
  - 语音下载
  - SILK -> WAV 转码
- 决定 Go 版媒体暂存目录：
  - 建议新增 `internal/storage` 或 `internal/media` 目录管理缓存文件
- 为每种媒体补齐结构化入站对象，而不是只拼文本提示

验收标准：

- 用户发图片后，系统能拿到本地文件路径或可传给模型的内容
- 用户发语音后，系统能下载并转码，至少能拿到转写与音频文件
- 用户发文件/视频后，系统能保留基础元信息和本地缓存路径

### 2. 让大模型真正处理图片

当前问题：

- 目前提示词里只会告诉模型“用户发了图片”，但没有把图片内容送进模型
- 这只解决了“不乱猜”，还没解决“看图理解”

要做的事：

- 先确认目标模型能力：
  - `deepseek`
  - `kimi`
  - `ollama`
- 为支持视觉的 provider 增加多模态请求结构：
  - 不再只传 `system + user string`
  - 需要支持 message content array / image_url / binary upload 等不同协议
- 在 `internal/ai` 下抽象统一的多模态 message 结构
- 在 `prompt_builder` 外新增“用户消息内容构造器”，把文本 + 图片一起传给模型

建议顺序：

- 先做 Kimi 或 DeepSeek 的图片输入
- 再考虑 Ollama 的本地视觉模型兼容

验收标准：

- 用户发图片并提问“这是什么”“帮我识别一下图里的内容”时，模型能基于图片回答

### 3. 迁移 typing 能力到业务链路

当前问题：

- `getconfig` 和 `sendtyping` API 已经补了，但业务上还没用

要做的事：

- 在 AI 回复生成前发一次 `typing`
- 在发送完成或失败后发一次 `cancel typing`
- 给 typing 加超时与降级逻辑，避免 typing 失败影响主流程

可参考 TypeScript：

- `packages/openclaw-weixin/src/messaging/process-message.ts`

验收标准：

- 用户发送消息后，在模型处理期间微信侧能看到“正在输入”

### 4. 迁移更多出站媒体能力

当前问题：

- Go 版目前只支持文本发送和图片发送
- TypeScript 版还支持：
  - 文件发送
  - 视频发送
  - 更完整的 media item 组织

要做的事：

- 对照 `packages/openclaw-weixin/src/messaging/send.ts`
- 对照 `packages/openclaw-weixin/src/messaging/send-media.ts`
- 对照 `packages/openclaw-weixin/src/cdn/upload.ts`
- 增加：
  - `SendFile`
  - `SendVideo`
  - 通用 `SendMedia`
- 若后面要支持模型生成图片或远程图片 URL 回复，也需要增加“下载远程图片到临时文件再上传”的流程

验收标准：

- 服务端可以主动发送本地图片、文件、视频给微信用户

### 5. 重新设计“二维码”分享内容

当前问题：

- 当前 `二维码` 指令返回的是“机器人标识信息二维码”
- 这能满足“回一张图”，但未必等于“用户扫码即可分享机器人”
- 目前还没确认腾讯官方 iLink 是否提供“机器人分享二维码 / 名片二维码 / 加好友二维码”接口

要做的事：

- 先确认产品目标：
  - 是分享机器人身份信息
  - 还是分享一个网页落地页
  - 还是分享一个可以再次连接/登录的二维码
  - 还是分享某个公众号/服务入口
- 如果有明确的分享 URL 或协议数据，把 `buildShareQRCodeContent` 改成真实业务内容
- 如果官方后续存在“分享卡片”或“联系机器人”能力，再改成真正的可扫码使用入口

验收标准：

- 用户把机器人发出去后，扫码方能完成预期行为，而不只是看到元信息

## 中优先级迁移项

### 6. 把 TypeScript 版消息结构完整搬到 Go

当前问题：

- 虽然 Go 版 `types.go` 已经补了一部分字段，但还不是完全对齐
- 未来如果协议字段变化，Go 版可能再次落后

要做的事：

- 对照 `packages/openclaw-weixin/src/api/types.ts`
- 补齐以下结构的完整字段：
  - `WeixinMessage`
  - `MessageItem`
  - `ImageItem`
  - `VoiceItem`
  - `FileItem`
  - `VideoItem`
  - `RefMessage`
  - `GetUploadUrlResp`
  - `GetConfigResp`
  - `SendTypingReq/Resp`
- 评估是否需要统一生成协议定义，避免 TS/Go 两边手写漂移

### 7. 迁移引用消息 / ref_msg 语义

当前问题：

- TypeScript 版对引用消息做了专门处理
- Go 版现在还没把引用内容喂给模型

要做的事：

- 对照 `packages/openclaw-weixin/src/messaging/inbound.ts`
- 把被引用文本、被引用附件摘要拼进历史上下文
- 确保模型知道用户是在“回复某条历史消息”

验收标准：

- 用户引用上一条消息追问时，模型能理解上下文

### 8. 迁移群聊能力

当前问题：

- 现在逻辑仍以单聊为主
- 协议里已有 `group_id`

要做的事：

- 明确群聊里 `from_user_id`、`to_user_id`、`group_id` 的业务含义
- 设计群聊会话 key
- 设计“是否响应群聊”的策略：
  - @机器人才回复
  - 指令触发
  - 全量监听

验收标准：

- 群聊上下文与单聊上下文不串线

### 9. 会话与上下文管理增强

当前问题：

- 现在历史窗口较短，且主要是纯文本历史
- 媒体消息进入后，需要重新定义历史摘要策略

要做的事：

- 设计统一的历史消息结构：
  - 文本
  - 语音转写
  - 图片说明
  - 文件摘要
- 支持更稳健的历史裁剪和摘要
- 评估是否需要持久化更多入站消息元信息

## 低优先级与工程优化

### 10. 增加配置项

建议增加到 `config.yaml`：

- `weixin.cdn_base_url`
- `weixin.enable_typing`
- `weixin.enable_image_understanding`
- `weixin.enable_voice_transcode`
- `weixin.share_qrcode_payload`
- `weixin.allowed_message_types`

### 11. 增加测试覆盖

当前问题：

- 本次主要靠编译和现有测试兜底
- 业务逻辑测试还不够

建议新增测试：

- `extractInboundMessage` 单测：
  - 文本
  - 语音转写
  - 图片
  - 文件
  - 混合消息
- `buildSystemPrompt` 多模态上下文测试
- `SendImage` 请求体测试
- `UploadImageBuffer` 加密与请求参数测试
- `二维码` 分支的业务测试

### 12. 增加日志与可观测性

要做的事：

- 为媒体上传、下载、解密、转码加 debug 日志
- 为 typing、媒体发送、模型多模态输入加 action 级埋点
- 避免日志直接打印敏感 token / context_token / 文件明文内容

## 推荐迁移顺序

建议下次按这个顺序继续：

1. 迁移媒体下载与解密
2. 接入图片理解到模型调用链
3. 接入 typing
4. 补齐文件/视频出站
5. 处理引用消息
6. 再考虑群聊与更完整会话管理

## 下次开工前建议先看的文件

- TypeScript 参考实现：
  - `packages/openclaw-weixin/src/api/types.ts`
  - `packages/openclaw-weixin/src/api/api.ts`
  - `packages/openclaw-weixin/src/messaging/inbound.ts`
  - `packages/openclaw-weixin/src/messaging/process-message.ts`
  - `packages/openclaw-weixin/src/messaging/send.ts`
  - `packages/openclaw-weixin/src/messaging/send-media.ts`
  - `packages/openclaw-weixin/src/media/media-download.ts`
  - `packages/openclaw-weixin/src/cdn/upload.ts`
  - `packages/openclaw-weixin/src/cdn/cdn-upload.ts`
- Go 当前实现：
  - `golang/weixin-echo/internal/app/polling.go`
  - `golang/weixin-echo/internal/app/helpers.go`
  - `golang/weixin-echo/internal/app/media_reply.go`
  - `golang/weixin-echo/internal/ai/prompt_builder.go`
  - `golang/weixin-echo/internal/ilinkapi/api.go`
  - `golang/weixin-echo/internal/ilinkapi/types.go`
  - `golang/weixin-echo/internal/ilinkapi/media.go`

## 每次迁移后的最低验证清单

- `go test ./...`
- `go build ./...`
- 手工验证登录流程不回归
- 手工验证文本收发不回归
- 手工验证 `二维码` 指令仍能回图
- 若本次涉及媒体：
  - 验证图片消息
  - 验证语音消息
  - 验证文件消息
  - 验证 typing 状态
