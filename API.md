# MiaoSpeed 后端控制文档

本文档描述如何作为控制端（Bot / 主控服务）连接并驱动 miaospeed 后端，包括通信通道、请求结构、签名算法、矩阵清单与响应格式。

部署相关见 [DEPLOY.md](DEPLOY.md)。

## 1. 通信模型

- 通道：**WebSocket**。未设置 `-path` 时接受**任意路径**的 Upgrade 请求，即 `ws://<host>:9966/`（启用 `-mtls` 时为 `wss://`）。设置 `-path koipy` 后只接受 `/koipy`。
- 消息：全部为**一条 JSON 文本帧**。控制端每下发一个任务发送一个 `SlaveRequest`；后端回推若干条 `SlaveResponse`。
- 同一连接可连续下发多个任务；**连接断开时，该连接上所有未完成任务会被取消**。
- 普通 HTTP 请求访问端口会返回 `400 Bad Request`，属正常现象。

## 2. 鉴权与签名

后端启动时通过 `-token` 设置「启动TOKEN」；构建时固化「编译TOKEN」（默认 `MIAOKO4|580JxAo049R|GEnERAl|1X571R930|T0kEN`，Docker 构建可用 secret `BUILDTOKEN` 覆盖）。

### 2.1 签名算法（aws-v4 风格链式 SHA512）

1. 将请求结构体克隆一份，置 `Challenge = ""`；
2. 序列化为**紧凑 JSON**（字段名 = Go 结构体字段名，字段顺序 = 结构体声明顺序，无空格）；
3. 计算：

```
tokens = [启动TOKEN] + 编译TOKEN.split("|")        // 空段按 "SOME_TOKEN" 处理
h = sha512()
h.update(requestJSON)
for t in tokens:
    h.update(t + h.digest())                      // 注意：先写 token，再写当前摘要
Challenge = base64url(h.digest())                 // 保留 padding
```

> **陷阱 1（Vendor 字段）**：后端签名前会克隆请求，而上游的 `Clone()` 不复制 `Vendor` 字段——**参与签名的 JSON 中 `Vendor` 永远是空串 `""`**（实际发送的请求里 `Vendor` 照常填 `"Clash"`）。两端行为一致，但控制端自行实现签名时必须照此处理，否则验签失败。
>
> **陷阱 2（HTML 转义）**：Go 的 JSON 序列化会把字符串中的 `<` `>` `&` 转义为 `\u003c` 等，其余字符按 UTF-8 原样输出。非 Go 客户端若签名不一致，优先检查这里；让节点配置（Payload）中避免这三个字符可绕开。

### 2.2 签名字段顺序（序列化模板）

```jsonc
{
  "Basics":   {"ID":"","Slave":"","SlaveName":"","Invoker":"","Version":""},
  "Options":  {"Filter":"","Matrices":[{"Type":"","Params":""}]},
  "Configs":  {"STUNURL":"","DownloadURL":"","DownloadDuration":0,"DownloadThreading":0,
               "PingAverageOver":0,"PingAddress":"","TaskRetry":0,"DNSServers":[],
               "TaskTimeout":0,"Scripts":[{"ID":"","Type":"","Content":"","TimeoutMillis":0}]},
  "Vendor":   "",        // 注意：仅签名用模板；上游 Clone() 丢弃该字段，签名时恒为 ""
  "Nodes":    [{"Name":"","Payload":""}],
  "RandomSequence": "",
  "Challenge": ""
}
```

空值处理：字符串为 `""`，数值为 `0`，空数组建议显式给 `[]`（Go 中 nil 切片会输出 `null`，与 `[]` 签名结果不同——**以控制端实际发出的字节为准重新计算签名即可**，两端保持一致）。

### 2.3 Python 参考实现

```python
import json, hashlib, base64

def sign(req: dict, token: str, buildtoken: str = "MIAOKO4|580JxAo049R|GEnERAl|1X571R930|T0kEN") -> str:
    req = dict(req)
    req["Challenge"] = ""
    req["Vendor"] = ""  # 上游 Clone() 不复制 Vendor，签名时恒为空串（发送的请求里照填 "Clash"）
    # 字段插入顺序必须与 2.2 模板一致
    payload = json.dumps(req, separators=(",", ":"), ensure_ascii=False)
    tokens = [token] + [t or "SOME_TOKEN" for t in buildtoken.strip().split("|")]
    h = hashlib.sha512()
    h.update(payload.encode("utf-8"))
    for t in tokens:
        h.update(t.encode("utf-8") + h.digest())
    return base64.urlsafe_b64encode(h.digest()).decode("ascii")
```

### 2.4 白名单

若后端以 `-whitelist 1111,2222` 启动，则 `Basics.Invoker` 必须命中白名单，否则返回 `the bot id is not in the whitelist` 并关闭连接。未设置白名单时不校验。

## 3. 请求结构 `SlaveRequest`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `Basics.ID` | string | 任务 ID，响应中原样回显 |
| `Basics.Slave` / `SlaveName` | string | 后端标识/名称（仅记录用） |
| `Basics.Invoker` | string | 调用方（bot）ID，白名单校验对象 |
| `Basics.Version` | string | 调用方版本（仅记录用） |
| `Options.Filter` | string | 保留字段，当前未生效 |
| `Options.Matrices` | array | 要测试的矩阵列表，见 §4 |
| `Configs` | object | 任务配置，见 §3.2 |
| `Vendor` | string | `Clash`（经代理核心测试）或 `Local`（本机直连） |
| `Nodes` | array | 节点列表：`Name` + `Payload`（Clash 格式的单节点 YAML） |
| `RandomSequence` | string | 随机串，参与签名防重放 |
| `Challenge` | string | 签名结果，见 §2 |

`Nodes[].Payload` 示例（YAML 文本，无需 `name` 字段，后端会以 `Name` 补齐）：

```yaml
type: ss
server: 1.2.3.4
port: 8388
cipher: aes-128-gcm
password: pass
```

支持的 `type` 与内核（mihomo，nw 定制版）一致：`ss`、`ssr`、`vmess`、`vless`、`trojan`、`hysteria`、`hysteria2`、`tuic`、`snell`、`socks5`、`http`、`wireguard`、`ssh`、`mieru`、`anytls`、`sudoku`、`masque`、`trusttunnel`、`shadowquic`、`openvpn`、`tailscale`、`zerotier`、`easytier`、`gost-relay`、`nowhere`。

### 3.2 `Configs` 任务配置

| 字段 | 类型 | 默认 | 范围 | 说明 |
| --- | --- | --- | --- | --- |
| `STUNURL` | string | `udp://stun.voipstunt.com:3478` | - | UDP 矩阵的 STUN 服务器 |
| `DownloadURL` | string | `DYNAMIC:INTL` | - | 测速文件。`DYNAMIC:INTL`=按出口自动选微软/Google 文件；`DYNAMIC:FAST`=fast.com 动态节点；或直接填 http(s) URL |
| `DownloadDuration` | int | `3` | 1–30 秒 | 测速时长 |
| `DownloadThreading` | uint | `1` | 1–32 | 测速线程数 |
| `PingAverageOver` | uint16 | `1` | 1–16 | Ping 多次取均值 |
| `PingAddress` | string | `http://gstatic.com/generate_204` | - | Ping 目标 URL |
| `TaskRetry` | uint | `3` | 1–10 | 失败重试次数 |
| `DNSServers` | []string | `[]` | - | 自定义 DNS。支持 `ip:53`、`https://` DoH，以及 `mihomo://` 加上 base64 的 mihomo DNS 配置。解析失败时回退系统解析 |
| `TaskTimeout` | uint | `5000` | 10–10000 毫秒 | 单任务超时 |
| `Scripts` | []Script | `[]` | - | 自定义脚本，见 §4.4 |
| `ApiVersion` | int | `0` | 0–3 | 签名结构版本。`0`/`1` 与旧客户端相同，不包含本字段和上行字段；`2` 带上 `ApiVersion`；`3` 再带上上行字段 |
| `UploadURL` | string | `https://speed.cloudflare.com/__up` | - | 上行测速地址。`DYNAMIC:INTL` 使用默认地址。仅 `ApiVersion=3` 且服务端打开 `-upload` 时实际测速 |
| `UploadDuration` | int | `3` | 1–30 秒 | 上行测速时长 |
| `UploadThreading` | uint | `1` | 1–32 | 上行测速线程数 |

越界取值会被后端重置为默认值。

## 4. 矩阵（`Options.Matrices`）

请求中每个矩阵为 `{"Type": "...", "Params": "..."}`；`Params` 一般留空，仅 `TEST_SCRIPT` 用于指定脚本 ID。

矩阵→宏→返回 Payload（`MatrixResponse.Payload` 为 JSON 字符串）对照表：

| Matrix Type | 触发宏 | Payload 结构 | 说明 |
| --- | --- | --- | --- |
| `TEST_PING_CONN` | PING | `{"Value": 233}` | HTTP 建连耗时，毫秒 |
| `TEST_PING_RTT` | PING | `{"Value": 233}` | TLS/RTT 耗时，毫秒 |
| `TEST_PING_MAX_RTT` | PING | `{"Value": 300}` | 多次 Ping 中的最大 RTT，毫秒 |
| `TEST_PING_TOTAL_CONN` | PING | `{"Values":[233,240]}` | 每次 HTTP 建连耗时 |
| `TEST_PING_TOTAL_RTT` | PING | `{"Values":[180,190]}` | 每次 RTT |
| `TEST_PING_SD_CONN` | PING | `{"Value": 12.5}` | HTTP 建连耗时标准差 |
| `TEST_PING_SD_RTT` | PING | `{"Value": 8.2}` | RTT 标准差 |
| `TEST_HTTP_CODE` | PING | `{"Values":[204,204]}` | 每次 Ping 的 HTTP 状态码 |
| `TEST_PING_PACKET_LOSS` | PING | `{"Value": 0}` | 丢包率，百分比 |
| `UDP_TYPE` | UDP | `{"Value": "FullCone"}` | NAT 类型：`FullCone` / `RestrictedCone` / `PortRestrictedCone` / `Symmetric` / `SymmetricFirewall` / `Unknown` |
| `SPEED_AVERAGE` | SPEED | `{"Value": 12345678}` | 平均下载速度，**字节/秒** |
| `SPEED_MAX` | SPEED | `{"Value": 23456789}` | 下载峰值速度（单秒最大值），字节/秒 |
| `SPEED_PER_SECOND` | SPEED | `{"Max":..,"Average":..,"Speeds":[..]}` | 每秒下载速度序列，字节/秒 |
| `USPEED_AVERAGE` | USPEED | `{"Value": 12345678}` | 平均上传速度，字节/秒 |
| `USPEED_MAX` | USPEED | `{"Value": 23456789}` | 上传峰值速度，字节/秒 |
| `USPEED_PER_SECOND` | USPEED | `{"Max":..,"Average":..,"Speeds":[..]}` | 每秒上传速度序列，字节/秒 |
| `GEOIP_INBOUND` | GEO | MultiStacks（见 §4.2） | 入口 GeoIP |
| `GEOIP_OUTBOUND` | GEO | MultiStacks（见 §4.2） | 出口 GeoIP |
| `TEST_HIJACK_DETECTION` | HIJACK | `{"SpeedIP":"1.1.1.1","RealIP":"1.1.1.1"}` | 用不同 SNI 访问 Google 拦截页，比较看到的 IP |
| `DEBUG_SLEEP` | SLEEP | `{"Value":"slept for 10s seconds"}` | 调试用，固定睡眠 10 秒。与下载测速一样受 `-nospeed` 限制 |
| `TEST_SCRIPT` | SCRIPT | `{"Key":"<脚本ID>","Text":"...","Color":"...","Background":"...","TimeElapsed":123}` | 自定义脚本结果 |

同一宏只执行一次：例如同时请求 `TEST_PING_CONN` 与 `TEST_PING_RTT`，只跑一轮 Ping，两个矩阵各自提取结果。上传矩阵同理，只跑一轮 `USPEED`。

脚本可以声明 `MS_MATRIX_ENTRY = {name: "TEST_PING_MAX_RTT", params: ""}`。此时 `TEST_SCRIPT` 会改跑该矩阵，并用脚本里的 `matrice_formatter(matrix)` 或 `matrix_extract(matrix)` 把结果格式化成脚本文本。这样旧客户端仍用 `TEST_SCRIPT` 就能取到它不认识的矩阵。

### 4.2 GeoIP 的 MultiStacks 结构

```jsonc
{
  "Domain": "",
  "MainStack": null,                 // 已废弃
  "IPv4Stack": [GeoInfo, ...],
  "IPv6Stack": [GeoInfo, ...]
}
```

`GeoInfo`（注意此处为小写 JSON 键）：

```jsonc
{
  "ip": "1.2.3.4", "country": "...", "country_code": "...", "continent_code": "...",
  "organization": "...", "isp": "...", "asn": 13335, "asn_organization": "...",
  "latitude": 0, "longitude": 0, "timezone": "...", "stackType": ""
}
```

GeoIP 查询优先走 `-mmdb` 本地库；未配置时使用内置脚本查询 ip-api.com（出口为代理出口 IP）。请求 `Configs.Scripts` 中 `Type: "ip"` 的脚本可覆盖内置的 IP/GeoIP 解析逻辑（入口函数分别为 `ip_resolve()` 与 `handler(ip)`）。

### 4.4 自定义脚本（`TEST_SCRIPT`）

`Configs.Scripts` 中 `Type: "media"` 的脚本会并发执行（goja ES5.1 引擎），脚本需定义 `handler()`，返回字符串或 `{text, color, background}`。`TEST_SCRIPT` 矩阵的 `Params` 填对应 `Script.ID`，多脚本时每个矩阵条目取一个。

脚本沙箱内置函数：

- `fetch(url, params)` — HTTP 请求（经被测节点）。`params`：`method`、`body`、`headers`、`cookies`、`timeout`(ms)、`retry`、`noRedir`、`useHost`(true=不走代理)、`sni`（覆盖 TLS Server Name）。返回 `{status, statusCode, headers, cookies, body, redirects, url, method}` 或 `null`。跟随重定向时，后续连接按重定向后的地址经代理拨号。
- `netcat(addr, data, params)` — 原始 TCP/UDP 收发。
- `proxy` — 当前节点信息 `{Name, Address, Type}`。
- 预定义工具：`get(obj, "a.b", default)`、`safeParse` / `safeStringify`、`println`。

## 5. 响应 `SlaveResponse`

后端按事件推送多条消息：

| 场景 | 载荷 |
| --- | --- |
| 每个节点测完 | `{"ID":"<任务ID>","MiaoSpeedVersion":"5.0.0","Progress":{"Index":0,"Record":{...},"Queuing":2}}` |
| 全部完成 | `{"ID":"...","Result":{"Request":{...},"Results":[...]}}` |
| 鉴权/白名单/禁用失败 | `{"ID":"","Error":"cannot verify the request, please check your token"}`，随后**服务端关闭连接** |

`Progress.Record` / `Results[]` 元素结构（`SlaveEntrySlot`）：

```jsonc
{
  "Grouping": "",
  "ProxyInfo": {"Name":"节点名","Address":"server:port","Type":"Shadowsocks"},
  "InvokeDuration": 1234,            // 该节点耗时，毫秒
  "Matrices": [
    {"Type":"TEST_PING_CONN","Payload":"{\"Value\":233}"},
    {"Type":"SPEED_AVERAGE","Payload":"{\"Value\":1048576}"}
  ]
}
```

注意 `Payload` 是**字符串里套 JSON**，需二次解析。

## 6. 服务端队列行为

- 含 SPEED 宏的任务进 **SpeedPoll**（并发 1，串行执行；`-pausesecond` 可在任务间插入冷却）；其余任务进 **ConnPoll**（并发 = `-connthread`，默认 64）。
- 后端以 `-nospeed` 启动时，任何包含测速矩阵的任务直接返回 `speedtest is disabled on backend`。
- `Progress.Queuing` 可用于前端展示排队进度。

## 7. 最小完整示例

请求（签名前）：

```jsonc
{
  "Basics": {"ID":"task-001","Slave":"","SlaveName":"","Invoker":"my-bot","Version":"1.0"},
  "Options": {"Filter":"","Matrices":[{"Type":"TEST_PING_CONN","Params":""},{"Type":"SPEED_AVERAGE","Params":""}]},
  "Configs": {"STUNURL":"","DownloadURL":"","DownloadDuration":10,"DownloadThreading":4,
              "PingAverageOver":1,"PingAddress":"","TaskRetry":3,"DNSServers":[],"TaskTimeout":5000,"Scripts":[]},
  "Vendor": "Clash",
  "Nodes": [{"Name":"节点A","Payload":"type: ss\nserver: 1.2.3.4\nport: 8388\ncipher: aes-128-gcm\npassword: pass\n"}],
  "RandomSequence": "f3a9c1...",
  "Challenge": "<用 §2 算法计算>"
}
```

成功响应序列：

```
<- {"ID":"task-001","MiaoSpeedVersion":"5.0.0","Progress":{"Index":0,"Record":{...},"Queuing":0}}
<- {"ID":"task-001","MiaoSpeedVersion":"5.0.0","Result":{"Request":{...},"Results":[{...}]}}
```
