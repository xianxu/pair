export class Element {
  constructor(tag) {
    this.tagName = tag; this.children = []; this.listeners = {}; this.dataset = {};
    this.style = { setProperty() {} }; this.value = ''; this.className = '';
    this.classList = { toggle: (v,on) => { const set = new Set(this.className.split(' ')); on ? set.add(v) : set.delete(v); this.className = [...set].join(' '); }, add: (v) => this.classList.toggle(v,true), remove: (v) => this.classList.toggle(v,false), contains: (v) => this.className.split(' ').includes(v) };
  }
  hasPointerCapture(id) { return this.captured===id; }
  setPointerCapture(id) { this.captured=id; }
  releasePointerCapture(id) { if(this.captured===id) this.captured=null; }
  append(...es) { for (const e of es) { e.remove?.(); e.parentNode=this; this.children.push(e); } }
  appendChild(e) { this.append(e); return e; }
  remove() { if(this.parentNode) { this.parentNode.children=this.parentNode.children.filter(e=>e!==this); this.parentNode=null; } }
  insertBefore(e,before) { e.remove?.(); const i=this.children.indexOf(before); if(i<0)this.append(e);else {e.parentNode=this;this.children.splice(i,0,e);}return e; }
  get nextSibling() { const es=this.parentNode?.children||[];return es[es.indexOf(this)+1]||null; }
  contains(e) { return e===this||this.children.some(c=>c.contains(e)); }
  removeEventListener(k,f) { this.listeners[k]=(this.listeners[k]||[]).filter(x=>x!==f); }
  replaceChildren(...es) { for(const e of [...this.children])e.remove?.();this.children=[];this.append(...es); }
  setAttribute() {}
  addEventListener(k,f) { (this.listeners[k] ||= []).push(f); }
  querySelectorAll(s) { return this.children.flatMap((e) => [...(s === 'button' && e.tagName === 'button' ? [e] : []), ...e.querySelectorAll(s)]); }
  querySelector(s) { return this.children.find((e) => e.className.split(' ').includes(s.slice(1))) || this.children.map((e) => e.querySelector(s)).find(Boolean); }
  getBoundingClientRect() { return {left:0,top:0,right:100,bottom:100,width:100,height:100}; }
  focus() { this.focused=true; }
  blur() { this.focused=false; }
}

