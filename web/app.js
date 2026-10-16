const DEFAULT_PLAN = [
  {id:'scope',title:'问题定义',description:'明确研究对象、决策目标与证据门槛。',status:'pending'},
  {id:'discover',title:'市场与产品',description:'搜索市场、需求、产品和竞争信号。',status:'pending'},
  {id:'verify',title:'交叉验证',description:'补充官方、时间与反向证据。',status:'pending'},
  {id:'read',title:'阅读与提取',description:'从网页正文提取可回溯段落。',status:'pending'},
  {id:'synthesize',title:'报告与证据链',description:'连接结论、来源和不确定性。',status:'pending'}
];
const STEP_LABELS = {scope:'定义研究问题', discover:'搜索市场与产品', verify:'补充交叉证据', read:'阅读与提取证据', synthesize:'综合分析与报告'};
const $ = id => document.getElementById(id);
const refs = {landing:$('landing'),workspace:$('workspace'),question:$('question'),start:$('startButton'),questionTitle:$('workspaceQuestion'),questionNode:$('questionNode'),subtitle:$('workspaceSubtitle'),state:$('runState'),plan:$('planList'),activity:$('activityText'),activityStage:$('activityStage'),log:$('liveLog'),found:$('foundMetric'),read:$('readMetric'),evidence:$('evidenceMetric'),conflicts:$('conflictMetric'),sourceTotal:$('sourceTotal'),sourceList:$('sourceList'),high:$('highCount'),medium:$('mediumCount'),low:$('lowCount'),reportTitle:$('reportTitle'),reportBody:$('reportBody'),reportActions:$('reportActions'),viewProduct:$('viewProductButton'),modelBadge:$('modelBadge'),evidenceList:$('evidenceList'),evidenceStatus:$('evidenceStatus'),error:$('errorBanner'),kbState:$('knowledgeState'),kbDocuments:$('kbDocuments'),kbChunks:$('kbChunks'),kbTickets:$('kbTickets'),kbInterviews:$('kbInterviews'),kbInsights:$('kbInsights'),kbRequirements:$('kbRequirements'),kbResearchNotes:$('kbResearchNotes'),kbDatasets:$('kbDatasets'),kbForm:$('knowledgeUploadForm'),kbFile:$('knowledgeFile'),kbType:$('knowledgeType'),kbDepartment:$('knowledgeDepartment'),kbVersion:$('knowledgeVersion'),kbUpload:$('knowledgeUploadButton'),kbSeed:$('knowledgeSeedButton'),kbQuery:$('knowledgeQuery'),kbSearch:$('knowledgeSearchButton'),kbMessage:$('knowledgeMessage'),kbResults:$('knowledgeResults'),historyQuery:$('historyQuery'),historyType:$('historyType'),historySearch:$('historySearchButton'),historyState:$('historyState'),historyMessage:$('historyMessage'),historyResults:$('historyResults'),productGenerate:$('productGenerateButton'),productExport:$('productExportButton'),productStatus:$('productStatus'),productOutput:$('productOutput'),productSummaryTitle:$('productSummaryTitle'),productSummary:$('productSummary'),productSummaryEvidence:$('productSummaryEvidence'),productFeedback:$('productFeedback'),productDemand:$('productDemand'),productScenarios:$('productScenarios'),productPainPoints:$('productPainPoints'),productFeatures:$('productFeatures'),productPrdTitle:$('productPrdTitle'),productPrdProblem:$('productPrdProblem'),productPrd:$('productPrd'),productPrdEvidence:$('productPrdEvidence'),deliveryTabs:$('deliveryTabs'),deliveryTitle:$('deliveryTitle'),deliverySummary:$('deliverySummary'),deliverySummaryEvidence:$('deliverySummaryEvidence'),deliveryWorkflow:$('deliveryWorkflow'),decisionStatement:$('decisionStatement'),decisionEvidence:$('decisionEvidence'),deliveryNotice:$('deliveryNotice'),deliveryFeedback:$('deliveryFeedback'),deliveryResearchEvidence:$('deliveryResearchEvidence'),deliveryDemand:$('deliveryDemand'),deliveryScenarios:$('deliveryScenarios'),deliveryPainPoints:$('deliveryPainPoints'),deliveryOptions:$('deliveryOptions'),deliveryCompetition:$('deliveryCompetition'),deliveryMvp:$('deliveryMvp'),deliveryOutOfScope:$('deliveryOutOfScope'),prdInfo:$('prdInfo'),prdBackground:$('prdBackground'),prdGoals:$('prdGoals'),prdNonGoals:$('prdNonGoals'),prdSuccessMetrics:$('prdSuccessMetrics'),prdUsers:$('prdUsers'),prdArchitecture:$('prdArchitecture'),prdUserFlow:$('prdUserFlow'),prdSystemFlow:$('prdSystemFlow'),prdPermissions:$('prdPermissions'),prdRequirements:$('prdRequirements'),prdNfr:$('prdNfr'),prdQuestions:$('prdQuestions'),solutionPages:$('solutionPages'),solutionApis:$('solutionApis'),solutionData:$('solutionData'),solutionRag:$('solutionRag'),solutionConstraints:$('solutionConstraints'),deliveryEvaluation:$('deliveryEvaluation'),deliveryChains:$('deliveryChains'),readinessScore:$('readinessScore'),readinessVerdict:$('readinessVerdict'),readinessChecks:$('readinessChecks'),readinessActions:$('readinessActions'),productPersist:$('productPersistButton'),productSubmit:$('productSubmitButton'),deliveryPersistence:$('deliveryPersistence'),deliveryApprovals:$('deliveryApprovals'),prdFlowDiagram:$('prdFlowDiagram'),productFlowDiagram:$('productFlowDiagram'),prdPrototypes:$('prdPrototypes'),solutionFlowDiagram:$('solutionFlowDiagram'),solutionDataModel:$('solutionDataModel'),solutionDataDictionary:$('solutionDataDictionary'),solutionApiContracts:$('solutionApiContracts'),solutionDependencies:$('solutionDependencies'),deliveryEstimate:$('deliveryEstimate'),deliveryAssumptions:$('deliveryAssumptions'),deliverySchedule:$('deliverySchedule'),sopSteps:$('sopSteps'),testRoles:$('testRoles'),testDataSets:$('testDataSets'),testStrategy:$('testStrategy'),testCases:$('testCases'),testExitRules:$('testExitRules'),visualDownload:$('visualDownloadButton'),memoryState:$('memoryState'),memoryMetrics:$('memoryMetrics'),memoryPinned:$('memoryPinned'),memorySummary:$('memorySummary'),memoryEvents:$('memoryEvents'),coeditActor:$('coeditActor'),coeditInput:$('coeditInput'),coeditSend:$('coeditSendButton'),coeditApplyAll:$('coeditApplyAllButton'),coeditStatus:$('coeditStatus'),coeditConversation:$('coeditConversation'),coeditProposalState:$('coeditProposalState'),coeditProposal:$('coeditProposal'),generatedImageStatus:$('generatedImageStatus'),generatedImageGallery:$('generatedImageGallery'),generatedImageRetry:$('generatedImageRetryButton')};
let counts = {found:0,read:0,evidence:0};
let currentResult = null;let currentDeliveryVisuals = null;let currentDeliveryCopilot = null;let currentCoEditProposal = null;let coEditHistory = [];
let deliveryResearchResult = null;let scenarioRebuildRequest = null;
let coEditRequest = null;let coEditProposalContext = null;let coEditEpoch = 0;let coEditDeliveryLoading = false;
let researchRequest = null;
let researchViewEpoch=0,historyListEpoch=0,historyDetailEpoch=0,researchHistoryItem=null,deliveryDraftDirty=false,historyApprovalLocked=false;
let historyPage={query:'',offset:0,limit:20,total:0};
const deliveryImageStates = new WeakMap();

function escapeHTML(value='') { return String(value).replace(/[&<>'"]/g,char => ({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[char])); }
function renderPlan(plan=DEFAULT_PLAN) { refs.plan.innerHTML = plan.map(item => `<li class="${item.status || 'pending'}" data-step="${item.id}"><span class="step-symbol"></span><span class="step-name">${escapeHTML(item.title)}</span><span class="step-description">${escapeHTML(item.description)}</span></li>`).join(''); }
function setStep(step,status='active') { const items=[...refs.plan.querySelectorAll('li')], index=items.findIndex(item=>item.dataset.step===step); if(index<0)return; items.forEach((item,i)=>item.className=i<index?'done':i===index?status:'pending'); }
function setRunState(label,state='running') { refs.state.className=`run-state ${state}`; refs.state.textContent=label; }
function updateMetrics() { refs.found.textContent=counts.found; refs.read.textContent=counts.read; refs.evidence.textContent=counts.evidence; refs.sourceTotal.textContent=`${counts.found} ITEMS`; }
function renderCoverage(coverage=null) {
  const container=$('researchCoverage'),status=$('coverageStatus'),summary=$('coverageSummary'),dimensions=$('coverageDimensions'),gaps=$('coverageGaps'),stop=$('coverageStopReason');
  if(!container)return;
  const number=value=>Math.max(0,Number(value)||0);
  if(!coverage){container.dataset.status='waiting';status.textContent='等待研究开始';summary.textContent='读取资料后，将逐轮检查来源数量、来源多样性和研究维度。';dimensions.innerHTML='';gaps.innerHTML='';stop.textContent='';return;}
  const sufficient=coverage.status==='sufficient';container.dataset.status=sufficient?'sufficient':'insufficient';status.textContent=sufficient?'覆盖门槛已达到':'仍有待补证据';
  summary.textContent=`第 ${number(coverage.round)} / ${number(coverage.maxRounds)} 轮 · 有效来源 ${number(coverage.evidenceSources)} / ${number(coverage.targetSources)} · 成功读取 ${number(coverage.sourcesRead)} · 来源域名 ${number(coverage.domains)} / ${number(coverage.targetDomains)} · 证据 ${number(coverage.evidenceCount)} / ${number(coverage.targetEvidence)}`;
  dimensions.innerHTML=(Array.isArray(coverage.dimensions)?coverage.dimensions:[]).map(item=>`<article class="coverage-dimension ${item.status==='covered'?'covered':'gap'}"><strong>${escapeHTML(item.name||item.id||'研究维度')} <span>${item.status==='covered'?'已覆盖':'待补充'}</span></strong><p>${number(item.sources)} 个来源 · ${number(item.domains)} 个域名</p><small>${escapeHTML((Array.isArray(item.evidenceIds)?item.evidenceIds:[]).join(' · ')||'暂无匹配证据')}</small></article>`).join('');
  gaps.innerHTML=(Array.isArray(coverage.gaps)?coverage.gaps:[]).map(gap=>`<li>${escapeHTML(gap)}</li>`).join('');
  const stopReasons={coverage_target:'已达到本次研究的覆盖目标',time_budget:'已达到本次研究的时间预算',source_budget:'已达到本次研究的来源预算',no_new_evidence:'继续搜索暂未发现新增证据',round_budget:'已达到本次研究的轮次预算',query_budget:'已达到本次研究的查询预算'};
  stop.textContent=coverage.stopReason?`本次研究停止原因：${stopReasons[coverage.stopReason]||coverage.stopReason}`:'';
}
function safeSourceURL(value='') { try { const url=new URL(value);return ['http:','https:'].includes(url.protocol)?url.href:''; } catch { return ''; } }
function logClass(kind='') { return kind.includes('read')?'read':(kind==='search'||kind==='sources')?'search':kind; }
function addLog(event) { const item=document.createElement('li'); item.className=logClass(event.kind); const time=new Date(event.at||Date.now()).toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'}); item.innerHTML=`<i class="event-dot"></i><span>${escapeHTML(event.message||'正在处理…')}</span><time class="event-time">${time}</time>`; refs.log.prepend(item); while(refs.log.children.length>6)refs.log.lastElementChild.remove(); }
function resetRun(question) {
  invalidateCoEdit();
  const imageState=currentDeliveryCopilot&&deliveryImageStates.get(currentDeliveryCopilot);if(imageState&&imageState.timer)clearTimeout(imageState.timer);currentDeliveryCopilot=null;
  renderCoverage();
  deliveryResearchResult=null;scenarioRebuildRequest=null;syncScenarioRebuildState();
  counts={found:0,read:0,evidence:0};currentResult=null;currentCoEditProposal=null;coEditHistory=[];updateMetrics();renderPlan(DEFAULT_PLAN);refs.error.classList.remove('visible');refs.log.innerHTML='';
  refs.questionTitle.textContent=question;refs.questionNode.textContent=question;refs.subtitle.textContent='Agent 正在按计划搜索、阅读、比较与检查证据。';refs.activity.textContent='正在建立研究任务…';refs.activityStage.textContent='准备中';setRunState('建立计划中');
  refs.conflicts.textContent='0';refs.high.textContent='0';refs.medium.textContent='0';refs.low.textContent='0';refs.sourceList.innerHTML='<li><div class="source-info"><div class="source-title">正在建立来源清单…</div><div class="source-meta">优先读取官方资料与可验证页面</div></div></li>';
  refs.productOutput.hidden=true;refs.productStatus.textContent='研究完成后会自动组合本地历史资料，生成需求调研、功能方案与 PRD。';refs.reportTitle.textContent='Final Report';refs.modelBadge.textContent='WAITING FOR EVIDENCE';refs.reportBody.innerHTML='<p class="report-empty">先完成来源阅读、相关性检查与冲突扫描，再生成带引用的结论。</p>';refs.reportActions.style.display='none';refs.evidenceStatus.textContent='COLLECTING EVIDENCE';refs.evidenceList.innerHTML='<div class="empty-evidence">正在从来源正文中提取可追溯段落…</div>';
}
function handleProgress(event) {
  if(event.coverage)renderCoverage(event.coverage);
  refs.activity.textContent=event.message||'正在研究…';refs.activityStage.textContent=STEP_LABELS[event.step]||'研究进行中';addLog(event);
  if(event.step)setStep(event.step,event.kind==='warning'?'blocked':'active');
  if(event.coverage){counts.found=Math.max(0,Number(event.discovered)||0);counts.read=Math.max(0,Number(event.coverage.sourcesRead)||0);counts.evidence=Math.max(0,Number(event.coverage.evidenceCount)||0);}else{if(event.discovered)counts.found+=event.discovered;if(event.readCount)counts.read+=event.readCount;if(event.evidenceCount)counts.evidence=event.evidenceCount;}updateMetrics();
}
function confidenceFor(trust) { return trust>=76?'高':trust>=59?'中':'低'; }
function confidenceClass(label) { return label==='高'?'high':label==='中'?'medium':'low'; }
function renderSources(sources,stats) {
  const safeSources=Array.isArray(sources)?sources:[];
  const safeStats=stats&&typeof stats==='object'?stats:{};
  counts={found:Number(safeStats.discovered)||0,read:Number(safeStats.read)||0,evidence:Number(safeStats.evidenceCount)||0};updateMetrics();
  refs.sourceList.innerHTML=safeSources.length?safeSources.map((source,index)=>{source=source||{};const confidence=confidenceFor(Number(source.trust)||0);const paperMeta=[source.authors,source.publishedAt].filter(Boolean).join(' · ');const url=safeSourceURL(source.url);const title=escapeHTML(source.title||source.domain||'未命名来源');return `<li><span class="source-no">${index+1}</span><div class="source-info">${url?`<a class="source-title source-link" href="${escapeHTML(url)}" target="_blank" rel="noopener noreferrer" title="${title}">${title} ↗</a>`:`<div class="source-title" title="${title}">${title}</div>`}<div class="source-meta"><span class="confidence ${confidenceClass(confidence)}">${confidence}</span><span class="source-domain">${escapeHTML(source.status||'状态未知')} · ${escapeHTML(source.kind)} · ${escapeHTML(source.domain)}${paperMeta?` · ${escapeHTML(paperMeta)}`:''}</span></div>${source.error?`<div class="source-meta">${escapeHTML(source.error)}</div>`:''}</div></li>`;}).join(''):'<li><div class="source-info"><div class="source-title">没有可读取的来源</div><div class="source-meta">可尝试更具体的问题后重试</div></div></li>';
  refs.high.textContent=Number(safeStats.highConfidence)||0;refs.medium.textContent=Number(safeStats.mediumConfidence)||0;refs.low.textContent=Math.max(0,(Number(safeStats.evidenceCount)||0)-(Number(safeStats.highConfidence)||0)-(Number(safeStats.mediumConfidence)||0));
}function normalizeReportMarkdown(markdown='') {
  return String(markdown)
    .replace(/\r/g,'')
    .replace(/\\\s*([#*_`-])/g,'$1')
    .replace(/([^\n])\s+(#{1,6}\s+)/g,'$1\n\n$2')
    .replace(/(^|\n)(#{1,6}\s+[^\n]+)/g,'$1\n\n$2\n\n')
    .replace(/(^|\n)(---+)\s*(?=\n|$)/g,'$1\n\n$2\n\n')
    .replace(/\n{3,}/g,'\n\n')
    .trim();
}
function inlineMarkdown(text='') { return escapeHTML(String(text)).replace(/\*\*(.+?)\*\*/g,'<strong>$1</strong>').replace(/`(.+?)`/g,'<code>$1</code>').replace(/\[(E\d+)\]/g,'<strong>[$1]</strong>'); }function renderMarkdown(markdown='') {
  const normalized=normalizeReportMarkdown(markdown);
  if(!normalized)return '<p class="report-empty">报告没有返回可展示的文本。</p>';
  return normalized.split(/\n{2,}/).filter(Boolean).map(block=>{
    const heading=block.match(/^(#{1,6})\s+([\s\S]+)$/);
    if(heading){const tag=heading[1].length<=2?'h2':'h3';return `<${tag}>${inlineMarkdown(heading[2].trim())}</${tag}>`;}
    if(/^---+$/.test(block.trim()))return '<hr class="report-divider">';
    const lines=block.split('\n').map(line=>line.trim()).filter(Boolean);
    if(lines.some(line=>/^(?:- |\* )/.test(line))){
      const parts=[], bullets=[];
      const flush=()=>{if(bullets.length){parts.push(`<ul>${bullets.map(line=>`<li>${inlineMarkdown(line.replace(/^(?:- |\* )/,''))}</li>`).join('')}</ul>`);bullets.length=0;}};
      lines.forEach(line=>{if(/^(?:- |\* )/.test(line)){bullets.push(line);}else{flush();parts.push(`<p>${inlineMarkdown(line)}</p>`);}});
      flush();return parts.join('');
    }
    return `<p>${inlineMarkdown(block).replace(/\n/g,'<br>')}</p>`;
  }).join('');
}function renderEvidence(evidence) {
  const safeEvidence=Array.isArray(evidence)?evidence:[];
  refs.evidenceStatus.textContent=`${safeEvidence.length} EVIDENCE ITEMS`;
  refs.evidenceList.innerHTML=safeEvidence.length?safeEvidence.map(item=>{item=item||{};const tags=Array.isArray(item.tags)?item.tags:[];return `<article class="evidence-card"><div class="evidence-top"><span>${escapeHTML(item.id)} · ${escapeHTML(tags.join(' / '))}</span><span class="evidence-score">${Number(item.score)||0}/100 · ${escapeHTML(item.confidence)}</span></div><p class="evidence-quote">${escapeHTML(item.quote)}</p><a class="evidence-source" href="${escapeHTML(item.url)}" target="_blank" rel="noreferrer" title="打开原始来源">↗ ${escapeHTML(item.sourceTitle)}</a></article>`;}).join(''):'<div class="empty-evidence">没有提取到可回溯的正文证据，因此系统不会给出确定性结论。</div>';
}function complete(result,options={}) {
  renderCoverage(result.coverage);
  currentResult=result;renderPlan(result.plan);renderSources(result.sources,result.stats);renderEvidence(result.evidence);refs.conflicts.textContent=result.stats.conflictCount;
  refs.reportTitle.textContent=result.report.title;refs.reportBody.innerHTML=renderMarkdown(result.report.analysis);refs.modelBadge.textContent=result.report.modelUsed;refs.reportActions.style.display='flex';
  refs.subtitle.textContent=`${result.stats.queries} 次搜索 · ${result.stats.discovered} 个候选来源 · ${result.stats.evidenceCount} 条证据 · 完成于 ${new Date(result.completedAt).toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'})}`;
  const withWarnings=result.status==='completed_with_warnings';
  refs.activity.textContent=withWarnings?'研究已完成，但部分步骤存在警告或降级。请检查活动警告及报告模型标记，不要将保守证据底稿当作完整模型结论。':'研究完成。每条证据都可回到原始网址核验。';refs.activityStage.textContent=withWarnings?'完成 · 有警告':'完成';setRunState(withWarnings?'研究完成（有警告）':'研究完成','done');addLog({kind:withWarnings?'warning':'finished',message:withWarnings?'报告已生成，部分步骤存在警告或降级，请检查报告模型标记与各交付状态。':'报告与 Claim ↔ Evidence Map 已生成。',at:result.completedAt});
  if(!options.history){setResearchLocation(result.runId?`#research=${encodeURIComponent(result.runId)}`:'');if(result.archiveStatus==='failed'){refs.error.textContent='本次报告自动保存失败，请先下载报告；暂时无法保证从历史研究恢复。';refs.error.classList.add('visible');}loadAgentMemory(result.question||'');loadProductAnalysis(result);}
}
function researchResultIsComplete(result) {
  const object=value=>value!==null&&typeof value==='object'&&!Array.isArray(value);
  return object(result)&&(!result.status||['completed','completed_with_warnings'].includes(result.status))&&object(result.report)&&typeof result.report.analysis==='string'&&Boolean(result.report.analysis.trim())&&typeof result.report.title==='string'&&object(result.stats)&&(result.plan===undefined||(Array.isArray(result.plan)&&result.plan.every(object)));
}
// An SSE event is committed only by its terminating blank line. An EOF tail is
// deliberately not dispatched: it may be an incomplete complete/failure frame.
function createResearchEventParser(onEvent) {
  let buffer='',name='',data=[];
  return text=>{
    buffer+=text;
    let newline;
    while((newline=buffer.indexOf('\n'))!==-1){
      let line=buffer.slice(0,newline);buffer=buffer.slice(newline+1);if(line.endsWith('\r'))line=line.slice(0,-1);
      if(line===''){
        const eventName=name,raw=data.join('\n');name='';data=[];
        if(!['progress','started','complete','failure'].includes(eventName))continue;
        if(!raw)throw new Error('研究数据流缺少事件内容，请手动重试。');
        let payload;try{payload=JSON.parse(raw);}catch{throw new Error('研究数据流格式异常，未收到可验证的完整结果，请手动重试。');}
        if(!payload||typeof payload!=='object'||Array.isArray(payload))throw new Error('研究数据流事件内容无效，请手动重试。');
        if(onEvent(eventName,payload)===false)return false;
        continue;
      }
      if(line.startsWith(':'))continue;
      const colon=line.indexOf(':'),field=colon<0?line:line.slice(0,colon);let value=colon<0?'':line.slice(colon+1);if(value.startsWith(' '))value=value.slice(1);
      if(field==='event')name=value;else if(field==='data')data.push(value);
    }
    return true;
  };
}
async function startResearch(question) {
  if(researchRequest)return;
  question=String(question||'').trim();if(question.length<8){refs.question.focus();return;}
  if(!confirmLeaveResearch())return;
  researchViewEpoch++;historyDetailEpoch++;historyListEpoch++;researchHistoryItem=null;historyApprovalLocked=false;deliveryDraftDirty=false;setResearchLocation('');setHistoryVisibility(false);syncResearchHistoryActions();
  const request={question,runId:'',terminal:'',controller:new AbortController()};researchRequest=request;refs.start.disabled=true;
  const isCurrent=()=>researchRequest===request;let reader=null;
  const rememberRunId=payload=>{
    if(typeof payload.runId!=='string'||!payload.runId.trim())return;
    if(request.runId&&request.runId!==payload.runId)throw new Error('研究数据流的任务编号不一致，已停止接收。');
    request.runId=payload.runId;
  };
  try {
    refs.landing.style.display='none';refs.workspace.classList.add('visible');resetRun(question);
    const response=await fetch('/api/research',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({question}),signal:request.controller.signal});
    if(!isCurrent())return;
    rememberRunId({runId:response.headers.get('X-Research-Run-ID')||''});
    if(!response.ok){const body=await response.json().catch(()=>({}));if(!isCurrent())return;rememberRunId(body||{});throw new Error(body&&body.error||'无法开始研究任务。');}
    if(!/^text\/event-stream(?:\s*;|\s*$)/i.test(response.headers.get('content-type')||''))throw new Error('服务未返回研究事件流，任务未完成，请手动重试。');
    if(!response.body||typeof response.body.getReader!=='function')throw new Error('研究数据流为空，任务未完成，请手动重试。');
    reader=response.body.getReader();const decoder=new TextDecoder('utf-8',{fatal:true});
    const consume=createResearchEventParser((name,payload)=>{
      if(!isCurrent()||request.terminal)return false;
      rememberRunId(payload);
      if(name==='failure'){
        request.terminal=payload.status==='cancelled'?'cancelled':'failed';
        throw new Error(payload.error||(request.terminal==='cancelled'?'研究已中止。':'研究任务失败。'));
      }
      if(name==='complete'){
        if(!researchResultIsComplete(payload))throw new Error('完成事件缺少有效报告或必要数据，未将此次研究标记为成功。');
        complete(payload);request.terminal='completed';
        if(request.runId)refs.subtitle.textContent+=` · 任务 ID：${request.runId}`;
        return false;
      }
      handleProgress(payload);
      if(request.runId)refs.subtitle.textContent=`Agent 正在按计划搜索、阅读、比较与检查证据。任务 ID：${request.runId}`;
      return true;
    });
    while(isCurrent()&&!request.terminal){
      const chunk=await reader.read();if(!isCurrent())return;
      if(chunk.done){consume(decoder.decode());if(!request.terminal)throw new Error('研究连接提前结束，未收到完成确认；任务未完成，请手动重试。');break;}
      if(consume(decoder.decode(chunk.value,{stream:true}))===false)break;
    }
  } catch(error) {
    if(!isCurrent()||request.terminal==='completed')return;
    const cancelled=request.terminal==='cancelled'||error?.name==='AbortError';request.terminal=cancelled?'cancelled':'failed';currentResult=null;
    const message=cancelled?'研究已中止，尚未生成完整结果。':error?.message||'研究任务失败，请稍后手动重试。';
    refs.error.textContent=message+(request.runId?` 任务 ID：${request.runId}`:'');refs.error.classList.add('visible');
    refs.activity.textContent=cancelled?'研究已中止；已读取的数量保留供排查，可手动重新开始。':'研究没有完成；已读取的数量保留供排查，请手动重试。';refs.activityStage.textContent=cancelled?'已中止':'已中断';
    refs.subtitle.textContent=(cancelled?'本次研究已中止。':'本次研究未完成。')+(request.runId?`任务 ID：${request.runId}`:'');
    refs.reportBody.innerHTML='<p class="report-empty">未收到并验证完整报告，不展示部分结果。请手动重新开始研究。</p>';refs.reportActions.style.display='none';refs.modelBadge.textContent='RESEARCH INCOMPLETE';refs.evidenceStatus.textContent='RESEARCH INCOMPLETE';refs.productOutput.hidden=true;
    setRunState(cancelled?'研究已中止':'研究中断 · 可重试','error');addLog({kind:'warning',message:refs.error.textContent});
  } finally {
    if(reader){try{Promise.resolve(reader.cancel()).catch(()=>{});}catch{}try{reader.releaseLock();}catch{}}
    if(isCurrent()){researchRequest=null;refs.start.disabled=false;}
  }
}
function setResearchLocation(hash='',push=false) {
  const browser=globalThis.window;if(!browser?.history||!browser.location)return;
  const url=`${browser.location.pathname}${browser.location.search}${hash}`;
  browser.history[push?'pushState':'replaceState'](null,'',url);
}
function researchNavigationNotice(message='') { const node=$('researchNavigationNotice');if(node)node.textContent=message; }
function confirmLeaveResearch() {
  if(researchRequest){researchNavigationNotice('当前研究尚未结束，请完成后再切换课题；不会中断或覆盖正在进行的任务。');return false;}
  if(deliveryDraftDirty&&!(globalThis.window?.confirm('当前 PRD 有未保存的人工修改。取消可先点击“保存 PRD 草案”；确定离开将丢弃这些修改。')))return false;
  researchNavigationNotice('');return true;
}
function setHistoryVisibility(visible) { const panel=$('researchHistory');if(panel)panel.hidden=!visible; }
function syncResearchHistoryActions() { const button=$('researchAgainButton');if(button)button.hidden=!researchHistoryItem; }
function historyDate(value) { const date=new Date(value||'');return Number.isNaN(date.getTime())?'时间未记录':date.toLocaleString('zh-CN',{hour12:false}); }
function historyStatus(item={}) { return ({completed:'已完成',completed_with_warnings:'已完成 · 有警告',failed:'失败',cancelled:'已取消',running:'进行中'})[item.status]||'历史记录'; }
async function showResearchHistory(options={}) {
  if(!options.skipConfirmation&&!confirmLeaveResearch())return;
  historyDetailEpoch++;researchViewEpoch++;
  refs.landing.style.display='none';refs.workspace.classList.remove('visible');setHistoryVisibility(true);
  setResearchLocation('#history',Boolean(options.push));
  return loadResearchHistory(0);
}
async function loadResearchHistory(offset=0) {
  const panel=$('researchHistory'),message=$('researchHistoryMessage'),results=$('researchHistoryResults');if(!panel||!message||!results)return;
  const token=++historyListEpoch;const query=String($('researchHistoryQuery')?.value||'').trim();
  message.textContent='正在读取本机保存的研究记录，不会调用模型…';
  try {
    const response=await fetch(`/api/research/history?q=${encodeURIComponent(query)}&limit=20&offset=${Math.max(0,offset)}`,{cache:'no-store'});
    const body=await response.json().catch(()=>({}));if(token!==historyListEpoch||panel.hidden)return;
    if(!response.ok)throw new Error(body.error||'无法读取历史研究。');
    const items=Array.isArray(body.items)?body.items:[];historyPage={query,offset:Math.max(0,Number(body.offset)||0),limit:20,total:Math.max(0,Number(body.total)||0)};
    message.textContent=`共 ${historyPage.total} 条研究${query?'（搜索结果）':''}。打开历史仅读取本机数据，不会重新搜索或消耗模型额度。`;
    results.innerHTML=items.map(item=>`<article class="research-history-card"><div class="research-history-card-head"><h3>${escapeHTML(item.title||item.question||'未命名课题')}</h3><span>${escapeHTML(historyStatus(item))}</span></div><p class="research-history-question">${escapeHTML(item.question)}</p><p>${escapeHTML(item.summary||'暂无摘要。')}</p><div class="research-history-card-foot"><span>${escapeHTML(historyDate(item.completedAt||item.startedAt))} · ${item.hasSnapshot?'完整报告与证据':'旧记录仅摘要'}${item.hasDelivery?' · 已存 PRD':''}</span><button class="secondary" type="button" data-open-research="${escapeHTML(item.runId)}">${item.hasSnapshot?'打开研究':'查看摘要'} →</button></div></article>`).join('')||'<p class="empty-evidence">没有匹配的研究记录。可换一个关键词，或开始新的研究。</p>';
    $('researchHistoryPrevious').disabled=historyPage.offset===0;$('researchHistoryNext').disabled=historyPage.offset+items.length>=historyPage.total;
    $('researchHistoryPage').textContent=historyPage.total?`${historyPage.offset+1}–${historyPage.offset+items.length} / ${historyPage.total}`:'0 条';
  }catch(error){if(token!==historyListEpoch||panel.hidden)return;message.textContent=error.message||'历史研究读取失败，请重试。';results.innerHTML='';$('researchHistoryPrevious').disabled=true;$('researchHistoryNext').disabled=true;}
}
async function openResearchHistory(runId,options={}) {
  if(!options.skipConfirmation&&!confirmLeaveResearch())return;
  if(!/^[a-zA-Z0-9_-]{1,80}$/.test(String(runId||''))){researchNavigationNotice('历史研究编号无效。');return;}
  const token=++historyDetailEpoch;historyListEpoch++;researchViewEpoch++;const viewEpoch=researchViewEpoch;
  refs.landing.style.display='none';refs.workspace.classList.remove('visible');setHistoryVisibility(true);
  $('researchHistoryMessage').textContent='正在打开已保存的报告、证据和 PRD…';
  try {
    const response=await fetch(`/api/research/history/${encodeURIComponent(runId)}`,{cache:'no-store'});const body=await response.json().catch(()=>({}));
    if(token!==historyDetailEpoch||viewEpoch!==researchViewEpoch||researchRequest)return;
    if(!response.ok)throw new Error(response.status===404?'这条历史研究不存在或已不可读取；不会自动重新研究。':body.error||'无法打开历史研究。');
    if(!body.item||body.item.runId!==runId)throw new Error('历史记录的任务编号不一致，未加载。');
    if(body.result&&(!researchResultIsComplete(body.result)||body.result.runId!==runId))throw new Error('历史报告数据不完整或任务编号不一致；未重新调用模型。');
    if(body.delivery?.researchRunId&&body.delivery.researchRunId!==runId)throw new Error('历史 PRD 与研究编号不一致，未加载。');
    resetRun(body.item.question||'历史研究');deliveryDraftDirty=false;coEditDeliveryLoading=false;researchHistoryItem=body.item;historyApprovalLocked=true;
    refs.question.value=body.item.question||'';refs.productGenerate.disabled=!body.result;refs.productExport.disabled=true;refs.productPersist.disabled=true;refs.productSubmit.disabled=true;refs.viewProduct.disabled=true;
    if(body.result){
      complete(body.result,{history:true});deliveryResearchResult=body.result;
      refs.subtitle.textContent=`历史研究 · ${historyDate(body.item.completedAt||body.item.startedAt)} · 任务 ID：${runId} · 仅恢复已保存内容`;
      refs.activity.textContent='正在查看已保存的研究，没有重新搜索、生成报告或生成图片。';refs.activityStage.textContent='历史快照';
    }else{
      currentResult=null;renderPlan([]);renderSources([],{});renderEvidence([]);renderCoverage();
      refs.reportTitle.textContent=body.item.title||body.item.question||'历史摘要';refs.reportBody.innerHTML=renderMarkdown(body.item.summary||'未保存可恢复的摘要。');refs.reportActions.style.display='none';refs.modelBadge.textContent='LEGACY SUMMARY';
      refs.subtitle.textContent=`历史记录 · ${historyDate(body.item.completedAt||body.item.startedAt)} · ${runId}`;refs.activity.textContent='旧记录仅摘要，无法恢复完整报告。不会用摘要伪造原始证据或重新调用模型。';refs.activityStage.textContent='历史摘要';setRunState('旧记录 · 仅摘要','done');
    }
    if(body.delivery){
      renderDeliveryCopilot(body.delivery);refs.productStatus.textContent='已恢复这个课题保存的 PRD，可继续编辑或下载。历史浏览不会生成新内容或图片。';
      refs.deliveryPersistence.textContent='这是该课题的历史草案快照。审批状态未随快照恢复，不能据此提交审批。';refs.deliveryApprovals.innerHTML='';refs.productSubmit.disabled=true;
      const imageState=deliveryImageState(body.delivery);imageState.paused=Boolean(imageState.id||imageState.requestId);renderGeneratedImages(imageState.images,'已恢复保存的图片。需要查看后台任务进度时，请点击“继续查询进度”；不会自动重新生成。');
    }else{refs.productOutput.hidden=true;refs.productStatus.textContent=body.result?'本次研究尚未保存 PRD。可点击“生成交付工作流”主动生成；打开历史不会自动生成。':'旧记录没有完整研究快照，不能恢复或据此生成原 PRD。';}
    if(body.notice){refs.error.textContent=String(body.notice);refs.error.classList.add('visible');}
    setHistoryVisibility(false);refs.workspace.classList.add('visible');syncResearchHistoryActions();setResearchLocation(`#research=${encodeURIComponent(runId)}`,Boolean(options.push));
  }catch(error){if(token!==historyDetailEpoch||viewEpoch!==researchViewEpoch)return;$('researchHistoryMessage').textContent=error.message||'无法打开历史记录。';setResearchLocation('#history');}
}
function newResearchQuestion(options={}) {
  if(!options.skipConfirmation&&!confirmLeaveResearch())return;
  researchViewEpoch++;historyDetailEpoch++;historyListEpoch++;resetRun('新研究');researchHistoryItem=null;historyApprovalLocked=false;deliveryDraftDirty=false;coEditDeliveryLoading=false;
  refs.question.value='';refs.workspace.classList.remove('visible');refs.landing.style.display='';setHistoryVisibility(false);syncResearchHistoryActions();setResearchLocation('',Boolean(options.push));refs.question.focus();
}
async function researchAgain() {
  if(!researchHistoryItem||researchRequest)return;
  if(!globalThis.window?.confirm('重新研究会重新搜索并调用模型，可能消耗模型和图片额度。历史记录会保留，新结果作为另一条研究保存。确认继续吗？'))return;
  return startResearch(researchHistoryItem.question);
}
async function restoreResearchLocation(options={}) {
  const hash=globalThis.window?.location?.hash||'';
  if(hash==='#history')return showResearchHistory(options);
  if(hash.startsWith('#research=')){let id='';try{id=decodeURIComponent(hash.slice(10));}catch{}return openResearchHistory(id,options);}
}
function setKnowledgeMessage(message='',isError=false) { refs.kbMessage.textContent=message;refs.kbMessage.classList.toggle('error',isError); }
function renderKnowledgeStatus(status={}) { refs.kbDocuments.textContent=Number(status.documents)||0;refs.kbChunks.textContent=Number(status.chunks)||0;refs.kbTickets.textContent=Number(status.supportTickets)||0;refs.kbInterviews.textContent=Number(status.interviews)||0;refs.kbInsights.textContent=Number(status.insights)||0;refs.kbRequirements.textContent=Number(status.productRequirements)||0;refs.kbResearchNotes.textContent=Number(status.researchNotes)||0;refs.kbDatasets.textContent=Number(status.structuredDatasets)||0; }
async function refreshKnowledgeStatus() { try { const response=await fetch('/api/knowledge/status');if(!response.ok)throw new Error('无法读取本地数据表状态。');renderKnowledgeStatus(await response.json());refs.kbState.textContent='LOCAL SQLITE · READY'; } catch(error) { refs.kbState.textContent='LOCAL DB · UNAVAILABLE';setKnowledgeMessage(error.message||'无法读取本地数据表状态。',true); } }
function renderKnowledgeHits(hits) { const safeHits=Array.isArray(hits)?hits:[];refs.kbResults.innerHTML=safeHits.map(hit=>`<article class="knowledge-hit"><h3>${escapeHTML(hit.documentTitle)} <span class="mono">${escapeHTML(hit.documentType)}</span></h3><p>${escapeHTML(hit.snippet)}</p><div class="knowledge-hit-meta">${escapeHTML(hit.department||'未分类')} · v${escapeHTML(hit.version||'1')} · ${escapeHTML(hit.sectionTitle||'正文')} · score ${Number(hit.score)||0}</div></article>`).join(''); }
async function searchKnowledge() { const query=refs.kbQuery.value.trim();if(!query){refs.kbQuery.focus();return;} refs.kbSearch.disabled=true;setKnowledgeMessage('正在从本地 documents 与 document_chunks 检索…');try { const response=await fetch('/api/knowledge/search',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({query,status:'active'})});const body=await response.json().catch(()=>({}));if(!response.ok)throw new Error(body.error||'内部资料检索失败。');renderKnowledgeStatus(body.status);renderKnowledgeHits(body.hits);setKnowledgeMessage((body.hits||[]).length?`已找到 ${(body.hits||[]).length} 条可回溯内部证据（${body.retrieval}）。`:'没有匹配的内部资料。'); } catch(error) { setKnowledgeMessage(error.message||'内部资料检索失败。',true); } finally { refs.kbSearch.disabled=false; } }
async function uploadKnowledge(event) { event.preventDefault();const file=refs.kbFile.files&&refs.kbFile.files[0];if(!file){setKnowledgeMessage('请选择要导入的文件。',true);return;}refs.kbUpload.disabled=true;setKnowledgeMessage(`正在导入 ${file.name} 并建立可检索分块…`);try { const formData=new FormData();formData.append('file',file);formData.append('documentType',refs.kbType.value);formData.append('department',refs.kbDepartment.value);formData.append('version',refs.kbVersion.value);formData.append('status','active');formData.append('accessLevel','team');const response=await fetch('/api/knowledge/upload',{method:'POST',body:formData});const body=await response.json().catch(()=>({}));if(!response.ok)throw new Error(body.error||'文件导入失败。');renderKnowledgeStatus(body.status);refs.kbForm.reset();refs.kbVersion.value='1';setKnowledgeMessage(`已导入「${body.document.title}」，生成 ${body.document.chunkCount} 个可回溯分块。`); } catch(error) { setKnowledgeMessage(error.message||'文件导入失败。',true); } finally { refs.kbUpload.disabled=false; } }
async function seedKnowledge() { refs.kbSeed.disabled=true;setKnowledgeMessage('正在写入 AI Agent 行业模拟资料…');try { const response=await fetch('/api/knowledge/seed',{method:'POST'});const body=await response.json().catch(()=>({}));if(!response.ok)throw new Error(body.error||'加载模拟样本失败。');renderKnowledgeStatus(body.status);setKnowledgeMessage(body.created?'已导入 AI Agent 行业模拟样本：研究笔记、3 份访谈、8 条工单、PRD 与试点指标。它们均标注为“模拟”。':'模拟样本已存在，未重复写入。所有样本都仅用于演示。'); } catch(error) { setKnowledgeMessage(error.message||'加载模拟样本失败。',true); } finally { refs.kbSeed.disabled=false; } }
refs.kbForm.addEventListener('submit',uploadKnowledge);refs.kbSeed.addEventListener('click',seedKnowledge);refs.kbSearch.addEventListener('click',searchKnowledge);refs.kbQuery.addEventListener('keydown',event=>{if(event.key==='Enter'){event.preventDefault();searchKnowledge();}});refreshKnowledgeStatus();loadAgentMemory();
function renderAgentMemory(data={}) {
  const context=data.context||{},policy=context.policy||{};const events=Array.isArray(context.recentEvents)?context.recentEvents:[];const compressed=Boolean(context.compressionTriggered);const budget=Number(policy.contextWindowTokens)||0;const used=Number(context.estimatedTokens)||0;
  refs.memoryState.textContent=compressed?'CONTEXT COMPRESSED':'RAW CONTEXT READY';refs.memoryMetrics.innerHTML=[[''+(Number(data.runs)||0),'RUNS'],[''+(Number(data.events)||0),'TOOL EVENTS'],[`${used} / ${budget||'—'}`,'EST. TOKENS'],[compressed?'ON':'OFF','COMPRESSION']].map(([value,label])=>`<div class="memory-metric"><strong>${escapeHTML(value)}</strong><span>${escapeHTML(label)}</span></div>`).join('');
  refs.memoryPinned.innerHTML=asArray(context.pinnedFacts).map(item=>`<li>${escapeHTML(item)}</li>`).join('')||'<li>尚无固定事实。</li>';refs.memorySummary.textContent=context.compressedToolSummary||'暂无需要压缩的较早工具轨迹。';
  refs.memoryEvents.innerHTML=events.length?events.map(item=>`<article class="memory-event"><strong>${escapeHTML(item.tool||item.category||'tool')}</strong><span>${escapeHTML(item.outputSummary||item.inputSummary||'无摘要')}</span></article>`).join(''):'<div class="empty-evidence">完成一次研究后，这里会记录 Agent 的工具轨迹。</div>';
}
async function loadAgentMemory(question='') { try { const response=await fetch(`/api/agent-memory/overview?question=${encodeURIComponent(question)}`);const body=await response.json().catch(()=>({}));if(!response.ok)throw new Error(body.error||'无法读取 Agent Memory。');renderAgentMemory(body); } catch(error) { refs.memoryState.textContent='MEMORY UNAVAILABLE';refs.memorySummary.textContent=error.message||'无法读取 Agent Memory。'; } }function setHistoryMessage(message='',isError=false) { refs.historyMessage.textContent=message;refs.historyMessage.classList.toggle('error',isError); }
function historyTypeLabel(type='') { return ({document:'文档',ticket:'客服工单',interview:'客户访谈',insight:'需求洞察',prd:'PRD',research_note:'研究笔记',dataset:'数据集'})[type]||type||'资料'; }
function renderHistoryRecords(records) { const items=Array.isArray(records)?records:[];refs.historyResults.innerHTML=items.length?items.map(item=>`<article class="history-row"><h3>${escapeHTML(historyTypeLabel(item.type))} · ${escapeHTML(item.title)}</h3><p>${escapeHTML(item.summary)}</p><div class="history-meta">${escapeHTML(item.metadata||'')} · relevance ${Number(item.score)||0}</div></article>`).join(''):'<div class="empty-evidence">没有找到符合条件的历史资料。</div>'; }
async function searchHistory() { refs.historySearch.disabled=true;setHistoryMessage('正在跨本地文档、工单、访谈、洞察、PRD、笔记和数据集检索…');try { const response=await fetch('/api/knowledge/history',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({query:refs.historyQuery.value.trim(),type:refs.historyType.value})});const body=await response.json().catch(()=>({}));if(!response.ok)throw new Error(body.error||'历史资料检索失败。');renderHistoryRecords(body.records);refs.historyState.textContent=`${Number(body.total)||0} RECORDS`;setHistoryMessage(body.query?`已从全部历史资料中找到 ${Number(body.total)||0} 条匹配记录。`:`已列出 ${Number(body.total)||0} 条最近载入的历史资料。`); } catch(error) { setHistoryMessage(error.message||'历史资料检索失败。',true); } finally { refs.historySearch.disabled=false; } }
function renderProductEvidence(evidence) { const items=Array.isArray(evidence)?evidence:[];return items.length?items.map(item=>`<span class="evidence-pill ${item.type==='gap'?'gap':''}" title="${escapeHTML(item.type||'evidence')}">${escapeHTML(item.label||'资料待补充')}</span>`).join(''):'<span class="evidence-pill gap">证据待补充</span>'; }
function renderProductItems(target,items,render) { const safe=Array.isArray(items)?items:[];target.innerHTML=safe.length?safe.map(render).join(''):'<div class="empty-evidence">尚无对应资料。</div>'; }
function scenarioFieldLabel(field='') {
  return ({trigger:'触发条件',currentGap:'现状缺口',outcome:'期望结果',actor:'使用者',context:'具体情境',inputs:'输入材料',currentWorkflow:'现状流程',impact:'影响与代价',proposedWorkflow:'拟议步骤',acceptance:'验收条件',exceptions:'异常边界',validation:'验证动作'})[field]||field||'场景背景';
}
function renderScenario(item={}) {
  item=item&&typeof item==='object'?item:{};
  const links=asArray(item.evidenceLinks).filter(link=>link&&typeof link==='object');
  const sourcePages=new Set(links.map(link=>safeSourceURL(link.url)).filter(Boolean));
  const basis=({research_hypothesis:'研究推导 · 待用户验证',unverified_hypothesis:'产品假设 · 未验证',needs_research:'资料不足 · 待补研究'})[item.basis]||'历史场景 · 待重新验证';
  const list=(title,items,ordered=false)=>{const values=asArray(items).filter(value=>typeof value==='string'&&value.trim());const tag=ordered?'ol':'ul';return `<section><h5>${escapeHTML(title)}</h5>${values.length?`<${tag}>${values.map(value=>`<li>${escapeHTML(value)}</li>`).join('')}</${tag}>`:'<p class="scenario-missing">待补充，不能据此进入开发。</p>'}</section>`;};
  const evidence=links.length?`<p class="scenario-reference-count">${links.length} 条关联摘录${sourcePages.size?` · ${sourcePages.size} 个来源页面`:''}。同源摘录不构成独立交叉验证；引用是场景线索，不代表客户已确认。</p>${links.map(link=>{const url=safeSourceURL(link.url);return `<article class="scenario-evidence"><strong>${escapeHTML(link.id||'证据')} → ${escapeHTML(scenarioFieldLabel(link.field))}</strong><blockquote>${escapeHTML(link.quote||'未提供可核对原文')}</blockquote><p>${escapeHTML(link.explanation||'支持关系待补充')}</p><small>${url?`<a href="${escapeHTML(url)}" target="_blank" rel="noopener noreferrer">${escapeHTML(link.sourceTitle||url)} ↗</a>`:escapeHTML(link.sourceTitle||'来源页面待补充')}${link.publishedAt?` · ${escapeHTML(link.publishedAt)}`:''}</small></article>`;}).join('')}`:`<p class="scenario-missing">尚无字段级原文关联。下列旧引用不能单独证明触发条件或用户痛点。</p>${renderProductEvidence(item.evidence)}`;
  return `<article class="scenario-card"><header><h4>${escapeHTML(item.name||'待定义场景')} · ${escapeHTML(item.actor||'目标用户待确认')}</h4><span class="scenario-basis">${escapeHTML(basis)}</span></header>${item.context?`<p class="scenario-context">${escapeHTML(item.context)}</p>`:''}<dl class="scenario-core"><div><dt>触发条件</dt><dd>${escapeHTML(item.trigger||'尚无依据，待调研')}</dd></div><div><dt>现状缺口</dt><dd>${escapeHTML(item.currentGap||'尚无依据，待调研')}</dd></div><div><dt>期望结果</dt><dd>${escapeHTML(item.outcome||'待定义可观察结果')}</dd></div></dl><details class="scenario-detail"><summary>展开操作、边界与验证方式</summary>${list('输入材料',item.inputs)}${list('现状工作流（研究线索，待确认）',item.currentWorkflow,true)}<section><h5>影响与代价</h5><p>${escapeHTML(item.impact||'影响程度与基线待真实用户验证，不预设收益数字。')}</p></section>${list('拟议工作流（产品方案，非已验证事实）',item.proposedWorkflow,true)}${list('异常与边界',item.exceptions)}${list('候选验收条件',item.acceptance)}<section><h5>下一步验证</h5><p>${escapeHTML(item.validation||'尚缺验证动作，请补充目标用户、要观察的行为和判定方式。')}</p></section></details><details class="scenario-detail"><summary>查看证据原文与字段关联${links.length?`（${links.length} 条摘录）`:''}</summary>${evidence}</details></article>`;
}
function renderScenarioLists(scenarios) {
  if(refs.deliveryScenarios)renderProductItems(refs.deliveryScenarios,scenarios,renderScenario);
  if(refs.productScenarios)renderProductItems(refs.productScenarios,scenarios,renderScenario);
}
function syncScenarioRebuildState(message='',isError=false) {
  const button=$('scenarioRebuildButton'),status=$('scenarioRebuildStatus');if(!button||!status)return;
  const eligible=Boolean(currentResult&&currentDeliveryCopilot&&deliveryResearchResult===currentResult&&asArray(currentResult.evidence).length);
  const loading=Boolean(scenarioRebuildRequest&&scenarioRebuildRequest.result===currentResult&&scenarioRebuildRequest.delivery===currentDeliveryCopilot);
  button.disabled=!eligible||loading;button.textContent=loading?'正在重建场景…':'用本轮证据重建场景';
  status.textContent=message||(loading?'正在将本轮研究问题与证据关联到具体操作场景…':eligible?'仅更新场景分析，保留已编辑 PRD；不会生成图片。':'需要本轮已完成研究及有效证据，才能重建场景。');status.classList.toggle('is-error',Boolean(isError));
}
async function rebuildResearchScenarios() {
  if(!currentResult||!currentDeliveryCopilot||deliveryResearchResult!==currentResult||!asArray(currentResult.evidence).length){syncScenarioRebuildState('请先完成含有效证据的研究，再重建该问题的场景。',true);return;}
  if(scenarioRebuildRequest&&scenarioRebuildRequest.result===currentResult&&scenarioRebuildRequest.delivery===currentDeliveryCopilot)return;
  const request={result:currentResult,delivery:currentDeliveryCopilot,runId:currentResult.runId};scenarioRebuildRequest=request;syncScenarioRebuildState();
  const isCurrent=()=>scenarioRebuildRequest===request&&currentResult===request.result&&currentResult.runId===request.runId&&currentDeliveryCopilot===request.delivery&&deliveryResearchResult===request.result;
  try{
    const response=await fetch('/api/research/scenarios',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(request.result)});const body=await response.json().catch(()=>({}));if(!isCurrent())return;
    if(!response.ok)throw new Error(body.error||'场景重建失败，原有内容已保留。');if(!Array.isArray(body.scenarios)||!body.scenarios.length)throw new Error('未返回有效场景，原有内容已保留。');
    request.result.scenarios=body.scenarios;request.result.scenarioStatus=body.scenarioStatus||'';request.delivery.discovery=request.delivery.discovery||{};request.delivery.discovery.scenarios=body.scenarios;
    deliveryDraftDirty=true;renderScenarioLists(body.scenarios);scenarioRebuildRequest=null;syncScenarioRebuildState(`${body.scenarioStatus||'场景已根据本轮证据重建。'} 已保留人工编辑的 PRD；保存草案后写入本地。`);
  }catch(error){if(!isCurrent())return;scenarioRebuildRequest=null;syncScenarioRebuildState(error.message||'场景重建失败，原有内容已保留。',true);}
  finally{if(scenarioRebuildRequest===request){scenarioRebuildRequest=null;syncScenarioRebuildState();}}
}
function renderProductAnalysis(data) {
  const summary=data&&data.summary?data.summary:{};const prd=data&&data.prd?data.prd:{};
  refs.productSummaryTitle.textContent=data.title||'AI Agent 产品发现包';refs.productSummary.textContent=`${summary.opportunity||''} ${summary.decision||''}`.trim();refs.productSummaryEvidence.innerHTML=renderProductEvidence(summary.evidence);
  renderProductItems(refs.productFeedback,data.userFeedback,item=>`<article class="product-item"><h4>${escapeHTML(item.audience)}</h4><p>${escapeHTML(item.feedback)}</p>${renderProductEvidence(item.evidence)}</article>`);
  renderProductItems(refs.productDemand,data.demandResearch,item=>`<article class="product-item"><h4>${escapeHTML(item.need)} <span class="priority">${escapeHTML(item.confidence||'待验证')}</span></h4><p>${escapeHTML(item.finding)}</p>${renderProductEvidence(item.evidence)}</article>`);
  renderProductItems(refs.productScenarios,data.scenarios,renderScenario);
  renderProductItems(refs.productPainPoints,data.painPoints,item=>`<article class="product-item"><h4>${escapeHTML(item.title)} <span class="priority">${escapeHTML(item.severity)}</span></h4><p>${escapeHTML(item.description)}</p>${renderProductEvidence(item.evidence)}</article>`);
  renderProductItems(refs.productFeatures,data.featurePlan,item=>`<article class="product-item"><h4><span class="priority">${escapeHTML(item.priority)}</span>${escapeHTML(item.feature)}</h4><p>${escapeHTML(item.outcome)}<br>范围：${escapeHTML(item.scope)}</p>${renderProductEvidence(item.evidence)}</article>`);
  refs.productPrdTitle.textContent=prd.name||'PRD 草案';refs.productPrdProblem.textContent=`问题：${prd.problem||'待补充。'}`;
  const prdBlocks=[['目标',prd.goals],['功能需求',prd.functionalRequirements],['目标用户',prd.targetUsers],['非目标',prd.nonGoals],['成功指标',prd.successMetrics],['风险与待验证项',prd.risks]];
  refs.productPrd.innerHTML=prdBlocks.map(([title,items])=>`<section class="prd-block"><h4>${escapeHTML(title)}</h4><ul>${(Array.isArray(items)?items:[]).map(item=>`<li>${escapeHTML(item)}</li>`).join('')}</ul></section>`).join('');refs.productPrdEvidence.innerHTML=renderProductEvidence(prd.evidence);
  refs.productOutput.hidden=false;refs.productStatus.textContent=(data.scope||'')+' '+(data.dataNotice||'');refs.productStatus.classList.remove('error');
}
function asArray(value) { return Array.isArray(value)?value:[]; }
function bulletList(target,items) { target.innerHTML=asArray(items).map(item=>`<li>${escapeHTML(item)}</li>`).join('')||'<li>待补充</li>'; }
function sectionList(title,items) { const safe=asArray(items);return safe.length?`<h5>${escapeHTML(title)}</h5><ul>${safe.map(item=>`<li>${escapeHTML(item)}</li>`).join('')}</ul>`:''; }
function clipVisualText(value,limit=42) { const chars=[...String(value||'')];return chars.length>limit?chars.slice(0,limit).join('')+'…':chars.join(''); }
function svgSafe(value) { return String(value||'').replace(/[&<>"']/g,char=>({"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&apos;"})[char]); }
function renderFlowNodes(items,empty='尚未生成流程图。') { const flow=asArray(items);return flow.map((item,index)=>`<article class="prd-flow-node"><strong>${escapeHTML(item.id||String(index+1).padStart(2,'0'))} · ${escapeHTML(item.title)}</strong><span>Owner · ${escapeHTML(item.owner)}</span><span>Output · ${escapeHTML(item.output)}</span></article>${index<flow.length-1?'<span class="prd-flow-arrow" aria-hidden="true">→</span>':''}`).join('')||`<div class="empty-evidence">${escapeHTML(empty)}</div>`; }
function renderDeliveryVisuals(visuals) {
  currentDeliveryVisuals=visuals||null;const flow=asArray(visuals&&visuals.flow);const productFlow=asArray(visuals&&visuals.productFlow);const sop=asArray(visuals&&visuals.sop);const plan=visuals&&visuals.testPlan?visuals.testPlan:{};
  refs.prdFlowDiagram.innerHTML=renderFlowNodes(flow,'尚未生成标准交付工作流。');
  refs.productFlowDiagram.innerHTML=renderFlowNodes(productFlow,'尚未生成产品内业务流程。');
  refs.sopSteps.innerHTML=sop.map(item=>`<article class="sop-step"><div class="sop-order">${Number(item.order)||'—'}</div><div><h4>${escapeHTML(item.title)} <span class="priority">${escapeHTML(item.owner)}</span></h4><p>${escapeHTML(item.action)}</p><div class="sop-meta">输入：${escapeHTML(item.input)}<br>输出：${escapeHTML(item.output)}<br>质量门：${escapeHTML(item.qualityGate)}</div></div></article>`).join('')||'<div class="empty-evidence">尚未生成 SOP。</div>';
  refs.testRoles.innerHTML=asArray(plan.roles).map(item=>`<article class="test-card"><h4>${escapeHTML(item.role)}</h4><p><strong>${escapeHTML(item.owner)}</strong></p><p>怎么测：${escapeHTML(item.howToTest)}</p><p class="sop-meta">通过标准：${escapeHTML(item.exit)}</p></article>`).join('')||'<div class="empty-evidence">尚未生成测试角色。</div>';
  refs.testDataSets.innerHTML=asArray(plan.dataSets).map(item=>`<article class="test-card"><h4>${escapeHTML(item.name)}</h4><p>${escapeHTML(item.purpose)}</p><p class="sop-meta">样例：${escapeHTML(item.example)}<br>边界：${escapeHTML(item.notice)}</p></article>`).join('')||'<div class="empty-evidence">尚未生成测试数据。</div>';
  refs.testStrategy.textContent=plan.strategy||'待补充测试策略。';refs.testCases.innerHTML=asArray(plan.cases).map(item=>`<article class="test-case"><h4><span class="priority">${escapeHTML(item.level)}</span>${escapeHTML(item.id)} · ${escapeHTML(item.title)}</h4><ol>${asArray(item.steps).map(step=>`<li>${escapeHTML(step)}</li>`).join('')}</ol><p><strong>预期：</strong>${escapeHTML(item.expected)}</p></article>`).join('')||'<div class="empty-evidence">尚未生成测试用例。</div>';bulletList(refs.testExitRules,plan.exitRules);refs.visualDownload.disabled=!flow.length;
}
function renderGeneratedImages(images=[],message='') {
  if(!refs.generatedImageGallery)return;
  const state=currentDeliveryCopilot&&deliveryImageState(currentDeliveryCopilot);
  const activeState=state&&imageContextMatches(state)?state:null;
  const stale=Boolean(activeState&&imageInputsStale(activeState));
  if(stale)activeState.copilot.generatedImages=[];
  const items=asArray(activeState?activeState.images:images).filter(Boolean);
  const ready=items.filter(item=>item.status==='ready').length;
  const pending=items.filter(item=>['queued','generating'].includes(item.status)).length;
  const unknown=items.filter(item=>item.status==='unknown').length;
  const failed=items.filter(item=>item.status==='failed').length;
  const busy=activeState&&(activeState.submitting||activeState.polling||activeState.status==='running');
  const resume=$('generatedImageResumeButton');
  refs.generatedImageRetry.textContent='重试未完成图片';
  refs.generatedImageRetry.disabled=!currentDeliveryCopilot||coEditDeliveryLoading||stale||Boolean(busy)||Boolean(activeState&&activeState.postUnknown)||ready===2||!(failed||unknown||!items.length);
  if(resume){resume.hidden=!(activeState&&(activeState.paused||(stale&&(activeState.id||activeState.requestId))));resume.disabled=Boolean(activeState&&(activeState.submitting||activeState.polling));resume.textContent=stale?'查询旧版图片进度':'继续查询进度';}
  const summary=!items.length?'尚未生成图片。提交图片任务后，页面会分别显示原型图与系统架构图的状态。':`${ready} 张可预览 · ${pending} 张排队或生成中 · ${failed} 张失败 · ${unknown} 张结果未知。`;
  refs.generatedImageStatus.textContent=(stale?'当前 PRD 内容已经修改。下方仅是修改前版本的图片，不代表当前规格，也不会写入当前草案。旧任务可继续查询；不会自动重画或重试。 ':'')+(message||(activeState&&activeState.message)||`${summary} 图片是 AI 视觉概念，不替代结构化规格。`);
  refs.generatedImageGallery.innerHTML=items.length?items.map(item=>{
    const url=safeGeneratedImageURL(item.url);const isReady=item.status==='ready'&&url;
    const status=(stale?'旧版 · ':'')+(({queued:'等待生成',generating:'正在生成',ready:'已生成',failed:'生成失败',unknown:'结果未知'})[item.status]||'尚未生成');
    const note=item.status==='unknown'?`${item.error||'等待响应超时或连接中断。'} 供应商可能仍在生成，也可能已经计费；系统不会自动重提。`:item.error||({queued:'等待前一张处理完成；不会重复提交。',generating:'图片模型正在处理，完成后会自动显示。',ready:'返回的图片地址无效，请检查服务日志。'})[item.status]||'尚未提交图片任务。';
    const download=safeGeneratedImageURL(item.downloadUrl)||url;
    return `<article class="generated-image-card ${['failed','unknown'].includes(item.status)?'failed':''}"><header><strong>${escapeHTML(item.title||'PRD 视觉方案')}</strong><span>${escapeHTML(status)}</span></header>${isReady?`<img src="${escapeHTML(url)}" alt="${escapeHTML((stale?'旧版参考：':'')+(item.title||'PRD 视觉方案'))}" loading="lazy">`:`<p class="generated-image-note">${escapeHTML(note)}</p>`}<footer><span>${isReady?(stale?'仅供旧版参考':'本机保存，可下载'):escapeHTML(item.kind||'image')}</span>${isReady?`<a href="${escapeHTML(download)}" download>${stale?'下载旧版 PNG':'下载 PNG'}</a>`:''}</footer></article>`;
  }).join(''):'<div class="empty-evidence">尚未生成图片。</div>';
}
function safeGeneratedImageURL(value='') { const text=String(value||'');return /^\/api\/generated-images\/[a-zA-Z0-9._-]+(?:\?download=1)?$/.test(text)?text:safeSourceURL(text); }
function imageContextMatches(state) { return currentDeliveryCopilot===state.copilot&&currentResult===state.research; }
function imageInputFingerprint(copilot) {
  const prd=copilot.prd||{},info=prd.info||{};
  const stable=value=>Array.isArray(value)?value.map(stable):value&&typeof value==='object'?Object.fromEntries(Object.keys(value).sort().filter(key=>value[key]!==undefined).map(key=>[key,stable(value[key])])):value;
  const text=JSON.stringify(stable({prd:{...prd,info:{prdID:info.prdID||'',product:info.product||'',feature:info.feature||''}},solution:copilot.solution||{},decision:copilot.decision||{}}));
  let first=2166136261,second=5381;
  for(let i=0;i<text.length;i++){first=Math.imul(first^text.charCodeAt(i),16777619);second=Math.imul(second,33)^text.charCodeAt(i);}
  return `v1-${text.length}-${(first>>>0).toString(16)}-${(second>>>0).toString(16)}`;
}
function imageInputsStale(state) { return Boolean(state.fingerprint&&state.fingerprint!==imageInputFingerprint(state.copilot)); }
function deliveryImageState(copilot) {
  let state=deliveryImageStates.get(copilot);
  if(!state||state.research!==currentResult){
    if(state&&state.timer)clearTimeout(state.timer);
    const awaitingReceipt=Boolean(copilot.imageRequestId);
    state={copilot,research:currentResult,id:copilot.imageJobId||'',status:copilot.imageJobStatus||'',images:asArray(copilot.generatedImages),requestId:copilot.imageRequestId||'',fingerprint:copilot.imageInputFingerprint||((copilot.imageJobId||copilot.imageRequestId||asArray(copilot.generatedImages).length)?imageInputFingerprint(copilot):''),submitting:false,polling:false,paused:awaitingReceipt,postUnknown:awaitingReceipt,message:awaitingReceipt?'此前提交结果尚未确认；只查询已有任务，不会自动重新生成。':'',timer:null};
    if(state.fingerprint)copilot.imageInputFingerprint=state.fingerprint;
    deliveryImageStates.set(copilot,state);
  }
  return state;
}
function mergeGeneratedImages(previous,incoming) {
  const byKind=new Map(asArray(previous).filter(Boolean).map(item=>[item.kind||item.id,item]));
  asArray(incoming).filter(Boolean).forEach(item=>{const key=item.kind||item.id;const prior=byKind.get(key);if(!prior||prior.status!=='ready'||item.status==='ready')byKind.set(key,item);});
  return [...byKind.values()];
}
function applyImageJob(state,job) {
  if(!job||!job.id)throw new Error('图片任务响应缺少任务 ID，不能确认是否已提交。');
  state.id=job.id;state.status=job.status;state.images=mergeGeneratedImages(state.images,job.images);state.paused=false;state.postUnknown=false;
  const timeout=Number(job.timeoutSeconds)>0?`单张最多等待 ${Math.ceil(Number(job.timeoutSeconds)/60)} 分钟。`:'';
  const ready=state.images.filter(item=>item.status==='ready').length;
  state.message=[job.status==='running'?`${ready} 张已就绪；其余图片由后台继续处理。${timeout} 页面只查询进度，不会重复提交。`:`图片任务已结束，${ready} 张可以预览。`,job.warning||''].filter(Boolean).join(' ');
  if(state.images.some(item=>item.status==='unknown'))state.message+=' 结果未知不等于供应商生成失败；可能已经计费，系统不会自动重试。';
  if(!imageContextMatches(state))return;
  state.copilot.imageJobId=state.id;state.copilot.imageJobStatus=state.status;state.copilot.imageRequestId='';state.copilot.generatedImages=imageInputsStale(state)?[]:state.images;renderGeneratedImages(state.images);
}
function scheduleImageJobPoll(state) {
  if(state.timer)clearTimeout(state.timer);state.timer=null;
  if(imageContextMatches(state)&&state.status==='running'&&!state.paused)state.timer=setTimeout(()=>{state.timer=null;void pollDeliveryImageJob(state);},2500);
}
async function pollDeliveryImageJob(state) {
  if(!imageContextMatches(state)||state.polling||state.submitting||!state.id)return;
  state.polling=true;state.paused=false;renderGeneratedImages(state.images);
  try {
    const response=await fetchImageJob(`/api/delivery-copilot/image-jobs/${encodeURIComponent(state.id)}`,{cache:'no-store'});const body=await response.json().catch(()=>({}));
    if(!response.ok)throw new Error(body.error||'暂时无法读取任务进度。');
    applyImageJob(state,body);
  }catch(error){state.paused=true;state.message=`进度查询已暂停：${error.message||'网络连接中断。'} 后台任务可能仍在运行，已生成的图片会保留；点击“继续查询进度”只查询，不重新生成。`;}
  finally{state.polling=false;if(imageContextMatches(state)){renderGeneratedImages(state.images);scheduleImageJobPoll(state);}}
}
async function resumeDeliveryImageJob(copilot=currentDeliveryCopilot) {
  if(!copilot)return;const state=deliveryImageState(copilot);if(state.submitting||state.polling)return;
  if(state.id&&!state.postUnknown){await pollDeliveryImageJob(state);return;}
  if(!state.requestId){state.paused=true;state.message='提交结果暂未确认。为避免重复生成与计费，系统不会自动重新提交；请检查后台图片任务记录。';renderGeneratedImages(state.images);return;}
  state.polling=true;state.paused=false;state.message='只读查询此前提交的图片任务，不会重新调用图片模型。';renderGeneratedImages(state.images);
  try {
    const response=await fetchImageJob(`/api/delivery-copilot/image-jobs/lookup?requestId=${encodeURIComponent(state.requestId)}`,{cache:'no-store'});const body=await response.json().catch(()=>({}));
    if(!response.ok)throw new Error(response.status===404?'暂未找到该提交记录，仍不能确认供应商是否处理；请稍后继续查询或检查后台日志。':body.error||'无法查询此前的提交记录。');
    applyImageJob(state,body);
  }catch(error){state.paused=true;state.postUnknown=true;state.message=`提交结果仍未知：${error.message||'网络异常。'} 没有重新提交图片生成请求。`;}
  finally{state.polling=false;if(imageContextMatches(state)){renderGeneratedImages(state.images);scheduleImageJobPoll(state);}}
}
async function fetchImageJob(url,options={}) {
  const controller=new AbortController();const timer=setTimeout(()=>controller.abort(),30000);
  try{return await fetch(url,{...options,signal:controller.signal});}finally{clearTimeout(timer);}
}
function newImageRequestID() { return globalThis.crypto&&typeof globalThis.crypto.randomUUID==='function'?globalThis.crypto.randomUUID():`image-${Date.now()}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`; }
async function generateDeliveryImages(copilot,options={}) {
  if(!copilot||!copilot.prd||copilot!==currentDeliveryCopilot)return;
  const state=deliveryImageState(copilot);if(state.submitting||state.polling)return;
  if(imageInputsStale(state)){renderGeneratedImages(state.images);if(state.id||state.requestId)await resumeDeliveryImageJob(copilot);return;}
  if(state.status==='running'||(!options.retry&&state.id)){await resumeDeliveryImageJob(copilot);return;}
  if(state.postUnknown){await resumeDeliveryImageJob(copilot);return;}
  const ready=state.images.filter(item=>item.status==='ready');if(ready.length===2){renderGeneratedImages(state.images);return;}
  const legacyImages=!state.id&&state.images.length>0;
  const retryKinds=options.retry&&!legacyImages?state.images.filter(item=>['failed','unknown'].includes(item.status)).map(item=>item.kind).filter(kind=>['prototype','architecture'].includes(kind)):[];
  const hasUnknown=retryKinds.some(kind=>state.images.some(item=>item.kind===kind&&item.status==='unknown'));
  if(hasUnknown&&!window.confirm('部分图片的生成结果未知，供应商可能仍在生成或已计费。再次提交可能重复生成并重复计费。确认只重试这些未完成图片吗？'))return;
  state.fingerprint=imageInputFingerprint(copilot);copilot.imageInputFingerprint=state.fingerprint;
  state.requestId=newImageRequestID();copilot.imageRequestId=state.requestId;
  state.submitting=true;state.paused=false;state.message=legacyImages?'正在恢复旧版图片记录，本次不会调用图片模型；恢复后可只重试未完成图片。':retryKinds.length?'正在提交未完成图片的重试任务，已生成图片会保留。':'正在创建或恢复当前 PRD 的图片任务；相同任务不会重复生成。';renderGeneratedImages(state.images);
  let submitted=false;
  try {
    const response=await fetchImageJob('/api/delivery-copilot/image-jobs',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({copilot,requestId:state.requestId,retryKinds,acknowledgeUnknown:hasUnknown})});
    const body=await response.json().catch(()=>({}));
    if(!response.ok){submitted=response.status>=400&&response.status<500&&response.status!==408;throw new Error(body.error||'无法创建图片任务。');}
    applyImageJob(state,body);submitted=true;
  }catch(error){
    state.postUnknown=!submitted;state.paused=!submitted;if(submitted){state.requestId='';copilot.imageRequestId='';}
    state.message=submitted?`图片任务未提交：${error.message||'请检查服务配置。'} 已有图片保持不变。`:`图片任务的提交结果未知：${error.message||'网络连接中断。'} 后台可能已经开始生成，请勿反复点击提交，系统不会自动重试。`;
  }finally{state.submitting=false;if(imageContextMatches(state)){renderGeneratedImages(state.images);scheduleImageJobPoll(state);}}
}
function renderCompetitorAnalysis(target,analysis={}) {
  if(!target)return;const profiles=asArray(analysis.competitors);const opportunities=asArray(analysis.opportunities);const status=analysis.status||'insufficient_evidence';const coverage=analysis.coverage||'当前没有可验证的竞品资料。';const questions=asArray(analysis.openQuestions);
  const profileHTML=profiles.map(profile=>`<article class="competition-profile"><h4>${escapeHTML(profile.name||'未命名竞品')} <span>${escapeHTML(profile.category||'未分类')} · ${escapeHTML(profile.confidence||'低')}置信度</span></h4><p>定位：${escapeHTML(profile.positioning||'公开资料未充分说明')}<br>目标用户：${escapeHTML(profile.targetUser||'待核验')}<br>主流程：${escapeHTML(profile.primaryWorkflow||'待核验')}<br>定价：${escapeHTML(profile.pricing||'公开资料未核验到定价')}</p>${asArray(profile.capabilities).length?`<p><strong>已观察能力：</strong>${asArray(profile.capabilities).map(escapeHTML).join('；')}</p>`:''}${asArray(profile.gaps).length?`<p><strong>待验证缺口：</strong>${asArray(profile.gaps).map(escapeHTML).join('；')}</p>`:''}${renderProductEvidence(profile.evidence)}</article>`).join('');
  const opportunityHTML=opportunities.map(item=>`<article class="competition-opportunity"><strong>待验证机会：${escapeHTML(item.hypothesis)}</strong><p>空白判断：${escapeHTML(item.whiteSpace||'待补充')}<br>验证动作：${escapeHTML(item.validation)}</p>${renderProductEvidence(item.evidence)}</article>`).join('');
  const questionHTML=questions.length?`<article class="competition-opportunity"><strong>还需补证</strong><ul>${questions.map(question=>`<li>${escapeHTML(question)}</li>`).join('')}</ul></article>`:'';
  target.innerHTML=`<p class="competition-status ${escapeHTML(status)}">${escapeHTML(coverage)}</p>${profileHTML||'<div class="empty-evidence">没有足以支持竞品画像的公开资料；系统不会用常识或模型记忆补全。</div>'}${opportunityHTML}${questionHTML}`;
}function renderPrototypeAndData(prd,solution) {
  const prototype=prd&&prd.prototype?prd.prototype:{};
  refs.prdPrototypes.innerHTML=asArray(prototype.screens).map(screen=>`<article class="prototype-card"><header class="prototype-head"><strong>${escapeHTML(screen.id)} · ${escapeHTML(screen.name)}</strong><span>${escapeHTML(screen.route)}</span></header><p class="prototype-purpose">${escapeHTML(screen.purpose)}</p>${asArray(screen.regions).map(region=>`<section class="prototype-region"><strong>${escapeHTML(region.name)}</strong><span>${escapeHTML(region.content)}</span><span>交互：${escapeHTML(region.behavior)}</span></section>`).join('')}<footer class="prototype-footer">主操作：${escapeHTML(screen.primaryAction)}<br>状态：${asArray(screen.states).map(escapeHTML).join(' · ')}</footer></article>`).join('')||'<div class="empty-evidence">尚未生成原型规格。</div>';
  refs.solutionFlowDiagram.innerHTML=renderFlowNodes(solution.solutionFlow||[],'尚未生成方案数据流。');
  refs.solutionDataModel.innerHTML=asArray(solution.dataModel).map(item=>`<article class="delivery-list-item"><strong>${escapeHTML(item.name)} <span class="priority">PK · ${escapeHTML(item.primaryKey)}</span></strong><p>${escapeHTML(item.purpose)}<br>关系：${asArray(item.relationships).map(escapeHTML).join('；')}<br>访问：${escapeHTML(item.access)}<br>保留：${escapeHTML(item.retention)}</p></article>`).join('')||'<div class="empty-evidence">尚未生成数据模型。</div>';
  refs.solutionDataDictionary.innerHTML=asArray(solution.dataDictionary).map(item=>`<tr><td>${escapeHTML(item.entity)}.${escapeHTML(item.field)}</td><td>${escapeHTML(item.type)}<br>${escapeHTML(item.required)}</td><td>${escapeHTML(item.description)}</td><td>${escapeHTML(item.example)}<br>${escapeHTML(item.classification)}</td></tr>`).join('')||'<tr><td colspan="4">尚未生成数据字典。</td></tr>';
  refs.solutionApiContracts.innerHTML=asArray(solution.apiContracts).map(api=>`<article class="delivery-list-item"><strong><span class="priority">${escapeHTML(api.method)}</span>${escapeHTML(api.path)} · ${escapeHTML(api.name)}</strong><p>授权：${escapeHTML(api.auth)}<br>请求：${asArray(api.request).map(escapeHTML).join('；')}<br>响应：${asArray(api.response).map(escapeHTML).join('；')}<br>失败与降级：${asArray(api.failures).map(escapeHTML).join('；')}</p></article>`).join('')||'<div class="empty-evidence">尚未生成接口契约。</div>';
  refs.solutionDependencies.innerHTML=asArray(solution.dependencies).map(item=>`<article class="delivery-list-item"><strong>${escapeHTML(item.name)} <span class="priority">${escapeHTML(item.needBy)}</span></strong><p>Owner：${escapeHTML(item.owner)}<br>风险：${escapeHTML(item.risk)}<br>替代方案：${escapeHTML(item.fallback)}</p></article>`).join('')||'<div class="empty-evidence">尚未生成依赖清单。</div>';
  const schedule=prd&&prd.deliveryPlan?prd.deliveryPlan:{};
  refs.deliveryEstimate.textContent=schedule.estimate||'尚未生成工期估算。';bulletList(refs.deliveryAssumptions,schedule.assumptions);
  refs.deliverySchedule.innerHTML=asArray(schedule.phases).map(phase=>`<tr><td>${escapeHTML(phase.window)}</td><td>${escapeHTML(phase.stage)}<br><span class="sop-meta">${escapeHTML(phase.owner)}</span></td><td>${escapeHTML(phase.outcome)}</td><td>进入：${escapeHTML(phase.entryGate)}<br>退出：${escapeHTML(phase.exitGate)}</td><td>${escapeHTML(phase.delayRisk)}</td></tr>`).join('')||'<tr><td colspan="5">尚未生成里程碑。</td></tr>';
}
function downloadDeliveryVisualPNG() {
  const visuals=currentDeliveryVisuals||{};const flow=asArray(visuals.flow);const sop=asArray(visuals.sop);const plan=visuals.testPlan||{};if(!flow.length)return;
  const width=1920,height=1520,cardWidth=238,margin=58;const nodeSVG=flow.map((item,index)=>{const x=margin+index*264;const arrow=index<flow.length-1?`<text x="${x+244}" y="202" fill="#91a66a" font-size="30">→</text>`:'';return `<rect x="${x}" y="116" width="${cardWidth}" height="156" rx="6" fill="#fbfcf8" stroke="#cbdab0"/><rect x="${x}" y="116" width="${cardWidth}" height="5" fill="#9abb4e"/><text x="${x+14}" y="151" fill="#45593a" font-size="18" font-family="Microsoft YaHei,Arial">${svgSafe(item.id)} · ${svgSafe(clipVisualText(item.title,14))}</text><text x="${x+14}" y="190" fill="#70806a" font-size="14" font-family="Microsoft YaHei,Arial">Owner · ${svgSafe(clipVisualText(item.owner,21))}</text><text x="${x+14}" y="226" fill="#70806a" font-size="14" font-family="Microsoft YaHei,Arial">Output · ${svgSafe(clipVisualText(item.output,20))}</text>${arrow}`;}).join('');
  const sopSVG=sop.slice(0,6).map((item,index)=>{const y=386+index*128;return `<circle cx="80" cy="${y}" r="18" fill="#edf7d5"/><text x="73" y="${y+6}" fill="#4b641b" font-size="16" font-family="Arial">${index+1}</text><text x="116" y="${y-7}" fill="#405234" font-size="19" font-family="Microsoft YaHei,Arial">${svgSafe(clipVisualText(item.title,27))} · ${svgSafe(clipVisualText(item.owner,22))}</text><text x="116" y="${y+25}" fill="#697566" font-size="15" font-family="Microsoft YaHei,Arial">${svgSafe(clipVisualText(item.action,77))}</text><text x="116" y="${y+51}" fill="#8a9283" font-size="13" font-family="Microsoft YaHei,Arial">质量门：${svgSafe(clipVisualText(item.qualityGate,73))}</text><line x1="116" y1="${y+70}" x2="914" y2="${y+70}" stroke="#e1e8d8"/>`;}).join('');
  const roleSVG=asArray(plan.roles).slice(0,5).map((item,index)=>{const y=392+index*124;return `<text x="1010" y="${y}" fill="#405234" font-size="18" font-family="Microsoft YaHei,Arial">${svgSafe(clipVisualText(item.role,18))} · ${svgSafe(clipVisualText(item.owner,20))}</text><text x="1010" y="${y+30}" fill="#697566" font-size="14" font-family="Microsoft YaHei,Arial">怎么测：${svgSafe(clipVisualText(item.howToTest,48))}</text><text x="1010" y="${y+55}" fill="#8a9283" font-size="13" font-family="Microsoft YaHei,Arial">通过：${svgSafe(clipVisualText(item.exit,52))}</text><line x1="1010" y1="${y+74}" x2="1830" y2="${y+74}" stroke="#e1e8d8"/>`;}).join('');
  const caseSVG=asArray(plan.cases).slice(0,5).map((item,index)=>{const y=1115+index*60;return `<text x="84" y="${y}" fill="#485a40" font-size="15" font-family="Microsoft YaHei,Arial">${svgSafe(item.id)} · ${svgSafe(clipVisualText(item.title,35))}</text><text x="760" y="${y}" fill="#76836e" font-size="14" font-family="Microsoft YaHei,Arial">${svgSafe(clipVisualText(item.expected,75))}</text>`;}).join('');
  const svg=`<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}"><rect width="100%" height="100%" fill="#f7f9f0"/><text x="58" y="58" fill="#2f3d28" font-size="35" font-family="Georgia,Microsoft YaHei,Arial">${svgSafe(visuals.title||'PRD Delivery Map')}</text><text x="58" y="87" fill="#75816e" font-size="16" font-family="Microsoft YaHei,Arial">Evidence-driven workflow · SOP · QA plan · generated locally</text><text x="58" y="332" fill="#50633d" font-size="22" font-family="Microsoft YaHei,Arial">团队 SOP</text><text x="1010" y="332" fill="#50633d" font-size="22" font-family="Microsoft YaHei,Arial">测试角色与方法</text><rect x="58" y="1018" width="1804" height="390" rx="8" fill="#fbfcf8" stroke="#dce6ce"/><text x="84" y="1061" fill="#50633d" font-size="22" font-family="Microsoft YaHei,Arial">关键测试用例</text><text x="760" y="1061" fill="#50633d" font-size="18" font-family="Microsoft YaHei,Arial">预期结果</text>${nodeSVG}${sopSVG}${roleSVG}${caseSVG}<text x="84" y="1455" fill="#7e8877" font-size="14" font-family="Microsoft YaHei,Arial">${svgSafe(visuals.dataNotice||'测试资料仅限合成或脱敏数据。')}</text></svg>`;
  const blob=new Blob([svg],{type:'image/svg+xml;charset=utf-8'}),url=URL.createObjectURL(blob),image=new Image();image.onload=()=>{const canvas=document.createElement('canvas');canvas.width=width;canvas.height=height;const context=canvas.getContext('2d');context.drawImage(image,0,0);canvas.toBlob(file=>{if(!file)return;const link=document.createElement('a');link.href=URL.createObjectURL(file);link.download='deepsearch-prd-delivery-map.png';link.click();setTimeout(()=>URL.revokeObjectURL(link.href),0);},'image/png');URL.revokeObjectURL(url);};image.onerror=()=>{const link=document.createElement('a');link.href=url;link.download='deepsearch-prd-delivery-map.svg';link.click();setTimeout(()=>URL.revokeObjectURL(url),0);};image.src=url;
}function activateDeliveryTab(name) { if(!refs.deliveryTabs)return;refs.deliveryTabs.querySelectorAll('[data-delivery-tab]').forEach(button=>button.classList.toggle('active',button.dataset.deliveryTab===name));document.querySelectorAll('[data-delivery-panel]').forEach(panel=>{panel.hidden=panel.dataset.deliveryPanel!==name;}); }
function renderDeliveryCopilot(data) {
  if(currentDeliveryCopilot!==data)invalidateCoEdit();
  currentDeliveryCopilot=data||null;const discovery=data&&data.discovery?data.discovery:{};const summary=discovery.summary||{};const decision=data.decision||{};const prd=data.prd||{};const info=prd.info||{};const background=prd.background||{};const solution=data.solution||{};const evaluation=data.evaluation||{};const visuals=data.visuals||{};const readiness=data.readiness||{};
  refs.deliveryTitle.textContent=data&&data.workflow?'AI Product Delivery Copilot':'产品交付工作流';refs.deliverySummary.textContent=`${summary.opportunity||''} ${summary.decision||''}`.trim();refs.deliverySummaryEvidence.innerHTML=renderProductEvidence(summary.evidence);
  refs.deliveryNotice.textContent=discovery.dataNotice||'尚未加载本地资料。';refs.decisionStatement.textContent=`${decision.decision||''} ${decision.reasoning||''}`.trim();refs.decisionEvidence.innerHTML=renderProductEvidence(decision.evidence);
  refs.deliveryWorkflow.innerHTML=asArray(data.workflow).map(stage=>`<article class="workflow-stage ${escapeHTML(stage.status||'draft')}"><strong>${escapeHTML(stage.title)}</strong><span>${escapeHTML(stage.description)}</span></article>`).join('');
  renderProductItems(refs.deliveryFeedback,discovery.userFeedback,item=>`<article class="delivery-list-item"><strong>${escapeHTML(item.audience)}</strong><p>${escapeHTML(item.feedback)}</p>${renderProductEvidence(item.evidence)}</article>`);
  refs.deliveryResearchEvidence.innerHTML=renderProductEvidence(summary.evidence);
  renderProductItems(refs.deliveryDemand,discovery.demandResearch,item=>`<article class="delivery-list-item"><strong>${escapeHTML(item.need)} <span class="priority">${escapeHTML(item.confidence||'待验证')}</span></strong><p>${escapeHTML(item.finding)}</p>${renderProductEvidence(item.evidence)}</article>`);
  renderProductItems(refs.deliveryScenarios,discovery.scenarios,renderScenario);syncScenarioRebuildState(currentResult&&deliveryResearchResult===currentResult?currentResult.scenarioStatus||'':'');
  renderProductItems(refs.deliveryPainPoints,discovery.painPoints,item=>`<article class="delivery-list-item"><strong>${escapeHTML(item.title)} <span class="priority">${escapeHTML(item.severity)}</span></strong><p>${escapeHTML(item.description)}</p>${renderProductEvidence(item.evidence)}</article>`);
  renderProductItems(refs.deliveryOptions,decision.options,item=>`<article class="delivery-list-item"><strong>${escapeHTML(item.name)} <span class="priority">${escapeHTML(item.verdict)}</span></strong><p>收益：${escapeHTML(item.benefit)}<br>取舍：${escapeHTML(item.tradeoff)}</p></article>`);renderCompetitorAnalysis(refs.deliveryCompetition,discovery.competition);bulletList(refs.deliveryMvp,decision.mvp);bulletList(refs.deliveryOutOfScope,decision.outOfScope);
  const infoFields=[['PRD ID',info.prdID],['Product',info.product],['Feature',info.feature],['Owner',info.owner],['Reviewers',info.reviewers],['Status',info.status],['Version',info.version],['Created',info.createdDate],['Target Release',info.targetRelease],['Last Updated',info.lastUpdated]];refs.prdInfo.innerHTML=infoFields.map(([label,value])=>`<div class="info-cell"><span>${escapeHTML(label)}</span><strong title="${escapeHTML(value||'待补充')}">${escapeHTML(value||'待补充')}</strong></div>`).join('');
  refs.prdBackground.innerHTML=`<h4>业务背景</h4><p>${escapeHTML(background.businessContext||'待补充')}</p><h4>问题陈述（不是研究摘要）</h4><p>${escapeHTML(background.userProblem||'待补充')}</p><h4>当前业务现状</h4><p>${escapeHTML(background.currentSituation||'待补充')}</p><h4>基线与待补充数据</h4><p>${escapeHTML(background.baselineMetrics||'待补充')}</p>${renderProductEvidence(background.evidence)}`;
  bulletList(refs.prdGoals,prd.goals);bulletList(refs.prdNonGoals,prd.nonGoals);renderProductItems(refs.prdSuccessMetrics,prd.successMetrics,item=>`<article class="delivery-list-item"><strong>${escapeHTML(item.name)}</strong><p>口径：${escapeHTML(item.definition)}<br>基线：${escapeHTML(item.baseline)}<br>目标：${escapeHTML(item.target)}<br>周期：${escapeHTML(item.window)} · Owner：${escapeHTML(item.owner)}</p></article>`);
  renderProductItems(refs.prdUsers,prd.users,item=>`<article class="delivery-list-item"><strong>${escapeHTML(item.role)} · Buyer: ${escapeHTML(item.buyer)}</strong><p>场景：${escapeHTML(item.scenario)}<br>触发：${escapeHTML(item.trigger)}<br>目标：${escapeHTML(item.goal)}</p>${renderProductEvidence(item.evidence)}</article>`);
  refs.prdArchitecture.innerHTML=asArray(prd.architecture).map(item=>`<article class="architecture-module"><strong>${escapeHTML(item.name)}</strong><ul>${asArray(item.children).map(child=>`<li>${escapeHTML(child)}</li>`).join('')}</ul></article>`).join('');bulletList(refs.prdUserFlow,prd.userFlow);bulletList(refs.prdSystemFlow,prd.systemFlow);
  refs.prdPermissions.innerHTML=asArray(prd.permissions).map(item=>`<tr><td>${escapeHTML(item.capability)}</td><td>${escapeHTML(item.user)}</td><td>${escapeHTML(item.csAgent)}</td><td>${escapeHTML(item.admin)}</td></tr>`).join('');
  refs.prdRequirements.innerHTML=asArray(prd.requirements).map(item=>`<article class="requirement-card"><h4><span class="priority">${escapeHTML(item.priority)}</span>${escapeHTML(item.id)} · ${escapeHTML(item.title)}</h4><h5>User Story</h5><p>${escapeHTML(item.userStory)}</p>${sectionList('Preconditions',item.preconditions)}${sectionList('Main Flow',item.mainFlow)}${sectionList('Business Rules',item.businessRules)}${sectionList('Exceptions',item.exceptions)}${sectionList('Permissions',item.permissions)}${sectionList('Acceptance Criteria',item.acceptance)}${sectionList('Tracking',item.tracking)}${renderProductEvidence(item.evidence)}</article>`).join('')||'<div class="empty-evidence">尚无 Requirement。</div>';
  renderProductItems(refs.prdNfr,prd.nonFunctional,item=>`<article class="delivery-list-item"><strong>${escapeHTML(item.area)} <span class="priority">${escapeHTML(item.status)}</span></strong><p>${escapeHTML(item.requirement)}<br>验证：${escapeHTML(item.verification)}</p></article>`);bulletList(refs.prdQuestions,prd.openQuestions);
  bulletList(refs.solutionPages,solution.pages);bulletList(refs.solutionApis,solution.apis);bulletList(refs.solutionData,solution.dataEntities);bulletList(refs.solutionRag,solution.ragConstraints);bulletList(refs.solutionConstraints,solution.technicalConstraints);renderDeliveryVisuals(visuals);renderPrototypeAndData(prd,{...solution,solutionFlow:visuals.solutionFlow});renderGeneratedImages(data.generatedImages||[]);
  refs.deliveryEvaluation.innerHTML=[['Tracking',evaluation.tracking],['Test Focus',evaluation.testFocus],['Launch Checklist',evaluation.launchChecklist],['Monitoring',evaluation.monitoring],['Rollback',evaluation.rollback]].map(([title,items])=>`<article class="delivery-list-item"><strong>${escapeHTML(title)}</strong><ul>${asArray(items).map(item=>`<li>${escapeHTML(item)}</li>`).join('')}</ul></article>`).join('');
  renderProductItems(refs.deliveryChains,data.chains,item=>`<article class="delivery-list-item"><strong>${escapeHTML(item.requirement)}</strong><p>Source：${escapeHTML(item.source)}<br>Insight：${escapeHTML(item.insight)}</p>${renderProductEvidence(item.evidence)}</article>`);
  refs.readinessScore.textContent=`${Number(readiness.score)||0}%`;refs.readinessVerdict.textContent=readiness.verdict||'待 Reviewer 检查';refs.readinessChecks.innerHTML=asArray(readiness.checks).map(item=>`<article class="check-item ${escapeHTML(item.status||'attention')}"><strong>${escapeHTML(item.area)} · ${escapeHTML(item.status)}</strong><span>${escapeHTML(item.note)}</span></article>`).join('');bulletList(refs.readinessActions,readiness.nextActions);
  renderCoEditState();refs.productOutput.hidden=false;refs.productExport.disabled=false;refs.productPersist.disabled=false;refs.productSubmit.disabled=true;refs.viewProduct.disabled=false;refs.productStatus.textContent=discovery.dataNotice||'交付工作流已生成。';refs.productStatus.classList.remove('error');activateDeliveryTab('overview');
}
function coEditTargetLabel(target='') { return ({'background.userProblem':'问题陈述','background.currentSituation':'当前业务现状','background.baselineMetrics':'基线与数据缺口','goals':'目标','nonGoals':'非目标','decision.decision':'方案决策','decision.mvp':'MVP 范围','decision.outOfScope':'非范围','openQuestions':'开放问题','discovery.userFeedback':'真实客户反馈','requirements.new':'新增功能需求'})[target]||target||'PRD 草案'; }
function coEditEvidenceLabel(status='') { return ({human_input:'人工输入待验证',assumption:'待验证假设',evidence_gap:'证据缺口'})[status]||'待人工确认'; }
function setCoEditStatus(message='',isError=false) { if(!refs.coeditStatus)return;refs.coeditStatus.textContent=message;refs.coeditStatus.classList.toggle('error',Boolean(isError)); }
const COEDIT_LIST_TARGETS = ['goals','nonGoals','decision.mvp','decision.outOfScope'];
function coEditDraftText(change={}) { return String(change._draftAfter??(change.after||(change.feedback&&change.feedback.feedback)||(change.requirement&&change.requirement.title)||'')).trim(); }
function coEditListTarget(target,copilot=currentDeliveryCopilot) { const object=target.startsWith('decision.')?copilot&&copilot.decision:copilot&&copilot.prd;return asArray(object&&object[target.split('.').pop()]); }
function isCoEditListReplace(change={}) { return change.operation==='replace'&&COEDIT_LIST_TARGETS.includes(change.target); }
function sameCoEditList(a,b) { return Array.isArray(a)&&Array.isArray(b)&&a.length===b.length&&a.every((item,index)=>typeof item==='string'&&item===b[index]); }
function coEditContextMatches(context) { return Boolean(context&&context.epoch===coEditEpoch&&context.delivery===currentDeliveryCopilot&&context.result===currentResult&&context.runId===(currentResult&&currentResult.runId)); }
function invalidateCoEdit() { coEditEpoch+=1;coEditRequest=null;coEditProposalContext=null;currentCoEditProposal=null;syncCoEditControls(); }
function syncCoEditControls() {
  const busy=Boolean(coEditRequest&&coEditContextMatches(coEditRequest));
  if(refs.coeditSend)refs.coeditSend.disabled=!currentDeliveryCopilot||coEditDeliveryLoading||busy;
  const button=$('scopeDetailButton');if(button){button.disabled=!currentDeliveryCopilot||coEditDeliveryLoading||busy;button.textContent=busy&&coEditRequest.mode==='scope_detail'?'正在细化范围与目标…':'细化范围与目标';}
}
function coEditReplacementItems(change={}) { return Array.isArray(change._draftItems)?change._draftItems:asArray(change.afterItems); }
function rememberCoEditDrafts() {
  asArray(currentCoEditProposal&&currentCoEditProposal.proposal&&currentCoEditProposal.proposal.changes).forEach((change,index)=>{
    if(change._applied)return;
    if(isCoEditListReplace(change)){const fields=[...refs.coeditProposal.querySelectorAll(`[data-coedit-list="${index}"]`)];if(fields.length)change._draftItems=fields.map(field=>String(field.value||'').trim());}
    else{const field=refs.coeditProposal.querySelector(`[data-coedit-after="${index}"]`);if(field)change._draftAfter=String(field.value||'').trim();}
  });
}
function renderCoEditState() {
  if(!refs.coeditConversation||!refs.coeditProposal)return;syncCoEditControls();
  refs.coeditConversation.innerHTML=coEditHistory.length?coEditHistory.map(entry=>`<article class="coedit-message ${escapeHTML(entry.role||'agent')}"><strong>${escapeHTML(entry.role==='human'?entry.actor||'协作者':'Agent')}</strong><p>${escapeHTML(entry.content||'')}</p></article>`).join(''):'<p class="coedit-empty">在左侧输入一条观点。建议写清楚：谁、在什么场景、遇到什么阻碍、造成什么代价。</p>';
  const proposal=currentCoEditProposal&&currentCoEditProposal.proposal?currentCoEditProposal.proposal:null;const changes=asArray(proposal&&proposal.changes);const pending=changes.filter(change=>!change._applied).length;
  refs.coeditProposalState.textContent=changes.length?`${pending} 条待确认`:'等待输入';refs.coeditApplyAll.disabled=!pending;
  if(!changes.length){refs.coeditProposal.innerHTML='<div class="empty-evidence">暂无修改建议。</div>';return;}
  const questions=asArray(proposal.questions);
  refs.coeditProposal.innerHTML=`${proposal.summary?`<p class="coedit-summary">${escapeHTML(proposal.summary)}</p>`:''}${questions.length?`<div class="coedit-questions"><strong>需要确认</strong><ul>${questions.map(question=>`<li>${escapeHTML(question)}</li>`).join('')}</ul></div>`:''}${changes.map((change,index)=>{
    const applied=Boolean(change._applied),replace=isCoEditListReplace(change),detail=coEditDraftText(change);
    const before=replace?`<p><b>当前整段（应用时会核对）：</b></p><ul>${asArray(change.beforeItems).map(item=>`<li>${escapeHTML(item)}</li>`).join('')||'<li>空列表</li>'}</ul>`:change.before?`<p><b>原内容：</b>${escapeHTML(change.before)}</p>`:'';
    const after=replace?`<p><b>细化后（逐项可编辑，将替换上方整段，不追加）：</b></p><ol class="coedit-list-editor">${coEditReplacementItems(change).map((item,itemIndex)=>`<li><label for="coedit-item-${index}-${itemIndex}">第 ${itemIndex+1} 项</label><textarea id="coedit-item-${index}-${itemIndex}" data-coedit-list="${index}" ${applied?'disabled':''}>${escapeHTML(item)}</textarea></li>`).join('')}</ol>`:`<label for="coedit-after-${index}">修改后（可编辑）</label><textarea id="coedit-after-${index}" data-coedit-after="${index}" ${applied?'disabled':''}>${escapeHTML(detail)}</textarea>`;
    return `<article class="coedit-change ${applied?'applied':''}"><header><strong>${escapeHTML(change.id||`C-${index+1}`)} · ${escapeHTML(coEditTargetLabel(change.target))}</strong><span>${escapeHTML(coEditEvidenceLabel(change.evidenceStatus))}</span></header>${before}${after}<p class="coedit-reason">${escapeHTML(change.rationale||'由人工输入触发，等待确认。')}</p><footer>${applied?'<span class="coedit-applied">已应用到当前草案</span>':`<button class="product-action" type="button" data-coedit-action="apply" data-coedit-index="${index}">确认并应用</button>`}</footer></article>`;
  }).join('')}`;
}
function currentCoEditAfter(index) { const field=refs.coeditProposal.querySelector(`[data-coedit-after="${index}"]`);return field?String(field.value||'').trim():coEditDraftText(asArray(currentCoEditProposal&&currentCoEditProposal.proposal&&currentCoEditProposal.proposal.changes)[index]||{}); }
function markPRDAsHumanEdited() { const prd=currentDeliveryCopilot&&currentDeliveryCopilot.prd;if(!prd)return;deliveryDraftDirty=true;prd.info=prd.info||{};prd.info.lastUpdated=`${new Date().toLocaleDateString('zh-CN')} · 人工共创待保存`;prd.info.status='draft · 人工待确认'; }
function nextHumanRequirementID(requirements=[]) { const used=new Set(asArray(requirements).map(item=>String(item&&item.id||'')));let number=asArray(requirements).length+1;while(used.has(`REQ-${String(number).padStart(3,'0')}`))number+=1;return `REQ-${String(number).padStart(3,'0')}`; }
function applyCoEditChange(index) {
  const proposal=currentCoEditProposal&&currentCoEditProposal.proposal;const change=asArray(proposal&&proposal.changes)[index];if(!change||change._applied||!currentDeliveryCopilot)return;
  if(!coEditContextMatches(coEditProposalContext)){setCoEditStatus('草案或研究任务已切换；旧建议不能应用，请重新生成建议。',true);return;}
  rememberCoEditDrafts();
  const after=currentCoEditAfter(index);const prd=currentDeliveryCopilot.prd||(currentDeliveryCopilot.prd={});const decision=currentDeliveryCopilot.decision||(currentDeliveryCopilot.decision={});const discovery=currentDeliveryCopilot.discovery||(currentDeliveryCopilot.discovery={});prd.background=prd.background||{};
  if(isCoEditListReplace(change)){
    if(!sameCoEditList(coEditListTarget(change.target),change.beforeItems)){setCoEditStatus(`${coEditTargetLabel(change.target)}已被修改，本条建议未应用；请基于当前草案重新细化。`,true);return;}
    const items=coEditReplacementItems(change);if(!items.length||items.some(item=>typeof item!=='string'||!item.trim())){setCoEditStatus('细化后的每一项都必须有内容，请补充后再确认。',true);return;}
    const next=items.map(item=>item.trim());if(new Set(next).size!==next.length){setCoEditStatus('细化列表中有完全重复的条目，请修改后再确认。',true);return;}
    const object=change.target.startsWith('decision.')?decision:prd;object[change.target.split('.').pop()]=next;change.afterItems=next;
  }else{
    if(change.target!=='requirements.new'&&change.target!=='discovery.userFeedback'&&!after){setCoEditStatus('“修改后”不能为空；请补充后再确认。',true);return;}
    switch(change.target){
      case 'background.userProblem':prd.background.userProblem=after;break;
      case 'background.currentSituation':prd.background.currentSituation=after;break;
      case 'background.baselineMetrics':prd.background.baselineMetrics=after;break;
      case 'goals':prd.goals=[...new Set([...asArray(prd.goals),after])];break;
      case 'nonGoals':prd.nonGoals=[...new Set([...asArray(prd.nonGoals),after])];break;
      case 'decision.decision':decision.decision=after;break;
      case 'decision.mvp':decision.mvp=[...new Set([...asArray(decision.mvp),after])];break;
      case 'decision.outOfScope':decision.outOfScope=[...new Set([...asArray(decision.outOfScope),after])];break;
      case 'openQuestions':prd.openQuestions=[...asArray(prd.openQuestions),after];break;
      case 'discovery.userFeedback':{const feedback=Object.assign({},change.feedback||{});feedback.audience=feedback.audience||'协作者';feedback.feedback=after||feedback.feedback||'';feedback.evidence=[{label:`人工输入 · ${feedback.audience}`,type:'human_input'}];if(!feedback.feedback){setCoEditStatus('人工观点不能为空。',true);return;}discovery.userFeedback=[...asArray(discovery.userFeedback),feedback];break;}
      case 'requirements.new':{if(!change.requirement){setCoEditStatus('这条建议缺少完整需求结构，无法直接应用。',true);return;}const requirement=Object.assign({},change.requirement);requirement.id=nextHumanRequirementID(prd.requirements);requirement.evidence=[{label:'人工输入待验证',type:'human_input'}];prd.requirements=[...asArray(prd.requirements),requirement];break;}
      default:setCoEditStatus('这条建议的目标不受支持，未修改草案。',true);return;
    }
  }
  change.after=after||coEditDraftText(change);change._applied=true;markPRDAsHumanEdited();setCoEditStatus(`已应用 ${change.id||'该条'}；这是浏览器中的草案，点击“保存 PRD 草案”后才会写入本地数据库。`);renderDeliveryCopilot(currentDeliveryCopilot);activateDeliveryTab('coedit');
}
function applyAllCoEditChanges() {
  rememberCoEditDrafts();const changes=asArray(currentCoEditProposal&&currentCoEditProposal.proposal&&currentCoEditProposal.proposal.changes);changes.forEach((change,index)=>{if(!change._applied)applyCoEditChange(index);});
  const remaining=changes.filter(change=>!change._applied);if(remaining.length)setCoEditStatus(`${remaining.length} 条建议未应用（内容冲突或缺失），已保留当前草案，请逐条检查或重新生成。`,true);
}
async function requestCoEdit(options={}) {
  if(coEditRequest&&coEditContextMatches(coEditRequest))return;
  if(!currentDeliveryCopilot||coEditDeliveryLoading){setCoEditStatus('请先完成产品交付工作流生成。',true);return;}
  const mode=options.mode==='scope_detail'?'scope_detail':'';
  const message=mode?'请只详细化当前草案的 MVP 范围、非范围、Goals、Non-goals 四块，保留原有目标用户和业务范围。MVP 逐项说明谁在什么触发条件下、输入哪些字段、执行什么步骤、输出什么结果、人工确认及失败边界、如何验收；Goals 写明可观察业务结果和测量办法，未知基线或目标标为待验证；非范围和 Non-goals 写清不做什么、原因和替代办法。不要新增未经选择的功能或编造证据，不重新搜索，不生成图片，不修改这四块以外的内容。请返回逐条可审阅的整列表替换建议，不自动应用。':String(refs.coeditInput.value||'').trim();
  const actor=refs.coeditActor.value||'协作者';if(message.length<4){setCoEditStatus('请至少输入 4 个字符，说明观点、约束、反对意见或修改目标。',true);return;}
  const inputAtSend=refs.coeditInput.value;const request={delivery:currentDeliveryCopilot,result:currentResult,runId:currentResult&&currentResult.runId,epoch:coEditEpoch,mode};coEditRequest=request;
  const isCurrent=()=>coEditRequest===request&&coEditContextMatches(request);
  setCoEditStatus(mode?'正在细化这四块范围与目标；只生成建议，不改写草案、不重搜或生成图片…':'正在把人工观点转为可审阅的 PRD 修改建议…');coEditHistory.push({role:'human',actor,content:message});currentCoEditProposal=null;coEditProposalContext=null;renderCoEditState();
  try {
    const response=await fetch('/api/delivery-copilot/coedit',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({copilot:request.delivery,actor,message,mode})});const body=await response.json().catch(()=>({}));
    if(!isCurrent())return;if(!response.ok)throw new Error(body.error||'无法生成 PRD 修改建议。');
    currentCoEditProposal=body;coEditProposalContext=request;coEditHistory.push({role:'agent',content:[body.reply,body.warning].filter(Boolean).join(' ')||'我已生成待人工确认的修改建议。'});
    if(!mode&&refs.coeditInput.value===inputAtSend)refs.coeditInput.value='';setCoEditStatus(body.warning||'已生成建议。请逐条核对原内容与修改后内容，再确认应用；未确认的内容保持不变。',Boolean(body.warning));renderCoEditState();
  }catch(error){if(!isCurrent())return;coEditHistory.push({role:'agent',content:error.message||'生成建议失败。'});setCoEditStatus(error.message||'生成建议失败。',true);renderCoEditState();}
  finally{if(coEditRequest===request){coEditRequest=null;syncCoEditControls();}}
}
function approvalLabel(role='') { return ({product_owner:'产品 Owner',design_lead:'设计负责人',engineering_lead:'研发负责人',qa_lead:'测试负责人',security_lead:'安全负责人'})[role]||role||'待指定角色'; }
function renderDeliveryState(state) {
  if(!state||!state.persisted){refs.deliveryPersistence.textContent='草案尚未保存到本地数据库。保存后将创建版本、需求条目、证据链接、评审记录与五个角色的审批队列。';refs.deliveryApprovals.innerHTML='';refs.productSubmit.disabled=true;return;}
  const score=Number(state.readinessScore)||0;refs.deliveryPersistence.textContent=`本地版本 v${state.version||'—'} · ${state.status||'draft'} · ${Number(state.requirements)||0} 条需求 · ${Number(state.evidenceLinks)||0} 个证据链接 · ${Number(state.reviewCount)||0} 条评审记录 · Readiness ${score}/100。`;
  refs.deliveryApprovals.innerHTML=asArray(state.approvals).map(item=>`<article class="delivery-list-item"><strong>${escapeHTML(approvalLabel(item.role))} <span class="priority">${escapeHTML(item.status||'pending')}</span></strong><p>${item.reviewer?`Reviewer：${escapeHTML(item.reviewer)}<br>`:''}${item.comment?escapeHTML(item.comment):'待分配审批人；仅在 Readiness 达到 80 分后可提交研发评审。'}</p></article>`).join('')||'<div class="empty-evidence">暂无审批队列。</div>';
  refs.productPersist.disabled=false;refs.productSubmit.disabled=historyApprovalLocked||state.status!=='draft'||score<80;
}
async function loadDeliveryState() { if(historyApprovalLocked)return;const epoch=researchViewEpoch;try { const response=await fetch('/api/delivery-copilot/state');const body=await response.json().catch(()=>({}));if(epoch!==researchViewEpoch||historyApprovalLocked)return;if(!response.ok)throw new Error(body.error||'无法读取 PRD 工作流状态。');renderDeliveryState(body); } catch(error) { if(epoch===researchViewEpoch)refs.deliveryPersistence.textContent=error.message||'无法读取 PRD 工作流状态。'; } }
async function downloadDeliveryPRD() {
  refs.productExport.disabled=true;refs.productStatus.textContent='正在生成包含流程、原型、数据字典、接口、测试与证据链的 Word 文档…';refs.productStatus.classList.remove('error');
  try { const options=currentDeliveryCopilot?{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(currentDeliveryCopilot)}:{};const response=await fetch('/api/delivery-copilot/export?format=docx',options);const body=await response.blob();if(!response.ok){let message='PRD Word 文档导出失败。';try{message=(await body.text())||message;}catch(_){ }throw new Error(message);}const link=document.createElement('a');link.href=URL.createObjectURL(body);link.download='deepsearch-enterprise-prd.docx';link.click();setTimeout(()=>URL.revokeObjectURL(link.href),0);refs.productStatus.textContent='已下载完整 PRD Word 文档。'; }
  catch(error) { refs.productStatus.textContent=error.message||'PRD Word 文档导出失败。';refs.productStatus.classList.add('error'); }
  finally { refs.productExport.disabled=false; }
}
async function persistDelivery() {
  if(!currentDeliveryCopilot){refs.productStatus.textContent='请先生成一份 PRD 草案。';refs.productStatus.classList.add('error');return;}
  const delivery=currentDeliveryCopilot,epoch=researchViewEpoch,payload=JSON.stringify(delivery);const isCurrent=()=>delivery===currentDeliveryCopilot&&epoch===researchViewEpoch;
  refs.productPersist.disabled=true;refs.productStatus.textContent='正在保存这个课题的 PRD 草案、需求和证据链…';refs.productStatus.classList.remove('error');
  try { const response=await fetch('/api/delivery-copilot/persist',{method:'POST',headers:{'Content-Type':'application/json'},body:payload});const body=await response.json().catch(()=>({}));if(!isCurrent())return;if(!response.ok)throw new Error(body.error||'保存 PRD 草案失败。');if(JSON.stringify(delivery)===payload)deliveryDraftDirty=false;renderDeliveryState(body);void refreshKnowledgeStatus();refs.productStatus.textContent=`已保存 PRD v${body.version}：${Number(body.requirements)||0} 条需求、${Number(body.evidenceLinks)||0} 个证据链接。${deliveryDraftDirty?'保存期间新增的修改仍需再次保存。':'可从历史研究中重新打开。'}${historyApprovalLocked?'历史浏览不恢复全局审批状态。':''}`;
  } catch(error) { if(!isCurrent())return;refs.productStatus.textContent=error.message||'保存 PRD 草案失败。';refs.productStatus.classList.add('error'); } finally { if(isCurrent())refs.productPersist.disabled=false; }
}
async function submitDeliveryForReview() {
  if(historyApprovalLocked){refs.productStatus.textContent='历史快照未恢复该版本的精确审批状态，不能通过全局审批接口提交。';return;}
  const epoch=researchViewEpoch,delivery=currentDeliveryCopilot;const isCurrent=()=>epoch===researchViewEpoch&&delivery===currentDeliveryCopilot&&!historyApprovalLocked;
  refs.productSubmit.disabled=true;refs.productStatus.textContent='正在检查 Reviewer 门槛并提交研发评审…';refs.productStatus.classList.remove('error');
  try { const response=await fetch('/api/delivery-copilot/submit-review',{method:'POST'});const body=await response.json().catch(()=>({}));if(!isCurrent())return;if(body.state)renderDeliveryState(body.state);if(!response.ok)throw new Error(body.error||'提交研发评审失败。');renderDeliveryState(body);refs.productStatus.textContent=`PRD v${body.version} 已进入研发评审，等待 ${Number(body.pendingApprovals)||0} 个角色审批。`;
  } catch(error) { if(!isCurrent())return;refs.productStatus.textContent=error.message||'提交研发评审失败。';refs.productStatus.classList.add('error'); } finally { if(isCurrent())await loadDeliveryState(); }
}
async function loadProductAnalysis(researchResult=null) {
  if(researchResult&&typeof researchResult.preventDefault==='function'){
    if(researchHistoryItem&&!globalThis.window?.confirm('将为这份历史报告生成新的交付工作流，并可能触发图片生成、消耗模型额度。确认继续吗？'))return;
    if(deliveryDraftDirty&&!globalThis.window?.confirm('生成交付工作流会替换未保存的 PRD 修改，请先保存。确定继续生成吗？'))return;
    researchResult=currentResult;
  }
  const viewEpoch=researchViewEpoch;
  coEditDeliveryLoading=true;invalidateCoEdit();
  coEditHistory=[];currentCoEditProposal=null;deliveryResearchResult=null;syncScenarioRebuildState();
  const isResearchPRD=Boolean(researchResult&&researchResult.question),startResult=currentResult;const blueprintStatus=String(researchResult&&researchResult.blueprintStatus||'');
  refs.productGenerate.disabled=true;refs.productStatus.textContent=isResearchPRD?'正在依据本次研究问题、报告、证据、冲突与限制生成专属 PRD…':'正在从本地历史资料生成交付工作流…';refs.productStatus.classList.remove('error');
  try { const options=isResearchPRD?{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(researchResult)}:{};const response=await fetch('/api/delivery-copilot',options);const body=await response.json().catch(()=>({}));if(currentResult!==startResult||researchViewEpoch!==viewEpoch)return;if(!response.ok)throw new Error(body.error||'交付工作流生成失败。');deliveryResearchResult=isResearchPRD?researchResult:null;renderDeliveryCopilot(body);deliveryDraftDirty=Boolean(body.historyNotice);if(isResearchPRD){renderDeliveryState({persisted:false});if(blueprintStatus){refs.productStatus.textContent=blueprintStatus;refs.productStatus.classList.toggle('error',!researchResult.productBlueprint);}}else{await loadDeliveryState();}if(body.historyNotice){refs.productStatus.textContent+=` ${body.historyNotice}`;refs.productStatus.classList.add('error');}if(currentResult===startResult&&researchViewEpoch===viewEpoch)void generateDeliveryImages(body); }
  catch(error) { if(currentResult!==startResult||researchViewEpoch!==viewEpoch)return;refs.productStatus.textContent=error.message||'交付工作流生成失败。';refs.productStatus.classList.add('error'); }
  finally { if(researchViewEpoch===viewEpoch){refs.productGenerate.disabled=false;coEditDeliveryLoading=false;syncCoEditControls();} }
}
$('scenarioRebuildButton')?.addEventListener('click',rebuildResearchScenarios);
$('scopeDetailButton')?.addEventListener('click',()=>{activateDeliveryTab('coedit');refs.coeditProposal.scrollIntoView({behavior:'smooth',block:'nearest'});void requestCoEdit({mode:'scope_detail'});});
function viewProductBlueprint(){if(!currentDeliveryCopilot)return;activateDeliveryTab('prd');const zone=refs.productOutput.closest('.product-zone');if(zone)zone.scrollIntoView({behavior:'smooth',block:'start'});}refs.deliveryTabs.addEventListener('click',event=>{const button=event.target.closest('[data-delivery-tab]');if(button)activateDeliveryTab(button.dataset.deliveryTab);});refs.visualDownload.addEventListener('click',downloadDeliveryVisualPNG);refs.viewProduct.addEventListener('click',viewProductBlueprint);refs.productExport.addEventListener('click',downloadDeliveryPRD);refs.productPersist.addEventListener('click',persistDelivery);refs.productSubmit.addEventListener('click',submitDeliveryForReview);refs.coeditSend.addEventListener('click',requestCoEdit);refs.coeditApplyAll.addEventListener('click',applyAllCoEditChanges);refs.generatedImageRetry.addEventListener('click',()=>generateDeliveryImages(currentDeliveryCopilot,{retry:true}));$('generatedImageResumeButton').addEventListener('click',()=>resumeDeliveryImageJob());refs.coeditProposal.addEventListener('click',event=>{const button=event.target.closest('[data-coedit-action="apply"]');if(button)applyCoEditChange(Number(button.dataset.coeditIndex));});refs.coeditInput.addEventListener('keydown',event=>{if((event.metaKey||event.ctrlKey)&&event.key==='Enter'){event.preventDefault();requestCoEdit();}});refs.historySearch.addEventListener('click',searchHistory);refs.historyQuery.addEventListener('keydown',event=>{if(event.key==='Enter'){event.preventDefault();searchHistory();}});refs.productGenerate.addEventListener('click',loadProductAnalysis);
document.querySelectorAll('.example').forEach(button=>button.addEventListener('click',()=>{refs.question.value=button.dataset.question;refs.question.focus();}));
refs.start.addEventListener('click',()=>startResearch(refs.question.value));
refs.question.addEventListener('keydown',event=>{if((event.metaKey||event.ctrlKey)&&event.key==='Enter')startResearch(refs.question.value);});
$('copyButton').addEventListener('click',async()=>{if(!currentResult)return;try{await navigator.clipboard.writeText(currentResult.report.analysis);}catch(_){}});
$('downloadButton').addEventListener('click',()=>{if(!currentResult)return;const body=`# ${currentResult.report.title}\n\n${currentResult.report.analysis}\n\n## 方法\n${currentResult.report.method}`;const blob=new Blob([body],{type:'text/markdown;charset=utf-8'}),link=document.createElement('a');link.href=URL.createObjectURL(blob);link.download='deepsearch-report.md';link.click();URL.revokeObjectURL(link.href);});
$('downloadDocxButton').addEventListener('click',async()=>{
  if(!currentResult)return;
  try{
    const response=await fetch('/api/export/docx',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(currentResult)});
    if(!response.ok){const body=await response.json().catch(()=>({}));throw new Error(body.error||'Word 文档导出失败。');}
    const blob=await response.blob(),link=document.createElement('a');link.href=URL.createObjectURL(blob);link.download='deepsearch-report.docx';link.click();setTimeout(()=>URL.revokeObjectURL(link.href),0);
  }catch(error){refs.error.textContent=error.message||'Word 文档导出失败，请稍后重试。';refs.error.classList.add('visible');}
});
renderPlan();
$('openResearchHistoryButton')?.addEventListener('click',()=>showResearchHistory({push:true}));
$('newResearchButton')?.addEventListener('click',()=>newResearchQuestion({push:true}));
$('researchHistoryForm')?.addEventListener('submit',event=>{event.preventDefault();void loadResearchHistory(0);});
$('researchHistoryResults')?.addEventListener('click',event=>{const button=event.target.closest('[data-open-research]');if(button)void openResearchHistory(button.dataset.openResearch,{push:true});});
$('researchHistoryPrevious')?.addEventListener('click',()=>loadResearchHistory(Math.max(0,historyPage.offset-historyPage.limit)));
$('researchHistoryNext')?.addEventListener('click',()=>loadResearchHistory(historyPage.offset+historyPage.limit));
$('researchAgainButton')?.addEventListener('click',researchAgain);
if(globalThis.window?.addEventListener){
  window.addEventListener('beforeunload',event=>{if(deliveryDraftDirty||researchRequest){event.preventDefault();event.returnValue='';}});
  window.addEventListener('hashchange',()=>{if(!confirmLeaveResearch()){setResearchLocation(currentResult?.runId?`#research=${encodeURIComponent(currentResult.runId)}`:'');return;}if(window.location.hash)void restoreResearchLocation({skipConfirmation:true});else newResearchQuestion({skipConfirmation:true});});
  void restoreResearchLocation();
}
