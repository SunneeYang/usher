import "./style.css";
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
  version: "",
  configPath: "",
  lastResult: null,
};

const app = document.getElementById("app");

function subtitleText() {
  if (!state.version) {
    return "正在加载...";
  }
  return `v${state.version} · ${state.configPath}`;
}

function renderResultBody(result) {
  if (!result) {
    return "";
  }

  if (!result.success) {
    return `<div class="status error">${escapeHtml(result.error || "扫描失败")}</div>`;
  }

  const warning = result.error
    ? `<div class="status error" style="margin-bottom:10px">${escapeHtml(result.error)}</div>`
    : "";

  const sources = (result.sources || [])
    .map(
      (s) => `
      <div class="source-block">
        <h3>${escapeHtml(s.label)}</h3>
        <div class="meta">${s.videos} 个视频 · ${s.subdirs} 个子目录</div>
        <div class="chips">
          ${(s.topDirs || [])
            .slice(0, 6)
            .map((d) => `<span class="chip">${escapeHtml(d.name)}: ${d.videos}</span>`)
            .join("")}
        </div>
      </div>`
    )
    .join("");

  const change = result.change || {};
  const added = change.added || [];
  const removed = change.removed || [];

  return `
    ${warning}
    <div class="status success">
      完成 · ${result.videoCount} 个视频 · 耗时 ${result.durationMs}ms
    </div>
    <p class="meta result-summary">${escapeHtml(change.summary || "")}</p>
    ${sources}
    ${
      added.length
        ? `<div class="field"><strong>新增</strong><ul class="path-list">${added
            .slice(0, 5)
            .map((p) => `<li>${escapeHtml(p)}</li>`)
            .join("")}</ul></div>`
        : ""
    }
    ${
      removed.length
        ? `<div class="field"><strong>删除</strong><ul class="path-list">${removed
            .slice(0, 5)
            .map((p) => `<li>${escapeHtml(p)}</li>`)
            .join("")}</ul></div>`
        : ""
    }
    <div class="actions">
      <button class="secondary" id="reveal-output">在 Finder 中显示</button>
      <button class="secondary" id="open-playlist">用播放器打开</button>
    </div>
  `;
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

function render() {
  const { config, scanning, lastResult } = state;
  const sourceDirs = Array.isArray(config.sourceDirs) ? config.sourceDirs : [];
  const showCustomPlayer = config.player === "custom";

  app.innerHTML = `
    <header class="titlebar">
      <h1>usher</h1>
      <p id="subtitle">${escapeHtml(subtitleText())}</p>
    </header>

    <section class="card">
      <h2>视频目录</h2>
      <ul class="dir-list" id="dir-list">
        ${
          sourceDirs.length === 0
            ? '<li style="justify-content:center;color:var(--muted)">尚未添加目录</li>'
            : sourceDirs
                .map(
                  (dir, i) => `
            <li>
              <span>${escapeHtml(dir)}</span>
              <button class="danger" data-remove="${i}">移除</button>
            </li>`
                )
                .join("")
        }
      </ul>
      <div class="actions">
        <button class="secondary" id="add-dir" ${scanning ? "disabled" : ""}>添加目录</button>
      </div>
    </section>

    <section class="card">
      <h2>输出与选项</h2>
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
      <div class="options">
        <label><input type="checkbox" id="open-after-scan" ${config.openAfterScan ? "checked" : ""} ${scanning ? "disabled" : ""} /> 生成后自动用播放器打开</label>
        <label><input type="checkbox" id="sort" ${config.sort ? "checked" : ""} ${scanning ? "disabled" : ""} /> 按路径排序</label>
        <label><input type="checkbox" id="shuffle" ${config.shuffle ? "checked" : ""} ${scanning ? "disabled" : ""} /> 随机打乱</label>
        <label><input type="checkbox" id="skip-hidden" ${config.skipHidden ? "checked" : ""} ${scanning ? "disabled" : ""} /> 跳过隐藏文件</label>
        <label><input type="checkbox" id="fresh" ${scanning ? "disabled" : ""} /> 全量重扫 (-fresh)</label>
      </div>
      <div class="actions">
        <button class="primary" id="run-scan" ${scanning ? "disabled" : ""}>${scanning ? "扫描中..." : "生成播放列表"}</button>
      </div>
    </section>

    <section class="card ${lastResult ? "" : "hidden"}" id="result-card">
      <h2>结果</h2>
      <div id="result-body">${renderResultBody(lastResult)}</div>
    </section>
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
  };
}

async function init() {
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
  } catch (err) {
    app.innerHTML = `<div class="status error">初始化失败: ${escapeHtml(err)}</div>`;
  }
}

init();
