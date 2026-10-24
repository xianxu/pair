// Transient UI pixels only. Hover never changes DOM attributes, selection or
// feedback visibility stamps; geometry updates belong to resize/normal paint.
export function mountFeedbackGlow(doc, win, {parent, box}) {
  const canvas=doc.createElement('canvas');
  canvas.className='feedback-glow';
  canvas.style.cssText='position:absolute;inset:0;width:100%;height:100%;pointer-events:none;z-index:34';
  parent.append(canvas);
  let disposed=false,rect=null,scale=1;
  const context=canvas.getContext?.('2d');
  const clear=()=>{if(!disposed&&context&&rect)context.clearRect(0,0,rect.width,rect.height);};
  const resize=()=>{
    if(disposed)return;
    clear();rect=box.getBoundingClientRect();
    scale=Number.isFinite(win.devicePixelRatio)&&win.devicePixelRatio>0?win.devicePixelRatio:1;
    const width=Math.max(0,Math.ceil((rect.width||0)*scale)),height=Math.max(0,Math.ceil((rect.height||0)*scale));
    if(canvas.width!==width)canvas.width=width;
    if(canvas.height!==height)canvas.height=height;
    context?.setTransform(scale,0,0,scale,0,0);
  };
  resize();win.addEventListener('resize',resize);
  return {canvas,clear,resize,
    draw(paint){
      if(disposed||!context||!rect||!(rect.width>0&&rect.height>0))return false;
      clear();context.save();
      try{paint(context,rect);return true;}finally{context.restore();}
    },
    dispose(){if(disposed)return;clear();disposed=true;win.removeEventListener?.('resize',resize);canvas.remove?.();},
  };
}
