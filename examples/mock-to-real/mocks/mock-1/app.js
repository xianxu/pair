// MOCK_TO_REAL_HTML_757: development-only renderer; never part of the Go artifact.
const response = await fetch('/fixtures.json', {cache: 'no-store'});
if (!response.ok) throw new Error('Fixtures unavailable');
const fixture = await response.json();
document.querySelector('#project').textContent = fixture.project.name;
const screen = document.body.dataset.screen;
const view = document.querySelector('#view');
if (screen === 'index') {
  view.textContent = `${fixture.runs.length} illustrative runs ready to explore.`;
} else if (screen === 'runs') {
  for (const run of fixture.runs) {
    const link = document.createElement('a');
    link.href = '/runs/detail?id=' + encodeURIComponent(run.id);
    link.className = 'run';
    link.textContent = `${run.name} · ${run.status}`;
    view.append(link);
  }
} else {
  const id = new URLSearchParams(location.search).get('id') || fixture.runs[0].id;
  const run = fixture.runs.find(run => run.id === id);
  view.textContent = run ? `${run.name} · ${run.status}` : 'Run not found.';
}
document.documentElement.dataset.ready = 'true';
