<div align="center">

<img src="resources/icon.png" width="112" alt="">

# Clash Traffic

> EasyTier 修改版：[Linux 安装、同步配置与构建说明](docs/LINUX.md)。功能版关闭上游自动更新，避免丢失 EasyTier 功能。

看清 Clash Verge 的流量去了哪里 —— 按应用、地址、节点、订阅和设备统计。

[English](README.md) · **简体中文**

<picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/overview-dark.png"><img alt="概览：三台电脑近 7 天的流量" src="docs/screenshots/overview-light.png"></picture>

</div>

## 功能

- **每个应用、每个地址** —— 上传、下载和连接数；辅助进程归到所属应用，子域名可合并到主域名。
- **代理 / 直连** —— 一键只看机场计费的代理流量。
- **订阅** —— 每个订阅走了多少流量，旁边是机场返回的已用量和到期时间。
- **多台电脑，无需配置** —— 同一 Tailscale 网络里运行本应用的电脑自动互相发现、合并统计；也可用共享文件夹或 WebDAV。
- **EasyTier 自动发现** —— 自动发现直连和中继设备，使用独立同步密钥，复用现有增量流量统计同步。参见 [EasyTier 设置](#easytier-设置)。
- **访问记录与实时连接** —— 每个连接的应用、目标、节点和规则；实时速度和活动连接。
- **不打扰** —— 自动找到 Clash Verge，常驻菜单栏，开机启动，自动更新，导出 CSV，跟随浅色 / 深色模式。

数据只保存在你自己的电脑上，设备之间直接交换。界面支持简体中文、繁體中文、English、日本語、한국어 和 Русский，默认跟随系统，也可以在设置中切换。

## 截图

<table>
<tr><td width="50%" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/apps-dark.png"><img alt="应用" src="docs/screenshots/apps-light.png"></picture><br><sub><b>应用</b> — 以及其中一个应用访问的地址</sub></td><td width="50%" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/subscriptions-dark.png"><img alt="订阅" src="docs/screenshots/subscriptions-light.png"></picture><br><sub><b>订阅</b> — 机场用量与本应用的统计</sub></td></tr>
<tr><td width="50%" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/live-dark.png"><img alt="实时连接" src="docs/screenshots/live-light.png"></picture><br><sub><b>实时连接</b> — 速度与活动连接</sub></td><td width="50%" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/records-dark.png"><img alt="访问记录" src="docs/screenshots/records-light.png"></picture><br><sub><b>访问记录</b> — 每个连接，可搜索</sub></td></tr>
<tr><td width="50%" valign="top"><picture><source media="(prefers-color-scheme: dark)" srcset="docs/screenshots/devices-dark.png"><img alt="设备" src="docs/screenshots/devices-light.png"></picture><br><sub><b>设备</b> — 通过 Tailscale 自动发现</sub></td><td width="50%"></td></tr>
</table>

## 安装

从 [Releases](https://github.com/lufficc/clash-traffic/releases) 下载：macOS 通用 `.dmg`（macOS 13 及以上，支持 Apple 芯片和 Intel）、Windows 安装程序（x64 或 ARM64）、Linux `.deb` / `.tar.gz`。

安装包暂未签名。macOS 首次打开请右键选择「打开」；Windows 请选择「更多信息 → 仍要运行」。

支持 Clash Verge Rev，以及控制接口可访问的其他 mihomo 内核。为区分应用，它会把内核的 `find-process-mode` 设为 `always`；在 Clash Verge 的全局扩展覆写配置中加入 `find-process-mode: always` 可在重载后保持。

## EasyTier 设置

1. 连接 EasyTier，启用虚拟 IPv4 网卡、网络加密及本机管理 RPC（GUI 服务模式通常为 `127.0.0.1:15999`，独立内核通常为 `127.0.0.1:15888`）。准备与 EasyTier 匹配、支持 `-o json route` 的 `easytier-cli`。
2. 进入「设置 → 多设备同步」，启用 EasyTier，在自己的设备上填写同一个至少 16 字符的独立同步密钥，不要使用 EasyTier 网络密码。选择 EasyTier 会关闭 Tailscale 同步，但保留原有配置与历史统计。
3. 默认从 PATH、应用目录、运行中的 EasyTier 安装目录及常见安装位置查找 CLI。已有可用 CLI 路径可保留。将 RPC 和实例留空，自动读取 Windows 服务/进程参数、服务配置目录的实例名称和 ID，必要时检查本机监听端口。逐实例查询，兼容旧版服务的多实例情况。仅有一个可用网络时自动选择；有多个时，点击「检测 EasyTier → 使用此网络」，然后保存。手动填写的可用地址优先；失败时再尝试自动探测。
4. 在「设备 → EasyTier 自动同步」查看结果。每分钟发现并增量同步一次，也可点「立即扫描」。防火墙仅对 EasyTier 网卡及自己的设备放行 TCP **47321**（开发模式为 47322）。将 EasyTier 虚拟网络和底层传输排除在代理接管之外。

仅探测所选实例路由表里的虚拟 IPv4，包括经中继可达的设备；不扫描子网代理网段及公网中继传输地址。同步服务只绑定虚拟 IP，同时校验来源地址和同步密钥，不跟随 HTTP 重定向。同步内容为流量统计，不包含详细访问日志。CLI 断开后在下一次扫描停止共享；离线历史仍保留。

「检测 EasyTier」会显示 CLI 版本、识别的 RPC/实例以及脱敏的 CLI 具体错误，不再只显示 exit status 1。探测只读，不启动/停止服务，不修改 GUI 配置。默认端口回退顺序为 15999、15888。Windows 服务识别仅在 Windows 提供，其他平台保留 CLI 和默认端口探测。

适配依据为 EasyTier 官方 CLI 路由 JSON 的 `ipv4`、`hostname`、`next_hop_hostname`、`path_len` 字段。兼容 2.5/2.6 风格的平铺数组及新版按实例包装的结果。不支持缺少此输出格式的旧版 CLI、无 TUN 模式及纯 IPv6 网络。新增界面提供简体中文和英文，其余语言暂以英文显示新增文字。

请求与响应使用 HMAC-SHA256 签名，不向被发现的设备发送同步密钥。签名绑定目标地址，并校验 60 秒时间窗口，因此请保持各设备系统时间同步。密钥与应用现有凭据一样保存在本机设置文件中，分享备份时应移除。

构建关闭上游自动更新的本地功能版：

```powershell
$env:CLASH_TRAFFIC_EASYTIER_BUILD = "1"
bun run build
```

功能版沿用原应用标识及数据位置。运行前先退出原版，不要让两者同时使用同一数据目录。

## 构建

需要 Go 1.27+ 和 Bun：

```sh
bun install
bun run dev        # 开发，热重载
bun run build      # 打包本机平台到 build/
go test ./...      # 以及 bun run test:web
```

基于 [MyGo](https://mygo.egoist.dev/) 构建。
