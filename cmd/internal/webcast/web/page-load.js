// One exact identity across recording and live projection. Root stays a string
// so backend IDs above Number.MAX_SAFE_INTEGER cannot compare after rounding.
export function copyPageLoad(value) {
  const text=value=>typeof value==='string'&&value.length>0&&!/[\p{Cs}]/u.test(value);
  if(!value||!text(value.target)||!text(value.loader)||typeof value.root!=='string'||
    !/^[1-9][0-9]*$/.test(value.root)||value.root.length>19||
    (value.root.length===19&&value.root>'9223372036854775807'))return null;
  return {target:value.target,loader:value.loader,root:value.root};
}

export function samePageLoad(a,b) {
  const left=copyPageLoad(a),right=copyPageLoad(b);
  return !!left&&!!right&&left.target===right.target&&left.loader===right.loader&&left.root===right.root;
}

// The standalone adapters also accept legacy presentation models. Actual
// recording Models and injected live snapshots always expose an owned scope;
// an unknown/pending scope there never admits page feedback.
export function feedbackLoad(model) {
  if(!model)return null;
  if('page_load_ready' in model)return model.page_load_ready===true?copyPageLoad(model.page_load):null;
  if(typeof model.currentPageLoad==='function')return copyPageLoad(model.currentPageLoad());
  return undefined;
}
export function feedbackReady(model) {return feedbackLoad(model)!==null;}
export function onFeedbackPage(model,segment) {
  const load=feedbackLoad(model);
  if(load===null)return false;
  // Operator snapshots and their delivery bootstrap are already scoped by
  // Board. Full browser Models use retained creation-segment membership.
  return load===undefined||typeof model.onPageLoad!=='function'||model.onPageLoad(segment,load);
}
export function feedbackScopeKey(model) {
  const load=feedbackLoad(model);
  return load===undefined?undefined:load===null?null:JSON.stringify(load);
}
