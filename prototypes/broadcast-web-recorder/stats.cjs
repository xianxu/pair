const fs=require('fs'),zlib=require('zlib');
for (const r of process.argv.slice(2)) {
  const m=JSON.parse(fs.readFileSync(r+'/meta.json'));const out={run:r};
  if (m.A){const raw=fs.readFileSync(r+'/rrweb.json');const ev=JSON.parse(raw);
    out.A={evps:+(ev.length/m.elapsed).toFixed(0),KBpm:+(raw.length/m.elapsed*60/1024).toFixed(0),gzKBpm:+(zlib.gzipSync(raw).length/m.elapsed*60/1024).toFixed(0),
      firstFull:(ev.find(e=>e.type===2)||{}).timestamp-m.navStart, fullBytes:ev.filter(e=>e.type===2).map(e=>JSON.stringify(e).length)};}
  if (m.B){const raw=fs.readFileSync(r+'/cdp.jsonl');const L=raw.toString().trim().split('\n').map(JSON.parse);
    const by={};for(const l of L){by[l.type]=(by[l.type]||0)+JSON.stringify(l).length}
    const snaps=L.filter(l=>l.type==='snapshot'&&!l.s&&l.root.documentURL!=='about:blank');
    out.B={evps:+(L.length/m.elapsed).toFixed(0),KBpm:+(raw.length/m.elapsed*60/1024).toFixed(0),gzKBpm:+(zlib.gzipSync(raw).length/m.elapsed*60/1024).toFixed(0),
      firstSnap:snaps.length?snaps[0].t-m.navStart:null, topBytesByType:Object.entries(by).sort((a,b)=>b[1]-a[1]).slice(0,6).map(([k,v])=>k+':'+(v/1024|0)+'K').join(' '),
      assetsKB:(m.B.assets.bytes/1024|0)};}
  console.log(JSON.stringify(out));
}
