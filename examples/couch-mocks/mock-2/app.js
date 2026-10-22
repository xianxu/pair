const data = await fetch('/fixtures.json', {cache:'no-store'}).then(r => { if (!r.ok) throw new Error('Fixtures unavailable'); return r.json(); });
const state = new URLSearchParams(location.search).get('state') || 'normal';
document.querySelector('#project').textContent = data.project.name;
const view = document.querySelector('#view');
if (state === 'error') {
 view.textContent = data.error.message;
} else if (document.body.dataset.screen === 'runs') {
 const runs = state === 'empty' ? data.empty : state === 'long-list' ? data.longList : data.runs;
 if (!runs.length) view.textContent = 'No runs yet. Create your first run.';
 for (const run of runs) {
  const link = document.createElement('a'); link.className = 'run'; link.href = 'detail.html?id=' + encodeURIComponent(run.id);
  link.textContent = run.name + ' · ' + run.status; view.append(link);
 }
} else if (document.body.dataset.screen === 'detail') {
 const id = new URLSearchParams(location.search).get('id') || 'run-1';
 const run = data.runs.concat(data.longList).find(r => r.id === id);
 view.textContent = run ? run.name + ' · ' + run.status : 'Run not found.';
} else {
 view.textContent = data.runs.length + ' runs ready to explore.';
}
document.documentElement.dataset.ready = 'true';
