import "./style.css";
import { WindowSetSize } from "../wailsjs/runtime/runtime";
import {
  GetConfig,
  SaveConfig,
  RunScan,
  PickSourceDir,
  PickOutputFile,
  RevealInFinder,
  OpenPlaylist,
  GetPlayerOptions,
  PickPlayerApp,
  GetConfigPath,
  GetVersion,
} from "../wailsjs/go/main/DesktopApp";

const defaultConfig = {
  sourceDirs: [],
  outputFile: "playlist.m3u",
  shuffle: false,
  sort: true,
  skipHidden: true,
  scanWorkers: 32,
  scanCache: ".usher-scan-cache.json",
  cacheVerify: "none",
  player: "default",
  playerApp: "",
  openAfterScan: false,
};

const state = {
  config: { ...defaultConfig },
  playerOptions: [],
  scanning: false,
  showAdvanced: false,
  version: "",
  configPath: "",
  lastResult: null,
};

const app = document.getElementById("app");

const WINDOW_MIN = { w: 380, h: 420 };
const WINDOW_DEFAULT_W = 680;

function setLayoutMode(mode) {
  document.body.classList.toggle("layout-natural", mode === "natural");
  document.body.classList.toggle("layout-constrained", mode === "constrained");
}

function measureShellHeight() {
  const shell = document.querySelector(".app-shell");
  if (!shell) {
    return 0;
  }

  const undo = [];
  const push = (el, prop, value) => {
    undo.push([el, prop, el.style[prop]]);
    el.style[prop] = value;
  };

  push(shell, "height", "auto");

  const appBody = shell.querySelector(".app-body");
  if (appBody) {
    push(appBody, "flex", "none");
    push(appBody, "minHeight", "auto");
    push(appBody, "overflow", "visible");
    push(appBody, "height", "auto");
  }

  for (const el of shell.querySelectorAll(
    ".app-panel, .card--dirs, .card--result, #result-body, .dir-list"
  )) {
    push(el, "flex", "none");
    push(el, "minHeight", "auto");
    push(el, "maxHeight", "none");
    push(el, "overflow", "visible");
  }

  void shell.offsetHeight;
  const height = Math.ceil(shell.getBoundingClientRect().height);

  for (const [el, prop, value] of undo) {
    el.style[prop] = value;
  }

  return height;
}

let lastFitHeight = 0;

async function fitWindowToContent() {
  const height = measureShellHeight();
  if (height <= 0) {
    return;
  }

  const targetH = Math.max(WINDOW_MIN.h, height + 8);
  if (Math.abs(targetH - lastFitHeight) <= 2) {
    return;
  }
  lastFitHeight = targetH;

  try {
    await WindowSetSize(WINDOW_DEFAULT_W, targetH);
    setLayoutMode("constrained");
  } catch {
    setLayoutMode("natural");
  }
}

function subtitleText() {
  if (!state.version) {
    return "正在加载...";
  }
  return `v${state.version} · ${state.configPath}`;
}

function formatNumber(value) {
  if (value == null) {
    return "—";
  }
  return Number(value).toLocaleString("zh-CN");
}

function formatDuration(ms) {
  if (ms == null) {
    return "—";
  }
  if (ms < 1000) {
    return `${ms}ms`;
  }
  return `${(ms / 1000).toFixed(1)}s`;
}

function computeStats() {
  const sourceCount = Array.isArray(state.config.sourceDirs)
    ? state.config.sourceDirs.length
    : 0;
  const result = state.lastResult;

  if (state.scanning) {
    return { mode: "scanning", sourceCount };
  }

  if (!result?.success) {
    return { mode: "idle", sourceCount };
  }

  const change = result.change || {};
  return {
    mode: "result",
    sourceCount: result.sources?.length || sourceCount,
    videos: result.videoCount,
    added: change.added?.length ?? 0,
    removed: change.removed?.length ?? 0,
    duration: result.durationMs,
    hasHistory: change.hasHistory,
  };
}

function renderStatsBar() {
  const stats = computeStats();

  if (stats.mode === "scanning") {
    return `
      <section class="stats-bar stats-bar--scanning" aria-live="polite">
        <div class="stat-item stat-item--wide">
          <span class="stat-value">扫描中</span>
          <span class="stat-label">${stats.sourceCount} 个源目录</span>
        </div>
      </section>`;
  }

  if (stats.mode === "idle") {
    return `
      <section class="stats-bar stats-bar--idle">
        <div class="stat-item">
          <span class="stat-value">—</span>
          <span class="stat-label">视频</span>
        </div>
        <div class="stat-item">
          <span class="stat-value">${stats.sourceCount}</span>
          <span class="stat-label">源目录</span>
        </div>
        <div class="stat-item">
          <span class="stat-value">—</span>
          <span class="stat-label">耗时</span>
        </div>
        <div class="stat-item">
          <span class="stat-value muted">尚未扫描</span>
        </div>
      </section>`;
  }

  const changeLabel = stats.hasHistory
    ? `<span class="delta plus">+${stats.added}</span><span class="delta minus">-${stats.removed}</span>`
    : `<span class="stat-label">首次扫描</span>`;

  return `
    <section class="stats-bar">
      <div class="stat-item stat-item--hero">
        <span class="stat-value">${formatNumber(stats.videos)}</span>
        <span class="stat-label">视频</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">${stats.sourceCount}</span>
        <span class="stat-label">源目录</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">${formatDuration(stats.duration)}</span>
        <span class="stat-label">耗时</span>
      </div>
      <div class="stat-item stat-item--change">
        ${changeLabel}
      </div>
    </section>`;
}

function renderToolbar(scanning) {
  const hasResult = Boolean(state.lastResult?.success && state.lastResult?.outputFile);
  const disabled = scanning ? "disabled" : "";

  return `
    <section class="toolbar">
      <button class="primary" id="run-scan" ${disabled}>
        ${scanning ? "扫描中…" : "生成播放列表"}
      </button>
      <button class="secondary" id="add-dir" ${disabled}>添加目录</button>
      <div class="toolbar-spacer"></div>
      <button class="secondary" id="reveal-output" ${hasResult && !scanning ? "" : "disabled"}>
        在 Finder 中显示
      </button>
      <button class="secondary" id="open-playlist" ${hasResult && !scanning ? "" : "disabled"}>
        用播放器打开
      </button>
    </section>`;
}

function renderSourceGrid(sources) {
  if (!sources?.length) {
    return "";
  }

  return `
    <div class="source-grid">
      ${sources
        .map(
          (s) => `
        <article class="source-card">
          <div class="source-card-head">
            <h3>${escapeHtml(s.label)}</h3>
            <span class="source-card-count">${formatNumber(s.videos)}</span>
          </div>
          <p class="meta">${s.subdirs} 个子目录</p>
          <div class="chips">
            ${(s.topDirs || [])
              .slice(0, 4)
              .map((d) => `<span class="chip">${escapeHtml(d.name)} · ${d.videos}</span>`)
              .join("")}
          </div>
        </article>`
        )
        .join("")}
    </div>`;
}

function renderResultBody(result) {
  if (!result) {
    return `<p class="meta empty-hint">生成播放列表后，源目录统计与变更明细会显示在这里。</p>`;
  }

  if (!result.success) {
    return `<div class="status error">${escapeHtml(result.error || "扫描失败")}</div>`;
  }

  const warning = result.error
    ? `<div class="status error result-warning">${escapeHtml(result.error)}</div>`
    : "";

  const change = result.change || {};
  const added = change.added || [];
  const removed = change.removed || [];

  return `
    ${warning}
    ${change.summary ? `<p class="meta result-summary">${escapeHtml(change.summary)}</p>` : ""}
    ${renderSourceGrid(result.sources)}
    ${
      added.length
        ? `<details class="change-details">
            <summary>新增 ${added.length} 个</summary>
            <ul class="path-list">${added
              .slice(0, 8)
              .map((p) => `<li>${escapeHtml(p)}</li>`)
              .join("")}</ul>
          </details>`
        : ""
    }
    ${
      removed.length
        ? `<details class="change-details">
            <summary>删除 ${removed.length} 个</summary>
            <ul class="path-list">${removed
              .slice(0, 8)
              .map((p) => `<li>${escapeHtml(p)}</li>`)
              .join("")}</ul>
          </details>`
        : ""
    }`;
}

function renderPlayerOptions(config) {
  const options = state.playerOptions.length
    ? state.playerOptions
    : [{ id: "default", label: "系统默认" }];
  return options
    .map(
      (opt) =>
        `<option value="${escapeAttr(opt.id)}" ${config.player === opt.id ? "selected" : ""}>${escapeHtml(opt.label)}</option>`
    )
    .join("");
}

function renderAdvancedPanel(config, scanning) {
  const disabled = scanning ? "disabled" : "";
  const open = state.showAdvanced ? "open" : "";

  return `
    <details class="advanced-panel" id="advanced-panel" ${open}>
      <summary id="toggle-more">更多选项</summary>
      <div class="advanced-body">
        <div class="options options--advanced">
          <label><input type="checkbox" id="shuffle" ${config.shuffle ? "checked" : ""} ${disabled} /> 随机打乱</label>
          <label><input type="checkbox" id="skip-hidden" ${config.skipHidden ? "checked" : ""} ${disabled} /> 跳过隐藏文件</label>
          <label><input type="checkbox" id="fresh" ${disabled} /> 全量重扫 (-fresh)</label>
        </div>
        <div class="field">
          <label>并行扫描线程 (scan_workers)</label>
          <input type="number" id="scan-workers" min="1" max="64" value="${config.scanWorkers}" ${disabled} />
        </div>
        <div class="field">
          <label>扫描缓存路径</label>
          <input type="text" id="scan-cache" value="${escapeAttr(config.scanCache)}" ${disabled} />
        </div>
        <div class="field">
          <label>缓存校验 (cache_verify)</label>
          <select id="cache-verify" class="select" ${disabled}>
            <option value="none" ${config.cacheVerify === "none" ? "selected" : ""}>none（NAS 推荐）</option>
            <option value="mtime" ${config.cacheVerify === "mtime" ? "selected" : ""}>mtime</option>
          </select>
        </div>
      </div>
    </details>`;
}

function render() {
  const { config, scanning, lastResult } = state;
  const sourceDirs = Array.isArray(config.sourceDirs) ? config.sourceDirs : [];
  const showCustomPlayer = config.player === "custom";

  app.innerHTML = `
    <div class="app-shell">
      <header class="titlebar">
        <div class="titlebar-brand">
          <img class="titlebar-logo" src="/logo-mark.svg" width="28" height="28" alt="" />
          <div class="titlebar-row">
            <h1>usher</h1>
            <p id="subtitle" title="${escapeAttr(subtitleText())}">${escapeHtml(subtitleText())}</p>
          </div>
        </div>
      </header>

      <div class="app-top">
        ${renderStatsBar()}
        ${renderToolbar(scanning)}
      </div>

      <div class="app-body">
        <div class="app-panel app-panel--config">
          <section class="card card--dirs">
            <h2>视频目录</h2>
            <ul class="dir-list" id="dir-list">
              ${
                sourceDirs.length === 0
                  ? '<li class="dir-empty">尚未添加目录</li>'
                  : sourceDirs
                      .map(
                        (dir, i) => `
                <li>
                  <span class="dir-path">${escapeHtml(dir)}</span>
                  <button class="danger" data-remove="${i}" ${scanning ? "disabled" : ""}>移除</button>
                </li>`
                      )
                      .join("")
              }
            </ul>
          </section>

          <section class="card card--output">
            <h2>输出</h2>
            <div class="field">
              <label>播放列表路径</label>
              <div class="row">
                <input type="text" id="output-file" value="${escapeAttr(config.outputFile)}" ${scanning ? "disabled" : ""} />
                <button class="secondary" id="pick-output" ${scanning ? "disabled" : ""}>选择</button>
              </div>
            </div>
            <div class="field">
              <label>播放器</label>
              <div class="row">
                <select id="player" class="select" ${scanning ? "disabled" : ""}>
                  ${renderPlayerOptions(config)}
                </select>
                ${
                  showCustomPlayer
                    ? `<button class="secondary" id="pick-player" ${scanning ? "disabled" : ""}>选择应用</button>`
                    : ""
                }
              </div>
              ${
                showCustomPlayer && config.playerApp
                  ? `<p class="meta player-app-path">${escapeHtml(config.playerApp)}</p>`
                  : ""
              }
            </div>
            <div class="options options--quick">
              <label><input type="checkbox" id="sort" ${config.sort ? "checked" : ""} ${scanning ? "disabled" : ""} /> 按路径排序</label>
              <label><input type="checkbox" id="open-after-scan" ${config.openAfterScan ? "checked" : ""} ${scanning ? "disabled" : ""} /> 生成后自动打开</label>
            </div>
            ${renderAdvancedPanel(config, scanning)}
          </section>
        </div>

        <div class="app-panel app-panel--result">
          <section class="card card--result" id="result-card">
            <h2>扫描结果</h2>
            <div id="result-body">${renderResultBody(lastResult)}</div>
          </section>
        </div>
      </div>
    </div>
  `;

  bindEvents();
}

function bindEvents() {
  document.getElementById("add-dir")?.addEventListener("click", async () => {
    const dir = await PickSourceDir();
    if (!dir) return;
    if (!Array.isArray(state.config.sourceDirs)) {
      state.config.sourceDirs = [];
    }
    if (!state.config.sourceDirs.includes(dir)) {
      state.config.sourceDirs.push(dir);
      await persistConfig();
      render();
    }
  });

  document.querySelectorAll("[data-remove]").forEach((btn) => {
    btn.addEventListener("click", async () => {
      const index = Number(btn.dataset.remove);
      state.config.sourceDirs.splice(index, 1);
      await persistConfig();
      render();
    });
  });

  document.getElementById("pick-output")?.addEventListener("click", async () => {
    const path = await PickOutputFile();
    if (path) {
      state.config.outputFile = path;
      await persistConfig();
      render();
    }
  });

  document.getElementById("output-file")?.addEventListener("change", async (e) => {
    state.config.outputFile = e.target.value;
    await persistConfig();
  });

  for (const id of ["sort", "shuffle", "skip-hidden"]) {
    document.getElementById(id)?.addEventListener("change", async (e) => {
      const key = id === "skip-hidden" ? "skipHidden" : id;
      state.config[key] = e.target.checked;
      await persistConfig();
    });
  }

  document.getElementById("scan-workers")?.addEventListener("change", async (e) => {
    const value = Math.min(64, Math.max(1, Number(e.target.value) || 32));
    state.config.scanWorkers = value;
    e.target.value = value;
    await persistConfig();
  });

  document.getElementById("scan-cache")?.addEventListener("change", async (e) => {
    state.config.scanCache = e.target.value;
    await persistConfig();
  });

  document.getElementById("cache-verify")?.addEventListener("change", async (e) => {
    state.config.cacheVerify = e.target.value;
    await persistConfig();
  });

  document.getElementById("advanced-panel")?.addEventListener("toggle", (e) => {
    state.showAdvanced = e.target.open;
    void fitWindowToContent();
  });

  document.getElementById("reveal-output")?.addEventListener("click", () => {
    if (state.lastResult?.outputFile) {
      RevealInFinder(state.lastResult.outputFile);
    }
  });

  document.getElementById("open-playlist")?.addEventListener("click", async () => {
    const path = state.lastResult?.outputFile || state.config.outputFile;
    try {
      await OpenPlaylist(path);
    } catch (err) {
      state.lastResult = {
        ...(state.lastResult || { success: true }),
        error: String(err),
      };
      render();
    }
  });

  document.getElementById("player")?.addEventListener("change", async (e) => {
    state.config.player = e.target.value;
    if (state.config.player !== "custom") {
      state.config.playerApp = "";
    }
    await persistConfig();
    render();
  });

  document.getElementById("pick-player")?.addEventListener("click", async () => {
    const appPath = await PickPlayerApp();
    if (!appPath) return;
    state.config.player = "custom";
    state.config.playerApp = appPath;
    await persistConfig();
    render();
  });

  document.getElementById("open-after-scan")?.addEventListener("change", async (e) => {
    state.config.openAfterScan = e.target.checked;
    await persistConfig();
  });

  document.getElementById("run-scan")?.addEventListener("click", async () => {
    if (!state.config.sourceDirs?.length) {
      state.lastResult = { success: false, error: "请至少添加一个视频目录" };
      render();
      return;
    }

    state.scanning = true;
    render();

    const fresh = document.getElementById("fresh")?.checked ?? false;
    try {
      await persistConfig();
      state.lastResult = await RunScan(fresh);
    } catch (err) {
      state.lastResult = { success: false, error: String(err) };
    } finally {
      state.scanning = false;
      render();
    }
  });
}

async function persistConfig() {
  await SaveConfig(state.config);
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function escapeAttr(value) {
  return escapeHtml(value).replaceAll("'", "&#39;");
}

function normalizeConfig(config) {
  return {
    ...defaultConfig,
    ...(config || {}),
    player: config?.player || "default",
    playerApp: config?.playerApp || "",
    openAfterScan: Boolean(config?.openAfterScan),
    scanWorkers: config?.scanWorkers > 0 ? config.scanWorkers : defaultConfig.scanWorkers,
    scanCache: config?.scanCache || defaultConfig.scanCache,
    cacheVerify: config?.cacheVerify || defaultConfig.cacheVerify,
  };
}

async function init() {
  setLayoutMode("natural");
  render();
  try {
    const [version, configPath, config, playerOptions] = await Promise.all([
      GetVersion(),
      GetConfigPath(),
      GetConfig(),
      GetPlayerOptions(),
    ]);
    state.version = version;
    state.configPath = configPath;
    state.config = normalizeConfig(config);
    state.playerOptions = Array.isArray(playerOptions) ? playerOptions : [];
    render();
    await fitWindowToContent();
  } catch (err) {
    app.innerHTML = `<div class="status error">初始化失败: ${escapeHtml(err)}</div>`;
  }
}

init();
