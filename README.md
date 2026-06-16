<div align="center">

<img src="assets/logo.png" width="120" height="120" alt="usher logo" />

# usher

**扫描 NAS 或本地目录中的视频文件，一键生成 M3U 播放列表**

适用于 VLC · PotPlayer · IINA · Infuse 等主流播放器

![platform](https://img.shields.io/badge/platform-macOS-111111?style=flat-square)
![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?style=flat-square&logo=go&logoColor=white)
![Wails](https://img.shields.io/badge/Wails-v2-DF0000?style=flat-square)
![M3U](https://img.shields.io/badge/output-%23EXTM3U-5b8cff?style=flat-square)

</div>

<div align="center">
  <img src="assets/screenshot.png" width="820" alt="usher 桌面界面" />
</div>

---

## 目录

- [功能特性](#功能特性)
- [快速开始](#快速开始)
  - [桌面版（推荐）](#桌面版推荐)
  - [命令行版](#命令行版)
- [配置说明](#配置说明)
  - [缓存与增量更新](#缓存与增量更新)
  - [运行日志](#运行日志)
  - [自定义扩展名](#自定义扩展名)
- [内置视频扩展名](#内置视频扩展名)
- [输出格式](#输出格式)
- [项目结构](#项目结构)
- [开发](#开发)

---

## 功能特性

| | |
|---|---|
| ⚡ **并发扫描** | 多源目录并行，支持子目录级并发 worker |
| 🗂️ **目录级缓存** | 按目录粒度缓存，重复运行大幅加速 |
| 🎬 **40+ 扩展名** | 内置常见视频格式，也可自定义覆盖 |
| 🔀 **排序 / 洗牌** | 按路径排序，或随机打乱播放顺序 |
| 🙈 **智能跳过** | 自动忽略隐藏文件/目录并去重 |
| 📊 **变更摘要** | 输出各源目录统计与相对上次的新增/删除 |

输出标准 `#EXTM3U` 格式，开箱即用。

---

## 快速开始

### 桌面版（推荐）

图形界面，无需手写 YAML——添加目录、选择播放器、一键生成。

> [!NOTE]
> **环境要求：** Go 1.22+ · [Wails v2](https://wails.io/docs/gettingstarted/installation) · Node.js

```bash
git clone https://github.com/SunneeYang/usher.git
cd usher
wails build
open build/bin/usher.app
```

配置保存在 `~/Library/Application Support/usher/config.yaml`。

开发模式（热重载）：

```bash
wails dev
```

### 命令行版

> [!NOTE]
> **环境要求：** Go 1.22+

```bash
go build -o usher-cli ./cmd/usher
```

复制示例配置并修改路径：

```bash
cp config.yaml.example config.yaml
```

`config.yaml` 示例：

```yaml
source_dirs:
  - /path/to/movies
  - /path/to/tv

output_file: playlist.m3u
shuffle: false
sort: true
skip_hidden: true
scan_workers: 32
scan_cache: .usher-scan-cache.json
```

运行：

```bash
./usher-cli                          # 使用当前目录 config.yaml
./usher-cli -config /path/to/config.yaml
./usher-cli -version
./usher-cli -perf                    # 开启详细性能日志
./usher-cli -fresh                   # 忽略缓存，全量重新扫描
```

---

## 配置说明

| 字段 | 必填 | 默认值 | 说明 |
|------|:---:|--------|------|
| `source_dirs` | ✅ | — | 要扫描的视频目录列表 |
| `output_file` | ✅ | `playlist.m3u` | 输出的 M3U 文件路径 |
| `video_extensions` | — | 见内置列表 | 视频扩展名；省略时使用内置完整列表 |
| `shuffle` | — | `false` | 是否随机打乱播放顺序 |
| `sort` | — | `true` | 未洗牌时按路径字母序排序 |
| `skip_hidden` | — | `true` | 跳过 `.` 开头的文件和目录 |
| `scan_workers` | — | `32` | 并行扫描 worker 数（上限 64）；NAS 网络盘可适当调高 |
| `scan_cache` | — | `.usher-scan-cache.json` | 扫描缓存文件；设为空字符串可禁用 |
| `cache_verify` | — | `none` | `none`=信任缓存极速；`mtime`=逐个校验目录变更 |

> [!IMPORTANT]
> 若在配置中指定 `video_extensions`，将**完全覆盖**内置默认值，而非合并。

### 缓存与增量更新

缓存按**目录**粒度存储，不是「加一个视频就全库作废」：

| 模式 | 行为 | 适用场景 |
|------|------|----------|
| `cache_verify: none` | 完全信任缓存，二次运行极快 | 日常更新播放列表 |
| `cache_verify: mtime` | 每个目录 `stat` 校验 mtime，变更目录自动重扫 | 希望自动感知少量新增 |
| `./usher -fresh` | 忽略缓存，全量重扫 | 大量文件变动后 |

新增一个视频时，通常只有**该视频所在目录**（及可能的上级目录）缓存失效，其余 3000+ 目录仍命中缓存。

> [!TIP]
> `mtime` 模式在 NAS 上对每个目录 `stat` 仍偏慢（约 3381 次网络请求），因此默认推荐 `none` + 有变动时手动 `-fresh`。

### 运行日志

每次扫描会输出源目录统计与视频变更：

```text
📁 [usher] library-1: 3666 个视频 (3381 子目录)
   一级子目录: Unsorted=2100, Movies=800, TV=500, ...
📁 [usher] library-2: 4905 个视频 (4487 子目录)
   一级子目录: ...
📊 [usher] 视频变更: +12 新增, -3 删除, 8556 未变 (共 8571)
   新增: /Volumes/library-2/new/a.mp4, ...
```

首次运行显示 `首次索引 N 个视频`；无变化时显示 `无变化，共 N 个视频`。

按当前测试目录推算（175 视频 / 534 条目 ≈ 1s），全量几十倍时预估：

| 场景 | 预计耗时 | 说明 |
|------|----------|------|
| 首次全量扫描 | ~30–60s | 取决于 NAS 并发与目录结构 |
| 二次运行（缓存命中） | 通常 < 5s | 仅检查目录 mtime，跳过未变化的子树 |
| 少量新增后 | 介于两者之间 | 只重扫 mtime 变化的目录 |

**推荐流程：**

1. 首次跑全量库，耐心等待并生成 `.usher-scan-cache.json`
2. 日常更新播放列表直接 `./usher`，走缓存
3. 大量文件变动后执行 `./usher -fresh` 强制全量重扫

### 自定义扩展名

```yaml
source_dirs:
  - /media/videos
output_file: playlist.m3u
video_extensions:
  - .mp4
  - .mkv
  - .myformat
```

> [!TIP]
> 扩展名可带或不带点号（`.mp4` 与 `mp4` 均可）。

---

## 内置视频扩展名

省略 `video_extensions` 时，默认匹配以下格式：

| 类别 | 扩展名 |
|------|--------|
| 常见容器 | `.mp4` `.m4v` `.mkv` `.avi` `.mov` `.wmv` `.flv` `.f4v` `.webm` |
| MPEG 系列 | `.mpg` `.mpeg` `.mpe` `.mp2` `.m2v` `.mpv` |
| 广播/录制流 | `.ts` `.m2ts` `.mts` |
| 光盘镜像 | `.vob` `.iso` |
| 开源容器 | `.ogv` `.ogm` |
| 移动端 | `.3gp` `.3g2` `.asf` |
| 老格式 | `.rm` `.rmvb` `.divx` `.xvid` `.dat` |
| 摄像机 | `.mod` `.tod` `.dv` |
| 电视录制 | `.wtv` `.dvr-ms` `.rec` `.str` |
| 裸流 | `.264` `.265` `.h264` `.h265` `.hevc` |
| 专业 | `.mxf` |

完整列表见 [`internal/config/extensions.go`](internal/config/extensions.go)。

---

## 输出格式

生成的 M3U 文件示例：

```text
#EXTM3U
#EXTINF:-1,movie.mp4
/movies/movie.mp4
#EXTINF:-1,episode.mkv
/tv/show/episode.mkv
```

---

## 项目结构

```text
usher/
├── main.go                 # Wails 桌面入口
├── app.go                  # 桌面 API（绑定前端）
├── cmd/usher/              # CLI 入口
├── frontend/               # 桌面 UI (Vite + vanilla JS)
├── wails.json
├── internal/
│   ├── app/                # CLI / 桌面共享业务逻辑
│   ├── config/
│   ├── scanner/
│   └── playlist/
└── go.mod
```

---

## 开发

```bash
go test ./...                        # 测试
go build -o usher-cli ./cmd/usher    # CLI
wails dev                            # 桌面（开发热重载）
wails build                          # 桌面（发布构建）
```
