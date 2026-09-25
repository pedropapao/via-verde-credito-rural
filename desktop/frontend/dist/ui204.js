/* ViaVerdeCAR 2.0.4 - interface operacional em uma única tela */
(()=>{
  const E=id=>document.getElementById(id);
  const A=v=>Array.isArray(v)?v:[];
  const H=v=>String(v??'').replace(/[&<>"']/g,m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]));
  const N=(v,d=2)=>(Number(v)||0).toLocaleString('pt-BR',{minimumFractionDigits:d,maximumFractionDigits:d});
  const MONEY=v=>(Number(v)||0).toLocaleString('pt-BR',{style:'currency',currency:'BRL'});
  let current=null;
  let activeTab='summary';
  let map204=null;
  let searchTimer=null;

  const paths={
    leaf:'M19.5 3.5C13 3.8 7.2 6.7 4.8 11.7c-1.9 4 .2 7.6 3.7 8.2 4.1.7 7.4-2.3 8.3-6.5.7-3.5 1.4-6.4 2.7-9.9Z M7 18c2.4-4.4 5.3-7.2 9-9.4',
    home:'M3.5 11.2 12 4l8.5 7.2v8.3h-6v-5h-5v5h-6v-8.3Z',
    users:'M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z M3.5 20c0-3.4 2.2-5.6 5-5.6s5 2.2 5 5.6 M16 8.5a2.5 2.5 0 1 0 0-5 M15 14.6c3-.2 5.5 1.7 5.5 5.4',
    search:'M10.5 18a7.5 7.5 0 1 1 0-15 7.5 7.5 0 0 1 0 15Z M16 16l5 5',
    tree:'M12 3 8 9h2l-3 5h3l-2 4h8l-2-4h3l-3-5h2l-4-6Z M12 18v3',
    map:'M3.5 5.5 9 3l6 2.5L20.5 3v15.5L15 21l-6-2.5L3.5 21V5.5Z M9 3v15.5 M15 5.5V21',
    bank:'M3 9 12 4l9 5 M5 10v8 M9 10v8 M15 10v8 M19 10v8 M3 20h18',
    file:'M6 3h8l4 4v14H6V3Z M14 3v5h5 M9 12h6 M9 16h6',
    folder:'M3 7h7l2 2h9v10H3V7Z',
    database:'M4 6c0-2 3.6-3 8-3s8 1 8 3-3.6 3-8 3-8-1-8-3Z M4 6v6c0 2 3.6 3 8 3s8-1 8-3V6 M4 12v6c0 2 3.6 3 8 3s8-1 8-3v-6',
    info:'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z M12 10v6 M12 7h.01',
    history:'M4 9a8 8 0 1 1 2 8 M4 9V4 M4 9h5 M12 7v5l3 2',
    settings:'M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z M19 13.5l2-1.5-2-1.5-.5-2.1.8-2.3-2.4-1.4-1.7 1.7-2.2-.5L12 3.5l-2 2.4-2.2.5-1.7-1.7L3.7 6l.8 2.4L4 10.5 2 12l2 1.5.5 2.1-.8 2.4 2.4 1.4 1.7-1.7 2.2.5 2 2.3 2-2.3 2.2-.5 1.7 1.7 2.4-1.4-.8-2.4.5-2.1Z',
    help:'M9.8 9a2.4 2.4 0 1 1 3.1 2.3c-1 .4-1.4 1-1.4 2 M12 17h.01 M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z',
    alert:'M12 3 22 20H2L12 3Z M12 9v4 M12 17h.01',
    check:'M5 12.5 9.3 17 19 7',
    refresh:'M20 7v5h-5 M4 17v-5h5 M6.1 8.2A7 7 0 0 1 18.6 9 M5.4 15a7 7 0 0 0 12.5 1.8',
    download:'M12 3v11 M8 10l4 4 4-4 M4 19h16',
    print:'M7 8V3h10v5 M7 17v4h10v-4 M5 17H3v-7h18v7h-2 M17 13h.01',
    link:'M10 13a4 4 0 0 0 5.7 0l2.3-2.3a4 4 0 0 0-5.7-5.7L11 6.3 M14 11a4 4 0 0 0-5.7 0L6 13.3A4 4 0 1 0 11.7 19l1.3-1.3',
    layers:'M12 3 3 8l9 5 9-5-9-5Z M3 12l9 5 9-5 M3 16l9 5 9-5',
    clock:'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z M12 7v5l3 2',
    arrow:'M5 12h14 M14 7l5 5-5 5',
    close:'M6 6l12 12 M18 6 6 18',
    doc:'M7 3h7l4 4v14H7V3Z M14 3v5h5 M9.5 12h6 M9.5 16h6',
    chart:'M4 20V10 M10 20V4 M16 20v-7 M22 20H2'
  };
  const icon=(name,cls='')=>'<svg class="vv204-icon '+cls+'" viewBox="0 0 24 24" aria-hidden="true"><path d="'+paths[name]+'"/></svg>';

  function install(){
    document.body.classList.add('vv204');
    buildTopbar();
    buildWorkspace();
    bindGlobalKeys();
    setTimeout(renderLanding,250);
    setTimeout(renderLanding,1200);
    try{api()?.GetAppInfo?.().then(i=>{if(i?.version&&E('vv204Version'))E('vv204Version').textContent=i.version}).catch(()=>{})}catch(_){}
  }

  function buildTopbar(){
    const top=document.querySelector('.topbar');
    if(!top)return;
    top.innerHTML=`
      <button class="vv204-brand" id="vv204Home">
        <span class="vv204-brandmark">${icon('leaf')}</span>
        <span><strong>ViaVerdeCAR</strong><small>CONSULTA E ANÁLISE RURAL</small></span>
        <b id="vv204Version">2.0.5</b>
      </button>
      <div class="vv204-searchbar">
        ${icon('search')}
        <input id="vv204Search" placeholder="Digite o CAR, CPF, CNPJ, imóvel, matrícula ou município..." />
        <button id="vv204Analyze">${icon('search')}<span>ANALISAR</span></button>
      </div>
      <div class="vv204-top-actions">
        <button id="vv204Clients">${icon('users')}<span>Cadastros</span></button>
        <button id="vv204History">${icon('history')}<span>Histórico</span></button>
        <button id="vv204Settings">${icon('settings')}<span>Configurações</span></button>
        <button id="vv204Help">${icon('help')}<span>Ajuda</span></button>
      </div>
      <h1 id="pageTitle" class="vv204-compat">Visão geral</h1>
      <button id="backupBtn" class="vv204-compat"></button>
      <button id="goCarBtn" class="vv204-compat"></button>`;

    E('vv204Home').onclick=()=>{setView('dashboard');renderCurrentOrLanding()};
    E('vv204Clients').onclick=()=>setView('clients');
    E('vv204Settings').onclick=()=>setView('settings');
    E('vv204Help').onclick=()=>toast('Use a busca superior. Depois da análise, navegue pelas abas Resumo, CAR, Ambiental, Fundiário, Crédito Rural, Mapa, Documentos e Relatórios.');
    E('vv204History').onclick=()=>{setView('dashboard');if(current){activeTab='summary';renderShell();setTimeout(()=>E('vv204Timeline')?.scrollIntoView({behavior:'smooth',block:'start'}),80)}else toast('Analise um CAR para visualizar a linha do tempo do imóvel.')};
    E('vv204Analyze').onclick=()=>search(true);
    E('vv204Search').addEventListener('keydown',e=>{if(e.key==='Enter'){e.preventDefault();search(true)}});
    E('vv204Search').addEventListener('input',()=>{clearTimeout(searchTimer);searchTimer=setTimeout(()=>search(false),350)});
  }

  function buildWorkspace(){
    const dashboard=E('view-dashboard');
    if(!dashboard)return;
    [...dashboard.children].forEach(x=>x.classList.add('vv204-legacy-hidden'));
    const root=document.createElement('div');
    root.id='vv204Root';
    root.className='vv204-root';
    root.innerHTML='<section id="vv204Content"></section>';
    dashboard.prepend(root);
  }

  function bindGlobalKeys(){
    document.addEventListener('keydown',e=>{
      if((e.ctrlKey||e.metaKey)&&e.key.toLowerCase()==='k'){
        e.preventDefault();setView('dashboard');setTimeout(()=>E('vv204Search')?.focus(),50);
      }
    });
  }

  function renderCurrentOrLanding(){
    current?renderShell():renderLanding();
  }

  function renderLanding(){
    if(current)return;
    const box=E('vv204Content');if(!box)return;
    const recent=A(typeof state!=='undefined'?state.properties:[]).slice().sort((a,b)=>String(b.updated_at||'').localeCompare(String(a.updated_at||''))).slice(0,6);
    box.innerHTML=`
      <section class="vv204-landing">
        <div class="vv204-landing-main">
          <span class="vv204-landing-icon">${icon('search')}</span>
          <div>
            <span class="vv204-eyebrow">VISÃO GERAL AUTOMÁTICA</span>
            <h2>Informe o CAR uma vez. O imóvel inteiro fica organizado em uma única tela.</h2>
            <p>Depois da análise, use as abas grandes para ver somente o assunto que precisa: CAR, Ambiental, Fundiário, Crédito Rural, Mapa, Documentos ou Relatórios.</p>
          </div>
        </div>
        <div class="vv204-flow">
          <div>${icon('search')}<b>CAR</b></div><i></i><div>${icon('map')}<b>Mapa</b></div><i></i><div>${icon('tree')}<b>Ambiental</b></div><i></i><div>${icon('bank')}<b>Crédito</b></div><i></i><div>${icon('file')}<b>Dossiê</b></div>
        </div>
        <div class="vv204-recent-head"><strong>Imóveis recentes</strong><span>Abra novamente sem procurar no cadastro</span></div>
        <div class="vv204-recent">${recent.length?recent.map((p,i)=>`<button data-vv204-recent="${i}"><span>${p.car_number?'CAR VINCULADO':'CADASTRO LOCAL'}</span><strong>${H(p.name||'Imóvel')}</strong><small>${H([p.client_name,p.municipality,p.uf].filter(Boolean).join(' • '))}</small><b>${p.declared_area_ha?N(p.declared_area_ha)+' ha':'Área não informada'}</b></button>`).join(''):'<div class="vv204-empty">Nenhum imóvel cadastrado ainda.</div>'}</div>
      </section>`;
    box.querySelectorAll('[data-vv204-recent]').forEach(b=>b.onclick=()=>{
      const p=recent[Number(b.dataset.vv204Recent)];if(!p)return;
      if(p.car_number){E('vv204Search').value=p.car_number;runCAR(p.car_number,p.id,true)}
      else{setView('clients');selectClient(p.client_id)}
    });
  }

  async function search(run){
    const q=(E('vv204Search')?.value||'').trim();
    setView('dashboard');
    if(!q){current=null;renderLanding();return}
    try{
      const r=await api().SearchEverything(q);
      if(run&&r.can_analyze_car&&r.valid){
        const hit=A(r.hits).find(x=>x.property_id&&x.car);
        return runCAR(r.normalized,hit?.property_id||0,true);
      }
      if(r.mode==='cpf'||r.mode==='cnpj')return renderDocumentSearch(r);
      renderSearchResults(r);
    }catch(e){toast(String(e),true)}
  }

  function renderSearchResults(r){
    const hits=A(r.hits),box=E('vv204Content');
    box.innerHTML=`<section class="vv204-simple">
      <div class="vv204-section-head"><div><span>RESULTADOS</span><h3>${H(r.message||'Busca')}</h3></div><b>${hits.length} resultado(s)</b></div>
      <div class="vv204-result-list">${hits.length?hits.map((h,i)=>`<article><div><span>${H(h.kind||'registro')}</span><strong>${H(h.title)}</strong><small>${H(h.subtitle||h.car||h.cpf_cnpj||'')}</small></div><div>${h.car?'<button class="primary" data-vv204-an="'+i+'">Analisar CAR</button>':''}<button data-vv204-open="${i}">Abrir</button></div></article>`).join(''):'<div class="vv204-empty">Nenhum registro local encontrado.</div>'}</div>
    </section>`;
    box.querySelectorAll('[data-vv204-an]').forEach(b=>b.onclick=()=>{const h=hits[Number(b.dataset.vv204An)];runCAR(h.car,h.property_id||0,true)});
    box.querySelectorAll('[data-vv204-open]').forEach(b=>b.onclick=()=>{const h=hits[Number(b.dataset.vv204Open)];if(h.car)runCAR(h.car,h.property_id||0,true);else if(h.client_id){setView('clients');selectClient(h.client_id)}});
  }

  function renderDocumentSearch(r){
    const hits=A(r.hits),opts=A(r.official_options),box=E('vv204Content');
    box.innerHTML=`<section class="vv204-simple">
      <div class="vv204-section-head"><div><span>PESQUISA POR ${H(String(r.mode||'').toUpperCase())}</span><h3>${H(r.message)}</h3><p>${H(r.privacy_notice||'')}</p></div><b class="vv204-mono">${H(r.normalized||r.query)}</b></div>
      <div class="vv204-docsearch">
        <div><h4>Vínculos no ViaVerdeCAR</h4>${hits.length?hits.map((h,i)=>`<button data-vv204-doc-hit="${i}"><span><strong>${H(h.title)}</strong><small>${H(h.subtitle||h.car||'')}</small></span>${icon('arrow')}</button>`).join(''):'<div class="vv204-empty">Nenhum vínculo local encontrado.</div>'}</div>
        <div><h4>Fontes oficiais disponíveis</h4>${opts.map((o,i)=>`<article><div><span>${o.automatic?'AUTOMÁTICO':'OFICIAL'}</span><strong>${H(o.label)}</strong></div><p>${H(o.detail)}</p>${o.url?'<button data-vv204-source="'+i+'">Abrir fonte oficial</button>':''}</article>`).join('')}</div>
      </div>
      <div class="vv204-note"><strong>Importante:</strong> o ViaVerdeCAR não transforma coincidência de nome ou cadastro local em prova de titularidade do CAR.</div>
    </section>`;
    hits.forEach((h,i)=>{const b=box.querySelector('[data-vv204-doc-hit="'+i+'"]');if(b)b.onclick=()=>{if(h.car)runCAR(h.car,h.property_id||0,true);else if(h.client_id){setView('clients');selectClient(h.client_id)}}});
    opts.forEach((o,i)=>{const b=box.querySelector('[data-vv204-source="'+i+'"]');if(b)b.onclick=()=>openExternal(o.url)});
  }

  async function runCAR(car,propertyID,force){
    const box=E('vv204Content');
    activeTab='summary';
    box.innerHTML=`<section class="vv204-loading"><div class="vv204-spinner"></div><div><span>RAIO X AUTOMÁTICO</span><h3>Consultando o imóvel e cruzando as bases públicas</h3><p>${H(car)}</p></div><div class="vv204-progress"><i></i></div><div class="vv204-progress-labels"><b>CAR</b><b>Mapa</b><b>Ambiental</b><b>Fundiário</b><b>Crédito</b><b>Resumo</b></div></section>`;
    try{
      const r=await api().RunCARAutomation(car,Number(propertyID)||0,!!force);
      current=r;window.__vv204LastResult=r;
      state.car=r.car;
      if(r.car?.car){
        renderCAR(r.car);
        if(r.car.geojson)drawGeoJSON('car',r.car.geojson);
        if(E('carInput'))E('carInput').value=r.car.car;
      }
      await Promise.all([loadProperties(),loadDashboard()]);
      if(r.property_id)state.selectedProperty=state.properties.find(p=>p.id===r.property_id)||state.selectedProperty;
      renderShell();
    }catch(e){
      box.innerHTML='<section class="vv204-error">'+icon('alert')+'<div><strong>Não foi possível concluir o Raio X</strong><p>'+H(String(e))+'</p><span>Falha de uma fonte não é tratada como ausência de ocorrência.</span></div></section>';
      toast(String(e),true);
    }
  }

  function renderShell(){
    if(!current)return renderLanding();
    const r=current,car=r.car||{},env=r.environmental||{},xr=r.xray||{},sig=xr.sigef||{},sic=xr.sicor||{};
    const area=Number(car.area_ha||car.geometry_area_ha)||0;
    const envKeys=['ibama','mapbiomas','inpe_fire','funai','icmbio','mcr'];
    const envHits=envKeys.reduce((n,k)=>{const s=src(r,k);return n+(s.status==='hit'?(Number(s.count)||1):0)},0);
    const attention=countAttention(r);
    const carState=car.lookup_status==='cached'?'Cache local':car.lookup_status==='partial'?'Ficha parcial':car.lookup_status==='unavailable'?'Base indisponível':car.status||((car.found||car.has_geometry)?'Localizado':'Não localizado');

    E('vv204Content').innerHTML=`
      <section class="vv204-status-cards">
        ${topCard('home','CAR',carState,'Área',area?N(area)+' ha':'—','Município',H([car.municipality,car.uf].filter(Boolean).join(' / ')||'—'),toneCAR(car))}
        ${topCard('tree','Ambiental',String(envHits),'Ocorrências',envHits+' registro(s)','Resultado',envHits?'Conferir':'Sem ocorrência',envHits?'warn':'ok')}
        ${topCard('map','Fundiário',String(Number(sig.parcel_count)||0),'SIGEF',(Number(sig.parcel_count)||0)+' parcela(s)','Cobertura',sig.best_car_coverage_pct?N(sig.best_car_coverage_pct,1)+'%':'—','orange')}
        ${topCard('bank','Crédito Rural',String(Number(sic.operation_count)||0),'Operações',(Number(sic.operation_count)||0)+' registro(s)','Valor total',sic.total_credit_value?MONEY(sic.total_credit_value):'—','purple')}
        ${topCard('alert','Atenções',String(attention),'Pontos',attention+' item(ns)','Status',overall(r.overall_status),attention?'danger':'ok')}
      </section>
      <nav class="vv204-tabs" id="vv204Tabs">
        ${tabButton('summary','file','Resumo')}
        ${tabButton('car','leaf','CAR')}
        ${tabButton('environment','tree','Ambiental')}
        ${tabButton('land','map','Fundiário')}
        ${tabButton('credit','bank','Crédito Rural')}
        ${tabButton('map','map','Mapa')}
        ${tabButton('docs','folder','Documentos')}
        ${tabButton('reports','print','Relatórios')}
      </nav>
      <section id="vv204Pane" class="vv204-pane"></section>`;

    E('vv204Tabs').querySelectorAll('[data-vv204-tab]').forEach(b=>b.onclick=()=>{activeTab=b.dataset.vv204Tab;renderActiveTab()});
    renderActiveTab();
  }

  function topCard(ico,title,badge,l1,v1,l2,v2,tone){
    return `<article class="vv204-top-card ${tone}"><div class="vv204-top-card-head"><span>${icon(ico)}</span><strong>${H(title)}</strong><b>${H(badge)}</b></div><div class="vv204-top-card-body"><div><span>${H(l1)}</span><strong>${v1}</strong></div><div><span>${H(l2)}</span><strong>${v2}</strong></div></div></article>`;
  }

  function tabButton(key,ico,label){
    return `<button class="${activeTab===key?'active':''}" data-vv204-tab="${key}">${icon(ico)}<span>${H(label)}</span></button>`;
  }

  function renderActiveTab(){
    const pane=E('vv204Pane');if(!pane||!current)return;
    E('vv204Tabs')?.querySelectorAll('[data-vv204-tab]').forEach(b=>b.classList.toggle('active',b.dataset.vv204Tab===activeTab));
    if(map204){try{map204.remove()}catch(_){}map204=null}
    switch(activeTab){
      case 'car': pane.innerHTML=carTab(current);bindCarTab(current);break;
      case 'environment': pane.innerHTML=environmentTab(current);bindSourceLinks(pane);setTimeout(()=>drawMap(current,'vv204EnvMap',true),60);break;
      case 'land': pane.innerHTML=landTab(current);break;
      case 'credit': pane.innerHTML=creditTab(current);bindSourceLinks(pane);break;
      case 'map': pane.innerHTML=mapTab(current);setTimeout(()=>drawMap(current,'vv204BigMap',true),60);break;
      case 'docs': pane.innerHTML=documentsTab(current);bindDocumentActions(current);break;
      case 'reports': pane.innerHTML=reportsTab(current);bindReportActions(current);break;
      default: pane.innerHTML=summaryTab(current);bindSummary(current);setTimeout(()=>drawMap(current,'vv204SummaryMap',false),60);
    }
  }

  function summaryTab(r){
    const car=r.car||{}, env=r.environmental||{}, sig=r.xray?.sigef||{}, sic=r.xray?.sicor||{};
    const area=Number(car.area_ha||car.geometry_area_ha)||0;
    const envHits=['ibama','mapbiomas','inpe_fire','funai','icmbio','mcr'].reduce((n,k)=>{const s=src(r,k);return n+(s.status==='hit'?(Number(s.count)||1):0)},0);
    return `
      <div class="vv204-summary-grid">
        <article class="vv204-panel vv204-property-card">
          <div class="vv204-panel-head"><div><span>RESUMO DO IMÓVEL</span><h3>${H(car.property_name||state?.selectedProperty?.name||'Imóvel rural')}</h3></div><button id="vv204SummaryCar">${icon('leaf')}</button></div>
          <dl>
            <div><dt>CAR</dt><dd class="mono">${H(car.car||'—')}</dd></div>
            <div><dt>Município / UF</dt><dd>${H([car.municipality,car.uf].filter(Boolean).join(' / ')||'—')}</dd></div>
            <div><dt>Área</dt><dd>${area?N(area)+' ha':'—'}</dd></div>
            <div><dt>Módulos fiscais</dt><dd>${car.fiscal_modules?N(car.fiscal_modules,2):'—'}</dd></div>
            <div><dt>Situação</dt><dd><span class="vv204-pill ${toneCAR(car)}">${H(car.status||'—')}</span></dd></div>
            <div><dt>Condição</dt><dd>${H(car.condition||'—')}</dd></div>
            <div><dt>Inscrição</dt><dd>${H(dateBR(car.data_cadastro))}</dd></div>
            <div><dt>Atualização</dt><dd>${H(dateBR(car.data_atualizacao))}</dd></div>
          </dl>
          ${car.lookup_detail?'<div class="vv204-source-note"><strong>SICAR</strong><span>'+H(car.lookup_detail)+'</span></div>':''}
        </article>

        <article class="vv204-panel vv204-map-card">
          <div class="vv204-panel-head"><div><span>MAPA DO IMÓVEL</span><h3>Perímetro e ocorrências retornadas</h3></div><button id="vv204OpenBigMap">${icon('layers')}</button></div>
          <div id="vv204SummaryMap" class="vv204-summary-map"></div>
        </article>
      </div>

      <div class="vv204-key-results">
        <button data-go-tab="environment"><span class="green">${icon('tree')}</span><div><small>Ambiental</small><strong>${envHits?'Conferir '+envHits+' ocorrência(s)':'Sem ocorrência nas bases respondidas'}</strong></div>${icon('arrow')}</button>
        <button data-go-tab="land"><span class="orange">${icon('map')}</span><div><small>Fundiário</small><strong>${Number(sig.parcel_count)||0} parcela(s) SIGEF</strong></div>${icon('arrow')}</button>
        <button data-go-tab="credit"><span class="purple">${icon('bank')}</span><div><small>Crédito Rural</small><strong>${Number(sic.operation_count)||0} operação(ões) • ${sic.total_credit_value?MONEY(sic.total_credit_value):'sem valor agregado'}</strong></div>${icon('arrow')}</button>
      </div>

      <div class="vv204-quick-actions">
        <button id="vv204Refresh" class="primary">${icon('refresh')}<span><strong>Atualizar tudo</strong><small>Executar novamente todas as análises</small></span></button>
        <button id="vv204Kml">${icon('download')}<span><strong>Gerar KML</strong><small>Perímetro público SICAR</small></span></button>
        <button id="vv204Dossier" ${r.property_id?'':'disabled'}>${icon('file')}<span><strong>Gerar dossiê</strong><small>PDF + KML + evidências</small></span></button>
        <button id="vv204Report" ${r.property_id?'':'disabled'}>${icon('print')}<span><strong>Relatórios</strong><small>Saídas técnicas do imóvel</small></span></button>
      </div>

      <article id="vv204Timeline" class="vv204-panel vv204-timeline-panel">
        <div class="vv204-panel-head"><div><span>LINHA DO TEMPO</span><h3>Eventos encontrados para o imóvel</h3></div>${icon('clock')}</div>
        <div class="vv204-timeline">${timeline(r)}</div>
      </article>`;
  }

  function bindSummary(r){
    E('vv204SummaryCar').onclick=()=>{activeTab='car';renderShell()};
    E('vv204OpenBigMap').onclick=()=>{activeTab='map';renderShell()};
    document.querySelectorAll('[data-go-tab]').forEach(b=>b.onclick=()=>{activeTab=b.dataset.goTab;renderShell()});
    E('vv204Refresh').onclick=()=>runCAR(r.car.car,r.property_id||0,true);
    E('vv204Kml').onclick=()=>safeAction(()=>exportKML());
    E('vv204Dossier').onclick=()=>safeAction(()=>exportPackage());
    E('vv204Report').onclick=()=>{activeTab='reports';renderShell()};
  }

  function carTab(r){
    const c=r.car||{};
    const lookup=c.lookup_status==='cached'?'Cache local':c.lookup_status==='partial'?'Ficha parcial':c.lookup_status==='unavailable'?'Base indisponível':c.lookup_status==='not_found'?'Não localizado':'Consulta atual';
    return `
      <div class="vv204-tab-title"><div><span>CADASTRO AMBIENTAL RURAL</span><h2>CAR do imóvel</h2><p>Dados cadastrais e geometria retornados pela consulta pública.</p></div><span class="vv204-big-status ${toneCAR(c)}">${H(lookup)}</span></div>
      <div class="vv204-two">
        <article class="vv204-panel">
          <div class="vv204-data-grid">
            ${dataRow('Número do CAR',c.car,'mono')}
            ${dataRow('Município / UF',[c.municipality,c.uf].filter(Boolean).join(' / '))}
            ${dataRow('Nome público do imóvel',c.property_name)}
            ${dataRow('Situação',c.status)}
            ${dataRow('Condição',c.condition)}
            ${dataRow('Tipo do imóvel',c.property_type)}
            ${dataRow('Área SICAR',c.area_ha?N(c.area_ha,4)+' ha':'')}
            ${dataRow('Área geométrica',c.geometry_area_ha?N(c.geometry_area_ha,4)+' ha':'')}
            ${dataRow('Perímetro',c.perimeter_m?N(c.perimeter_m/1000,3)+' km':'')}
            ${dataRow('Módulos fiscais',c.fiscal_modules?N(c.fiscal_modules,2):'')}
            ${dataRow('Centro aproximado',(c.center_lat||c.center_lon)?Number(c.center_lat).toFixed(6)+', '+Number(c.center_lon).toFixed(6):'')}
            ${dataRow('Inscrição',dateBR(c.data_cadastro))}
            ${dataRow('Última atualização',dateBR(c.data_atualizacao))}
          </div>
        </article>
        <article class="vv204-panel vv204-source-card">
          <div class="vv204-panel-head"><div><span>SITUAÇÃO DA CONSULTA</span><h3>Fonte pública SICAR</h3></div>${icon('database')}</div>
          <div class="vv204-source-state"><span class="${toneCAR(c)}">${icon(c.public_confirmed?'check':'info')}</span><div><strong>${H(lookup)}</strong><p>${H(c.lookup_detail||'Consulta concluída.')}</p></div></div>
          <div class="vv204-source-state"><span class="${c.has_geometry?'ok':'warn'}">${icon(c.has_geometry?'check':'info')}</span><div><strong>Geometria</strong><p>${c.has_geometry?'Perímetro público disponível para mapa e KML.':'Geometria não disponível nesta consulta.'}</p></div></div>
          <div class="vv204-source-state"><span class="${c.auto_kml_path?'ok':'neutral'}">${icon(c.auto_kml_path?'check':'file')}</span><div><strong>KML SICAR</strong><p>${c.auto_kml_path?'Arquivo gerado automaticamente.':'Pode ser exportado quando a geometria estiver disponível.'}</p></div></div>
          <div class="vv204-car-actions">
            <button id="vv204CarKml" ${c.has_geometry?'':'disabled'}>${icon('download')}Gerar KML</button>
            ${!r.property_id&&c.found?'<button id="vv204CarLink">'+icon('link')+'Salvar / Vincular</button>':''}
            <button id="vv204Official" ${c.official_url?'':'disabled'}>${icon('database')}Fonte oficial</button>
          </div>
          <div id="vv204LinkBox" class="vv204-linkbox hidden"></div>
        </article>
      </div>`;
  }

  function bindCarTab(r){
    E('vv204CarKml').onclick=()=>safeAction(()=>exportKML());
    if(E('vv204Official'))E('vv204Official').onclick=()=>openExternal(r.car?.official_url);
    if(E('vv204CarLink'))E('vv204CarLink').onclick=()=>renderLinkBox(r);
  }

  function renderLinkBox(r){
    const box=E('vv204LinkBox'),clients=A(state?.clients);if(!box)return;
    box.classList.remove('hidden');
    box.innerHTML=clients.length?`<select id="vv204ClientSelect"><option value="">Selecione o cliente</option>${clients.map(c=>'<option value="'+c.id+'">'+H(c.name)+(c.cpf_cnpj?' • '+H(c.cpf_cnpj):'')+'</option>').join('')}</select><button id="vv204LinkSave">Vincular este CAR</button><small>O vínculo é confirmado pelo usuário; o aplicativo não deduz titularidade.</small>`:'<p>Cadastre um cliente antes de vincular este CAR.</p>';
    if(E('vv204LinkSave'))E('vv204LinkSave').onclick=async()=>{
      const id=Number(E('vv204ClientSelect').value)||0;if(!id){toast('Selecione o cliente.',true);return}
      try{
        const p=await api().SaveAnalyzedCARToClient(id,r.car.car);
        await Promise.all([loadClients(),loadProperties(),loadDashboard()]);
        toast('Imóvel salvo e vinculado.');
        await runCAR(r.car.car,p.id,false);
      }catch(e){toast(String(e),true)}
    };
  }

  function environmentTab(r){
    const env=r.environmental||{}, s=env.summary||{}, keys=[
      ['ibama','IBAMA • Embargos'],
      ['mapbiomas','MapBiomas • Desmatamento'],
      ['inpe_fire','INPE • Focos de calor'],
      ['funai','FUNAI • Terras Indígenas'],
      ['icmbio','ICMBio • Unidades de Conservação'],
      ['mcr','MMA • MCR/PRODES']
    ];
    const hits=keys.reduce((n,[k])=>{const x=src(r,k);return n+(x.status==='hit'?(Number(x.count)||1):0)},0);
    return `
      <div class="vv204-tab-title"><div><span>RAIO X AMBIENTAL</span><h2>Ocorrências e bases ambientais</h2><p>Uma fonte indisponível nunca é apresentada como “sem ocorrência”.</p></div><span class="vv204-big-status ${hits?'danger':'ok'}">${hits?hits+' ocorrência(s)':'Sem ocorrência nas bases respondidas'}</span></div>
      <div class="vv204-env-layout">
        <article class="vv204-panel">
          <div class="vv204-env-list">${keys.map(([k,label])=>envRow(r,k,label)).join('')}</div>
        </article>
        <article class="vv204-panel">
          <div class="vv204-panel-head"><div><span>MAPA AMBIENTAL</span><h3>Perímetro e ocorrências com geometria</h3></div>${icon('layers')}</div>
          <div id="vv204EnvMap" class="vv204-env-map"></div>
        </article>
      </div>
      <div class="vv204-metrics-row">
        <article><span>Alertas MapBiomas</span><strong>${Number(s.alerts)||0}</strong><small>${s.alert_area_in_car_ha?N(s.alert_area_in_car_ha,2)+' ha estimados no CAR':'Nenhuma área alertada calculada'}</small></article>
        <article><span>Alta prioridade</span><strong>${Number(s.high_attention_alerts)||0}</strong><small>alerta(s) para conferência</small></article>
        <article><span>Última detecção</span><strong>${H(dateBR(s.latest_detection))}</strong><small>MapBiomas Alerta</small></article>
      </div>`;
  }

  function envRow(r,key,label){
    const s=src(r,key),count=Number(s.count)||0;
    const tone=s.status==='hit'?'hit':s.status==='ok'?'ok':['unavailable','not_configured'].includes(s.status)?'off':['partial','cached'].includes(s.status)?'warn':'neutral';
    return `<div class="vv204-env-row ${tone}"><span class="dot"></span><div><strong>${H(label)}</strong><small>${H(s.detail||statusLabel(s.status))}</small></div><b>${H(statusLabel(s.status))}</b><em>${count?count+' registro(s)':'—'}</em>${s.source_url?'<button data-source-url="'+H(s.source_url)+'">'+icon('arrow')+'</button>':''}</div>`;
  }

  function landTab(r){
    const s=r.xray?.sigef||{};
    return `
      <div class="vv204-tab-title"><div><span>RAIO X FUNDIÁRIO</span><h2>SIGEF / INCRA e conferência de limites</h2><p>Resultado fundiário separado das análises ambientais e de crédito.</p></div><span class="vv204-big-status ${s.available?'ok':'neutral'}">${s.available?'Consulta realizada':'Sem resultado atual'}</span></div>
      <div class="vv204-land-grid">
        <article class="vv204-metric-large"><span>Parcelas SIGEF</span><strong>${Number(s.parcel_count)||0}</strong><small>parcelas encontradas</small></article>
        <article class="vv204-metric-large"><span>Melhor cobertura</span><strong>${s.best_car_coverage_pct?N(s.best_car_coverage_pct,1)+'%':'—'}</strong><small>sobre o CAR</small></article>
        <article class="vv204-metric-large"><span>Matrículas / registros</span><strong>${Number(s.registry_count)||0}</strong><small>retornados</small></article>
      </div>
      <article class="vv204-panel vv204-land-detail">
        <div class="vv204-panel-head"><div><span>RESULTADO</span><h3>Leitura fundiária</h3></div>${icon('map')}</div>
        <div class="vv204-land-message">${H(s.message||'Nenhum detalhe adicional retornado pelo SIGEF nesta consulta.')}</div>
        <div class="vv204-note"><strong>Importante:</strong> a sobreposição espacial é uma conferência auxiliar. Matrícula, domínio, certificação e situação registral devem ser confirmados pelos documentos e fontes competentes.</div>
      </article>`;
  }

  function creditTab(r){
    const sic=r.xray?.sicor||{}, ops=A(sic.operations), b=r.bcb||{}, market=b.market||{}, series=b.series||{}, inst=b.institutions||{}, ifd=b.ifdata||{}, bankRates=b.institution_rates||{};
    const rates=A(series.metrics).filter(x=>x.measure==='Taxa de juros'&&x.status==='available').slice(0,6);
    const macro=A(series.metrics).filter(x=>x.measure!=='Taxa de juros'&&x.status==='available').slice(0,6);
    const products=A(market.municipal_products).slice(0,6), programs=A(market.state_programs).slice(0,6), funding=A(market.national_sources).slice(0,6);
    return `
      <div class="vv204-tab-title"><div><span>CRÉDITO RURAL</span><h2>SICOR + Banco Central</h2><p>Operações ligadas ao imóvel e contexto oficial agregado do mercado.</p></div><span class="vv204-big-status purple">${Number(sic.operation_count)||0} operação(ões)</span></div>

      <div class="vv204-bcb-banner">
        <div><span>BANCO CENTRAL • DADOS ABERTOS</span><strong>Contexto oficial agregado de crédito rural</strong><small>MDCR/SICOR, SGS, instituições supervisionadas, IFData e taxas por instituição.</small></div>
        <b>${b.used_cache?'CACHE 8H':b.available?'ATUALIZADO':'PARCIAL'}</b>
      </div>

      <div class="vv204-credit-overview">
        <article><span>Operações SICOR</span><strong>${Number(sic.operation_count)||0}</strong><small>vinculadas ao contexto do imóvel</small></article>
        <article><span>Valor total SICOR</span><strong>${sic.total_credit_value?MONEY(sic.total_credit_value):'—'}</strong><small>soma das operações retornadas</small></article>
        <article><span>BCB</span><strong>${b.available?'Disponível':'Parcial'}</strong><small>${b.used_cache?'cache local de até 8h':'consulta pública atual'}</small></article>
      </div>

      <div class="vv204-credit-grid">
        <article class="vv204-panel">
          <div class="vv204-panel-head"><div><span>SICOR</span><h3>Operações encontradas</h3></div>${icon('bank')}</div>
          <div class="vv204-operation-list">${ops.length?ops.slice(0,10).map(o=>`<div><span><strong>${H(o.purpose||o.activity||o.product||'Operação SICOR')}</strong><small>${H([o.institution_name,o.program_name,o.year].filter(Boolean).join(' • '))}</small></span><b>${MONEY(o.credit_value)}</b></div>`).join(''):'<div class="vv204-empty">Nenhuma operação pública vinculada foi localizada.</div>'}</div>
        </article>

        <article class="vv204-panel">
          <div class="vv204-panel-head"><div><span>MDCR / SICOR</span><h3>Mercado no município</h3></div><button data-source-url="${H(market.source_url||'https://dadosabertos.bcb.gov.br/dataset/matrizdadoscreditorural')}">${icon('database')}</button></div>
          <div class="vv204-operation-list">${products.length?products.map(x=>`<div><span><strong>${H(x.label||x.kind||'Produto')}</strong><small>${H([x.kind,x.year].filter(Boolean).join(' • '))}</small></span><b>${MONEY(x.value)}</b></div>`).join(''):'<div class="vv204-empty">Sem produtos municipais retornados nesta consulta.</div>'}</div>
        </article>

        <article class="vv204-panel">
          <div class="vv204-panel-head"><div><span>SGS</span><h3>Taxas rurais oficiais agregadas</h3></div><button data-source-url="https://www.bcb.gov.br/estatisticas/">${icon('chart')}</button></div>
          <div class="vv204-series-list">${rates.length?rates.map(seriesRow).join(''):'<div class="vv204-empty">Séries de taxas indisponíveis nesta consulta.</div>'}</div>
        </article>

        <article class="vv204-panel">
          <div class="vv204-panel-head"><div><span>SGS</span><h3>Saldo, concessões e inadimplência</h3></div><button data-source-url="https://www.bcb.gov.br/estatisticas/">${icon('chart')}</button></div>
          <div class="vv204-series-list">${macro.length?macro.map(seriesRow).join(''):'<div class="vv204-empty">Séries macroeconômicas indisponíveis nesta consulta.</div>'}</div>
        </article>

        <article class="vv204-panel">
          <div class="vv204-panel-head"><div><span>PROGRAMAS E FONTES</span><h3>Distribuição do crédito rural</h3></div>${icon('database')}</div>
          <div class="vv204-split-list"><div><h4>Programas</h4>${programs.length?programs.map(x=>'<p><span>'+H(x.label||'Programa')+'</span><b>'+MONEY(x.value)+'</b></p>').join(''):'<small>Sem programas no recorte.</small>'}</div><div><h4>Fontes de recursos</h4>${funding.length?funding.map(x=>'<p><span>'+H(x.label||'Fonte')+'</span><b>'+MONEY(x.value)+'</b></p>').join(''):'<small>Sem fontes no recorte.</small>'}</div></div>
        </article>

        <article class="vv204-panel">
          <div class="vv204-panel-head"><div><span>INSTITUIÇÕES</span><h3>Entidades supervisionadas / IFData</h3></div><button data-source-url="${H(inst.source_url||'https://dadosabertos.bcb.gov.br/dataset/dados-cadastrais-de-entidades-autorizadas')}">${icon('bank')}</button></div>
          <div class="vv204-institutions">${A(inst.institutions).slice(0,7).map(x=>'<div><strong>'+H(x.name)+'</strong><span>'+H([x.type,x.situation,x.uf].filter(Boolean).join(' • '))+'</span></div>').join('')||'<div class="vv204-empty">Sem correspondência segura nas entidades supervisionadas.</div>'}</div>
          <div class="vv204-ifdata-note">IFData ${H(ifd.reference||'—')} • ${A(ifd.institutions).length} instituição(ões) relacionada(s)</div>
        </article>

        <article class="vv204-panel vv204-credit-wide">
          <div class="vv204-panel-head"><div><span>TAXAS POR INSTITUIÇÃO</span><h3>Médias publicadas pelo Banco Central</h3></div><button data-source-url="${H(bankRates.source_url||'https://dadosabertos.bcb.gov.br/dataset/taxas-de-juros-de-operacoes-de-credito')}">${icon('bank')}</button></div>
          <div class="vv204-bank-rates">${A(bankRates.rates).slice(0,12).map(x=>'<div><span><strong>'+H(x.institution||'Instituição')+'</strong><small>'+H([x.segment,x.modality,x.end_date].filter(Boolean).join(' • '))+'</small></span><b>'+N(x.annual_rate,2)+'% a.a.</b></div>').join('')||'<div class="vv204-empty">'+H(bankRates.message||'Nenhuma modalidade rural/agro retornada.')+'</div>'}</div>
          <div class="vv204-note">As taxas são médias observadas nas operações publicadas pelo BCB. Não representam oferta ou taxa garantida para o produtor.</div>
        </article>
      </div>`;
  }

  function seriesRow(x){
    const ch=Number(x.change_pct)||0;
    return `<div><span><strong>${H(x.label)}</strong><small>${H(x.latest_date||'')} • SGS ${H(x.sgs_code)}</small></span><b>${N(x.latest_value,2)} ${H(x.unit||'')}</b><em class="${ch>0?'up':ch<0?'down':'flat'}">${ch>0?'↑':ch<0?'↓':'→'} ${N(Math.abs(ch),1)}%</em></div>`;
  }

  function mapTab(r){
    return `
      <div class="vv204-tab-title"><div><span>MAPA</span><h2>Imóvel e camadas de análise</h2><p>O mapa ocupa a área principal para facilitar a conferência visual.</p></div><span class="vv204-big-status ok">Mapa ampliado</span></div>
      <article class="vv204-panel vv204-big-map-panel">
        <div class="vv204-panel-head"><div><span>CAMADAS</span><h3>CAR, ocorrências ambientais e glebas SICOR disponíveis</h3></div>${icon('layers')}</div>
        <div id="vv204BigMap" class="vv204-big-map"></div>
      </article>`;
  }

  function documentsTab(r){
    const c=r.car||{},p=state?.selectedProperty,k=state?.kml;
    return `
      <div class="vv204-tab-title"><div><span>DOCUMENTOS</span><h2>Arquivos e evidências do imóvel</h2><p>KMLs, evidências ambientais e pacote técnico.</p></div></div>
      <div class="vv204-doc-grid">
        <article class="vv204-panel vv204-doc-card"><span>${icon('map')}</span><div><h3>KML SICAR</h3><p>${c.has_geometry?'Geometria pública disponível para exportação.':'Geometria não disponível.'}</p></div><button id="vv204DocKml" ${c.has_geometry?'':'disabled'}>Exportar KML</button></article>
        <article class="vv204-panel vv204-doc-card"><span>${icon('folder')}</span><div><h3>KML do cliente</h3><p>${k?.geojson?'KML externo carregado para comparação.':p?'Você pode anexar um KML externo ao imóvel salvo.':'Salve/vincule o imóvel antes de anexar.'}</p></div><button id="vv204AttachKml" ${p?'':'disabled'}>Anexar KML</button></article>
        <article class="vv204-panel vv204-doc-card"><span>${icon('database')}</span><div><h3>Evidências ambientais</h3><p>Exporta os dados estruturados usados na análise ambiental.</p></div><button id="vv204Evidence" ${r.property_id?'':'disabled'}>Exportar JSON</button></article>
        <article class="vv204-panel vv204-doc-card"><span>${icon('file')}</span><div><h3>Dossiê completo</h3><p>Pacote com PDF, KMLs, JSON e histórico disponível.</p></div><button id="vv204Package" ${r.property_id?'':'disabled'}>Gerar dossiê</button></article>
      </div>`;
  }

  function bindDocumentActions(r){
    E('vv204DocKml').onclick=()=>safeAction(()=>exportKML());
    E('vv204AttachKml').onclick=()=>safeAction(()=>attachKML());
    E('vv204Evidence').onclick=()=>safeAction(()=>api().ExportEnvironmentalEvidenceJSON(r.property_id,false).then(p=>toast('Evidências salvas em '+p)));
    E('vv204Package').onclick=()=>safeAction(()=>exportPackage());
  }

  function reportsTab(r){
    const can=!!r.property_id;
    return `
      <div class="vv204-tab-title"><div><span>RELATÓRIOS</span><h2>Saídas técnicas do imóvel</h2><p>Gere somente o documento necessário para o trabalho atual.</p></div></div>
      <div class="vv204-report-grid">
        <article class="vv204-report-card"><span>${icon('print')}</span><h3>Demonstrativo CAR</h3><p>PDF profissional com ficha cadastral, situação SICAR, métricas e mapa vetorial do imóvel.</p><button id="vv204ReportCAR" ${can?'':'disabled'}>Gerar Demonstrativo PDF</button></article>
        <article class="vv204-report-card"><span>${icon('tree')}</span><h3>Laudo Ambiental</h3><p>Laudo multipágina com MapBiomas, INPE, perfil territorial, mapa, fontes e limitações.</p><button id="vv204ReportEnv" ${can?'':'disabled'}>Gerar Laudo PDF</button></article>
        <article class="vv204-report-card"><span>${icon('database')}</span><h3>Caderno de Evidências</h3><p>PDF de rastreabilidade com status das bases, mapa, alertas, fontes e avisos da execução.</p><button id="vv204ReportEvidence" ${can?'':'disabled'}>Gerar Evidências PDF</button></article>
        <article class="vv204-report-card featured"><span>${icon('file')}</span><h3>Dossiê Técnico do Imóvel</h3><p>PDF consolidado com CAR, ambiental, fundiário, SICOR, Banco Central, fontes e ressalvas.</p><button id="vv204ReportPackage" ${can?'':'disabled'}>Gerar Dossiê PDF</button></article>
      </div>`;
  }

  function bindReportActions(r){
    E('vv204ReportCAR').onclick=()=>safeAction(()=>exportReport());
    E('vv204ReportEnv').onclick=()=>safeAction(()=>api().ExportEnvironmentalTechnicalReport(r.property_id,false).then(p=>toast('Laudo PDF salvo em '+p)));
    E('vv204ReportEvidence').onclick=()=>safeAction(()=>api().ExportEnvironmentalEvidencePDF(r.property_id,false).then(p=>toast('Caderno de evidências salvo em '+p)));
    E('vv204ReportPackage').onclick=()=>safeAction(()=>api().ExportPropertyTechnicalDossierPDF(r.property_id,false).then(p=>toast('Dossiê técnico PDF salvo em '+p)));
  }

  function dataRow(label,value,cls=''){return `<div><dt>${H(label)}</dt><dd class="${cls}">${H(value||'—')}</dd></div>`}

  function timeline(r){
    const c=r.car||{},events=[];
    if(c.data_cadastro)events.push({d:c.data_cadastro,t:'Inscrição no CAR',s:c.status||''});
    A(r.environmental?.alerts).forEach(a=>events.push({d:a.detected_at||a.published_at,t:'Alerta MapBiomas'+(a.alert_code?' • '+a.alert_code:''),s:a.area_ha?N(a.area_ha)+' ha':''}));
    A(r.xray?.sicor?.operations).forEach(o=>events.push({d:o.issue_date||String(o.year||''),t:'Operação de crédito (SICOR)',s:[o.purpose,o.credit_value?MONEY(o.credit_value):''].filter(Boolean).join(' • ')}));
    if(c.data_atualizacao)events.push({d:c.data_atualizacao,t:'Atualização do CAR',s:c.condition||''});
    events.sort((a,b)=>dateValue(b.d)-dateValue(a.d));
    return events.length?events.slice(0,8).map((x,i)=>`<div class="vv204-time t${i%4}"><i></i><b>${H(shortDate(x.d))}</b><span><strong>${H(x.t)}</strong><small>${H(x.s||'')}</small></span></div>`).join(''):'<div class="vv204-empty">Nenhum evento com data disponível.</div>';
  }

  function drawMap(r,targetId,controls){
    const node=E(targetId);if(!node||!window.L)return;
    try{map204?.remove()}catch(_){}map204=null;
    map204=L.map(node,{zoomControl:true,attributionControl:true}).setView([-18.5,-44],5);
    const sat=L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',{maxZoom:19,attribution:'Esri'}).addTo(map204);
    const street=L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Street_Map/MapServer/tile/{z}/{y}/{x}',{maxZoom:19,attribution:'Esri'});
    const overlays={},fit=[];
    const add=(label,raw,style,visible=true)=>{if(!raw)return;try{const obj=typeof raw==='string'?JSON.parse(raw):raw;const layer=L.geoJSON(obj,{style});overlays[label]=layer;if(visible){layer.addTo(map204);fit.push(layer)}}catch(_){}};
    add('CAR (SICAR)',r.car?.geojson,{color:'#e93c35',weight:3,fillColor:'#ffffff',fillOpacity:.025},true);
    const live=r.environmental?.environment||{}, ce=r.car?.environment||{},env=(live.ibama_checked||live.funai_checked||live.icmbio_checked||live.mcr_checked)?live:ce;
    A(env.ibama_embargos).forEach((x,i)=>add('Embargo IBAMA '+(i+1),x.geojson,{color:'#dc2626',weight:3,fillColor:'#ef4444',fillOpacity:.22},true));
    A(env.indigenous_findings).forEach((x,i)=>add('Terra Indígena '+(i+1),x.geojson,{color:'#f59e0b',weight:3,fillColor:'#fbbf24',fillOpacity:.16},true));
    A(env.federal_uc_findings).forEach((x,i)=>add('UC Federal '+(i+1),x.geojson,{color:'#16a34a',weight:3,fillColor:'#22c55e',fillOpacity:.15},true));
    A(r.environmental?.alerts).forEach((x,i)=>add('MapBiomas '+(x.alert_code||i+1),x.geometry_geojson,{color:'#f97316',weight:3,fillColor:'#fb923c',fillOpacity:.20},true));
    A(r.xray?.sicor?.operations).forEach((o,oi)=>A(o.glebas).forEach((g,gi)=>add('SICOR '+(oi+1)+'.'+(gi+1),g.geojson,{color:'#7c3aed',weight:2,dashArray:'7 5',fillColor:'#8b5cf6',fillOpacity:.09},false)));
    if(controls)L.control.layers({'Satélite':sat,'Mapa':street},overlays,{collapsed:false,position:'topright'}).addTo(map204);
    if(fit.length){const g=L.featureGroup(fit),b=g.getBounds();if(b.isValid())map204.fitBounds(b.pad(.08),{maxZoom:16})}
  }

  function bindSourceLinks(root){
    root.querySelectorAll('[data-source-url]').forEach(b=>b.onclick=()=>openExternal(b.dataset.sourceUrl));
  }

  function src(r,key){return A(r.sources).find(x=>x.key===key)||{}}
  function statusLabel(s){return {ok:'Sem ocorrência',hit:'Ocorrência encontrada',available:'Disponível',empty:'Sem dados no recorte',on_demand:'Sob demanda',unavailable:'Base indisponível',not_configured:'Não configurado',partial:'Consulta parcial',cached:'Cache local',not_found:'Não localizado',not_saved:'Não salvo'}[s]||'Não consultado'}
  function toneCAR(c){if(c.lookup_status==='cached'||c.lookup_status==='partial')return'warn';if(c.lookup_status==='unavailable'||c.lookup_status==='not_found')return'danger';return'blue'}
  function countAttention(r){return A(r.warnings).length+A(r.sources).filter(s=>s.status==='hit').reduce((n,s)=>n+(Number(s.count)||1),0)+A(r.sources).filter(s=>['unavailable','partial','cached'].includes(s.status)).length}
  function overall(v){return {complete:'Completa',partial:'Parcial',not_found:'Não localizado',error:'Erro'}[v]||'Conferir'}
  function dateBR(v){if(!v)return'—';const d=new Date(v);return isNaN(d)?String(v):d.toLocaleDateString('pt-BR')}
  function shortDate(v){if(!v)return'—';const d=new Date(v);if(!isNaN(d))return d.toLocaleDateString('pt-BR',{month:'short',year:'numeric'});return String(v)}
  function dateValue(v){const d=new Date(v);if(!isNaN(d))return d.getTime();const y=Number(String(v||'').match(/20\d{2}/)?.[0]||0);return y?Date.UTC(y,0,1):0}
  async function safeAction(fn){try{await fn()}catch(e){if(!String(e).toLowerCase().includes('cancelad'))toast(String(e),true)}}

  document.addEventListener('DOMContentLoaded',()=>setTimeout(install,160));
})();