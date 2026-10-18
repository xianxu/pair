'use strict';
const assert = require('node:assert/strict');
const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { test } = require('node:test');

const driver = path.join(__dirname, 'driver.cjs');
function run(request) {
  return spawnSync(process.execPath, [driver], {
    input: JSON.stringify(request), encoding: 'utf8', timeout: 10000,
  });
}
function phases(stderr) {
  return stderr.split('\n').filter(line => line.startsWith('[pair-oracle] '))
    .map(line => JSON.parse(line.slice('[pair-oracle] '.length)));
}

test('phase breadcrumbs remain separate from the unchanged frame JSON', () => {
  const result = run({ Cols: 12, Rows: 2, Chunks: ['native 界'] });
  assert.equal(result.status, 0, result.stderr);
  assert.equal(JSON.parse(result.stdout)[0].Lines[0].trim(), 'native 界');
  const records = phases(result.stderr);
  assert.deepEqual(records.map(record => record.phase), ['startup', 'import', 'write', 'snapshot', 'dispose']);
  assert.equal(records[0].node, process.version);
  assert.equal(records[0].arch, process.arch);
  assert.equal(records[0].execPath, process.execPath);
});

test('each chunk has one ordered write and snapshot breadcrumb', () => {
  for (const count of [4, 8]) {
    const result = run({ Cols: 12, Rows: 2, Chunks: Array(count).fill('x') });
    assert.equal(result.status, 0, result.stderr);
    const expected = ['startup', 'import'];
    for (let index = 0; index < count; index++) expected.push('write', 'snapshot');
    expected.push('dispose');
    const records = phases(result.stderr);
    assert.deepEqual(records.map(record => record.phase), expected);
    assert.deepEqual(records.filter(record => record.phase === 'write').map(record => record.chunk),
      Array.from({ length: count }, (_, index) => index));
    assert.equal(JSON.parse(result.stdout).length, count);
  }
});

test('ordinary request rejection keeps the observed phase and no fake frame', () => {
  const result = run({ Cols: 0, Rows: 2, Chunks: ['x'] });
  assert.equal(result.status, 1);
  assert.equal(result.stdout, '');
  assert.match(result.stderr, /invalid oracle dimensions/);
  assert.deepEqual(phases(result.stderr).map(record => record.phase), ['startup', 'import']);
});

test('the configured Node report contains runtime evidence without environment data', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'pair-oracle-report-'));
  try {
    const result = spawnSync(process.execPath, [
      '--report-on-fatalerror', '--report-on-signal', '--report-exclude-env',
      '--report-exclude-network', `--report-directory=${dir}`, '--report-filename=node-report.json',
      '-e', 'process.report.writeReport(); process.stdout.write(JSON.stringify({fatal:process.report.reportOnFatalError,signal:process.report.reportOnSignal}));',
    ], { encoding: 'utf8', timeout: 10000,
      env: { ...process.env, PAIR_ORACLE_REPORT_SENTINEL: 'must-not-reach-the-report' } });
    assert.equal(result.status, 0, result.stderr);
    assert.deepEqual(JSON.parse(result.stdout), { fatal: true, signal: true });
    const raw = fs.readFileSync(path.join(dir, 'node-report.json'), 'utf8');
    const report = JSON.parse(raw);
    assert.equal(report.header.nodejsVersion, process.version);
    assert.ok(!Object.hasOwn(report, 'environmentVariables'));
    assert.ok(!raw.includes('must-not-reach-the-report'));
    assert.ok(!Object.hasOwn(report.header, 'networkInterfaces'));
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
});
