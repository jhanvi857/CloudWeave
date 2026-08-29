'use client'

import { useMemo, useState } from 'react'

const concepts = [
  ['01', 'Chunking', 'FastCDC turns objects into addressable pieces. Each chunk gets a content identity before it leaves the client.'],
  ['02', 'Placement', 'A virtual hash ring chooses replicas with minimal movement when nodes join or leave.'],
  ['03', 'Quorum', 'N, W, and R make durability and availability explicit instead of hiding them behind defaults.'],
  ['04', 'Recovery', 'Heartbeats mark failures. Repair workers copy missing replicas until the target is restored.'],
]

const flow = [
  ['Client', 'PUT /files/report.csv', 'The SDK streams the object into the coordinator.'],
  ['Chunker', 'FastCDC + SHA-256', 'The payload becomes independently addressable chunks.'],
  ['Ring', 'N = 3 replicas', 'Consistent hashing selects distinct healthy nodes.'],
  ['Quorum', 'W = 2 acknowledgements', 'The write commits after enough replicas confirm storage.'],
]

function FlowDemo() {
  const [step, setStep] = useState(0)
  const [healthy, setHealthy] = useState(true)
  const current = flow[step]
  return (
    <div className="demo-panel">
      <div className="demo-head"><div><span className="eyebrow">Interactive model</span><h3>Follow one object through the cluster</h3></div><span className={healthy ? 'status status-ok' : 'status status-warn'}>{healthy ? '3 nodes healthy' : '2 nodes healthy'}</span></div>
      <div className="flow-track">{flow.map((item, index) => <button type="button" key={item[0]} className={index === step ? 'flow-node active' : 'flow-node'} onClick={() => setStep(index)}><span className="node-number">{String(index + 1).padStart(2, '0')}</span><strong>{item[0]}</strong><small>{item[1]}</small></button>)}</div>
      <div className="demo-output"><div><span className="eyebrow">Current event</span><strong>{current[1]}</strong><p>{current[2]}</p></div><code>event_{String(step + 1).padStart(2, '0')} / {healthy ? 'ack_received' : 'repair_pending'}</code></div>
      <div className="demo-actions"><button type="button" className="button button-dark" onClick={() => setStep((step + 1) % flow.length)}>Next step</button><button type="button" className="button button-quiet" onClick={() => setHealthy(!healthy)}>{healthy ? 'Simulate node loss' : 'Restore node'}</button></div>
    </div>
  )
}

function QuorumDemo() {
  const [w, setW] = useState(2)
  const [r, setR] = useState(2)
  const overlap = w + r > 3
  return <div className="quorum-card"><div className="eyebrow">Quorum calculator</div><h3>Make consistency visible</h3><p className="muted">Tune the acknowledgement threshold and read overlap for a three-node cluster.</p><div className="sliders"><label>Write quorum <output>W = {w}</output><input aria-label="Write quorum" type="range" min="1" max="3" value={w} onChange={(event) => setW(Number(event.target.value))} /></label><label>Read quorum <output>R = {r}</output><input aria-label="Read quorum" type="range" min="1" max="3" value={r} onChange={(event) => setR(Number(event.target.value))} /></label></div><div className={overlap ? 'callout callout-good' : 'callout callout-warn'}><span className="callout-dot" />{overlap ? 'Read and write sets overlap. A successful read sees the newest committed value.' : 'Low overlap favors availability, but a read can miss the latest replica.'}</div></div>
}

export default function Home() {
  const [query, setQuery] = useState('')
  const visibleConcepts = useMemo(() => concepts.filter((item) => item.join(' ').toLowerCase().includes(query.toLowerCase())), [query])
  return <main id="top">
    <header className="site-header"><a className="brand" href="#top"><span className="brand-mark">CW</span><span>CloudWeave</span><span className="brand-sub">documentation</span></a><nav><a href="#concepts">Concepts</a><a href="#playground">Playground</a><a href="#reference">Reference</a><a className="button button-outline" href="http://localhost:8080/dashboard">Open live dashboard <span aria-hidden="true">{'->'}</span></a></nav><a className="mobile-live" href="http://localhost:8080/dashboard">Live dashboard {'->'}</a></header>
    <div className="docs-layout"><aside className="sidebar"><div className="sidebar-inner"><label className="search"><span aria-hidden="true">⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search docs" aria-label="Search documentation" /></label><div className="side-group"><span>Get started</span><a className="selected" href="#introduction">Introduction</a><a href="#quickstart">Quickstart</a></div><div className="side-group"><span>Core concepts</span><a href="#concepts">Chunking and hashes</a><a href="#concepts">Consistent hashing</a><a href="#playground">Quorum and repair</a></div><div className="side-group"><span>Reference</span><a href="#architecture">Architecture</a><a href="#reference">API and CLI</a><a href="#benchmarks">Benchmarks</a></div><div className="sidebar-footer"><span className="live-dot" />System preview<br /><small>Run the node dashboard locally</small></div></div></aside><div className="content">
      <section id="introduction" className="hero"><div className="hero-copy"><span className="eyebrow blue">Distributed object storage</span><h1>See distributed storage <em>in action.</em></h1><p className="lead">CloudWeave is a compact object store built to make the hard parts observable: placement, quorum, failure, repair, and the path from bytes to durable replicas.</p><div className="hero-actions"><a className="button button-dark" href="#playground">Explore the playground <span aria-hidden="true">v</span></a><a className="text-link" href="http://localhost:8080/dashboard">View live cluster <span aria-hidden="true">{'->'}</span></a></div></div><div className="hero-map" aria-label="Object flow diagram"><div className="map-label">REQUEST PATH / PUT OBJECT</div><div className="map-line"><span className="map-box">client<br /><small>SDK</small></span><i>{'->'}</i><span className="map-box map-box-blue">object<br /><small>stream</small></span><i>{'->'}</i><span className="map-box">chunks<br /><small>SHA-256</small></span><i>{'->'}</i><span className="map-box map-box-green">nodes<br /><small>N = 3</small></span></div><div className="map-foot"><code>report.csv</code><span>W = 2 acknowledged</span></div></div></section>
      <section id="concepts" className="section"><div className="section-heading"><div><span className="eyebrow">The system, in four ideas</span><h2>Small pieces. Clear guarantees.</h2></div><p>Read the overview, then use the playground to change the system and see what follows.</p></div><div className="concept-grid">{visibleConcepts.map((item) => <article className="concept" key={item[0]}><span className="concept-index">{item[0]}</span><h3>{item[1]}</h3><p>{item[2]}</p><a href="#playground">Explore concept <span aria-hidden="true">{'->'}</span></a></article>)}</div></section>
      <section id="playground" className="section section-muted"><div className="section-heading"><div><span className="eyebrow">Explore the mechanics</span><h2>A cluster you can reason about.</h2></div><p>These controls are local documentation simulations. For live node health, open the existing dashboard.</p></div><div className="demo-grid"><FlowDemo /><QuorumDemo /></div></section>
      <section id="architecture" className="section architecture"><div className="architecture-copy"><span className="eyebrow">Architecture</span><h2>One request, several deliberate boundaries.</h2><p>Clients speak object semantics. The coordinator handles placement and quorum. Nodes keep the storage path intentionally simple.</p><a className="text-link" href="#reference">Read the API reference <span aria-hidden="true">{'->'}</span></a></div><div className="stack-list">{['HTTP API', 'Coordinator', 'Consistent hash ring', 'Node transport', 'Disk store + LRU'].map((item, index) => <div className="stack-row" key={item}><span>{String(index + 1).padStart(2, '0')}</span><strong>{item}</strong><code>{index === 0 ? 'PUT /files/:key' : index === 1 ? 'W / R quorum' : index === 2 ? 'virtual nodes' : index === 3 ? 'pooled HTTP' : 'content-addressed'}</code></div>)}</div></section>
      <section id="reference" className="section reference"><span className="eyebrow">Reference</span><h2>Build with the primitives.</h2><div className="reference-grid"><a href="#quickstart"><strong>Quickstart</strong><span>Start a local cluster and upload an object.</span><b>{'->'}</b></a><a href="#reference"><strong>HTTP API and CLI</strong><span>Endpoints, flags, and the cweave command.</span><b>{'->'}</b></a><a id="benchmarks" href="#benchmarks"><strong>Benchmarks</strong><span>Throughput measurements and methodology.</span><b>{'->'}</b></a></div></section>
      <footer><span className="brand"><span className="brand-mark">CW</span>CloudWeave</span><span>Distributed object storage, explained.</span><a href="http://localhost:8080/dashboard">Live system preview {'->'}</a></footer>
    </div></div>
  </main>
}
