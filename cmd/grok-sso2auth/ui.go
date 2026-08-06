package main

func uiHTML() []byte {
	return []byte(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<meta name="color-scheme" content="light dark"/>
<title>Grok SSO 导入</title>
<script>
(function(){
  "use strict";
  const STORAGE_KEY="cli-proxy-theme";
  const TOKENS=[
    "--bg-secondary","--bg-primary","--bg-tertiary","--bg-hover","--bg-quinary",
    "--floating-surface","--floating-shadow","--text-primary","--text-secondary","--text-tertiary",
    "--border-color","--border-primary","--border-hover","--primary-color","--primary-hover",
    "--primary-active","--primary-contrast","--success-color","--success-badge-bg",
    "--success-badge-text","--success-badge-border","--failure-badge-bg","--failure-badge-text",
    "--failure-badge-border","--amber-color","--amber-text","--amber-10","--amber-30","--shadow"
  ];
  const root=document.documentElement;
  const systemDark=()=>Boolean(window.matchMedia&&window.matchMedia("(prefers-color-scheme: dark)").matches);
  function storedPreference(){
    try{
      const raw=localStorage.getItem(STORAGE_KEY);
      if(!raw)return "auto";
      const value=JSON.parse(raw);
      return String(value&&value.state&&value.state.theme||value&&value.theme||value||"auto");
    }catch(_){return "auto";}
  }
  function resolvePreference(preference){
    if(preference==="dark"||preference==="white")return preference;
    if(preference==="light")return "";
    return systemDark()?"dark":"white";
  }
  function fallbackTheme(){return resolvePreference(storedPreference());}
  function themeBackground(applied){return applied==="dark"?"#151412":applied==="white"?"#ffffff":"#faf9f5";}
  function parentTheme(){
    if(window.parent===window)return null;
    try{
      const parentRoot=window.parent.document.documentElement;
      const styles=window.parent.getComputedStyle(parentRoot);
      const current=parentRoot.getAttribute("data-theme")||"";
      const anticipated=resolvePreference(storedPreference());
      const applied=current||(anticipated||"");
      const parentReady=current===applied;
      if(parentReady){
        for(const token of TOKENS){
          const value=styles.getPropertyValue(token).trim();
          if(value)root.style.setProperty(token,value);
        }
      }
      return {applied:applied,background:parentReady?styles.getPropertyValue("--bg-secondary").trim():themeBackground(applied),root:parentRoot};
    }catch(_){return null;}
  }
  function bootstrapBackground(applied,inherited){
    const background=(inherited&&inherited.background)||themeBackground(applied);
    root.style.backgroundColor=background;
    try{
      if(window.frameElement){
        window.frameElement.style.backgroundColor=background;
        window.frameElement.style.colorScheme=applied==="dark"?"dark":"light";
      }
    }catch(_){}
  }
  function sync(){
    const inherited=parentTheme();
    const applied=inherited?inherited.applied:fallbackTheme();
    if(applied==="dark"||applied==="white")root.setAttribute("data-theme",applied);
    else root.removeAttribute("data-theme");
    root.style.colorScheme=applied==="dark"?"dark":"light";
    bootstrapBackground(applied,inherited);
    root.setAttribute("data-theme-source",inherited?"parent":"local");
    return {applied:applied||"light",source:inherited?"parent":"local",parentRoot:inherited&&inherited.root};
  }
  window.__cpaThemeBridge={storageKey:STORAGE_KEY,sync:sync};
  const initial=sync();
  if(initial&&initial.parentRoot){
    try{new MutationObserver(()=>sync()).observe(initial.parentRoot,{attributes:true,attributeFilter:["data-theme"]});}catch(_){}
  }
  window.addEventListener("storage",event=>{if(event.key===STORAGE_KEY)sync();});
  if(window.matchMedia){
    const media=window.matchMedia("(prefers-color-scheme: dark)");
    media.addEventListener&&media.addEventListener("change",()=>sync());
  }
})();
</script>
<style>
  :root {
    color-scheme:light;
    --bg-secondary:#faf9f5; --bg-primary:#f0eee8; --bg-tertiary:#e9e6df; --bg-hover:var(--bg-tertiary); --bg-quinary:#f6f4ee;
    --floating-surface:#fffdf9; --floating-shadow:0 12px 26px rgba(0,0,0,.14);
    --text-primary:#2d2a26; --text-secondary:#6d6760; --text-tertiary:#a29c95;
    --border-color:#e3e1db; --border-primary:#d5d2cb; --border-hover:#cecac4;
    --primary-color:#8b8680; --primary-hover:#7f7a74; --primary-active:#726d67; --primary-contrast:#fff;
    --success-color:#10b981; --success-badge-bg:#d1fae5; --success-badge-text:#065f46; --success-badge-border:#6ee7b7;
    --failure-badge-bg:rgba(198,87,70,.14); --failure-badge-text:#8a3a30; --failure-badge-border:rgba(198,87,70,.35);
    --amber-color:#d97706; --amber-text:#92400e; --amber-10:color-mix(in srgb,var(--amber-color) 10%,transparent); --amber-30:color-mix(in srgb,var(--amber-color) 30%,transparent);
    --shadow:0 1px 2px 0 rgb(0 0 0/.08);
    --bg:var(--bg-secondary); --surface:var(--bg-primary); --surface-soft:var(--bg-secondary); --surface-strong:var(--floating-surface);
    --border:var(--border-color); --border-strong:var(--border-primary); --text:var(--text-primary); --muted:var(--text-secondary);
    --subtle:var(--text-tertiary); --accent:var(--primary-color); --accent-hover:var(--primary-hover);
    --success:var(--success-badge-text); --success-bg:var(--success-badge-bg); --success-border:var(--success-badge-border);
    --warning:var(--amber-text); --warning-bg:var(--amber-10); --warning-border:var(--amber-30);
    --danger:var(--failure-badge-text); --danger-bg:var(--failure-badge-bg); --danger-border:var(--failure-badge-border);
    --sans:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"PingFang SC","Noto Sans SC",sans-serif;
    --mono:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;
  }
  [data-theme="white"]{
    --bg-secondary:#fff; --bg-primary:#fff; --bg-tertiary:#f6f6f6; --bg-hover:var(--bg-tertiary); --bg-quinary:#fff;
    --floating-surface:#fff; --floating-shadow:0 12px 26px rgba(0,0,0,.12);
    --border-color:#e5e5e5; --border-primary:#d9d9d9; --border-hover:#ccc;
  }
  [data-theme="dark"]{
    color-scheme:dark;
    --bg-secondary:#151412; --bg-primary:#1d1b18; --bg-tertiary:#262320; --bg-hover:#2e2a26; --bg-quinary:#191714;
    --floating-surface:#2a2723; --floating-shadow:0 14px 30px rgba(0,0,0,.4);
    --text-primary:#f6f4f1; --text-secondary:#c9c3bb; --text-tertiary:#9c958d;
    --border-color:#3a3530; --border-primary:#4a453f; --border-hover:#5a544d;
    --primary-color:#8b8680; --primary-hover:#9a948e; --primary-active:#a6a099; --primary-contrast:#fff;
    --success-badge-bg:rgba(6,78,59,.3); --success-badge-text:#6ee7b7; --success-badge-border:#059669;
    --failure-badge-bg:rgba(198,87,70,.24); --failure-badge-text:#f1b0a6; --failure-badge-border:rgba(198,87,70,.5);
    --amber-color:#f59e0b; --amber-text:#fcd34d; --amber-10:color-mix(in srgb,var(--amber-color) 14%,transparent); --amber-30:color-mix(in srgb,var(--amber-color) 38%,transparent);
    --shadow:0 1px 3px 0 rgb(0 0 0/.3);
  }
  *{box-sizing:border-box} html{background:var(--bg)} body{margin:0;min-height:100vh;background:var(--bg);color:var(--text);font:14px/1.55 var(--sans)}
  [hidden]{display:none!important}
  main{width:min(1000px,100%);margin:0 auto;padding:32px clamp(18px,4vw,48px) 48px}
  .hero{display:flex;justify-content:space-between;align-items:flex-start;gap:28px;margin-bottom:24px}
  .eyebrow{margin:0 0 7px;color:var(--muted);font-size:12px;font-weight:650;letter-spacing:.09em;text-transform:uppercase}
  h1{margin:0;font-size:clamp(30px,4vw,40px);line-height:1.13;letter-spacing:-.04em}
  .lead{max-width:680px;margin:10px 0 0;color:var(--muted)}
  .session{display:inline-flex;align-items:center;gap:7px;min-height:30px;padding:5px 10px;border:1px solid var(--border);border-radius:999px;background:var(--surface-soft);color:var(--muted);font-size:12px;white-space:nowrap}
  .session::before{content:"";width:7px;height:7px;border-radius:50%;background:currentColor}
  .session.ready{color:var(--success);border-color:var(--success-border);background:var(--success-bg)}
  .session.warn{color:var(--warning);border-color:var(--warning-border);background:var(--warning-bg)}
  .card{margin-bottom:16px;padding:24px;border:1px solid var(--border);border-radius:12px;background:var(--surface);box-shadow:var(--shadow)}
  .card-head{display:flex;justify-content:space-between;align-items:flex-start;gap:16px;margin-bottom:18px}
  .title-row{display:flex;align-items:center;gap:10px}.step{display:grid;place-items:center;width:25px;height:25px;border-radius:50%;background:var(--text);color:var(--bg);font-size:12px;font-weight:700}
  h2{margin:0;font-size:18px;line-height:1.4}.card-head p{margin:2px 0 0;color:var(--muted);font-size:13px}
  label{display:block;margin-bottom:6px;font-weight:650}.required{color:var(--danger)}
  textarea,input{width:100%;border:1px solid var(--border);border-radius:8px;background:var(--surface-soft);color:var(--text);font:inherit;outline:0;transition:.15s border-color,.15s box-shadow,.15s background}
  textarea{min-height:190px;padding:12px 14px;resize:vertical;font:13px/1.6 var(--mono)} input{height:40px;padding:8px 11px}
  textarea:focus,input:focus{border-color:var(--accent);background:var(--surface-strong);box-shadow:0 0 0 3px color-mix(in srgb,var(--accent) 18%,transparent)}
  .field-meta{display:flex;justify-content:space-between;gap:16px;margin-top:8px;color:var(--muted);font-size:12px}.field-error{color:var(--danger)}
  .format{margin-top:16px;padding:12px 14px;border:1px solid var(--border);border-radius:8px;background:var(--surface-soft);color:var(--muted);font-size:12px}
  code{font:12px var(--mono);color:var(--text)}
  details{margin-top:16px;border:1px solid var(--border);border-radius:9px;background:var(--surface-soft);overflow:hidden}
  summary{padding:12px 14px;cursor:pointer;font-weight:650;user-select:none}details[open] summary{border-bottom:1px solid var(--border)}
  .advanced{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:14px;padding:16px}
  .advanced small{display:block;margin-top:5px;color:var(--muted);line-height:1.4}
  .toggle-row{grid-column:1/-1;display:flex;align-items:center;gap:9px}.toggle-row input{width:16px;height:16px;margin:0;accent-color:var(--accent)}.toggle-row label{margin:0;font-weight:550;color:var(--muted)}
  .submit-row{display:flex;align-items:center;justify-content:space-between;gap:18px;margin-top:18px}
  .estimate{color:var(--muted);font-size:13px}.estimate strong{color:var(--text)}
  button{display:inline-flex;align-items:center;justify-content:center;min-height:40px;padding:9px 16px;border:1px solid var(--accent);border-radius:8px;background:var(--accent);color:var(--primary-contrast);font:600 13px var(--sans);cursor:pointer;transition:.15s background,.15s transform}
  button:hover:not(:disabled){background:var(--accent-hover)}button:active:not(:disabled){transform:scale(.98)}button:disabled{opacity:.5;cursor:not-allowed}
  button.busy::before{content:"";width:14px;height:14px;margin-right:8px;border:2px solid currentColor;border-right-color:transparent;border-radius:50%;animation:spin .75s linear infinite}
  .notice{display:none;margin-bottom:16px;padding:12px 14px;border:1px solid var(--warning-border);border-radius:9px;background:var(--warning-bg);color:var(--warning);font-size:13px}.notice.show{display:block}
  .notice strong{display:block;margin-bottom:2px}.notice a{color:inherit}
  .progress{display:none;align-items:center;gap:12px;margin-bottom:16px;padding:12px 14px;border:1px solid var(--border);border-radius:9px;background:var(--surface-soft)}.progress.show{display:flex}
  .spinner{width:18px;height:18px;border:2px solid var(--border-strong);border-right-color:var(--accent);border-radius:50%;animation:spin .8s linear infinite}.progress strong{display:block}.progress small{color:var(--muted)}
  .summary{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px;margin-bottom:16px}.metric{padding:12px 14px;border:1px solid var(--border);border-radius:8px;background:var(--surface-soft)}
  .metric span{display:block;color:var(--muted);font-size:12px}.metric strong{display:block;margin-top:2px;font-size:23px;font-variant-numeric:tabular-nums}.metric.ok strong{color:var(--success)}.metric.fail strong{color:var(--danger)}
  .result-placeholder{padding:27px 16px;border:1px dashed var(--border-strong);border-radius:9px;background:var(--surface-soft);color:var(--muted);text-align:center}
  .table-wrap{display:none;overflow:auto;border:1px solid var(--border);border-radius:9px;background:var(--surface-strong)}.table-wrap.show{display:block}
  table{width:100%;border-collapse:collapse;font-size:13px}th,td{padding:10px 12px;border-bottom:1px solid var(--border);text-align:left;vertical-align:top}th{background:var(--bg-tertiary);color:var(--muted);font-size:12px;font-weight:550;white-space:nowrap}tr:last-child td{border-bottom:0}
  .badge{display:inline-flex;padding:3px 8px;border:1px solid transparent;border-radius:999px;font-size:12px;white-space:nowrap}.badge.ok{border-color:var(--success-border);background:var(--success-bg);color:var(--success)}.badge.fail{border-color:var(--danger-border);background:var(--danger-bg);color:var(--danger)}.badge.rate{margin-left:5px;border-color:var(--warning-border);background:var(--warning-bg);color:var(--warning)}
  .account{max-width:230px;overflow-wrap:anywhere}.error{max-width:330px;color:var(--danger);overflow-wrap:anywhere}.muted{color:var(--muted)}
  .footer{margin-top:18px;color:var(--subtle);font-size:12px}.footer a{color:var(--muted)}
  @keyframes spin{to{transform:rotate(360deg)}}
  @media(prefers-reduced-motion:reduce){button{transition:none}.spinner,button.busy::before{animation:none}}
  @media(max-width:720px){main{padding:22px 14px 36px}.hero{flex-direction:column;gap:14px}.card{padding:18px}.card-head{flex-direction:column}.advanced{grid-template-columns:repeat(2,minmax(0,1fr))}.submit-row{align-items:stretch;flex-direction:column}button{width:100%}.summary{grid-template-columns:repeat(2,minmax(0,1fr))}}
  @media(max-width:420px){.advanced{grid-template-columns:1fr}.summary{grid-template-columns:1fr}.field-meta{flex-direction:column;gap:2px}}
</style>
</head>
<body>
<main>
  <header class="hero">
    <div>
      <p class="eyebrow">CLIProxyAPI · Plugin Management</p>
      <h1>Grok SSO 导入</h1>
      <p class="lead">把 Grok / xAI SSO Cookie 转换为 xAI OAuth 凭证，并直接写入 CLIProxyAPI。无需离开管理中心，也无需再次输入 Management Key。</p>
    </div>
    <span id="sessionBadge" class="session">正在读取管理会话</span>
  </header>

  <div id="authNotice" class="notice" role="alert">
    <strong>未找到可复用的管理认证</strong>
    请返回管理中心重新登录并勾选“记住密码”，再重新打开此页面。插件不会要求你重复输入 Management Key。
  </div>

  <section class="card">
    <div class="card-head">
      <div>
        <div class="title-row"><span class="step">1</span><h2>粘贴 SSO</h2></div>
        <p>支持单个 Cookie，或一行一个账号的批量列表。</p>
      </div>
    </div>
    <label for="ssoInput">SSO Cookie / 批量列表 <span class="required">*</span></label>
    <textarea id="ssoInput" spellcheck="false" autocomplete="off" placeholder="eyJhbGciOi...&#10;&#10;或：&#10;name@example.com----eyJhbGciOi..."></textarea>
    <div class="field-meta"><span id="inputMessage">尚未添加账号</span><span>空行和以 # 开头的行会被忽略</span></div>
    <div class="format">支持格式：<code>sso</code>、<code>email----sso</code>、<code>email----password----sso</code>。密码字段只用于兼容列表格式，不会被读取或上传。</div>

    <details>
      <summary>高级设置</summary>
      <div class="advanced">
        <div><label for="baseDelay">基础间隔（秒）</label><input id="baseDelay" type="number" min="1" max="600" step="1" value="45"/><small>账号之间的起始等待</small></div>
        <div><label for="maxDelay">最大间隔（秒）</label><input id="maxDelay" type="number" min="30" max="900" step="1" value="180"/><small>限流时的间隔上限</small></div>
        <div><label for="stageRetries">阶段重试</label><input id="stageRetries" type="number" min="1" max="20" step="1" value="8"/><small>Device / Verify / Approve</small></div>
        <div><label for="accountRetries">账号级尝试</label><input id="accountRetries" type="number" min="1" max="10" step="1" value="3"/><small>仅限流时重跑整个账号</small></div>
        <div class="toggle-row"><input id="validateSSO" type="checkbox" checked/><label for="validateSSO">转换前验证 SSO 是否仍然有效（推荐）</label></div>
      </div>
    </details>

    <div class="submit-row">
      <div id="estimate" class="estimate">添加账号后显示预计批量等待时间</div>
      <button id="startButton" type="button" disabled>开始转换并导入</button>
    </div>
  </section>

  <section class="card" aria-live="polite">
    <div class="card-head">
      <div>
        <div class="title-row"><span class="step">2</span><h2>导入结果</h2></div>
        <p id="resultCaption">完成后会显示每个账号的凭证文件名与状态。</p>
      </div>
    </div>
    <div id="progress" class="progress"><span class="spinner"></span><div><strong>正在转换并导入</strong><small id="elapsed">已运行 0 秒，请勿关闭页面</small></div></div>
    <div id="summary" class="summary" hidden>
      <div class="metric"><span>账号总数</span><strong id="totalMetric">0</strong></div>
      <div class="metric ok"><span>成功导入</span><strong id="okMetric">0</strong></div>
      <div class="metric fail"><span>失败</span><strong id="failMetric">0</strong></div>
      <div class="metric"><span>最终间隔</span><strong id="delayMetric">—</strong></div>
    </div>
    <div id="placeholder" class="result-placeholder">还没有运行记录</div>
    <div id="tableWrap" class="table-wrap">
      <table><thead><tr><th>#</th><th>状态</th><th>账号</th><th>凭证文件</th><th>尝试</th><th>说明</th></tr></thead><tbody id="resultBody"></tbody></table>
    </div>
  </section>

  <p class="footer">SSO 与 OAuth Token 只在当前页面、本机 CLIProxyAPI 和 xAI 认证服务之间传输。请勿把管理中心暴露给不可信网络。 · <a href="https://help.router-for.me/cn/plugin/development.html" target="_blank" rel="noreferrer">插件开发文档 ↗</a></p>
</main>
<script>
(function(){
  "use strict";
  const API_URL="/v0/management/plugins/grok-sso2auth/convert-import";
  const AUTH_STORAGE_KEY="cli-proxy-auth";
  const ENCRYPTED_PREFIX="enc::v1::";
  const LEGACY_KEYS=["managementKey","grok_sso2auth.management_key","grok_sso2auth_mgmt_key"];
  const state={managementKey:"",busy:false,timer:0,startedAt:0};
  const $=id=>document.getElementById(id);

  function escapeHTML(value){return String(value==null?"":value).replace(/[&<>"']/g,c=>({"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"}[c]));}
  function decodeProtected(raw){
    if(!raw||!raw.startsWith(ENCRYPTED_PREFIX))return raw||"";
    try{
      const encoded=raw.slice(ENCRYPTED_PREFIX.length);
      const bytes=Uint8Array.from(atob(encoded),c=>c.charCodeAt(0));
      const key=new TextEncoder().encode("cli-proxy-api-webui::secure-storage|"+location.host+"|"+navigator.userAgent);
      for(let i=0;i<bytes.length;i++)bytes[i]^=key[i%key.length];
      return new TextDecoder().decode(bytes);
    }catch(_){return "";}
  }
  function keyFromValue(raw){
    const decoded=decodeProtected((raw||"").trim());
    if(!decoded)return "";
    try{
      const value=JSON.parse(decoded);
      if(typeof value==="string")return value.trim();
      return String(value&&value.state&&value.state.managementKey||value&&value.managementKey||"").trim();
    }catch(_){return decoded.trim();}
  }
  function loadManagementKey(){
    try{
      const primary=keyFromValue(localStorage.getItem(AUTH_STORAGE_KEY));
      if(primary)return primary;
      for(const name of LEGACY_KEYS){const key=keyFromValue(localStorage.getItem(name));if(key)return key;}
    }catch(_){}
    return "";
  }
  function setSession(ready){
    $("sessionBadge").textContent=ready?"已复用管理中心认证":"管理认证不可用";
    $("sessionBadge").className="session "+(ready?"ready":"warn");
    $("authNotice").classList.toggle("show",!ready);
    updateButton();
  }
  function parsedLines(){return $("ssoInput").value.split(/\r?\n/).map(v=>v.trim()).filter(v=>v&&!v.startsWith("#"));}
  function numberValue(id,fallback,min,max){const n=Number($(id).value);return Number.isFinite(n)?Math.min(max,Math.max(min,n)):fallback;}
  function updateInput(){
    const count=parsedLines().length;
    $("inputMessage").textContent=count?"已识别 "+count+" 个账号":"尚未添加账号";
    $("inputMessage").className=count?"":"field-error";
    const base=numberValue("baseDelay",45,1,600);
    const minimum=Math.max(0,count-1)*base;
    $("estimate").innerHTML=count?"<strong>"+count+" 个账号</strong> · 批量间最低等待约 "+formatDuration(minimum):"添加账号后显示预计批量等待时间";
    updateButton();
  }
  function updateButton(){$("startButton").disabled=state.busy||!state.managementKey||parsedLines().length===0;}
  function formatDuration(seconds){seconds=Math.round(seconds);if(seconds<60)return seconds+" 秒";const m=Math.floor(seconds/60),s=seconds%60;return m+" 分"+(s?" "+s+" 秒":"");}
  function setBusy(busy){
    state.busy=busy;$("startButton").classList.toggle("busy",busy);$("startButton").textContent=busy?"正在处理":"开始转换并导入";
    $("progress").classList.toggle("show",busy);
    if(state.timer){clearInterval(state.timer);state.timer=0;}
    if(busy){state.startedAt=Date.now();state.timer=setInterval(()=>{$("elapsed").textContent="已运行 "+formatDuration((Date.now()-state.startedAt)/1000)+"，请勿关闭页面";},1000);}
    updateButton();
  }
  function renderResult(data){
    const items=Array.isArray(data.items)?data.items:[];
    $("summary").hidden=false;$("placeholder").hidden=true;$("tableWrap").classList.add("show");
    $("totalMetric").textContent=data.total||items.length;$("okMetric").textContent=data.ok||0;$("failMetric").textContent=data.fail||0;
    $("delayMetric").textContent=Number.isFinite(Number(data.final_delay_sec))?Math.round(Number(data.final_delay_sec))+"s":"—";
    $("resultCaption").textContent=(data.ok||0)+" / "+(data.total||items.length)+" 个账号已成功导入。";
    $("resultBody").innerHTML=items.map(item=>{
      const status=item.ok?'<span class="badge ok">成功</span>':'<span class="badge fail">失败</span>';
      const rate=item.rate_limited?'<span class="badge rate">遇到限流</span>':'';
      return '<tr><td>'+escapeHTML(item.index)+'</td><td>'+status+rate+'</td><td class="account">'+escapeHTML(item.email||"—")+'</td><td><code>'+escapeHTML(item.file_name||"—")+'</code></td><td>'+escapeHTML(item.attempts||1)+'</td><td class="'+(item.ok?'muted':'error')+'">'+escapeHTML(item.error||"已写入 auth-dir")+'</td></tr>';
    }).join("");
  }
  function renderRequestError(message){
    $("summary").hidden=true;$("tableWrap").classList.remove("show");$("placeholder").hidden=false;
    $("placeholder").className="result-placeholder field-error";$("placeholder").textContent=message;
    $("resultCaption").textContent="请求未完成，请按提示检查后重试。";
  }
  async function start(){
    if(state.busy)return;
    state.managementKey=loadManagementKey();setSession(Boolean(state.managementKey));
    if(!state.managementKey)return;
    const lines=parsedLines();if(!lines.length){updateInput();return;}
    const payload={
      sso:$("ssoInput").value,validate_sso:$("validateSSO").checked,
      base_delay_sec:numberValue("baseDelay",45,1,600),max_delay_sec:numberValue("maxDelay",180,30,900),
      max_retries:Math.round(numberValue("stageRetries",8,1,20)),account_retries:Math.round(numberValue("accountRetries",3,1,10))
    };
    setBusy(true);$("placeholder").className="result-placeholder";
    const controller=new AbortController();
    const timeout=Math.max(180000,lines.length*120000+Math.max(0,lines.length-1)*payload.max_delay_sec*1000);
    const timer=setTimeout(()=>controller.abort(),timeout);
    try{
      const response=await fetch(API_URL,{method:"POST",headers:{"Authorization":"Bearer "+state.managementKey,"Content-Type":"application/json"},body:JSON.stringify(payload),signal:controller.signal});
      let data={};try{data=await response.json();}catch(_){}
      if(response.status===401||response.status===403){setSession(false);throw new Error("管理认证已失效。请返回管理中心重新登录并勾选“记住密码”。");}
      if(!response.ok&&!(data&&Array.isArray(data.items)))throw new Error(data.error||("请求失败（HTTP "+response.status+"）"));
      renderResult(data);
    }catch(error){renderRequestError(error&&error.name==="AbortError"?"处理超时。可减少账号数量后重试。":(error.message||String(error)));}
    finally{clearTimeout(timer);setBusy(false);}
  }
  $("ssoInput").addEventListener("input",updateInput);
  ["baseDelay","maxDelay","stageRetries","accountRetries"].forEach(id=>$(id).addEventListener("input",updateInput));
  $("startButton").addEventListener("click",start);
  state.managementKey=loadManagementKey();setSession(Boolean(state.managementKey));updateInput();
})();
</script>
</body>
</html>`)
}
