/* ViaVerdeCAR 1.3.1 — limpeza de consulta e organização por abas */
(function(){
  const g=id=>document.getElementById(id);
  const s131={active:'summary',ready:false};

  function prepare131(){
    const view=g('view-car'),toolbar=view?.querySelector('.car-toolbar');
    if(!view||!toolbar||g('carWorkspace131'))return;

    addResetButton131(toolbar);

    const workspace=document.createElement('div');
    workspace.id='carWorkspace131';
    workspace.className='car-workspace131';
    workspace.innerHTML=
      '<nav class="car-tabs131" aria-label="Seções do CAR">'+
        '<button class="car-tab131 active" data-car-tab131="summary">Resumo</button>'+
        '<button class="car-tab131" data-car-tab131="map">Mapa</button>'+
        '<button class="car-tab131" data-car-tab131="planning">Áreas e acesso</button>'+
        '<button class="car-tab131" data-car-tab131="environment">Ambiental</button>'+
        '<button class="car-tab131" data-car-tab131="docs">Histórico e saídas</button>'+
      '</nav>'+
      '<section class="car-tab-panel131 active" id="carTabSummary131"></section>'+
      '<section class="car-tab-panel131" id="carTabMap131"></section>'+
      '<section class="car-tab-panel131" id="carTabPlanning131"></section>'+
      '<section class="car-tab-panel131" id="carTabEnvironment131"></section>'+
      '<section class="car-tab-panel131" id="carTabDocs131"></section>';
    toolbar.after(workspace);

    const summary=g('carTabSummary131'),mapTab=g('carTabMap131'),planning=g('carTabPlanning131'),envTab=g('carTabEnvironment131'),docs=g('carTabDocs131');
    const strip=view.querySelector('.professional-strip');
    const mapLayout=view.querySelector('.map-layout');
    const mapPanel=mapLayout?.querySelector('.map-panel');
    const analysis=mapLayout?.querySelector('.analysis-column');
    const carCard=analysis?.querySelector('.result-card:first-child');
    const kmlCard=g('kmlAnalysisCard');
    const feature=g('feature109Grid');
    const env=view.querySelector('.environment-panel');
    const themes=view.querySelector('.themes-panel');
    const bottom=view.querySelector('.professional-bottom');
    const conference=bottom?.querySelector('.conference-panel');
    const history=bottom?.querySelector('.history-panel');
    const report=bottom?.querySelector('.report-panel');
    const toolbarActions=toolbar.querySelector('.toolbar-actions');

    if(strip)summary.appendChild(strip);
    const empty=document.createElement('div');empty.id='carEmptyTip131';empty.className='car-empty-tip131';empty.innerHTML='<strong>Uma tela mais simples.</strong> Consulte um CAR acima. Depois use as abas para ver somente o que precisa: resumo, mapa, áreas, ambiental ou documentos.';summary.appendChild(empty);
    const summaryGrid=document.createElement('div');summaryGrid.className='summary-grid131';summary.appendChild(summaryGrid);
    if(carCard){prepareCompactDataCard131(carCard);summaryGrid.appendChild(carCard)}
    if(conference)summaryGrid.appendChild(conference);

    const mapStack=document.createElement('div');mapStack.className='map-tab-stack131';mapTab.appendChild(mapStack);
    if(toolbarActions){
      const actions=document.createElement('div');actions.className='map-kml-actions131';actions.appendChild(toolbarActions);mapStack.appendChild(actions);
    }
    if(mapPanel)mapStack.appendChild(mapPanel);
    if(kmlCard)mapStack.appendChild(kmlCard);

    if(feature)planning.appendChild(feature);
    const envStack=document.createElement('div');envStack.className='environment-stack131';envTab.appendChild(envStack);
    if(env)envStack.appendChild(env);
    if(themes)envStack.appendChild(themes);

    const docsGrid=document.createElement('div');docsGrid.className='docs-grid131';docs.appendChild(docsGrid);
    if(history)docsGrid.appendChild(history);
    if(report)docsGrid.appendChild(report);

    mapLayout?.remove();
    analysis?.remove();
    bottom?.remove();

    document.querySelectorAll('[data-car-tab131]').forEach(b=>b.onclick=()=>setCarTab131(b.dataset.carTab131));
    s131.ready=true;
    setCarTab131('summary');
    updateResetState131();

    const oldRender=renderCAR,oldReset=resetCARWorkspace,oldKML=renderKML,oldEnv=renderEnvironment,oldThemes=renderThemes;
    renderCAR=function(r){oldRender(r);g('carEmptyTip131')?.classList.toggle('hidden',!!r?.found);updateResetState131();};
    resetCARWorkspace=function(){oldReset();g('carEmptyTip131')?.classList.remove('hidden');updateResetState131();collapseDataCard131();};
    renderKML=function(r){oldKML(r);updateTabBadges131();};
    renderEnvironment=function(r){oldEnv(r);updateTabBadges131();};
    renderThemes=function(r){oldThemes(r);updateTabBadges131();};
  }

  function addResetButton131(toolbar){
    if(g('resetCarBtn131'))return;
    const action=toolbar.querySelector('.car-query-wrap .input-action');
    if(!action)return;
    const b=document.createElement('button');
    b.id='resetCarBtn131';b.type='button';b.className='btn ghost car-reset131';b.textContent='Nova consulta';b.disabled=true;
    b.title='Limpa a consulta atual sem apagar clientes, imóveis ou histórico salvo';
    b.onclick=clearCurrentCAR131;action.appendChild(b);
    const note=document.createElement('div');note.className='car-clean-note131';note.textContent='“Nova consulta” limpa somente a tela e os arquivos temporários; dados salvos permanecem.';
    toolbar.querySelector('.car-query-wrap')?.appendChild(note);
  }

  function prepareCompactDataCard131(card){
    card.classList.add('car-data-card131');
    const rows=[...(card.querySelector('.data-list')?.children||[])];
    rows.forEach((row,i)=>{if(i>=7)row.classList.add('extended-data131')});
    const title=card.querySelector('.panel-title');
    const badge=g('carStatusBadge');
    if(title&&badge&&!g('toggleCarData131')){
      const tools=document.createElement('div');tools.className='panel-tools131';
      badge.parentNode?.insertBefore(tools,badge);tools.appendChild(badge);
      const b=document.createElement('button');b.id='toggleCarData131';b.className='btn ghost data-toggle131';b.type='button';b.textContent='Mais dados';
      b.onclick=()=>{const open=card.classList.toggle('show-expanded131');b.textContent=open?'Menos dados':'Mais dados'};
      tools.appendChild(b);
    }
  }

  function collapseDataCard131(){
    const card=document.querySelector('.car-data-card131');card?.classList.remove('show-expanded131');
    if(g('toggleCarData131'))g('toggleCarData131').textContent='Mais dados';
  }

  function setCarTab131(name){
    if(!s131.ready)return;
    s131.active=name;
    document.querySelectorAll('[data-car-tab131]').forEach(b=>b.classList.toggle('active',b.dataset.carTab131===name));
    document.querySelectorAll('.car-tab-panel131').forEach(p=>p.classList.remove('active'));
    const target={summary:'carTabSummary131',map:'carTabMap131',planning:'carTabPlanning131',environment:'carTabEnvironment131',docs:'carTabDocs131'}[name];
    g(target)?.classList.add('active');
    if(name==='map')setTimeout(()=>{state.map?.invalidateSize();fitMap()},70);
  }

  async function clearCurrentCAR131(){
    const has=!!(state.car||state.kml||g('carInput')?.value?.trim()||state.selectedProperty);
    if(has&&!confirm('Iniciar uma nova consulta?\n\nA tela, o mapa e os arquivos temporários serão limpos. Clientes, imóveis, histórico e áreas já salvas NÃO serão apagados.'))return;
    const btn=g('resetCarBtn131');if(btn){btn.disabled=true;btn.textContent='Limpando...'}
    try{
      try{await api()?.ClearLastCARSession()}catch(e){}
      const selector=g('carPropertySelect');if(selector)selector.value='';
      await selectCarProperty(0);
      if(g('carInput'))g('carInput').value='';
      ['areaName109','areaPurpose109','autoAreaName110','autoAreaPurpose110','autoAreaTarget110'].forEach(id=>{if(g(id))g(id).value=''});
      if(state.map){
        state.map.eachLayer(layer=>{
          try{if(layer?._url&&String(layer._url).includes('terrascope.be/wms'))state.map.removeLayer(layer)}catch(e){}
        });
        state.map.setView([-18.5,-44.0],5);
      }
      g('sessionBadge110')?.remove();
      collapseDataCard131();
      setCarTab131('summary');
      toast('Consulta limpa. Os dados salvos do imóvel e o histórico foram preservados.');
    }catch(e){toast(String(e),true)}
    finally{if(btn){btn.textContent='Nova consulta';updateResetState131()}}
  }

  function updateResetState131(){
    const b=g('resetCarBtn131');if(!b)return;
    b.disabled=!(state.car||state.kml||state.selectedProperty||g('carInput')?.value?.trim());
    updateTabBadges131();
  }

  function setCount131(tab,count){
    const b=document.querySelector('[data-car-tab131="'+tab+'"]');if(!b)return;
    let span=b.querySelector('.tab-count131');
    if(count>0){if(!span){span=document.createElement('span');span.className='tab-count131';b.appendChild(span)}span.textContent=String(count)}
    else span?.remove();
  }
  function updateTabBadges131(){
    const env=state.car?.environment||{},themes=state.car?.themes?.themes||{};
    const alerts=(Number(env.ibama_embargo_count)||0)+(Number(env.indigenous_count)||0)+(Number(env.federal_uc_count)||0)+(env.mcr_listed?1:0);
    const availableThemes=Object.values(themes).filter(x=>x?.available).length;
    setCount131('environment',alerts||availableThemes);
    setCount131('docs',(state.history||[]).length);
    setCount131('planning',document.querySelectorAll('#areaList109 .area-row109').length);
  }

  document.addEventListener('DOMContentLoaded',()=>prepare131());
})();