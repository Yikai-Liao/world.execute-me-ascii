# world.execute(me); —ascii

![world.execute(me);](docs/images/mv-cover.png)

Mili《world.execute(me);》的字符动画。支持中英字幕、原曲同步播放和终端字符动画。

## Go 跨平台原型

`feat/go-player` 分支新增了原生 Go 播放器，目标为 Windows、macOS、Linux
的 amd64 和 arm64。运行编译后的程序无需 Python、Swift 或 FFmpeg。

**这是播放器原型，尚未完整迁移原版约 3,000 行逐句动画。** 当前包含启动、
立方体、圆、正弦波、无限符号、心形、断连、执行队列等原生场景；部分段落
使用简化图案。现有双语字幕、频谱数据和五个章节时间点直接内嵌，原版视觉
编排仍以 `scenes.py` 为参照。

需要 Go 1.25 或更新版本构建：

```sh
go build -o .build/world-execute ./cmd/world-execute

# 无声预览，无需音频文件或音频设备
.build/world-execute --demo --autoplay

# 使用自己的 MP3，默认跟随音频时间线
.build/world-execute --audio media/song.mp3

# 固定 120 FPS，或使用默认 auto 模式
.build/world-execute --audio media/song.mp3 --fps 120

# 输出指定时间的纯文本画面，无需交互式终端
.build/world-execute --snapshot 38 --width 80 --height 24
```

Windows 可将输出命名为 `world-execute.exe`，推荐在 Windows Terminal 中运行。
Linux 使用 PulseAudio（包括 PipeWire 的 PulseAudio 兼容服务），或回退到系统
的 ALSA `libasound.so.2`；构建三大桌面目标无需 Cgo。

终端窗口可随时改变大小。几何场景等比缩放，字幕根据宽度换行；很小的窗口
切换为精简播放状态。字符单元比例从窗口像素信息读取，无法读取时使用宽高比
0.5，可通过 `--cell-aspect 0.5` 手动设置。字幕空间有限时仅显示前一至两行，
不能保证极小窗口完整显示所有字幕。

`--fps auto` 以 60 FPS 为上限，根据连续帧的计算和输出耗时在 15–60 FPS
之间调节。`--fps N` 可固定为 5–240 FPS。这个值是绘制目标，不是显示器
实际刷新率；终端没有通用的刷新率查询或垂直同步接口。过期帧会跳过，画面
直接跟随当前音频时间，不积压补帧。原版故障效果的 24 Hz 节奏保留。

音频启动时预解码为 PCM，默认歌曲约占 37 MB，解码上限为 256 MiB。画布逐帧
复用。Oto 提供缓冲字节数，未提供硬件播放时间戳；当前播放位置由 PCM 读取
位置减去播放器缓冲及估计输出延迟得到。因此不同设备仍可能有同步偏差，
可用 `--audio-latency 80ms` 调整请求缓冲和延迟估计，用 `--offset` 或 `[ / ]`
调整字幕。暂停、跳转仍可能听到系统队列中极短的旧音频尾部。

操作沿用空格／Enter、左右、R、Q、H、1–5，并支持 `[ / ]` 字幕偏移、
`, / .` 上一句／下一句、`+ / -` 音量。`--volume 0` 可用于静音验证。

```sh
# 构建六个目标；输出至 dist/go，附带 SHA256SUMS.txt
python3 tools/build_go.py

# 将本地音乐内嵌到所有可执行文件中；音乐不会提交到 Git
python3 tools/build_go.py --audio media/song.mp3

# 只构建一个目标
python3 tools/build_go.py --target linux/amd64 --audio media/song.mp3

# 检查布局、字幕边界、操作与自动调帧
go test -race ./...
go vet ./...

# POSIX PTY 中验证 60/120 FPS、窗口变化和操作；音频验证静音进行
go build -o .build/world-execute ./cmd/world-execute
python3 tools/smoke_go.py --audio media/song.mp3

# 在真实终端中采集绘制耗时、分配量和 GC 次数
.build/world-execute --demo --autoplay --start 147.66 \
  --stop-after 157.66 --report .build/frame-report.json
```

直接依赖锁定为 tcell/v2 v2.13.10、Oto/v3 v3.5.1、go-mp3 v0.3.4 和
uniseg v0.4.7（Unicode 字符宽度）。GitHub Actions 检查 Linux、macOS、Windows
的测试，并生成六个目标的源码构建产物；这些 CI 产物不包含音乐。

## 创作提示词

整理了这支 MV 从初始构想到逐段调整的 **25 轮提示词**，以及一份方便复用的合并版。

- **[在线阅读创作过程](docs/prompts/creation-prompts.md)**：按轮次查看需求、场景位置与修改方向，文内目录可直接跳转。
- **[直接查看合并提示词](docs/prompts/creation-prompts.md#combined-prompt)**：适合整体阅读和复制使用。
- **[纯文本版](docs/prompts/creation-prompts.txt)**：方便保存到本地或在文本编辑器中打开。

## 单文件运行（推荐）

从本仓库的 **Releases** 下载 `world-execute-mv-macos.zip` 并解压。音乐、动画、字幕、频谱数据和音频播放组件已经内嵌在 `world-execute-mv.pyz` 内，无需另外下载或指定 MP3。

需要 **macOS 12 或更新版本、Python 3.9 或更新版本**。音频组件同时包含 Apple Silicon 和 Intel 架构。终端播放器自身只使用 Python 标准库。

在 macOS「终端」中进入解压目录运行：

```sh
python3 world-execute-mv.pyz
```

也可双击 `运行单文件.command`，在系统终端中播放。按空格开始。推荐全屏，终端至少 64 列 × 24 行，128 列 × 44 行及以上效果更好。

```sh
# 从 2:38.7 开始直接播放
python3 world-execute-mv.pyz --start 158.7 --autoplay
```

内嵌音乐是随程序封装的资源，不是加密或 DRM。播放时会解包到当前用户的临时目录，正常退出后清理；不会读取旧电脑 Downloads 中的文件。运行过程无需联网。

## 操作

| 按键 | 功能 |
| --- | --- |
| 空格 | 开始／暂停 |
| 左／右 | 后退／前进 5 秒 |
| R | 从头播放 |
| Q | 退出 |
| H | 显示全部帮助 |
| 1–5 | 跳转章节 |

## 从源码运行

仓库不保存音频文件。从源码运行或重新打包前，请将自己的音频放到本地 `media/song.mp3`。使用 Release 播放包无需此步骤。

首次构建音频组件需要 Apple Command Line Tools（含 Swift 编译器）：

```sh
xcode-select --install
```

然后执行：

```sh
./run.sh
```

启动脚本在缺少 `audio-clock` 时自动编译本机架构。双击 `播放MV.command` 可在 macOS 系统终端中运行源码版。

## 重新打包

```sh
python3 tools/build_bundle.py
python3 tests/test_bundle.py
```

构建输出在 `dist/`：

- `world-execute-mv.pyz`：内嵌音乐的单文件播放器。
- `world-execute-mv-macos.zip`：包含播放器、启动器、说明的分发包。
- `SHA256SUMS.txt`：下载校验值。

音频以 macOS 音频时钟驱动画面；暂停、跳转时字幕与动画跟随音频时间。构建会生成 universal 音频组件，并在单文件包内记录各资源 SHA-256 以检查完整性。

## 收录范围

本仓库为项目归档，包含当前播放器、字幕、频谱和构建工具。音乐仅内嵌于 Release 播放包。

原曲与歌词：Mili《world.execute(me);》。本项目是个人创作与备份，未对原曲、歌词或其他第三方素材授予额外使用许可。
