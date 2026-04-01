package server

import "net/http"

const uiHTML = `<!DOCTYPE html><html lang="en"><head>
<meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Wrangler — Stockyard</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=Libre+Baskerville:ital,wght@0,400;0,700;1,400&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
<style>:root{
  --bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;
  --rust:#c45d2c;--rust-light:#e8753a;--rust-dark:#8b3d1a;
  --leather:#a0845c;--leather-light:#c4a87a;
  --cream:#f0e6d3;--cream-dim:#bfb5a3;--cream-muted:#7a7060;
  --gold:#d4a843;--green:#5ba86e;--red:#c0392b;--blue:#4a90d9;
  --font-serif:'Libre Baskerville',Georgia,serif;
  --font-mono:'JetBrains Mono',monospace;
}
*{margin:0;padding:0;box-sizing:border-box}
body{background:var(--bg);color:var(--cream);font-family:var(--font-serif);min-height:100vh;overflow-x:hidden}
a{color:var(--rust-light);text-decoration:none}a:hover{color:var(--gold)}
.hdr{background:var(--bg2);border-bottom:2px solid var(--rust-dark);padding:.9rem 1.8rem;display:flex;align-items:center;justify-content:space-between}
.hdr-left{display:flex;align-items:center;gap:1rem}
.hdr-brand{font-family:var(--font-mono);font-size:.75rem;color:var(--leather);letter-spacing:3px;text-transform:uppercase}
.hdr-title{font-family:var(--font-mono);font-size:1.1rem;color:var(--cream);letter-spacing:1px}
.badge{font-family:var(--font-mono);font-size:.6rem;padding:.2rem .6rem;letter-spacing:1px;text-transform:uppercase;border:1px solid}
.badge-free{color:var(--green);border-color:var(--green)}
.main{max-width:1000px;margin:0 auto;padding:2rem 1.5rem}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(120px,1fr));gap:1rem;margin-bottom:2rem}
.card{background:var(--bg2);border:1px solid var(--bg3);padding:1rem 1.2rem}
.card-val{font-family:var(--font-mono);font-size:1.6rem;font-weight:700;color:var(--cream);display:block}
.card-lbl{font-family:var(--font-mono);font-size:.6rem;letter-spacing:2px;text-transform:uppercase;color:var(--leather);margin-top:.2rem}
.section{margin-bottom:2.5rem}
.section-title{font-family:var(--font-mono);font-size:.68rem;letter-spacing:3px;text-transform:uppercase;color:var(--rust-light);margin-bottom:.8rem;padding-bottom:.5rem;border-bottom:1px solid var(--bg3)}
table{width:100%;border-collapse:collapse;font-family:var(--font-mono);font-size:.75rem}
th{background:var(--bg3);padding:.5rem .8rem;text-align:left;color:var(--leather-light);font-weight:400;letter-spacing:1px;font-size:.62rem;text-transform:uppercase}
td{padding:.5rem .8rem;border-bottom:1px solid var(--bg3);color:var(--cream-dim);vertical-align:top;word-break:break-all}
tr:hover td{background:var(--bg2)}
.empty{color:var(--cream-muted);text-align:center;padding:2rem;font-style:italic}
.btn{font-family:var(--font-mono);font-size:.7rem;padding:.3rem .8rem;border:1px solid var(--leather);background:transparent;color:var(--cream);cursor:pointer;transition:all .2s}
.btn:hover{border-color:var(--rust-light);color:var(--rust-light)}
.btn-rust{border-color:var(--rust);color:var(--rust-light)}.btn-rust:hover{background:var(--rust);color:var(--cream)}
.btn-sm{font-size:.62rem;padding:.2rem .5rem}
.pill{display:inline-block;font-family:var(--font-mono);font-size:.58rem;padding:.1rem .4rem;border-radius:2px;text-transform:uppercase}
.pill-pending{background:#2a2a1a;color:var(--gold)}.pill-running{background:#1a2a3a;color:var(--blue)}
.pill-done{background:#1a3a2a;color:var(--green)}.pill-failed{background:#2a1f1a;color:var(--rust-light)}
.pill-dead{background:#2a1a1a;color:var(--red)}.pill-cancelled{background:var(--bg3);color:var(--cream-muted)}
.lbl{font-family:var(--font-mono);font-size:.62rem;letter-spacing:1px;text-transform:uppercase;color:var(--leather)}
input{font-family:var(--font-mono);font-size:.78rem;background:var(--bg3);border:1px solid var(--bg3);color:var(--cream);padding:.4rem .7rem;outline:none}
input:focus{border-color:var(--leather)}
.row{display:flex;gap:.8rem;align-items:flex-end;flex-wrap:wrap;margin-bottom:1rem}
.field{display:flex;flex-direction:column;gap:.3rem}
.tabs{display:flex;gap:0;margin-bottom:1.5rem;border-bottom:1px solid var(--bg3)}
.tab{font-family:var(--font-mono);font-size:.72rem;padding:.6rem 1.2rem;color:var(--cream-muted);cursor:pointer;border-bottom:2px solid transparent;letter-spacing:1px;text-transform:uppercase}
.tab:hover{color:var(--cream-dim)}.tab.active{color:var(--rust-light);border-bottom-color:var(--rust-light)}
.tab-content{display:none}.tab-content.active{display:block}
pre{background:var(--bg3);padding:.8rem 1rem;font-family:var(--font-mono);font-size:.72rem;color:var(--cream-dim);overflow-x:auto}
</style></head><body>
<div class="hdr">
  <div class="hdr-left">
    <svg viewBox="0 0 64 64" width="22" height="22" fill="none"><rect x="8" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="28" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="48" y="8" width="8" height="48" rx="2.5" fill="#e8753a"/><rect x="8" y="27" width="48" height="7" rx="2.5" fill="#c4a87a"/></svg>
    <span class="hdr-brand">Stockyard</span>
    <span class="hdr-title">Wrangler</span>
  </div>
  <div style="display:flex;gap:.8rem;align-items:center">
    <span class="badge badge-free">Free</span>
    <a href="/api/status" class="lbl" style="color:var(--leather)">API</a>
  </div>
</div>
<div class="main">

<div class="cards">
  <div class="card"><span class="card-val" id="s-queues">—</span><span class="card-lbl">Queues</span></div>
  <div class="card"><span class="card-val" id="s-pending">—</span><span class="card-lbl">Pending</span></div>
  <div class="card"><span class="card-val" id="s-running">—</span><span class="card-lbl">Running</span></div>
  <div class="card"><span class="card-val" id="s-done">—</span><span class="card-lbl">Done</span></div>
  <div class="card"><span class="card-val" id="s-dead">—</span><span class="card-lbl">Dead</span></div>
</div>

<div class="tabs">
  <div class="tab active" onclick="switchTab('queues')">Queues</div>
  <div class="tab" onclick="switchTab('jobs')">Jobs</div>
  <div class="tab" onclick="switchTab('dlq')">DLQ</div>
  <div class="tab" onclick="switchTab('usage')">Usage</div>
</div>

<div id="tab-queues" class="tab-content active">
  <div class="section">
    <div class="section-title">Queues</div>
    <div class="row">
      <div class="field"><span class="lbl">Name</span><input id="q-name" placeholder="emails" style="width:200px"></div>
      <button class="btn btn-rust" onclick="createQueue()">Create Queue</button>
    </div>
    <table><thead><tr><th>Name</th><th>Pending</th><th>Running</th><th>Done</th><th>Dead</th><th></th></tr></thead>
    <tbody id="queues-body"></tbody></table>
  </div>
</div>

<div id="tab-jobs" class="tab-content">
  <div class="section">
    <div class="section-title">Jobs</div>
    <div id="jobs-list"></div>
  </div>
</div>

<div id="tab-dlq" class="tab-content">
  <div class="section">
    <div class="section-title">Dead Letter Queue</div>
    <table><thead><tr><th>ID</th><th>Queue</th><th>Callback</th><th>Attempts</th><th>Error</th><th></th></tr></thead>
    <tbody id="dlq-body"></tbody></table>
  </div>
</div>

<div id="tab-usage" class="tab-content">
  <div class="section">
    <div class="section-title">Quick Start</div>
    <pre>
# Create a queue
curl -X POST http://localhost:8810/api/queues \
  -H "Content-Type: application/json" \
  -d '{"name":"emails"}'

# Enqueue a job
curl -X POST http://localhost:8810/api/queues/{id}/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "callback_url": "http://localhost:3000/workers/send-email",
    "payload": {"user_id": 123, "template": "welcome"},
    "max_attempts": 3,
    "backoff_seconds": 60
  }'

# List jobs
curl http://localhost:8810/api/queues/{id}/jobs?status=pending

# Dead letter queue
curl http://localhost:8810/api/dlq

# Retry a dead job
curl -X POST http://localhost:8810/api/jobs/{id}/retry
    </pre>
  </div>
</div>

</div>
<script>
let queues=[];

function switchTab(n){
  document.querySelectorAll('.tab').forEach(t=>t.classList.toggle('active',t.textContent.toLowerCase()===n));
  document.querySelectorAll('.tab-content').forEach(t=>t.classList.toggle('active',t.id==='tab-'+n));
  if(n==='dlq')loadDLQ();
  if(n==='jobs')loadJobs();
}

async function refresh(){
  try{
    const r=await fetch('/api/status');const s=await r.json();
    document.getElementById('s-queues').textContent=s.queues||0;
    document.getElementById('s-pending').textContent=s.pending||0;
    document.getElementById('s-running').textContent=s.running||0;
    document.getElementById('s-done').textContent=fmt(s.done||0);
    document.getElementById('s-dead').textContent=s.dead||0;
  }catch(e){}
  try{
    const r=await fetch('/api/queues');const d=await r.json();
    queues=d.queues||[];
    const tb=document.getElementById('queues-body');
    if(!queues.length){tb.innerHTML='<tr><td colspan="6" class="empty">No queues yet.</td></tr>';return;}
    tb.innerHTML=queues.map(q=>
      '<tr><td style="color:var(--cream);font-weight:600">'+esc(q.name)+'<br><span style="font-size:.58rem;color:var(--cream-muted)">'+q.id+'</span></td>'+
      '<td>'+q.pending+'</td><td>'+q.running+'</td><td>'+q.done+'</td><td>'+q.dead+'</td>'+
      '<td><button class="btn btn-sm" onclick="deleteQueue(\''+q.id+'\')">Delete</button></td></tr>'
    ).join('');
  }catch(e){}
}

async function createQueue(){
  const name=document.getElementById('q-name').value.trim();
  if(!name)return;
  const r=await fetch('/api/queues',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name})});
  if(r.ok){document.getElementById('q-name').value='';refresh();}
}

async function deleteQueue(id){
  if(!confirm('Delete queue and all jobs?'))return;
  await fetch('/api/queues/'+id,{method:'DELETE'});
  refresh();
}

async function loadJobs(){
  let html='';
  for(const q of queues){
    const r=await fetch('/api/queues/'+q.id+'/jobs?limit=20');
    const d=await r.json();
    const jobs=d.jobs||[];
    html+='<div class="section-title" style="margin-top:1rem">'+esc(q.name)+'</div>';
    if(!jobs.length){html+='<div class="empty">No jobs</div>';continue;}
    html+='<table><thead><tr><th>ID</th><th>Status</th><th>Attempts</th><th>Callback</th><th>Created</th></tr></thead><tbody>';
    html+=jobs.map(j=>'<tr><td style="font-size:.65rem">'+j.id+'</td><td><span class="pill pill-'+j.status+'">'+j.status+'</span></td><td>'+j.attempts+'/'+j.max_attempts+'</td><td style="font-size:.65rem">'+esc(j.callback_url)+'</td><td style="font-size:.65rem">'+timeAgo(j.created_at)+'</td></tr>').join('');
    html+='</tbody></table>';
  }
  document.getElementById('jobs-list').innerHTML=html||'<div class="empty">No queues yet</div>';
}

async function loadDLQ(){
  const r=await fetch('/api/dlq');const d=await r.json();
  const jobs=d.dead_jobs||[];
  const tb=document.getElementById('dlq-body');
  if(!jobs.length){tb.innerHTML='<tr><td colspan="6" class="empty">No dead jobs</td></tr>';return;}
  tb.innerHTML=jobs.map(j=>
    '<tr><td style="font-size:.65rem">'+j.id+'</td><td style="font-size:.65rem">'+j.queue_id+'</td>'+
    '<td style="font-size:.65rem">'+esc(j.callback_url)+'</td><td>'+j.attempts+'/'+j.max_attempts+'</td>'+
    '<td style="font-size:.65rem;color:var(--red)">'+esc(j.last_error)+'</td>'+
    '<td><button class="btn btn-sm" onclick="retryJob(\''+j.id+'\')">Retry</button></td></tr>'
  ).join('');
}

async function retryJob(id){
  await fetch('/api/jobs/'+id+'/retry',{method:'POST'});
  loadDLQ();refresh();
}

function fmt(n){if(n>=1e6)return(n/1e6).toFixed(1)+'M';if(n>=1e3)return(n/1e3).toFixed(1)+'K';return n;}
function esc(s){const d=document.createElement('div');d.textContent=s||'';return d.innerHTML;}
function timeAgo(s){if(!s)return'—';const d=new Date(s);const diff=Date.now()-d.getTime();if(diff<60000)return'now';if(diff<3600000)return Math.floor(diff/60000)+'m';if(diff<86400000)return Math.floor(diff/3600000)+'h';return Math.floor(diff/86400000)+'d';}

refresh();
setInterval(refresh,8000);
</script></body></html>`

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(uiHTML))
}
