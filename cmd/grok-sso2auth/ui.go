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
    --bg: #0f1419;
    --panel: #1a2332;
    --border: #2d3a4d;
    --text: #e7ecf3;
    --muted: #8b9bb4;
    --accent: #3b82f6;
    --accent2: #22c55e;
    --danger: #ef4444;
    --warn: #f59e0b;
    --mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    --sans: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "PingFang SC", "Noto Sans SC", sans-serif;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; padding: 0;
    font-family: var(--sans);
    background: radial-gradient(1200px 600px at 10% -10%, #1e3a5f 0%, transparent 50%),
                radial-gradient(900px 500px at 100% 0%, #14213d 0%, transparent 40%),
                var(--bg);
    color: var(--text);
    min-height: 100vh;
  }
  main { max-width: 920px; margin: 0 auto; padding: 2rem 1.25rem 4rem; }
  h1 { font-size: 1.55rem; margin: 0 0 .35rem; letter-spacing: -.02em; }
  .sub { color: var(--muted); margin-bottom: 1.5rem; font-size: .95rem; line-height: 1.5; }
  .card {
    background: color-mix(in srgb, var(--panel) 92%, black);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 1.15rem 1.25rem;
    margin-bottom: 1rem;
    box-shadow: 0 8px 30px rgba(0,0,0,.25);
  }
  label { display: block; font-size: .82rem; color: var(--muted); margin-bottom: .4rem; }
  textarea, input[type=text], input[type=number], input[type=password] {
    width: 100%;
    background: #0c1118;
    color: var(--text);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: .7rem .8rem;
    font-family: var(--mono);
    font-size: .85rem;
    outline: none;
  }
  textarea { min-height: 140px; resize: vertical; line-height: 1.45; }
  textarea:focus, input:focus { border-color: var(--accent); box-shadow: 0 0 0 3px rgba(59,130,246,.2); }
  .row { display: flex; gap: .75rem; flex-wrap: wrap; margin-top: .85rem; }
  .row > * { flex: 1; min-width: 140px; }
  .checks { display: flex; gap: 1rem; flex-wrap: wrap; margin-top: .85rem; color: var(--muted); font-size: .88rem; }
  .checks label { display: flex; align-items: center; gap: .4rem; margin: 0; color: var(--text); font-size: .88rem; }
  button {
    appearance: none; border: 0; border-radius: 10px;
    padding: .7rem 1.1rem; font-weight: 600; font-size: .92rem;
    cursor: pointer; transition: transform .05s ease, opacity .15s;
  }
  button:active { transform: translateY(1px); }
  button:disabled { opacity: .55; cursor: not-allowed; }
  .btn-primary { background: linear-gradient(180deg, #4f8ff7, var(--accent)); color: white; }
  .btn-ok { background: linear-gradient(180deg, #34d399, var(--accent2)); color: #052e16; }
  .btn-ghost { background: transparent; color: var(--text); border: 1px solid var(--border); }
  .actions { display: flex; gap: .6rem; flex-wrap: wrap; margin-top: 1rem; }
  pre {
    background: #0c1118; border: 1px solid var(--border); border-radius: 10px;
    padding: .9rem 1rem; overflow: auto; max-height: 420px;
    font-family: var(--mono); font-size: .78rem; line-height: 1.45; white-space: pre-wrap;
  }
  .ok { color: #4ade80; }
  .err { color: #f87171; }
  .pill {
    display: inline-block; padding: .15rem .55rem; border-radius: 999px;
    background: #132033; border: 1px solid var(--border); color: var(--muted);
    font-size: .75rem; margin-right: .35rem;
  }
  .hint { color: var(--muted); font-size: .8rem; margin-top: .5rem; line-height: 1.45; }
  a { color: #93c5fd; }
  .key-row { display: flex; gap: .5rem; align-items: end; }
  .key-row > div { flex: 1; }
  table { width: 100%; border-collapse: collapse; font-size: .85rem; }
  th, td { text-align: left; padding: .45rem .4rem; border-bottom: 1px solid var(--border); }
  th { color: var(--muted); font-weight: 600; font-size: .75rem; text-transform: uppercase; letter-spacing: .04em; }
</style>
</head>
<body>
<main>
  <h1>Grok SSO → CLIProxyAPI</h1>
  <p class="sub">
    将 xAI / Grok 的 <code>sso</code> Cookie 经 Device Flow 换成官方 <span class="pill">type=xai</span><span class="pill">auth_kind=oauth</span>
    凭证，并通过 <code>host.auth.save</code> 写入 CLIProxyAPI 的 auth-dir。
  </p>

  <div class="card">
    <div class="key-row">
      <div>
        <label for="mgmtKey">Management Key（调用 /v0/management 需要）</label>
        <input id="mgmtKey" type="password" placeholder="自动尝试读取管理台已保存的 key，也可手动填写" autocomplete="off"/>
      </div>
      <button class="btn-ghost" type="button" id="btnSaveKey">记住</button>
    </div>
    <p class="hint">同源管理台通常把 key 存在 localStorage。本页会尝试常见键名；跨源部署时请手动填写。</p>
  </div>

  <div class="card">
    <label for="sso">SSO Cookie / 列表</label>
    <textarea id="sso" placeholder="单行 JWT，或每行：&#10;email----password----sso&#10;email----sso&#10;eyJhbGciOi..."></textarea>
    <div class="row">
      <div>
        <label for="email">Email 覆盖（可选）</label>
        <input id="email" type="text" placeholder="user@example.com"/>
      </div>
      <div>
        <label for="delay">批量间隔秒（默认 45）</label>
        <input id="delay" type="number" min="0" step="1" value="45"/>
      </div>
    </div>
    <div class="checks">
      <label><input type="checkbox" id="validate"/> 预先在线验证 SSO</label>
    </div>
    <div class="actions">
      <button class="btn-primary" type="button" id="btnConvert">仅转换</button>
      <button class="btn-ok" type="button" id="btnConvertImport">转换并导入</button>
      <button class="btn-ghost" type="button" id="btnList">刷新账号列表</button>
    </div>
    <p class="hint">敏感操作走 <code>POST /v0/management/plugins/grok-sso2auth/*</code>（需 Management Key）。页面本身在 resource 路径，不承载写操作。</p>
  </div>

  <div class="card">
    <label>结果</label>
    <pre id="out">等待操作…</pre>
  </div>

  <div class="card">
    <label>当前 Auth 文件（摘要）</label>
    <div id="listWrap"><p class="hint">点击「刷新账号列表」</p></div>
  </div>
</main>
<script>
(function () {
  const KEYS = [
    "managementKey", "management_key", "managementPassword", "management_password",
    "apiPassword", "api_password", "cliproxy_management_key", "cpa_management_key", "password"
  ];
  const out = document.getElementById("out");
  const listWrap = document.getElementById("listWrap");
  const keyInput = document.getElementById("mgmtKey");

  function loadKey() {
    try {
      const saved = localStorage.getItem("grok_sso2auth_mgmt_key");
      if (saved) { keyInput.value = saved; return; }
      for (const k of KEYS) {
        const v = localStorage.getItem(k);
        if (v && String(v).trim()) { keyInput.value = String(v).trim(); return; }
      }
      // some UIs nest JSON
      for (const k of Object.keys(localStorage)) {
        const raw = localStorage.getItem(k);
        if (!raw || raw.length > 4000) continue;
        try {
          const obj = JSON.parse(raw);
          if (obj && typeof obj === "object") {
            for (const kk of KEYS) {
              if (obj[kk]) { keyInput.value = String(obj[kk]); return; }
            }
          }
        } catch (_) {}
      }
    } catch (_) {}
  }

  document.getElementById("btnSaveKey").onclick = () => {
    try {
      localStorage.setItem("grok_sso2auth_mgmt_key", keyInput.value.trim());
      toast("已保存到 localStorage");
    } catch (e) {
      toast("保存失败: " + e);
    }
  };

  function toast(msg) {
    out.textContent = msg;
  }

  function mgmtHeaders() {
    const h = { "Content-Type": "application/json" };
    const key = keyInput.value.trim();
    if (key) h["Authorization"] = "Bearer " + key;
    return h;
  }

  async function mgmt(method, path, body) {
    const url = "/v0/management" + path;
    const resp = await fetch(url, {
      method,
      headers: mgmtHeaders(),
      body: body == null ? undefined : JSON.stringify(body),
    });
    const text = await resp.text();
    let data;
    try { data = JSON.parse(text); } catch (_) { data = { raw: text }; }
    if (!resp.ok) {
      const err = new Error((data && (data.error || data.message)) || ("HTTP " + resp.status));
      err.status = resp.status;
      err.data = data;
      throw err;
    }
    return data;
  }

  function payload() {
    return {
      sso: document.getElementById("sso").value,
      email: document.getElementById("email").value.trim(),
      delay_sec: Number(document.getElementById("delay").value || 0),
      validate_sso: document.getElementById("validate").checked,
    };
  }

  function setBusy(busy) {
    ["btnConvert", "btnConvertImport", "btnList"].forEach(id => {
      document.getElementById(id).disabled = !!busy;
    });
  }

  function renderBatch(data) {
    const lines = [];
    lines.push("合计: " + (data.ok || 0) + "/" + (data.total || 0) + " 成功" + (data.fail ? ("，失败 " + data.fail) : ""));
    (data.items || []).forEach(it => {
      if (it.ok) {
        lines.push("✔ #" + it.index + " " + (it.email || "") + " → " + (it.file_name || "") + (it.path ? (" @ " + it.path) : ""));
      } else {
        lines.push("✘ #" + it.index + " " + (it.email || "") + " " + (it.error || "failed"));
      }
    });
    // keep full JSON under the summary for power users
    lines.push("");
    lines.push(JSON.stringify(data, null, 2));
    out.textContent = lines.join("\\n");
    out.className = (data.fail && data.ok === 0) ? "err" : "";
  }

  document.getElementById("btnConvert").onclick = async () => {
    setBusy(true);
    out.textContent = "转换中…（device flow 可能需要数十秒/账号）";
    try {
      const data = await mgmt("POST", "/plugins/grok-sso2auth/convert", payload());
      renderBatch(data);
    } catch (e) {
      out.className = "err";
      out.textContent = "转换失败: " + e.message + "\\n" + JSON.stringify(e.data || {}, null, 2);
    } finally { setBusy(false); }
  };

  document.getElementById("btnConvertImport").onclick = async () => {
    setBusy(true);
    out.textContent = "转换并导入中…";
    try {
      const data = await mgmt("POST", "/plugins/grok-sso2auth/convert-import", payload());
      renderBatch(data);
      try { await refreshList(); } catch (_) {}
    } catch (e) {
      out.className = "err";
      out.textContent = "导入失败: " + e.message + "\\n" + JSON.stringify(e.data || {}, null, 2);
    } finally { setBusy(false); }
  };

  async function refreshList() {
    const data = await mgmt("GET", "/plugins/grok-sso2auth/list");
    const files = data.files || [];
    if (!files.length) {
      listWrap.innerHTML = "<p class='hint'>暂无 auth 文件</p>";
      return;
    }
    const rows = files.map(f => "<tr><td>" + esc(f.name || "") + "</td><td>" + esc(f.type || f.provider || "") +
      "</td><td>" + esc(f.email || f.label || "") + "</td><td>" + esc(f.status || "") + "</td></tr>").join("");
    listWrap.innerHTML = "<table><thead><tr><th>Name</th><th>Type</th><th>Email</th><th>Status</th></tr></thead><tbody>" +
      rows + "</tbody></table><p class='hint'>共 " + files.length + " 条</p>";
  }

  document.getElementById("btnList").onclick = async () => {
    setBusy(true);
    try {
      await refreshList();
      out.className = "";
      out.textContent = "列表已刷新";
    } catch (e) {
      out.className = "err";
      out.textContent = "列表失败: " + e.message;
    } finally { setBusy(false); }
  };

  function esc(s) {
    return String(s).replace(/[&<>"']/g, c => ({ "&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;" }[c]));
  }

  loadKey();
})();
</script>
</body>
</html>
`)
}
