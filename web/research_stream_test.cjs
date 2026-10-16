// Offline only: run with `node --test web/research_stream_test.cjs`.
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { test } = require('node:test');

const source = fs.readFileSync(path.join(__dirname, 'app.js'), 'utf8').split('function setKnowledgeMessage(')[0];
const encoder = new TextEncoder();
const question = '为自媒体团队研究内容生产工作台';
const validResult = () => ({
  runId: 'run-123', question, plan: [], sources: [], evidence: [],
  stats: { queries: 1, discovered: 3, read: 2, evidenceCount: 1, conflictCount: 0 },
  report: { title: '测试报告', analysis: '有证据支持的完整结论。', modelUsed: 'offline-test' },
  completedAt: '2026-10-05T00:00:00Z',
});
const frame = (name, payload) => `event: ${name}\ndata: ${JSON.stringify(payload)}\n\n`;
const progress = () => frame('progress', { kind: 'started', runId: 'run-123', message: '正在阅读中文资料', discovered: 3, readCount: 2, evidenceCount: 1 });

function element() {
  const classes = new Set();
  const result = {
    textContent: '', style: {}, dataset: {}, children: [], disabled: false, hidden: false,
    classList: { add: value => classes.add(value), remove: value => classes.delete(value), contains: value => classes.has(value), toggle: (value, on) => on ? classes.add(value) : classes.delete(value) },
    querySelectorAll: () => [], focus() {},
    prepend(child) { child.remove = () => { this.children = this.children.filter(item => item !== child); };this.children.unshift(child); },
    get lastElementChild() { return this.children.at(-1); },
  };
  Object.defineProperty(result, 'innerHTML', { get() { return this.html || ''; }, set(value) { this.html = value;this.children = []; } });
  return result;
}

function harness(fetch) {
  const nodes = new Map(), calls = { memory: 0, product: 0 };
  const context = vm.createContext({
    document: { getElementById(id) { if (!nodes.has(id)) nodes.set(id, element());return nodes.get(id); }, createElement: element },
    fetch, TextDecoder, AbortController, URL, console,
    invalidateCoEdit() {}, syncScenarioRebuildState() {},
    loadAgentMemory() { calls.memory++; }, loadProductAnalysis() { calls.product++; },
    clearTimeout,
  });
  vm.runInContext(source, context, { filename: 'web/app.js' });
  return { context, nodes, calls, start: () => context.startResearch(question), state: expression => vm.runInContext(expression, context) };
}

function responseFor(text, options = {}) {
  const bytes = encoder.encode(text);
  const parts = options.bytewise ? [...bytes].map(byte => Uint8Array.of(byte)) : [bytes];
  let index = 0;
  return new Response(new ReadableStream({
    pull(controller) {
      if (index < parts.length) return controller.enqueue(parts[index++]);
      if (options.error) return controller.error(options.error);
      if (!options.open) controller.close();
    },
    cancel() { if (options.onCancel) options.onCancel(); },
  }), { headers: { 'Content-Type': 'text/event-stream; charset=utf-8' } });
}

function assertFailed(h, expected) {
  assert.match(h.nodes.get('runState').textContent, /中断|中止/);
  assert.match(h.nodes.get('errorBanner').textContent, expected);
  assert.equal(h.nodes.get('startButton').disabled, false);
  assert.equal(h.state('currentResult'), null);
  assert.equal(h.nodes.get('reportActions').style.display, 'none');
  assert.equal(h.calls.product, 0);
}

test('valid complete ends immediately without waiting for EOF and starts downstream work once', async () => {
  let cancelled = 0;
  const h = harness(async () => responseFor(progress() + frame('complete', validResult()), { open: true, onCancel: () => cancelled++ }));
  await h.start();
  assert.equal(h.nodes.get('runState').textContent, '研究完成');
  assert.equal(h.nodes.get('startButton').disabled, false);
  assert.match(h.nodes.get('workspaceSubtitle').textContent, /run-123/);
  assert.equal(h.calls.product, 1);
  assert.equal(h.calls.memory, 1);
  assert.equal(cancelled, 1);
});

test('EOF without terminal marks interruption, retains counts and exposes run ID', async () => {
  const h = harness(async () => responseFor(progress()));await h.start();
  assertFailed(h, /连接提前结束.*run-123/);
  assert.equal(h.nodes.get('foundMetric').textContent, 3);
  assert.equal(h.nodes.get('readMetric').textContent, 2);
  assert.equal(h.nodes.get('evidenceMetric').textContent, 1);
});

test('empty stream and null response body cannot complete', async () => {
  for (const response of [responseFor(''), new Response(null, { headers: { 'Content-Type': 'text/event-stream' } })]) {
    const h = harness(async () => response);await h.start();assertFailed(h, /提前结束|数据流为空/);
  }
});

test('a successful HTTP response with non-SSE content is rejected', async () => {
  const h = harness(async () => new Response('<html>proxy error</html>', { headers: { 'Content-Type': 'text/html' } }));
  await h.start();assertFailed(h, /未返回研究事件流/);
});

test('malformed JSON, null payload and missing data fail explicitly', async () => {
  for (const text of ['event: progress\ndata: {broken}\n\n', 'event: complete\ndata: null\n\n', 'event: complete\n\n']) {
    const h = harness(async () => responseFor(text));await h.start();assertFailed(h, /格式异常|内容无效|缺少事件内容/);
  }
});

test('unterminated complete frame at EOF is not treated as a successful report', async () => {
  for (const ending of ['', '\n']) {
    const text = `event: complete\ndata: ${JSON.stringify(validResult())}${ending}`;
    const h = harness(async () => responseFor(text));await h.start();assertFailed(h, /提前结束/);
  }
});

test('CRLF, comments, multiple data lines and byte-split UTF-8 remain valid', async () => {
  const jsonLines = JSON.stringify(validResult(), null, 2).split('\n').map(line => `data: ${line}`).join('\r\n');
  const text = `: heartbeat\r\n\r\nevent: complete\r\n${jsonLines}\r\n\r\n`;
  const h = harness(async () => responseFor(text, { bytewise: true }));await h.start();
  assert.equal(h.nodes.get('runState').textContent, '研究完成');
  assert.match(h.nodes.get('reportBody').innerHTML, /有证据支持的完整结论/);
  assert.equal(h.calls.product, 1);
});

test('failure is terminal even if a complete frame follows in the same chunk', async () => {
  const text = progress() + frame('failure', { error: '模型请求失败', runId: 'run-123', status: 'failed' }) + frame('complete', validResult());
  const h = harness(async () => responseFor(text));await h.start();assertFailed(h, /模型请求失败.*run-123/);
});

test('server cancellation is shown as stopped, never successful', async () => {
  const h = harness(async () => responseFor(frame('failure', { error: 'context cancelled', status: 'cancelled', runId: 'run-cancel' })));
  await h.start();assertFailed(h, /已中止.*run-cancel/);assert.equal(h.nodes.get('runState').textContent, '研究已中止');
});

test('AbortError and reader transport failure do not leave a running UI', async () => {
  const abort = new Error('aborted');abort.name = 'AbortError';
  for (const error of [abort, new Error('socket lost')]) {
    const h = harness(async () => responseFor(progress(), { error }));await h.start();assertFailed(h, /已中止|socket lost/);
  }
});

test('duplicate complete frames cannot run PRD or image downstream twice', async () => {
  const h = harness(async () => responseFor(frame('complete', validResult()) + frame('complete', validResult())));await h.start();
  assert.equal(h.calls.product, 1);assert.equal(h.calls.memory, 1);
});

test('invalid complete structures and blank report are rejected before rendering', async () => {
  const invalid = [null, {}, { ...validResult(), status: 'failed' }, { ...validResult(), stats: null }, { ...validResult(), report: { title: '半截', analysis: '' } }, { ...validResult(), plan: [null] }];
  for (const result of invalid) {
    const h = harness(async () => responseFor(frame('complete', result)));await h.start();assertFailed(h, /内容无效|完成事件缺少/);
  }
});

test('completed with warnings is visibly distinct from a clean success', async () => {
  const result = { ...validResult(), status: 'completed_with_warnings' };
  const h = harness(async () => responseFor(frame('complete', result)));await h.start();
  assert.equal(h.nodes.get('runState').textContent, '研究完成（有警告）');
  assert.match(h.nodes.get('activityText').textContent, /保守证据底稿/);
  assert.equal(h.calls.product, 1);
});

test('complete accepts absent optional plan and null source/evidence collections', async () => {
  const result = validResult();delete result.plan;result.sources = null;result.evidence = null;
  const h = harness(async () => responseFor(frame('complete', result)));await h.start();
  assert.equal(h.nodes.get('runState').textContent, '研究完成');assert.equal(h.calls.product, 1);
});

test('synchronous complete rendering failure clears any partial report', async () => {
  const h = harness(async () => responseFor(frame('complete', validResult())));
  vm.runInContext("complete = result => {currentResult=result;refs.reportBody.innerHTML='PARTIAL REPORT';throw new Error('render failed');}", h.context);
  await h.start();assertFailed(h, /render failed/);assert.doesNotMatch(h.nodes.get('reportBody').innerHTML, /PARTIAL/);
});

test('duplicate clicks cannot issue another POST while first request is active', async () => {
  let resolveFetch, count = 0;
  const h = harness(() => { count++;return new Promise(resolve => { resolveFetch = resolve; }); });
  const first = h.start();await h.start();assert.equal(count, 1);assert.equal(h.nodes.get('startButton').disabled, true);
  resolveFetch(responseFor(frame('complete', validResult())));await first;assert.equal(h.calls.product, 1);
});

test('a superseded request cannot render results or unlock the newer request', async () => {
  const pending = [];
  const h = harness(() => new Promise(resolve => pending.push(resolve)));
  const old = h.start();vm.runInContext('researchRequest = null', h.context);const current = h.start();
  pending[0](responseFor(frame('complete', validResult())));await old;
  assert.equal(h.calls.product, 0);assert.equal(h.nodes.get('startButton').disabled, true);
  pending[1](responseFor(frame('complete', validResult())));await current;
  assert.equal(h.calls.product, 1);assert.equal(h.nodes.get('startButton').disabled, false);
});

test('run ID mismatch is rejected and HTTP failures preserve diagnostic ID', async () => {
  const wrong = { ...validResult(), runId: 'wrong-run' };
  const h = harness(async () => responseFor(progress() + frame('complete', wrong)));await h.start();assertFailed(h, /任务编号不一致.*run-123/);
  const rejected = harness(async () => new Response(JSON.stringify({ error: 'invalid request', runId: 'rejected-run' }), { status: 400, headers: { 'Content-Type': 'application/json' } }));
  await rejected.start();assertFailed(rejected, /invalid request.*rejected-run/);
});

test('header-only run ID survives an empty stream or malformed first frame', async () => {
  for (const text of ['', 'event: progress\ndata: {broken}\n\n']) {
    const response = responseFor(text);response.headers.set('X-Research-Run-ID', 'header-only-run');
    const h = harness(async () => response);await h.start();assertFailed(h, /header-only-run/);
  }
});

test('non-200 header-only run ID is shown even without a JSON error body', async () => {
  const response = new Response('service unavailable', { status: 503, headers: { 'X-Research-Run-ID': 'header-error-run' } });
  const h = harness(async () => response);await h.start();assertFailed(h, /header-error-run/);
});

test('header and event run IDs must agree', async () => {
  const response = responseFor(progress() + frame('complete', validResult()));response.headers.set('X-Research-Run-ID', 'header-run');
  const h = harness(async () => response);await h.start();assertFailed(h, /任务编号不一致.*header-run/);
});

test('malformed UTF-8 cannot become a corrupted successful report', async () => {
  const stream = new ReadableStream({ start(controller) {controller.enqueue(Uint8Array.of(0xff));controller.close();} });
  const h = harness(async () => new Response(stream, { headers: { 'Content-Type': 'text/event-stream' } }));await h.start();assertFailed(h, /encoded data|encoding/i);
});
