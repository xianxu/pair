import { Replayer, inBrowser } from './replayer.js';
import { currentFrame } from './render.js';
import { paintSketchPaths, projectSketchPoint } from './sketch-render.js';
import { pressureWidth } from './annotation-anchor.js';
const inside=(p,b)=>p[0]>=b[0]&&p[1]>=b[1]&&p[0]<=b[0]+b[2]&&p[1]<=b[1]+b[3];
const overlaps=(a,b)=>a.x<b.x+b.w&&b.x<a.x+a.w&&a.y<b.y+b.h&&b.y<a.y+a.h;
// Preflight every accepted centerline before drawing anything. Pressure fringe
// crossing an ordinary clip boundary is declared, rather than losing a path.
export function preparePageInk(plan,shapes,placement) {
 if(!shapes.length||shapes.length>8)throw Error('sketch page: shape count');
 let clipped=false;const badges=[];
 const result=shapes.map((s,i)=>{
  const fail=(why)=>{throw Error(`sketch page index ${s.index} ${s.id}: ${why}`);};
  if(s.index!==i+1)fail('index ordering');
  if(s.source.doc!==plan.scene.doc||s.source.tab!==plan.scene.tab||s.source.t>plan.scene.t)fail('source generation');
  if(!s.paths.length)fail('missing paths');
  const paths=s.paths.map(p=>{
   const g=placement(p,s);if(!g)fail(`path ${p.id} unavailable or ambiguous`);
   const projected=p.pts.map(point=>projectSketchPoint(point,p.at.box,g.box));
   if(!projected.length)fail(`path ${p.id} empty`);
   for(const point of projected){
    if(!point||!point.every(Number.isFinite)||!inside(point,plan.region)||!inside(point,g.clip))fail(`path ${p.id} outside visible viewport`);
    const radius=pressureWidth(p.width,point[2])/2;
    for(const b of [plan.region,g.clip])if(point[0]-radius<b[0]||point[1]-radius<b[1]||point[0]+radius>b[0]+b[2]||point[1]+radius>b[1]+b[3])clipped=true;
   }
   return {...p,projected,target:g.box,clip:g.clip};
  });
  const p=paths[0].projected[0],size=20;
  if(plan.width<size||plan.height<size)fail('badge does not fit');
  // Nine fixed nearby candidates, at most eight prior-badge comparisons each.
  // Clamping near an edge can collapse candidates; never retry duplicates.
  const x=(p[0]-plan.region[0])*plan.scale,y=(p[1]-plan.region[1])*plan.scale-size-4;
  const gap=size+4,seen=new Set();let badge;
  for(const [dx,dy] of [[0,0],[1,0],[-1,0],[0,1],[1,1],[-1,1],[0,-1],[1,-1],[-1,-1]]){
   const candidate={x:Math.max(0,Math.min(plan.width-size,x+dx*gap)),y:Math.max(0,Math.min(plan.height-size,y+dy*gap)),w:size,h:size};
   const key=`${candidate.x},${candidate.y}`;if(seen.has(key))continue;seen.add(key);
   if(!badges.some(b=>overlaps(b,candidate))){badge=candidate;break;}
  }
  if(!badge)fail('badge has no unobstructed nearby position');
  badges.push(badge);return {...s,paths,badge};
 });
 return {shapes:result,clipped};
}
async function start(){
 const responses=await Promise.all(['records','scene'].map(url=>fetch(url,{cache:'no-store'})));
 if(responses.some(r=>!r.ok))throw Error('historical page unavailable');
 const records=(await responses[0].text()).split('\n').filter(Boolean).map(JSON.parse);
 const {plan,shapes,coverage}=await responses[1].json();
 const frame=document.getElementById('page');
 const replayer=new Replayer(records,hash=>`assets/${hash}`,inBrowser(frame));
 await replayer.replayAt(plan.scene.t,0);
 const ink=preparePageInk(plan,shapes,(p,s)=>replayer.renderer.placementOf({at:p.at,source:s.source},{withClip:true}));
 const drawn=currentFrame(frame);drawn.style.transform=`scale(${plan.scale})`;
 const canvas=document.getElementById('ink');canvas.width=plan.width;canvas.height=plan.height;
 const g=canvas.getContext('2d'),toClient=(x,y)=>[(x-plan.region[0])*plan.scale,(y-plan.region[1])*plan.scale];
 for(const s of ink.shapes){
  for(const p of s.paths){
   const [x,y]=toClient(p.clip[0],p.clip[1]);g.save();g.beginPath();g.rect(x,y,p.clip[2]*plan.scale,p.clip[3]*plan.scale);g.clip();
   paintSketchPaths(g,[{...p,color:s.color}],{scale:plan.scale,toClient,project:()=>p.target});g.restore();
  }
  const b=s.badge;g.fillStyle='#fff';g.strokeStyle='#111';g.lineWidth=2;g.beginPath();g.arc(b.x+10,b.y+10,9,0,Math.PI*2);g.fill();g.stroke();g.fillStyle='#111';g.font='bold 13px sans-serif';g.textAlign='center';g.textBaseline='middle';g.fillText(String(s.index),b.x+10,b.y+10);
 }
 await new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done)));
 return {coverage,clipped:ink.clipped};
}
if(typeof window!=='undefined')window.sketchPageImage={ready:start().catch(error=>({error:String(error?.message||error)}))};
