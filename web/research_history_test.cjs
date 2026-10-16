// Offline VM tests; all fetches are mocked and no models are called.
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');
const {test}=require('node:test');
const source=fs.readFileSync(path.join(__dirname,'app.js'),'utf8');
const json=(body,status=200)=>new Response(JSON.stringify(body),{status,headers:{'content-type':'application/json'}});
function element(){
  const classes=new Set();const listeners={};
  const node={value:'',textContent:'',innerHTML:'',style:{},dataset:{},children:[],hidden:false,disabled:false,
    classList:{add:value=>classes.add(value),remove:value=>classes.delete(value),contains:value=>classes.has(value),toggle:(value,on)=>on?classes.add(value):classes.delete(value)},
    addEventListener(name,fn){listeners[name]=fn;},listeners,querySelectorAll:()=>[],querySelector:()=>null,closest:()=>null,focus(){},scrollIntoView(){},
    prepend(child){this.children.unshift(child);child.remove=()=>{this.children=this.children.filter(item=>item!==child);};},get lastElementChild(){return this.children.at(-1);}};
  return node;
}
function payload(runId='run-a',options={}){
  const result={runId,question:`研究 ${runId} 的内容工作台需求`,plan:[],sources:[{title:'原始论文',url:'https://example.org/paper',trust:90,status:'read',kind:'paper',domain:'example.org'}],evidence:[{id:'E1',quote:'原始段落',sourceTitle:'原始论文',url:'https://example.org/paper',tags:[],score:90}],stats:{queries:2,discovered:5,read:2,evidenceCount:1,conflictCount:0},report:{title:`报告 ${runId}`,analysis:`完整结论 ${runId} [E1]`,modelUsed:'mock-model'},startedAt:'2026-10-05T01:00:00Z',completedAt:'2026-10-05T01:05:00Z',status:'completed'};
  return {item:{runId,question:result.question,title:result.report.title,status:'completed',startedAt:result.startedAt,completedAt:result.completedAt,summary:'已存研究摘要',hasSnapshot:true,hasDelivery:true},result,delivery:{researchRunId:runId,prd:{info:{prdID:`prd-${runId}`,product:'内容工作台'}},generatedImages:[{kind:'prototype',status:'ready',title:'已存原型图',url:'/api/generated-images/old.png'}]},...options};
}
async function settle(){await new Promise(resolve=>setImmediate(resolve));await new Promise(resolve=>setImmediate(resolve));}
function harness(handler,options={}){
  const nodes=new Map(),requests=[],confirmations=[],browserListeners={};
  const location={hash:options.hash||'',pathname:'/',search:''};
  const window={location,confirm(message){confirmations.push(message);return options.confirm!==false;},addEventListener:(event,fn)=>{browserListeners[event]=fn;},history:{replaceState(_a,_b,url){location.hash=url.includes('#')?url.slice(url.indexOf('#')):'';},pushState(_a,_b,url){location.hash=url.includes('#')?url.slice(url.indexOf('#')):'';}}};
  const context=vm.createContext({document:{getElementById(id){if(!nodes.has(id))nodes.set(id,element());return nodes.get(id);},createElement:element,querySelectorAll:()=>[]},window,URL,AbortController,TextDecoder,console,setTimeout:()=>1,clearTimeout(){},navigator:{},fetch:async(url,init={})=>{requests.push({url,method:init.method||'GET',init});if(url==='/api/knowledge/status'||url.startsWith('/api/agent-memory/overview'))return json({});return handler(url,init);}});
  vm.runInContext(source,context,{filename:'web/app.js'});
  return {context,nodes,requests,confirmations,location,browserListeners,eval:code=>vm.runInContext(code,context),open:id=>context.openResearchHistory(id),node:id=>context.document.getElementById(id)};
}
function assertReadOnly(h){
  assert.equal(h.requests.filter(request=>request.method!=='GET').length,0,'history must not POST');
  assert.equal(h.requests.filter(request=>request.url==='/api/delivery-copilot'||request.url==='/api/delivery-copilot/state'||request.url.includes('image-jobs')||request.url==='/api/research').length,0,'history must not invoke generation or global approval state');
}
test('opening a snapshot restores report, sources, evidence, saved PRD and image with GET only',async()=>{
  const h=harness(()=>json(payload()));await h.open('run-a');
  assert.equal(h.eval('currentResult.runId'),'run-a');assert.equal(h.eval('currentDeliveryCopilot.prd.info.prdID'),'prd-run-a');
  assert.match(h.node('reportBody').innerHTML,/完整结论 run-a/);assert.match(h.node('sourceList').innerHTML,/原始论文/);assert.match(h.node('evidenceList').innerHTML,/原始段落/);
  assert.match(h.node('generatedImageGallery').innerHTML,/old\.png/);assert.equal(h.node('productSubmitButton').disabled,true);assert.equal(h.location.hash,'#research=run-a');assertReadOnly(h);
});
test('refreshing a saved hash restores the same research without downstream generation',async()=>{
  const h=harness(()=>json(payload()),{hash:'#research=run-a'});await settle();
  assert.equal(h.eval('currentResult.runId'),'run-a');assert.equal(h.node('workspace').classList.contains('visible'),true);assertReadOnly(h);
});
test('legacy summary is explicit and cannot masquerade as a full report or trigger generation',async()=>{
  const record=payload();record.result=null;record.delivery=null;record.item.hasSnapshot=false;record.item.hasDelivery=false;
  const h=harness(()=>json(record));await h.open('run-a');assert.equal(h.eval('currentResult'),null);assert.match(h.node('activityText').textContent,/旧记录仅摘要，无法恢复完整报告/);assert.equal(h.node('reportActions').style.display,'none');assert.equal(h.node('productGenerateButton').disabled,true);assertReadOnly(h);
});
test('snapshot without saved PRD waits for an explicit action',async()=>{
  const h=harness(()=>json(payload('run-a',{delivery:null})));await h.open('run-a');assert.equal(h.node('productOutput').hidden,true);assert.match(h.node('productStatus').textContent,/不会自动生成/);assert.equal(h.node('productGenerateButton').disabled,false);assertReadOnly(h);
});
test('404 history does not silently rerun the question',async()=>{
  const h=harness(()=>json({error:'not found'},404));await h.open('missing');assert.match(h.node('researchHistoryMessage').textContent,/不存在.*不会自动重新研究/);assert.equal(h.eval('currentResult'),null);assertReadOnly(h);
});
test('newer history selection wins if responses arrive out of order',async()=>{
  let resolveA;const h=harness(url=>url.endsWith('run-a')?new Promise(resolve=>{resolveA=resolve;}):json(payload('run-b')));
  const old=h.open('run-a');await h.open('run-b');resolveA(json(payload('run-a')));await old;
  assert.equal(h.eval('currentResult.runId'),'run-b');assert.equal(h.eval('currentDeliveryCopilot.researchRunId'),'run-b');assert.equal(h.location.hash,'#research=run-b');assertReadOnly(h);
});
test('pending history cannot overwrite a new-question navigation',async()=>{
  let release;const h=harness(()=>new Promise(resolve=>{release=resolve;}));const old=h.open('run-a');h.context.newResearchQuestion();release(json(payload()));await old;
  assert.equal(h.eval('currentResult'),null);assert.equal(h.node('workspace').classList.contains('visible'),false);assert.equal(h.location.hash,'');assertReadOnly(h);
});
test('active research blocks all history and new-question navigation without request or overwrite',async()=>{
  const h=harness(()=>json(payload()));h.eval('researchRequest={question:"当前任务"}');const before=h.requests.length;await h.open('run-a');await h.context.showResearchHistory();h.context.newResearchQuestion();
  assert.equal(h.requests.length,before);assert.equal(h.eval('researchRequest.question'),'当前任务');assert.match(h.node('researchNavigationNotice').textContent,/尚未结束/);
});
test('unsaved human edits require confirmation before leaving, refresh warns too',async()=>{
  const h=harness(()=>json(payload()),{confirm:false});await h.open('run-a');h.context.markPRDAsHumanEdited();await h.context.showResearchHistory();assert.equal(h.eval('currentResult.runId'),'run-a');assert.equal(h.node('workspace').classList.contains('visible'),true);assert.match(h.confirmations[0],/未保存/);
  let prevented=false;const event={preventDefault(){prevented=true;}};h.browserListeners.beforeunload(event);assert.equal(prevented,true);assert.equal(event.returnValue,'');assertReadOnly(h);
});
test('rerun cancelled by user performs no POST and preserves history',async()=>{
  const h=harness(()=>json(payload()),{confirm:false});await h.open('run-a');await h.context.researchAgain();assert.match(h.confirmations[0],/消耗模型和图片额度/);assert.equal(h.eval('currentResult.runId'),'run-a');assertReadOnly(h);
});
test('history search sends escaped keywords, renders escaped content and paginates',async()=>{
  const item=payload().item;item.title='<img src=x onerror=alert(1)>';
  const h=harness(()=>json({items:[item],total:21,limit:20,offset:0}));h.node('researchHistoryQuery').value='AI & 工作台';await h.context.showResearchHistory();
  assert.match(h.requests.at(-1).url,/q=AI%20%26%20/);assert.match(h.node('researchHistoryResults').innerHTML,/&lt;img/);assert.equal(h.node('researchHistoryPrevious').disabled,true);assert.equal(h.node('researchHistoryNext').disabled,false);assertReadOnly(h);
});
test('mismatched result run ID is rejected before rendering',async()=>{
  const record=payload();record.result.runId='other';const h=harness(()=>json(record));await h.open('run-a');assert.equal(h.eval('currentResult'),null);assert.match(h.node('researchHistoryMessage').textContent,/编号不一致/);assertReadOnly(h);
});
test('history approval calls remain locked, even if invoked outside the disabled button',async()=>{
  const h=harness(()=>json(payload()));await h.open('run-a');await h.context.loadDeliveryState();await h.context.submitDeliveryForReview();assertReadOnly(h);assert.match(h.node('productStatus').textContent,/不能通过全局审批接口/);
});
test('a PRD explicitly belonging to a different research is rejected',async()=>{
  const record=payload();record.delivery.researchRunId='run-other';const h=harness(()=>json(record));await h.open('run-a');assert.equal(h.eval('currentDeliveryCopilot'),null);assert.match(h.node('researchHistoryMessage').textContent,/PRD 与研究编号不一致/);assertReadOnly(h);
});
test('a pending live delivery response cannot overwrite an opened historical PRD or create images',async()=>{
  let resolveGeneration;const h=harness(url=>url==='/api/delivery-copilot'?new Promise(resolve=>{resolveGeneration=resolve;}):json(payload('run-b')));
  h.eval(`currentResult=${JSON.stringify(payload('run-a').result)}`);const generating=h.context.loadProductAnalysis(h.eval('currentResult'));await h.open('run-b');resolveGeneration(json(payload('run-a').delivery));await generating;
  assert.equal(h.eval('currentDeliveryCopilot.researchRunId'),'run-b');assert.equal(h.requests.filter(request=>request.url.includes('image-jobs')).length,0);assert.equal(h.node('productSubmitButton').disabled,true);
});
test('search responses from an older query cannot replace newer results',async()=>{
  let resolveOld;const h=harness(url=>url.includes('q=older')?new Promise(resolve=>{resolveOld=resolve;}):json({items:[payload('newer').item],total:1,offset:0}));
  h.node('researchHistoryQuery').value='older';const pending=h.context.showResearchHistory();h.node('researchHistoryQuery').value='newer';await h.context.loadResearchHistory(0);resolveOld(json({items:[payload('older').item],total:1,offset:0}));await pending;
  assert.match(h.node('researchHistoryResults').innerHTML,/报告 newer/);assert.doesNotMatch(h.node('researchHistoryResults').innerHTML,/报告 older/);assertReadOnly(h);
});
