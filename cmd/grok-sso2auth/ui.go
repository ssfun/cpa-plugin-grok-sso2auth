package main

func uiHTML() []byte {
	return []byte(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>Grok SSO → CLIProxyAPI</title>
<style>
  :root {
    --bg: #0b1020;
    --panel: rgba(19, 28, 49, .88);
    --panel-strong: #151f36;
    --border: #2a3858;
    --text: #edf3ff;
    --muted: #95a5c4;
    --accent: #6ea8ff;
    --accent-strong: #3d7df0;
    --success: #47d7a0;
    --warning: #f7c56b;
    --danger: #ff7f91;
    --mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    --sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Noto Sans SC", sans-serif;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    color: var(--text);
    background:
      radial-gradient(900px 560px at -10% -12%, rgba(42, 103, 205, .38), transparent 62%),
      radial-gradient(900px 560px at 112% 0%, rgba(103, 55, 168, .27), transparent 58%),
      var(--bg);
    font-family: var(--sans);
    min-height: 100vh;
  }
  main { width: min(1120px, calc(100% - 32px)); margin: 0 auto; padding: 30px 0 64px; }
  .hero { display: flex; justify-content: space-between; gap: 24px; align-items: flex-start; margin-bottom: 22px; }
  .eyebrow { color: var(--accent); font-size: .78rem; letter-spacing: .12em; text-transform: uppercase; margin: 0 0 10px; }
  h1 { margin: 0; font-size: clamp(1.65rem, 4vw, 2.35rem); letter-spacing: -.04em; }
  .hero p { color: var(--muted); line-height: 1.6; max-width: 700px; margin: 10px 0 0; }
  .hero-meta { display: flex; flex-direction: column; align-items: flex-end; gap: 9px; color: var(--muted); font-size: .8rem; white-space: nowrap; }
  .online { display: inline-flex; align-items: center; gap: 8px; color: var(--success); }
  .online::before { content: ""; width: 8px; height: 8px; border-radius: 50%; background: currentColor; box-shadow: 0 0 0 4px rgba(71, 215, 160, .14); }
  .hero-meta a { color: var(--accent); text-decoration: none; }
  .hero-meta a:hover { text-decoration: underline; }
  .card {
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 18px;
    padding: 20px;
    margin-bottom: 16px;
    box-shadow: 0 16px 44px rgba(0, 0, 0, .18);
    backdrop-filter: blur(12px);
  }
  .card-title { display: flex; justify-content: space-between; gap: 14px; align-items: baseline; margin-bottom: 15px; }
  .card-title h2 { margin: 0; font-size: 1.02rem; }
  .card-title p { margin: 0; color: var(--muted); font-size: .78rem; }
  label { display: block; color: var(--muted); font-size: .79rem; margin-bottom: 7px; }
  input, textarea, select {
    width: 100%;
    color: var(--text);
    background: rgba(5, 10, 22, .78);
    border: 1px solid var(--border);
    border-radius: 11px;
    padding: 10px 12px;
    font: inherit;
    outline: none;
  }
  input, textarea { font-family: var(--mono); font-size: .83rem; }
  textarea { min-height: 150px; resize: vertical; line-height: 1.55; }
  input:focus, textarea:focus, select:focus { border-color: var(--accent); box-shadow: 0 0 0 3px rgba(110, 168, 255, .15); }
  .key-grid, .form-grid { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 12px; align-items: end; }
  .form-grid { grid-template-columns: minmax(0, 1.5fr) minmax(170px, .5fr); }
  .field { min-width: 0; }
  .hint { color: var(--muted); font-size: .77rem; line-height: 1.55; margin: 9px 0 0; }
  .hint code, code { color: #c5d8ff; font-family: var(--mono); font-size: .88em; }
  .actions { display: flex; flex-wrap: wrap; gap: 9px; margin-top: 14px; }
  button {
    border: 1px solid transparent;
    border-radius: 10px;
    color: white;
    padding: 9px 13px;
    font: inherit;
    font-size: .84rem;
    font-weight: 650;
    cursor: pointer;
    transition: transform .08s ease, opacity .15s ease, border-color .15s ease;
  }
  button:active { transform: translateY(1px); }
  button:disabled { cursor: not-allowed; opacity: .5; }
  .primary { background: linear-gradient(180deg, #6ea8ff, var(--accent-strong)); }
  .success { color: #08271e; background: linear-gradient(180deg, #70e6ba, var(--success)); }
  .ghost { color: var(--text); background: transparent; border-color: var(--border); }
  .ghost:hover { border-color: var(--accent); }
  .danger { color: #ffd8dd; background: rgba(164, 43, 68, .32); border-color: rgba(255, 127, 145, .4); }
  .checks { display: flex; flex-wrap: wrap; gap: 15px; margin-top: 13px; }
  .check { display: inline-flex; align-items: center; gap: 8px; color: var(--text); font-size: .82rem; cursor: pointer; }
  .check input { width: auto; accent-color: var(--accent-strong); }
  .key-state { display: inline-flex; align-items: center; min-height: 31px; color: var(--muted); font-size: .77rem; }
  .key-state.ready { color: var(--success); }
  .key-state.error { color: var(--danger); }
  .result-status { min-height: 24px; margin: 0; color: var(--muted); font-size: .78rem; }
  .result-status.result-error { color: var(--danger); }
  .result-status.result-success { color: var(--success); }
  .summary-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; margin-bottom: 13px; }
  .metric { border: 1px solid var(--border); border-radius: 12px; padding: 11px 13px; background: rgba(8, 14, 28, .44); }
  .metric span { display: block; color: var(--muted); font-size: .73rem; }
  .metric strong { display: block; margin-top: 3px; font-size: 1.2rem; }
  .metric.ok strong { color: var(--success); }
  .metric.fail strong { color: var(--danger); }
  .result-box { color: var(--muted); font-size: .8rem; }
  .result-table, .auth-table { width: 100%; border-collapse: collapse; font-size: .81rem; }
  .table-scroll { overflow-x: auto; }
  th { color: var(--muted); font-size: .7rem; font-weight: 650; letter-spacing: .06em; text-transform: uppercase; white-space: nowrap; }
  th, td { padding: 10px 8px; border-bottom: 1px solid rgba(42, 56, 88, .7); text-align: left; vertical-align: top; }
  tbody tr:hover { background: rgba(110, 168, 255, .05); }
  td small { display: block; color: var(--muted); margin-top: 3px; line-height: 1.4; }
  .mono { font-family: var(--mono); }
  .badge { display: inline-flex; border: 1px solid var(--border); border-radius: 999px; padding: 3px 8px; color: var(--muted); font-size: .7rem; white-space: nowrap; }
  .badge.ok { color: var(--success); border-color: rgba(71, 215, 160, .35); background: rgba(71, 215, 160, .08); }
  .badge.warn { color: var(--warning); border-color: rgba(247, 197, 107, .35); background: rgba(247, 197, 107, .08); }
  .badge.err { color: var(--danger); border-color: rgba(255, 127, 145, .35); background: rgba(255, 127, 145, .08); }
  .toolbar { display: grid; grid-template-columns: minmax(0, 1fr) 160px auto; gap: 9px; align-items: end; margin-bottom: 13px; }
  .toolbar label { margin-bottom: 5px; }
  .empty { border: 1px dashed var(--border); border-radius: 12px; color: var(--muted); text-align: center; padding: 27px 15px; font-size: .82rem; }
  details { border: 1px solid var(--border); border-radius: 12px; overflow: hidden; }
  summary { cursor: pointer; padding: 13px 15px; color: var(--text); font-weight: 650; font-size: .86rem; }
  details[open] summary { border-bottom: 1px solid var(--border); }
  .details-body { padding: 16px; }
  .detail-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }
  .detail-item { padding: 10px 12px; border-radius: 10px; background: rgba(8, 14, 28, .52); border: 1px solid rgba(42, 56, 88, .78); }
  .detail-item span { display: block; color: var(--muted); font-size: .72rem; margin-bottom: 4px; }
  .detail-item strong { display: block; font-size: .81rem; overflow-wrap: anywhere; }
  .detail-error { color: var(--danger); font-size: .82rem; white-space: pre-wrap; }
  pre { margin: 0; white-space: pre-wrap; word-break: break-word; font: .77rem/1.5 var(--mono); color: #d7e4ff; }
  .dialog { width: min(700px, calc(100% - 28px)); color: var(--text); background: var(--panel-strong); border: 1px solid var(--border); border-radius: 16px; padding: 0; box-shadow: 0 26px 80px rgba(0,0,0,.5); }
  .dialog::backdrop { background: rgba(2, 5, 15, .72); backdrop-filter: blur(4px); }
  .dialog-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 16px 18px; border-bottom: 1px solid var(--border); }
  .dialog-head h2 { margin: 0; font-size: 1rem; }
  .dialog-body { padding: 18px; max-height: min(70vh, 600px); overflow: auto; }
  .footer-note { color: var(--muted); font-size: .75rem; line-height: 1.6; margin-top: 18px; }
  @media (max-width: 720px) {
    main { width: min(100% - 20px, 1120px); padding-top: 19px; }
    .hero { flex-direction: column; gap: 13px; }
    .hero-meta { align-items: flex-start; }
    .key-grid, .form-grid, .toolbar, .detail-grid { grid-template-columns: 1fr; }
    .summary-grid { grid-template-columns: 1fr; }
    .card { padding: 15px; border-radius: 14px; }
  }
</style>
</head>
<body>
<main>
  <section class="hero">
    <div>
      <p class="eyebrow">CLIProxyAPI · plugin management</p>
      <h1>Grok SSO → Auth</h1>
      <p>把 xAI / Grok SSO Cookie 通过 Device Flow 换成 CLIProxyAPI 可用的 xAI OAuth 凭证，并使用宿主的 auth 回调安全写入 auth-dir。</p>
    </div>
    <div class="hero-meta">
      <span class="online">资源页已连接</span>
      <a href="https://help.router-for.me/cn/plugin/development.html" target="_blank" rel="noreferrer">插件开发文档 ↗</a>
    </div>
  </section>

  <section class="card">
    <div class="card-title">
      <h2>管理认证</h2>
      <p id="keyState" class="key-state">等待 Management Key</p>
    </div>
    <div class="key-grid">
      <div class="field">
        <label for="mgmtKey">Management Key</label>
        <input id="mgmtKey" type="password" autocomplete="off" placeholder="用于 /v0/management/*，不会写入插件配置"/>
      </div>
      <div class="actions" style="margin:0">
        <button class="primary" id="btnSaveKey" type="button">保存并连接</button>
        <button class="ghost" id="btnClearKey" type="button">清除</button>
      </div>
    </div>
    <p class="hint">资源页本身按 CLIProxyAPI 约定不鉴权；转换、导入、列表和运行时详情都通过宿主认证的 Management API 完成。保存只写入当前浏览器此插件自己的 localStorage 项。</p>
  </section>

  <section class="card">
    <div class="card-title">
      <h2>SSO 转换</h2>
      <p id="inputHint">支持单个 Cookie 或多行列表</p>
    </div>
    <div class="field">
      <label for="sso">SSO Cookie / 批量列表</label>
      <textarea id="sso" spellcheck="false" placeholder="单行 JWT，或每行一个：
email----password----sso
email----sso
eyJhbGciOi..."></textarea>
    </div>
    <div class="form-grid" style="margin-top:12px">
      <div class="field">
        <label for="email">Email 覆盖（可选）</label>
        <input id="email" type="text" autocomplete="off" placeholder="user@example.com"/>
      </div>
      <div class="field">
        <label for="delay">批量间隔（秒）</label>
        <input id="delay" type="number" min="0" max="3600" step="1" value="45"/>
      </div>
    </div>
    <div class="checks">
      <label class="check"><input id="validate" type="checkbox"/> 先访问 accounts.x.ai 验证 SSO</label>
      <label class="check"><input id="showRaw" type="checkbox"/> 显示完整转换 JSON（包含敏感凭证）</label>
    </div>
    <div class="actions">
      <button class="primary" id="btnConvert" type="button">仅转换</button>
      <button class="success" id="btnConvertImport" type="button">转换并导入</button>
    </div>
    <p class="hint">转换后的文件通过 <code>host.auth.save</code> 写入并注册到宿主运行时；页面默认不展示 access/refresh token。</p>
  </section>

  <section class="card">
    <div class="card-title">
      <h2>最近结果</h2>
      <p id="resultMeta">等待操作</p>
    </div>
    <div id="resultBox" class="result-box empty">提交转换后，结果会显示在这里。</div>
  </section>

  <section class="card">
    <div class="card-title">
      <h2>Auth 文件与运行时账号</h2>
      <p id="listMeta">尚未加载</p>
    </div>
    <div class="toolbar">
      <div class="field">
        <label for="filter">搜索文件名、邮箱或标签</label>
        <input id="filter" type="search" placeholder="例如 user@example.com"/>
      </div>
      <div class="field">
        <label for="providerFilter">提供商</label>
        <select id="providerFilter">
          <option value="all">全部</option>
          <option value="xai">xAI</option>
          <option value="other">其他</option>
        </select>
      </div>
      <button class="ghost" id="btnRefresh" type="button">刷新列表</button>
    </div>
    <div id="listWrap" class="empty">请输入 Management Key 后刷新。</div>
    <p class="hint">列表来自 <code>host.auth.list</code>，运行时详情来自 <code>host.auth.get_runtime</code>。物理路径和凭证 JSON 不会被列表接口返回。</p>
  </section>

  <section class="card">
    <details>
      <summary>导入已有 xAI OAuth JSON</summary>
      <div class="details-body">
        <div class="form-grid">
          <div class="field">
            <label for="authName">文件名（可选，必须以 .json 结尾）</label>
            <input id="authName" type="text" autocomplete="off" placeholder="留空时按 email / sub 自动命名"/>
          </div>
          <div class="field">
            <label>宿主能力</label>
            <div class="key-state ready">host.auth.save</div>
          </div>
        </div>
        <div class="field" style="margin-top:12px">
          <label for="authJson">JSON</label>
          <textarea id="authJson" spellcheck="false" placeholder='{"type":"xai","auth_kind":"oauth",...}'></textarea>
        </div>
        <div class="actions">
          <button class="success" id="btnImport" type="button">导入 JSON</button>
        </div>
        <p class="hint">导入请求会由 CLIProxyAPI 校验 JSON 和文件名，并在保存后 upsert 运行时 auth 记录。</p>
      </div>
    </details>
  </section>

  <p class="footer-note">安全提示：SSO 和 OAuth token 仅在浏览器与本机 CLIProxyAPI 之间传输。不要把资源页暴露到不可信网络，也不要勾选“显示完整转换 JSON”后分享页面或截图。</p>
</main>

<dialog id="detailDialog" class="dialog">
  <div class="dialog-head">
    <h2 id="detailTitle">运行时详情</h2>
    <button class="ghost" id="btnCloseDetail" type="button">关闭</button>
  </div>
  <div id="detailBody" class="dialog-body"></div>
</dialog>

<script>
(function () {
  "use strict";

  const API_ROOT = "/v0/management";
  const PLUGIN_ROOT = "/plugins/grok-sso2auth";
  const STORAGE_KEY = "grok_sso2auth.management_key";
  const LEGACY_KEYS = ["grok_sso2auth_mgmt_key", "managementKey", "management_key", "managementPassword", "management_password", "apiPassword", "api_password", "cliproxy_management_key", "cpa_management_key"];
  const state = { files: [], busy: false };
  const $ = (id) => document.getElementById(id);

  function esc(value) {
    return String(value == null ? "" : value).replace(/[&<>"']/g, (char) => ({
      "&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;", "'": "&#39;"
    }[char]));
  }

  function formatTime(value) {
    if (!value) return "—";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return String(value);
    return date.toLocaleString();
  }

  function getKey() {
    try {
      const own = localStorage.getItem(STORAGE_KEY);
      if (own && own.trim()) return own.trim();
      for (const key of LEGACY_KEYS) {
        const value = localStorage.getItem(key);
        if (value && value.trim()) return value.trim();
      }
    } catch (_) {}
    return "";
  }

  function setKeyState(message, kind) {
    const node = $("keyState");
    node.textContent = message;
    node.className = "key-state" + (kind ? " " + kind : "");
  }

  function setStatus(message, kind) {
    const node = $("resultMeta");
    node.textContent = message;
    node.className = "result-status" + (kind ? " result-" + kind : "");
  }

  function setBusy(busy) {
    state.busy = busy;
    ["btnConvert", "btnConvertImport", "btnImport", "btnRefresh", "btnSaveKey"].forEach((id) => {
      $(id).disabled = busy;
    });
  }

  async function request(method, route, body, timeoutMs) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs || 30000);
    const headers = {};
    const key = $("mgmtKey").value.trim();
    if (key) headers.Authorization = "Bearer " + key;
    if (body != null) headers["Content-Type"] = "application/json";
    let response;
    try {
      response = await fetch(API_ROOT + route, {
        method: method,
        headers: headers,
        body: body == null ? undefined : JSON.stringify(body),
        signal: controller.signal
      });
    } catch (error) {
      if (error && error.name === "AbortError") throw new Error("请求超时，请检查服务状态或批量间隔");
      throw error;
    } finally {
      clearTimeout(timer);
    }
    const text = await response.text();
    let data;
    try { data = text ? JSON.parse(text) : {}; } catch (_) { data = { raw: text }; }
    if (!response.ok) {
      const detail = data && (data.error || data.message || data.raw);
      const error = new Error(detail || ("HTTP " + response.status));
      error.status = response.status;
      error.data = data;
      throw error;
    }
    return data;
  }

  function parseInputLines() {
    return $("sso").value.split(/\r?\n/).map((line) => line.trim()).filter((line) => line && !line.startsWith("#"));
  }

  function updateInputHint() {
    const count = parseInputLines().length;
    $("inputHint").textContent = count ? (count + " 个待转换条目") : "支持单个 Cookie 或多行列表";
  }

  function conversionPayload() {
    const delay = Number($("delay").value || 0);
    return {
      sso: $("sso").value,
      email: $("email").value.trim(),
      delay_sec: Number.isFinite(delay) ? Math.max(0, Math.min(delay, 3600)) : 45,
      validate_sso: $("validate").checked
    };
  }

  function renderBatch(data, imported) {
    const items = Array.isArray(data.items) ? data.items : [];
    const ok = Number(data.ok || 0);
    const fail = Number(data.fail || 0);
    const total = Number(data.total || items.length);
    const rows = items.map((item) => {
      const status = item.ok ? '<span class="badge ok">成功</span>' : '<span class="badge err">失败</span>';
      const detail = item.ok
        ? esc(item.file_name || "")
        : esc(item.error || "转换失败");
      return "<tr><td>" + status + "</td><td>" + esc("#" + (item.index || "")) + "</td><td>" +
        esc(item.email || "未识别邮箱") + "<small>" + detail + "</small></td></tr>";
    }).join("");
    let raw = "";
    if ($("showRaw").checked) {
      raw = '<details style="margin-top:12px"><summary>完整 JSON（含敏感字段）</summary><div class="details-body"><pre>' +
        esc(JSON.stringify(data, null, 2)) + "</pre></div></details>";
    } else if (!imported) {
      raw = '<p class="hint">完整凭证 JSON 已由 API 返回，但当前未展示。需要复制 token 时再勾选“显示完整转换 JSON”。</p>';
    }
    $("resultBox").className = "result-box";
    $("resultBox").innerHTML =
      '<div class="summary-grid"><div class="metric ok"><span>成功</span><strong>' + ok +
      '</strong></div><div class="metric fail"><span>失败</span><strong>' + fail +
      '</strong></div><div class="metric"><span>总数</span><strong>' + total +
      '</strong></div></div>' +
      '<div class="table-scroll"><table class="result-table"><thead><tr><th>状态</th><th>序号</th><th>账号 / 结果</th></tr></thead><tbody>' +
      (rows || '<tr><td colspan="3">没有返回明细</td></tr>') + '</tbody></table></div>' + raw;
    setStatus(imported ? "转换并导入完成" : "转换完成", fail && !ok ? "error" : "success");
  }

  function authStatus(file) {
    if (file.disabled) return ["已禁用", "err"];
    if (file.unavailable) return ["暂不可用", "warn"];
    const status = String(file.status || "").toLowerCase();
    if (status === "active" || status === "ready" || status === "ok") return ["正常", "ok"];
    return [file.status || "未知", "warn"];
  }

  function renderAuthList() {
    const filter = $("filter").value.trim().toLowerCase();
    const provider = $("providerFilter").value;
    const files = state.files.filter((file) => {
      const haystack = [file.name, file.email, file.label, file.type, file.provider, file.account].join(" ").toLowerCase();
      if (filter && !haystack.includes(filter)) return false;
      const isXAI = String(file.type || file.provider || "").toLowerCase() === "xai";
      if (provider === "xai" && !isXAI) return false;
      if (provider === "other" && isXAI) return false;
      return true;
    });
    $("listMeta").textContent = files.length + " / " + state.files.length + " 个账号";
    if (!files.length) {
      $("listWrap").className = "empty";
      $("listWrap").textContent = state.files.length ? "没有匹配的账号。" : "暂无 auth 文件，或 Management Key 未连接。";
      return;
    }
    const rows = files.map((file) => {
      const status = authStatus(file);
      const identity = file.email || file.label || file.account || "未标注账号";
      const providerName = file.type || file.provider || "unknown";
      const source = file.runtime_only ? "runtime" : (file.source || "file");
      const stats = Number(file.success || 0) + " / " + Number(file.failed || 0);
      return '<tr><td><strong>' + esc(identity) + '</strong><small class="mono">' + esc(file.name || "") +
        '</small></td><td><span class="badge">' + esc(providerName) + '</span><small>' + esc(source) +
        '</small></td><td><span class="badge ' + status[1] + '">' + esc(status[0]) + '</span><small>' +
        esc(file.status_message || "") + '</small></td><td class="mono">' + esc(stats) +
        '</td><td>' + esc(formatTime(file.updated_at || file.modtime || file.last_refresh)) +
        '</td><td><button class="ghost" type="button" data-runtime="' + esc(file.auth_index || "") +
        '">运行时详情</button></td></tr>';
    }).join("");
    $("listWrap").className = "table-scroll";
    $("listWrap").innerHTML = '<table class="auth-table"><thead><tr><th>账号 / 文件</th><th>提供商</th><th>状态</th><th>成功 / 失败</th><th>更新时间</th><th>操作</th></tr></thead><tbody>' + rows + '</tbody></table>';
    $("listWrap").querySelectorAll("[data-runtime]").forEach((button) => {
      button.addEventListener("click", () => showRuntime(button.getAttribute("data-runtime")));
    });
  }

  async function refreshList() {
    if (!$("mgmtKey").value.trim()) {
      setKeyState("请输入 Management Key", "error");
      return;
    }
    try {
      const data = await request("GET", PLUGIN_ROOT + "/list", null, 20000);
      state.files = Array.isArray(data.files) ? data.files : [];
      setKeyState("Management API 已连接", "ready");
      renderAuthList();
    } catch (error) {
      state.files = [];
      renderAuthList();
      setKeyState("连接失败：" + (error.message || "请求失败"), "error");
    }
  }

  function renderDetail(file) {
    const status = authStatus(file);
    const recent = Array.isArray(file.recent_requests) && file.recent_requests.length
      ? '<div class="table-scroll" style="margin-top:14px"><table class="result-table"><thead><tr><th>时间</th><th>成功</th><th>失败</th></tr></thead><tbody>' +
        file.recent_requests.map((item) => '<tr><td>' + esc(item.time) + '</td><td>' + Number(item.success || 0) + '</td><td>' + Number(item.failed || 0) + '</td></tr>').join("") +
        '</tbody></table></div>' : '<p class="hint">暂无最近请求快照。</p>';
    $("detailBody").innerHTML = '<div class="detail-grid">' +
      '<div class="detail-item"><span>账号</span><strong>' + esc(file.email || file.label || file.name) + '</strong></div>' +
      '<div class="detail-item"><span>状态</span><strong><span class="badge ' + status[1] + '">' + esc(status[0]) + '</span></strong></div>' +
      '<div class="detail-item"><span>Auth Index</span><strong class="mono">' + esc(file.auth_index || "—") + '</strong></div>' +
      '<div class="detail-item"><span>提供商</span><strong>' + esc(file.type || file.provider || "—") + '</strong></div>' +
      '<div class="detail-item"><span>状态说明</span><strong>' + esc(file.status_message || "—") + '</strong></div>' +
      '<div class="detail-item"><span>最近刷新</span><strong>' + esc(formatTime(file.last_refresh)) + '</strong></div>' +
      '<div class="detail-item"><span>成功请求</span><strong>' + Number(file.success || 0) + '</strong></div>' +
      '<div class="detail-item"><span>失败请求</span><strong>' + Number(file.failed || 0) + '</strong></div>' +
      '</div>' + recent + '<p class="hint">此信息来自 CLIProxyAPI 的 <code>host.auth.get_runtime</code>，不会读取或展示物理凭证 JSON。</p>';
  }

  async function showRuntime(authIndex) {
    if (!authIndex) return;
    $("detailTitle").textContent = "运行时详情";
    $("detailBody").innerHTML = '<p class="hint">读取中…</p>';
    const dialog = $("detailDialog");
    if (!dialog.open) {
      if (typeof dialog.showModal === "function") dialog.showModal();
      else dialog.setAttribute("open", "open");
    }
    try {
      const data = await request("GET", PLUGIN_ROOT + "/auth-runtime?auth_index=" + encodeURIComponent(authIndex), null, 20000);
      renderDetail(data.auth || {});
    } catch (error) {
      $("detailBody").innerHTML = '<p class="detail-error">读取失败：' + esc(error.message || "请求失败") + '</p>';
    }
  }

  async function convert(importAfter) {
    if (!$("sso").value.trim()) {
      $("resultBox").className = "result-box empty";
      $("resultBox").textContent = "请先填写 SSO Cookie 或列表。";
      setStatus("缺少输入", "error");
      return;
    }
    setBusy(true);
    $("resultBox").className = "result-box empty";
    $("resultBox").textContent = importAfter ? "转换并导入中，Device Flow 可能需要数十秒…" : "转换中，Device Flow 可能需要数十秒…";
    setStatus("处理中…");
    try {
      const data = await request("POST", PLUGIN_ROOT + (importAfter ? "/convert-import" : "/convert"), conversionPayload(), 30 * 60 * 1000);
      renderBatch(data, importAfter);
      if (importAfter) await refreshList();
    } catch (error) {
      $("resultBox").className = "result-box empty detail-error";
      $("resultBox").textContent = "请求失败：" + (error.message || "未知错误");
      setStatus("操作失败", "error");
    } finally {
      setBusy(false);
    }
  }

  async function importJSON() {
    const raw = $("authJson").value.trim();
    if (!raw) {
      setStatus("请填写 JSON", "error");
      return;
    }
    let parsed;
    try { parsed = JSON.parse(raw); } catch (error) {
      setStatus("JSON 格式错误：" + error.message, "error");
      return;
    }
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      setStatus("JSON 必须是对象", "error");
      return;
    }
    if (parsed.type && String(parsed.type).toLowerCase() !== "xai") {
      setStatus("这个页面只导入 type=xai 的凭证", "error");
      return;
    }
    setBusy(true);
    setStatus("导入中…");
    try {
      const data = await request("POST", PLUGIN_ROOT + "/import", {
        name: $("authName").value.trim(),
        json: parsed
      }, 30000);
      $("authJson").value = "";
      $("authName").value = "";
      $("resultBox").className = "result-box";
      $("resultBox").textContent = "已导入：" + (data.name || "完成");
      setStatus("JSON 导入完成", "success");
      await refreshList();
    } catch (error) {
      setStatus("导入失败：" + (error.message || "未知错误"), "error");
    } finally {
      setBusy(false);
    }
  }

  $("mgmtKey").value = getKey();
  setKeyState($("mgmtKey").value ? "已读取本地保存的 Key" : "等待 Management Key");
  $("sso").addEventListener("input", updateInputHint);
  $("filter").addEventListener("input", renderAuthList);
  $("providerFilter").addEventListener("change", renderAuthList);
  $("btnSaveKey").addEventListener("click", async () => {
    const key = $("mgmtKey").value.trim();
    if (!key) { setKeyState("请输入 Management Key", "error"); return; }
    try { localStorage.setItem(STORAGE_KEY, key); } catch (_) {}
    setKeyState("已保存，连接中…", "ready");
    await refreshList();
  });
  $("btnClearKey").addEventListener("click", () => {
    try { localStorage.removeItem(STORAGE_KEY); } catch (_) {}
    $("mgmtKey").value = "";
    state.files = [];
    renderAuthList();
    setKeyState("已清除本页保存的 Key");
  });
  $("btnConvert").addEventListener("click", () => convert(false));
  $("btnConvertImport").addEventListener("click", () => convert(true));
  $("btnRefresh").addEventListener("click", refreshList);
  $("btnImport").addEventListener("click", importJSON);
  $("btnCloseDetail").addEventListener("click", () => $("detailDialog").close());
  $("detailDialog").addEventListener("click", (event) => {
    if (event.target === $("detailDialog")) $("detailDialog").close();
  });
  updateInputHint();
  if ($("mgmtKey").value) refreshList();
})();
</script>
</body>
</html>
`)
}
