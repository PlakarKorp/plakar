package procmon

import (
	"context"
	"encoding/json"
	"html"
	"net/http"
	"sync"
)

// ===== SSE hub and HTTP server =====

type sseMsg struct {
	Event string
	Data  any
}

type sseHub struct {
	mu         sync.RWMutex
	register   chan chan sseMsg
	unregister chan chan sseMsg
	clients    map[chan sseMsg]struct{}
	closed     chan struct{}
}

func newHub() *sseHub {
	return &sseHub{
		register:   make(chan chan sseMsg),
		unregister: make(chan chan sseMsg),
		clients:    make(map[chan sseMsg]struct{}),
		closed:     make(chan struct{}),
	}
}

func (h *sseHub) run() {
	for {
		select {
		case ch := <-h.register:
			h.clients[ch] = struct{}{}
		case ch := <-h.unregister:
			delete(h.clients, ch)
			close(ch)
		case <-h.closed:
			for ch := range h.clients {
				close(ch)
				delete(h.clients, ch)
			}
			return
		}
	}
}

func (h *sseHub) broadcast(ev string, data any) {
	msg := sseMsg{Event: ev, Data: data}
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
			// slow or stuck client — drop
		}
	}
}

var hub *sseHub // set when StartHTTP is called

// StartHTTP starts the live UI and SSE on addr (e.g. ":8080").
// It returns stopHTTP() that gracefully shuts the server down.
func StartHTTP(addr, base, title string) (func(context.Context) error, error) {
	hub = newHub()
	go hub.run()

	mux := http.NewServeMux()

	// UI at "/"
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(minUIHTML(base, title)))
	})

	// Samples
	mux.HandleFunc("/samples", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Samples())
	})

	// Markers
	mux.HandleFunc("/marks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Markers())
	})

	// SSE stream
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		ch := make(chan sseMsg, 16)
		hub.register <- ch
		defer func() { hub.unregister <- ch }()

		// initial hello
		_, _ = w.Write([]byte("event: hello\ndata: {\"ok\":true}\n\n"))
		flusher.Flush()

		notify := r.Context().Done()
		for {
			select {
			case msg := <-ch:
				b, _ := json.Marshal(msg.Data)
				_, _ = w.Write([]byte("event: " + msg.Event + "\n"))
				_, _ = w.Write([]byte("data: " + string(b) + "\n\n"))
				flusher.Flush()
			case <-notify:
				return
			}
		}
	})

	srv := &http.Server{Addr: addr, Handler: mux}

	// start
	go func() {
		_ = srv.ListenAndServe()
	}()

	// stop func
	stop := func(ctx context.Context) error {
		close(hub.closed)
		return srv.Shutdown(ctx)
	}
	return stop, nil
}
func minUIHTML(base, title string) string {
	tHTML := html.EscapeString(title)
	return `<!doctype html>
<html lang="en"><head>
<meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/>
<title>` + tHTML + `</title>
<style>
 body{font-family:system-ui,-apple-system,Segoe UI,Roboto,sans-serif;margin:16px}
 #chartWrap{width:100%;height:420px}
 canvas{width:100%!important;height:100%!important;display:block}
 .row{display:flex;gap:8px;align-items:center;margin-bottom:8px;flex-wrap:wrap}
 .badge{display:inline-block;padding:4px 8px;border-radius:999px;background:#eee;font-variant-numeric:tabular-nums}
 .section{margin-top:6px}
</style>
</head><body>
<h2>` + tHTML + `</h2>

<div class="row section">
  <span class="badge" id="status">connecting…</span>
</div>

<div class="row section">
  <strong>CPU:</strong>
  <span class="badge" id="cpu-min">min: –</span>
  <span class="badge" id="cpu-max">max: –</span>
  <span class="badge" id="cpu-avg">avg: –</span>
  <span class="badge" id="cpu-med">med: –</span>
</div>

<div class="row section">
  <strong>RAM:</strong>
  <span class="badge" id="ram-min">min: –</span>
  <span class="badge" id="ram-max">max: –</span>
  <span class="badge" id="ram-avg">avg: –</span>
  <span class="badge" id="ram-med">med: –</span>
</div>

<div id="chartWrap"><canvas id="chart"></canvas></div>

<script>const BASE='` + base + `';</script>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.4/dist/chart.umd.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/chartjs-plugin-annotation@3.0.1/dist/chartjs-plugin-annotation.min.js"></script>
<script>
Chart.register(window['chartjs-plugin-annotation']);

const chart = new Chart(document.getElementById('chart').getContext('2d'), {
  type:'line',
  data:{datasets:[
    {label:'CPU % (total)', data:[], parsing:false, pointRadius:0, tension:0, yAxisID:'yCPU'},
    {label:'RAM (MiB)',     data:[], parsing:false, pointRadius:0, tension:0, yAxisID:'yRAM'}
  ]},
  options:{
    responsive:true, maintainAspectRatio:false, animation:false,
    layout:{ padding:{top:24} }, // room for labels
    plugins:{ legend:{display:true}, annotation:{ annotations:{} } },
    scales:{
      x:{type:'linear', title:{display:true,text:'Elapsed (s)'}},
      yCPU:{position:'left',  title:{display:true,text:'CPU %'}},
      yRAM:{position:'right', title:{display:true,text:'MiB'}, grid:{drawOnChartArea:false}}
    },
    interaction:{mode:'nearest', intersect:false}
  }
});

// ----- stats helpers -----
const state = {
  cpu: [],  // floats
  ram: [],  // floats (MiB)
};

function format1(n){ return (Math.round(n*10)/10).toFixed(1); }
function computeStats(arr){
  if(arr.length===0) return null;
  let sum=0, min=arr[0], max=arr[0];
  for(const v of arr){ sum+=v; if(v<min) min=v; if(v>max) max=v; }
  const avg = sum/arr.length;
  // median: sort copy
  const tmp = arr.slice().sort((a,b)=>a-b);
  const mid = Math.floor(tmp.length/2);
  const med = (tmp.length%2)? tmp[mid] : (tmp[mid-1]+tmp[mid])/2;
  return {min, max, avg, med};
}
function updateBadges(){
  const cpuStats = computeStats(state.cpu);
  const ramStats = computeStats(state.ram);
  if(cpuStats){
    document.getElementById('cpu-min').textContent = 'min: ' + format1(cpuStats.min) + '%';
    document.getElementById('cpu-max').textContent = 'max: ' + format1(cpuStats.max) + '%';
    document.getElementById('cpu-avg').textContent = 'avg: ' + format1(cpuStats.avg) + '%';
    document.getElementById('cpu-med').textContent = 'med: ' + format1(cpuStats.med) + '%';
  }
  if(ramStats){
    document.getElementById('ram-min').textContent = 'min: ' + format1(ramStats.min) + ' MiB';
    document.getElementById('ram-max').textContent = 'max: ' + format1(ramStats.max) + ' MiB';
    document.getElementById('ram-avg').textContent = 'avg: ' + format1(ramStats.avg) + ' MiB';
    document.getElementById('ram-med').textContent = 'med: ' + format1(ramStats.med) + ' MiB';
  }
}

// ----- samples / markers -----
let t0=null;
const statusEl = document.getElementById('status');

function addSample(s){
  const ts = new Date(s.TS).getTime();
  if(t0===null) t0 = ts;
  const x  = (ts - t0)/1000;
  const cpu = s.CPUPercent;
  const mib = s.RSSBytes/1048576;

  chart.data.datasets[0].data.push({x:x,y:cpu});
  chart.data.datasets[1].data.push({x:x,y:mib});

  state.cpu.push(cpu);
  state.ram.push(mib);
}

const markers = []; // {id,label,x,color}
function addMarker(m){
  const ts = new Date(m.TS || m.time).getTime();
  if(t0===null) return; // wait until we know t0
  const x = (ts - t0)/1000;
  const color = (m.Color || m.color || '#ff6f00');
  markers.push({id: (m.id || markers.length+1), label: m.Label || m.label || '', x, color});
  redrawMarkers();
}

function redrawMarkers(){
  const anns = {};
  const scale = chart.scales.x;
  const minDx = 12;           // min pixel gap to consider overlapping
  const lineW = 1;
  const baseYOffset = -12;
  const step = 14;

  const entries = markers.map(m => ({...m, pixelX: scale.getPixelForValue(m.x)}))
                         .sort((a,b)=>a.pixelX-b.pixelX);

  const layers = [];
  for(const e of entries){
    let placed=false;
    for(const stack of layers){
      const last = stack[stack.length-1];
      if(Math.abs(e.pixelX - last.pixelX) <= minDx){ stack.push(e); placed=true; break; }
    }
    if(!placed) layers.push([e]);
  }
  let side = 1;
  for(const stack of layers){
    stack.forEach((e, i) => {
      const yAdjust = baseYOffset - i*step * side;
      anns['m_'+e.id] = {
        type: 'line',
        xMin: e.x, xMax: e.x,
        borderColor: e.color, borderWidth: lineW,
        label: {
          display: !!e.label,
          content: '• ' + e.label,
          position: 'start',
          yAdjust: yAdjust,
          backgroundColor: 'rgba(0,0,0,0.6)',
          color: '#fff', padding: 2, borderRadius: 3, font: { size: 10 }
        },
        drawTime: 'afterDatasetsDraw'
      };
    });
    side *= -1;
  }
  chart.options.plugins.annotation.annotations = anns;
}

// ----- initial load -----
fetch(BASE + '/samples').then(r=>{
  if(!r.ok) throw new Error('HTTP '+r.status);
  return r.json();
}).then(arr=>{
  if(arr.length>0){
    t0 = new Date(arr[0].TS).getTime();
  }
  arr.forEach(addSample);
  updateBadges();
  chart.update();
  statusEl.textContent = 'loaded history';
}).catch(err=>{
  statusEl.textContent = 'history error';
  console.error('samples fetch failed:', err);
});

fetch(BASE + '/marks').then(r=>r.json()).then(ms=>{
  ms.forEach(addMarker);
  chart.update();
});

// ----- live -----
const es = new EventSource(BASE + '/events');
es.onopen = () => statusEl.textContent = 'live';
es.onerror = () => statusEl.textContent = 'disconnected';

es.addEventListener('sample', ev=>{
  addSample(JSON.parse(ev.data));
  updateBadges();
  chart.update();
});

es.addEventListener('mark', ev=>{
  addMarker(JSON.parse(ev.data));
  chart.update();
});
</script>
</body></html>`
}
