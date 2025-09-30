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
 .row{display:flex;gap:12px;align-items:center;margin-bottom:8px;flex-wrap:wrap}
 .badge{display:inline-block;padding:4px 8px;border-radius:999px;background:#eee}
</style>
</head><body>
<h2>` + tHTML + `</h2>
<div class="row">
  <span class="badge" id="peakcpu">peak CPU: –</span>
  <span class="badge" id="peakram">peak RAM: –</span>
  <span class="badge" id="status">connecting…</span>
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
    plugins:{
      legend:{display:true},
      annotation:{ annotations:{} }
    },
    scales:{
      x:{type:'linear', title:{display:true,text:'Elapsed (s)'}},
      yCPU:{position:'left',  title:{display:true,text:'CPU %'}},
      yRAM:{position:'right', title:{display:true,text:'MiB'}, grid:{drawOnChartArea:false}}
    },
    interaction:{mode:'nearest', intersect:false}
  }
});

let peakCPU=0, peakMiB=0, t0=null;
const statusEl = document.getElementById('status');

function addSample(s){
  const ts = new Date(s.TS).getTime();
  if(t0===null) t0 = ts;
  const x  = (ts - t0)/1000;
  const cpu = s.CPUPercent;
  const mib = s.RSSBytes/1048576;
  chart.data.datasets[0].data.push({x:x,y:cpu});
  chart.data.datasets[1].data.push({x:x,y:mib});
  if(cpu>peakCPU){peakCPU=cpu; document.getElementById('peakcpu').textContent='peak CPU: '+peakCPU.toFixed(1)+'%';}
  if(mib>peakMiB){peakMiB=mib; document.getElementById('peakram').textContent='peak RAM: '+peakMiB.toFixed(1)+' MiB';}
}

const markers = []; // {id,label,x,color}
function addMarker(m){
  const ts = new Date(m.TS || m.time).getTime();
  if(t0===null) return; // history will set t0 first
  const x = (ts - t0)/1000;
  const color = (m.Color || m.color || '#ff6f00');
  markers.push({id: (m.id || markers.length+1), label: m.Label || m.label || '', x, color});
  redrawMarkers();
}

// Build non-overlapping vertical line labels:
// - convert each marker x to pixel
// - sort by pixel x
// - stack labels when closer than minDx
function redrawMarkers(){
  const anns = {};
  const scale = chart.scales.x;
  const minDx = 12;           // min pixel gap to consider overlapping
  const lineW = 1;            // line width
  const baseYOffset = -12;    // label offset from top grid
  const step = 14;            // vertical spacing between stacked labels

  // prepare entries with pixelX
  const entries = markers.map(m => ({
    ...m,
    pixelX: scale.getPixelForValue(m.x)
  })).sort((a,b)=>a.pixelX-b.pixelX);

  // stack groups
  const layers = []; // array of arrays; each inner array is a stack at roughly same x
  for(const e of entries){
    let placed = false;
    for(const stack of layers){
      // if far enough from last in this stack, create new stack; else add to same stack
      const last = stack[stack.length-1];
      if (Math.abs(e.pixelX - last.pixelX) <= minDx){
        stack.push(e); placed = true; break;
      }
    }
    if(!placed) layers.push([e]);
  }

  // assign yAdjust per stack (0, -step, -2*step, …), alternate sides to reduce clutter
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
          color: '#fff',
          padding: 2,
          borderRadius: 3,
          font: { size: 10 }
        },
        drawTime: 'afterDatasetsDraw'
      };
    });
    side *= -1; // alternate stacking direction
  }

  chart.options.plugins.annotation.annotations = anns;
}

fetch(BASE + '/samples').then(r=>{
  if(!r.ok) throw new Error('HTTP '+r.status);
  return r.json();
}).then(arr=>{
  if(arr.length>0){
    // establish t0 based on first sample TS – elapsed(0)
    const firstTs = new Date(arr[0].TS).getTime();
    t0 = firstTs;
  }
  arr.forEach(addSample);
  chart.update();
  statusEl.textContent = 'loaded history';
}).catch(err=>{
  statusEl.textContent = 'history error';
  console.error('samples fetch failed:', err);
});

// load existing markers, then draw
fetch(BASE + '/marks').then(r=>r.json()).then(ms=>{
  ms.forEach(addMarker);
  chart.update();
});

const es = new EventSource(BASE + '/events');
es.onopen = () => statusEl.textContent = 'live';
es.onerror = () => statusEl.textContent = 'disconnected';
es.addEventListener('sample', ev=>{
  addSample(JSON.parse(ev.data));
  chart.update();
});
es.addEventListener('mark', ev=>{
  addMarker(JSON.parse(ev.data));
  chart.update();
});
</script>
</body></html>`
}
