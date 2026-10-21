/* ViaVerdeCAR 2.0.2 - interface principal com SVG, sem glifos de fonte */
(()=>{
  const el=id=>document.getElementById(id);
  const arr=v=>Array.isArray(v)?v:[];
  const esc=v=>String(v??'').replace(/[&<>"']/g,m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]));
  const num=(v,d=2)=>(Number(v)||0).toLocaleString('pt-BR',{minimumFractionDigits:d,maximumFractionDigits:d});
  const money=v=>(Number(v)||0).toLocaleString('pt-BR',{style:'currency',currency:'BRL'});
  let result202=null, map202=null, timer202=null;

  const paths={
    leaf:'M19.5 3.5C13 3.8 7.2 6.7 4.8 11.7c-1.9 4 .2 7.6 3.7 8.2 4.1.7 7.4-2.3 8.3-6.5.7-3.5 1.4-6.4 2.7-9.9Z M7 18c2.4-4.4 5.3-7.2 9-9.4',
    home:'M3.5 11.2 12 4l8.5 7.2v8.3h-6v-5h-5v5h-6v-8.3Z',
    users:'M8.5 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z M3.5 20c0-3.4 2.2-5.6 5-5.6s5 2.2 5 5.6 M16 8.5a2.5 2.5 0 1 0 0-5 M15 14.6c3-.2 5.5 1.7 5.5 5.4',
    building:'M4 20V8l8-4 8 4v12 M8 20v-6h8v6 M8 10h.01 M12 10h.01 M16 10h.01',
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
    close:'M6 6l12 12 M18 6 6 18',
    arrow:'M5 12h14 M14 7l5 5-5 5'
  };
  function icon(name,cls=''){
    return '<svg class="vv202-icon '+cls+'" viewBox="0 0 24 24" aria-hidden="true"><path d="'+paths[name]+'" /></svg>';
  }

  function install202(){
    document.body.classList.add('vv202');
    rebuildShell202();
    buildDashboard202();
    installCarAction202();
    patchSetView202();
    bindSearch202();
    setTimeout(renderLanding202,250);
    setTimeout(renderLanding202,1200);
  }

  function rebuildShell202(){
    const side=document.querySelector('.sidebar');
    if(side){
      side.innerHTML=`
        <div class="vv202-brand">
          <div class="vv202-logo">${icon('leaf')}</div>
          <div><strong>ViaVerdeCAR</strong><span>CONSULTA E ANÁLISE RURAL</span></div>
          <b id="vv202Version">2.0.3</b>
        </div>
        <nav class="vv202-nav">
          ${navButton202('dashboard','home','Início',true)}
          ${navButton202('clients','users','Clientes')}
          ${navButton202('clients','building','Imóveis','property-panel')}
          ${navButton202('car','search','Consulta CAR','car-toolbar')}
          ${navButton202('car','tree','Raio X Ambiental','tab:environment')}
          ${navButton202('car','map','Raio X Fundiário','tab:xray')}
          ${navButton202('car','bank','Crédito Rural','tab:credit')}
          ${navButton202('car','map','Mapa','tab:map')}
          ${navButton202('car','file','Relatórios','tab:docs')}
          ${navButton202('car','folder','Dossiês','tab:docs')}
        </nav>
        <div class="vv202-side-tools">
          <button id="vv202Sources">${icon('database')}<span>Fontes de Dados</span></button>
          <button id="vv202Backup">${icon('database')}<span>Backup / Banco</span></button>
          <button id="vv202Settings">${icon('info')}<span>Sobre</span></button>
        </div>
        <div class="sidebar-footer vv202-footer">
          <div class="online-dot"></div>
          <div><strong id="connectionLabel">Internet</strong><small id="versionLabel">ViaVerdeCAR</small></div>
        </div>`;
      side.querySelectorAll('[data-vv202-view]').forEach(b=>b.onclick=async()=>{
        setView(b.dataset.vv202View);
        side.querySelectorAll('.vv202-nav button').forEach(x=>x.classList.remove('active'));
        b.classList.add('active');
        const section=b.dataset.vv202Section;
        if(section)setTimeout(()=>openSection202(b.dataset.vv202View,section),100);
      });
      el('vv202Backup').onclick=async()=>{try{const r=await api().BackupData();toast(r?.message||'Backup concluído.')}catch(e){if(!String(e).toLowerCase().includes('cancelado'))toast(String(e),true)}};
      el('vv202Settings').onclick=()=>setView('settings');
      el('vv202Sources').onclick=()=>{setView('dashboard');setTimeout(()=>el('vv202Cards')?.scrollIntoView({behavior:'smooth'}),80)};
    }

    const top=document.querySelector('.topbar');
    if(top){
      top.innerHTML=`
        <div class="vv202-searchbar">
          ${icon('search')}
          <input id="vv202TopInput" placeholder="Digite o CAR, CPF, CNPJ, nome do imóvel, matrícula ou município..." />
          <button id="vv202TopGo">${icon('search')}<span>ANALISAR</span></button>
        </div>
        <div class="vv202-top-actions">
          <button id="vv202History">${icon('history')}<span>Histórico</span></button>
          <button id="vv202Config">${icon('settings')}<span>Configurações</span></button>
          <button id="vv202Help">${icon('help')}<span>Ajuda</span></button>
        </div>
        <h1 id="pageTitle" class="vv202-compat">Visão geral</h1>
        <button id="backupBtn" class="vv202-compat"></button>
        <button id="goCarBtn" class="vv202-compat"></button>`;
      el('vv202TopGo').onclick=()=>search202(true);
      el('vv202TopInput').addEventListener('keydown',e=>{if(e.key==='Enter'){e.preventDefault();search202(true)}});
      el('vv202TopInput').addEventListener('input',()=>{
        clearTimeout(timer202);timer202=setTimeout(()=>search202(false),350);
      });
      el('vv202History').onclick=()=>{setView('car');setTimeout(()=>{openSection202('car','tab:docs');document.querySelector('.history-panel')?.scrollIntoView({behavior:'smooth',block:'start'});activateSideSection202('tab:docs')},120)};
      el('vv202Config').onclick=()=>setView('settings');
      el('vv202Help').onclick=()=>toast('CAR executa a análise automática. CPF/CNPJ pesquisa vínculos locais e apresenta fontes oficiais disponíveis.');
    }
    try{api()?.GetAppInfo?.().then(i=>{if(i?.version){if(el('vv202Version'))el('vv202Version').textContent=i.version;if(el('versionLabel'))el('versionLabel').textContent='Versão '+i.version}})}catch(_){}
  }

  function openSection202(view,section){
    if(view!=='car'){
      if(section)document.querySelector('#view-'+view+' .'+section)?.scrollIntoView({behavior:'smooth',block:'start'});
      return;
    }
    if(section.startsWith('tab:')){
      const key=section.slice(4);
      const tab=document.querySelector('[data-car-tab131="'+key+'"]');
      if(tab){tab.click();setTimeout(()=>document.getElementById('carWorkspace131')?.scrollIntoView({behavior:'smooth',block:'start'}),60)}
      return;
    }
    document.querySelector('#view-car .'+section)?.scrollIntoView({behavior:'smooth',block:'start'});
  }

  function activateSideSection202(section){
    const side=document.querySelector('.sidebar');if(!side)return;
    side.querySelectorAll('.vv202-nav button').forEach(x=>x.classList.remove('active'));
    const b=side.querySelector('[data-vv202-section="'+section+'"]');
    if(b)b.classList.add('active');
  }

  function navButton202(view,ico,label,section='',active=false){
    return '<button class="'+(active?'active':'')+'" data-vv202-view="'+view+'" data-vv202-section="'+section+'">'+icon(ico)+'<span>'+esc(label)+'</span></button>';
  }

  function syncNav202(view){
    const side=document.querySelector('.sidebar');
    if(!side)return;
    side.querySelectorAll('.vv202-nav button').forEach(b=>b.classList.remove('active'));
    const target=side.querySelector('[data-vv202-view="'+view+'"]');
    if(target)target.classList.add('active');
  }

  function patchSetView202(){
    if(window.__vv202SetViewPatched)return;
    window.__vv202SetViewPatched=true;
    const old=setView;
    setView=function(name){
      old(name);
      syncNav202(name);
    };
  }

  function installCarAction202(){
    const toolbar=document.querySelector('#view-car .car-toolbar');
    const actions=toolbar?.querySelector('.toolbar-actions')||toolbar;
    if(!actions||el('vv202AnalyzeAll'))return;
    const b=document.createElement('button');
    b.id='vv202AnalyzeAll';
    b.className='btn primary';
    b.innerHTML=icon('refresh')+'<span>Analisar tudo</span>';
    b.onclick=()=>{
      const car=(el('carInput')?.value||state?.selectedProperty?.car_number||'').trim();
      if(!car){toast('Informe um CAR para executar a análise completa.',true);return}
      setView('dashboard');
      if(el('vv202TopInput'))el('vv202TopInput').value=car;
      setTimeout(()=>runCAR202(car,state?.selectedProperty?.id||0,true),40);
    };
    actions.prepend(b);
  }

  function buildDashboard202(){
    const d=el('view-dashboard');if(!d)return;
    [...d.children].forEach(x=>x.classList.add('vv202-legacy-hidden'));
    const w=document.createElement('div');
    w.id='vvWorkspace202';w.className='vv202-workspace';
    w.innerHTML='<section id="vv202Content"></section>';
    d.prepend(w);
  }

  function bindSearch202(){
    document.addEventListener('keydown',e=>{
      if((e.ctrlKey||e.metaKey)&&e.key.toLowerCase()==='k'){
        e.preventDefault();setView('dashboard');setTimeout(()=>el('vv202TopInput')?.focus(),60);
      }
    });
  }

  function renderLanding202(){
    if(result202)return;
    const box=el('vv202Content');if(!box)return;
    const recent=arr(typeof state!=='undefined'?state.properties:[]).slice().sort((a,b)=>String(b.updated_at||'').localeCompare(String(a.updated_at||''))).slice(0,5);
    box.innerHTML=`
      <section class="vv202-landing">
        <div class="vv202-landing-copy">
          <div class="vv202-landing-icon">${icon('search')}</div>
          <div><span>VISÃO GERAL AUTOMÁTICA</span><h2>Informe o CAR uma vez. O ViaVerdeCAR organiza o restante.</h2><p>CAR, geometria, mapa, ambiente, SIGEF, crédito rural e dossiê em um fluxo único.</p></div>
        </div>
        <div class="vv202-flow">
          <div>${icon('search')}<b>CAR</b></div><i></i><div>${icon('map')}<b>Mapa</b></div><i></i><div>${icon('tree')}<b>Ambiental</b></div><i></i><div>${icon('bank')}<b>Crédito</b></div><i></i><div>${icon('file')}<b>Dossiê</b></div>
        </div>
        <div class="vv202-recent-title"><strong>Imóveis recentes</strong><span>Abra novamente com um clique</span></div>
        <div class="vv202-recent">${recent.length?recent.map((p,i)=>`<button data-vv202-recent="${i}"><span>${p.car_number?'CAR VINCULADO':'CADASTRO LOCAL'}</span><strong>${esc(p.name||'Imóvel')}</strong><small>${esc([p.client_name,p.municipality,p.uf].filter(Boolean).join(' / '))}</small><b>${p.declared_area_ha?num(p.declared_area_ha)+' ha':'Área não informada'}</b></button>`).join(''):'<div class="vv202-empty">Nenhum imóvel cadastrado ainda.</div>'}</div>
      </section>`;
    box.querySelectorAll('[data-vv202-recent]').forEach(b=>b.onclick=()=>{
      const p=recent[Number(b.dataset.vv202Recent)];if(!p)return;
      if(p.car_number){el('vv202TopInput').value=p.car_number;runCAR202(p.car_number,p.id,true)}
      else{setView('clients');selectClient(p.client_id)}
    });
  }

  async function search202(run){
    const q=(el('vv202TopInput')?.value||'').trim();
    setView('dashboard');
    if(!q){result202=null;renderLanding202();return}
    try{
      const r=await api().SearchEverything(q);
      if(run&&r.can_analyze_car&&r.valid){
        const hit=arr(r.hits).find(x=>x.property_id&&x.car);
        return runCAR202(r.normalized,hit?.property_id||0,true);
      }
      if(r.mode==='cpf'||r.mode==='cnpj')return renderDoc202(r);
      renderSearch202(r);
    }catch(e){toast(String(e),true)}
  }

  function renderSearch202(r){
    const hits=arr(r.hits),box=el('vv202Content');
    box.innerHTML=`<section class="vv202-simple-panel"><div class="vv202-section-head"><div><span>BUSCA LOCAL</span><h3>${esc(r.message||'Resultados')}</h3></div><b>${hits.length} resultado(s)</b></div><div class="vv202-results">${hits.length?hits.map((h,i)=>`<article><div><span>${esc(h.kind||'registro')}</span><strong>${esc(h.title)}</strong><small>${esc(h.subtitle||h.car||h.cpf_cnpj||'')}</small></div><div>${h.car?'<button data-vv202-an="'+i+'">Analisar CAR</button>':''}<button data-vv202-open="${i}">Abrir</button></div></article>`).join(''):'<div class="vv202-empty">Nenhum registro local encontrado.</div>'}</div></section>`;
    box.querySelectorAll('[data-vv202-an]').forEach(b=>b.onclick=()=>{const h=hits[Number(b.dataset.vv202An)];runCAR202(h.car,h.property_id||0,true)});
    box.querySelectorAll('[data-vv202-open]').forEach(b=>b.onclick=()=>{const h=hits[Number(b.dataset.vv202Open)];if(h.car)runCAR202(h.car,h.property_id||0,true);else if(h.client_id){setView('clients');selectClient(h.client_id)}});
  }

  function renderDoc202(r){
    const hits=arr(r.hits), opts=arr(r.official_options), box=el('vv202Content');
    box.innerHTML=`<section class="vv202-doc">
      <div class="vv202-section-head"><div><span>PESQUISA POR ${esc(String(r.mode||'').toUpperCase())}</span><h3>${esc(r.message)}</h3><p>${esc(r.privacy_notice||'')}</p></div><b class="vv202-mono">${esc(r.normalized||r.query)}</b></div>
      <div class="vv202-doc-grid">
        <div><h4>Vínculos conhecidos pelo ViaVerdeCAR</h4>${hits.length?hits.map((h,i)=>`<button class="vv202-doc-hit" data-vv202-doc-hit="${i}"><div><strong>${esc(h.title)}</strong><span>${esc(h.subtitle||'')}</span><small>${esc(h.car||h.cpf_cnpj||'')}</small></div>${icon('arrow')}</button>`).join(''):'<div class="vv202-empty">Nenhum cliente ou imóvel local vinculado a este documento.</div>'}</div>
        <div><h4>Fontes oficiais disponíveis</h4>${opts.map((o,i)=>`<article class="vv202-source"><div><span>${o.automatic?'AUTOMÁTICO':'OFICIAL'}</span><strong>${esc(o.label)}</strong></div><p>${esc(o.detail)}</p>${o.url?'<button data-vv202-source="'+i+'">Abrir fonte oficial</button>':''}</article>`).join('')}</div>
      </div>
      <div class="vv202-legal"><strong>Regra de titularidade:</strong> vínculo local ou coincidência de nome não é tratado como prova de que um CAR pertence ao CPF/CNPJ. O sistema só afirma a relação quando ela já foi vinculada ou comprovada por fonte/documento apropriado.</div>
    </section>`;
    hits.forEach((h,i)=>{const b=box.querySelector('[data-vv202-doc-hit="'+i+'"]');if(b)b.onclick=()=>{if(h.car)runCAR202(h.car,h.property_id||0,true);else if(h.client_id){setView('clients');selectClient(h.client_id)}}});
    opts.forEach((o,i)=>{const b=box.querySelector('[data-vv202-source="'+i+'"]');if(b)b.onclick=()=>openExternal(o.url)});
  }

  async function runCAR202(car,propertyID,force){
    const box=el('vv202Content');
    box.innerHTML=`<section class="vv202-loading"><div class="vv202-spinner"></div><div><span>RAIO X AUTOMÁTICO</span><h3>Consultando o imóvel e cruzando as bases públicas</h3><p>${esc(car)}</p></div><div class="vv202-progress"><i></i></div><div class="vv202-progress-labels"><b>CAR</b><b>Mapa</b><b>Ambiental</b><b>Fundiário</b><b>Crédito</b><b>Resumo</b></div></section>`;
    try{
      const r=await api().RunCARAutomation(car,Number(propertyID)||0,!!force);
      result202=r;window.__vv202LastResult=r;
      renderAnalysis202(r);
      state.car=r.car;
      if(r.car?.car){renderCAR(r.car);if(r.car.geojson)drawGeoJSON('car',r.car.geojson);if(el('carInput'))el('carInput').value=r.car.car}
      await Promise.all([loadProperties(),loadDashboard()]);
      if(r.property_id)state.selectedProperty=state.properties.find(p=>p.id===r.property_id)||state.selectedProperty;
    }catch(e){
      box.innerHTML='<section class="vv202-error">'+icon('alert')+'<div><strong>Não foi possível concluir o Raio X</strong><p>'+esc(String(e))+'</p><span>Uma falha de serviço não é tratada como ausência de ocorrência.</span></div></section>';
      toast(String(e),true);
    }
  }

  function renderAnalysis202(r){
    const car=r.car||{}, env=r.environmental||{}, xr=r.xray||{}, sig=xr.sigef||{}, sic=xr.sicor||{};
    const area=Number(car.area_ha||car.geometry_area_ha)||0;
    const keys=['ibama','mapbiomas','inpe_fire','funai','icmbio','mcr'];
    const envHits=keys.reduce((n,k)=>{const s=src202(r,k);return n+(s.status==='hit'?(Number(s.count)||1):0)},0);
    const affected=Number(env.summary?.alert_area_in_car_ha)||0;
    const attention=countAttention202(r);
    const carState=car.lookup_status==='cached'?'Cache local':car.lookup_status==='partial'?'Ficha parcial':car.lookup_status==='unavailable'?'Base indisponível':car.status||((car.found||car.has_geometry)?'Localizado':'Não localizado');

    el('vv202Content').innerHTML=`
      <section id="vv202Cards" class="vv202-cards">
        ${card202('building','CAR',carState,'Área',area?num(area)+' ha':'-', 'Município',esc([car.municipality,car.uf].filter(Boolean).join(' / ')||'-'),toneCAR202(car))}
        ${card202('leaf','Ambiental',envHits?String(envHits):'0','Ocorrências',envHits+' registro(s)','Área afetada',affected?num(affected)+' ha'+(area?' ('+num(affected/area*100,1)+'%)':''):'-',envHits?'greenwarn':'green')}
        ${card202('map','Fundiário',String(Number(sig.parcel_count)||0),'SIGEF',(Number(sig.parcel_count)||0)+' parcela(s)','Sobreposição',sig.best_car_coverage_pct?num(sig.best_car_coverage_pct,1)+'%':'-',(Number(sig.parcel_count)||0)?'orange':'orange')}
        ${card202('bank','Crédito Rural',String(Number(sic.operation_count)||0),'Operações',(Number(sic.operation_count)||0)+' registro(s)','Valor total',sic.total_credit_value?money(sic.total_credit_value):'-','purple')}
        ${card202('alert','Atenções',String(attention),'Pontos de verificação',attention+' item(ns)','Status',overall202(r.overall_status),attention?'red':'red')}
      </section>

      <section class="vv202-main">
        <article class="vv202-panel vv202-property">
          <div class="vv202-panel-head"><div><span>RESUMO DO IMÓVEL</span><h3>${esc(car.property_name||state?.selectedProperty?.name||'Imóvel rural')}</h3></div><button id="vv202OpenMap">${icon('map')}</button></div>
          <dl>
            <div><dt>CAR</dt><dd class="mono">${esc(car.car||'-')}</dd></div>
            <div><dt>Município/UF</dt><dd>${esc([car.municipality,car.uf].filter(Boolean).join(' / ')||'-')}</dd></div>
            <div><dt>Área</dt><dd>${area?num(area)+' ha':'-'}</dd></div>
            <div><dt>Módulos fiscais</dt><dd>${car.fiscal_modules?num(car.fiscal_modules,2):'-'}</dd></div>
            <div><dt>Situação</dt><dd><span class="vv202-pill ${toneCAR202(car)}">${esc(carState)}</span></dd></div>
            <div><dt>Condição</dt><dd>${esc(car.condition||'-')}</dd></div>
            <div><dt>Data de inscrição</dt><dd>${esc(date202(car.data_cadastro))}</dd></div>
            <div><dt>Última atualização</dt><dd>${esc(date202(car.data_atualizacao))}</dd></div>
          </dl>
          ${car.lookup_detail?'<div class="vv202-source-note"><strong>SICAR</strong><span>'+esc(car.lookup_detail)+'</span></div>':''}
          <div class="vv202-property-actions"><button id="vv202Kml">${icon('download')}Gerar KML (SICAR)</button><button id="vv202Map">${icon('map')}Abrir no mapa</button>${!r.property_id&&car.found?'<button id="vv202Link">'+icon('link')+'Salvar / Vincular</button>':''}</div>
          <div id="vv202LinkBox" class="vv202-linkbox hidden"></div>
        </article>

        <article class="vv202-panel vv202-map-panel">
          <div class="vv202-panel-head"><div><span>MAPA DO IMÓVEL E OCORRÊNCIAS</span><h3>Camadas retornadas nesta análise</h3></div><b>${icon('layers')} Camadas</b></div>
          <div id="vv202MapCanvas"></div>
        </article>

        <article class="vv202-panel vv202-timeline-panel">
          <div class="vv202-panel-head"><div><span>RASTREABILIDADE</span><h3>Linha do Tempo</h3></div>${icon('clock')}</div>
          <div class="vv202-timeline">${timeline202(r)}</div>
          <button id="vv202HistoryFull" class="vv202-full-btn">Ver linha do tempo completa</button>
        </article>
      </section>

      <nav class="vv202-tabs">
        <button class="active" data-tab202="summary">${icon('file')}Resumo</button>
        <button data-tab202="car">${icon('leaf')}CAR</button>
        <button data-tab202="environment">${icon('leaf')}Ambiental</button>
        <button data-tab202="land">${icon('map')}Fundiário</button>
        <button data-tab202="credit">${icon('bank')}Crédito Rural</button>
        <button data-tab202="map">${icon('map')}Mapa</button>
        <button data-tab202="docs">${icon('folder')}Documentos</button>
        <button data-tab202="reports">${icon('file')}Relatórios</button>
      </nav>

      <section class="vv202-bottom">
        <article class="vv202-panel"><div class="vv202-panel-head"><div><span>RAIO X AMBIENTAL</span><h3>Ocorrências e bases</h3></div><b class="vv202-badge red">${envHits} ocorrência(s)</b></div>${environment202(r)}</article>
        <article class="vv202-panel"><div class="vv202-panel-head"><div><span>RAIO X FUNDIÁRIO</span><h3>SIGEF / INCRA</h3></div><b class="vv202-badge orange">${Number(sig.parcel_count)||0} parcela(s)</b></div>${land202(r)}</article>
        <article class="vv202-panel"><div class="vv202-panel-head"><div><span>CRÉDITO RURAL</span><h3>SICOR</h3></div><b class="vv202-badge purple">${Number(sic.operation_count)||0} operação(ões)</b></div>${credit202(r)}</article>
        <article class="vv202-panel vv202-actions"><div class="vv202-panel-head"><div><span>AÇÕES RÁPIDAS</span><h3>Próximo passo</h3></div></div>
          <button id="vv202Dossier" class="primary" ${r.property_id?'':'disabled'}>${icon('file')}<span><b>Gerar dossiê completo</b><small>PDF + KML + mapas + evidências</small></span></button>
          <button id="vv202Refresh" class="blue">${icon('refresh')}<span><b>Atualizar tudo</b><small>Reexecuta todas as análises</small></span></button>
          <button id="vv202Export" ${car.has_geometry?'':'disabled'}>${icon('download')}<span><b>Exportar KML</b><small>CAR e perímetro público</small></span></button>
          <button id="vv202Report" ${r.property_id?'':'disabled'}>${icon('print')}<span><b>Imprimir relatório</b><small>Relatório técnico do imóvel</small></span></button>
        </article>
      </section>
      ${bcb202(r)}
      ${arr(r.warnings).length?'<details class="vv202-warnings"><summary>Avisos e limitações ('+arr(r.warnings).length+')</summary>'+arr(r.warnings).map(w=>'<p>'+esc(w)+'</p>').join('')+'</details>':''}
    `;
    bindAnalysis202(r);
    setTimeout(()=>drawMap202(r),90);
  }

  function card202(ico,title,badge,l1,v1,l2,v2,tone){
    return '<article class="vv202-card '+tone+'"><div class="vv202-card-head"><span>'+icon(ico)+'</span><strong>'+esc(title)+'</strong><b>'+esc(badge)+'</b></div><div class="vv202-card-body"><div><span>'+esc(l1)+'</span><strong>'+v1+'</strong></div><div><span>'+esc(l2)+'</span><strong>'+v2+'</strong></div></div></article>';
  }

  function bindAnalysis202(r){
    el('vv202OpenMap').onclick=()=>openWorkspace202(r,'map');
    el('vv202Map').onclick=()=>openWorkspace202(r,'map');
    el('vv202Kml').onclick=()=>{openWorkspace202(r).then(()=>setTimeout(()=>el('exportKmlBtn')?.click(),120))};
    if(el('vv202Link'))el('vv202Link').onclick=()=>linkBox202(r);
    el('vv202Refresh').onclick=()=>runCAR202(r.car.car,r.property_id||0,true);
    el('vv202Export').onclick=()=>{openWorkspace202(r).then(()=>setTimeout(()=>el('exportKmlBtn')?.click(),120))};
    el('vv202Dossier').onclick=()=>{openWorkspace202(r).then(()=>setTimeout(()=>el('packageBtn')?.click(),160))};
    el('vv202Report').onclick=()=>{openWorkspace202(r).then(()=>setTimeout(()=>el('reportBtn')?.click(),160))};
    el('vv202HistoryFull').onclick=()=>{openWorkspace202(r).then(()=>setTimeout(()=>document.querySelector('.history-panel')?.scrollIntoView({behavior:'smooth'}),120))};
    document.querySelectorAll('[data-tab202]').forEach(b=>b.onclick=()=>tab202(b.dataset.tab202,r));
    document.querySelectorAll('[data-bcb-source202]').forEach(b=>b.onclick=()=>openExternal(b.dataset.bcbSource202));
  }

  async function openWorkspace202(r,section=''){
    setView('car');
    if(r.property_id){
      if(el('carPropertySelect'))el('carPropertySelect').value=String(r.property_id);
      await selectCarProperty(r.property_id);
    }else{
      state.car=r.car;renderCAR(r.car);if(r.car.geojson)drawGeoJSON('car',r.car.geojson);if(el('carInput'))el('carInput').value=r.car.car||'';
    }
    if(section==='map')setTimeout(()=>{openSection202('car','tab:map');activateSideSection202('tab:map')},100);
  }

  function tab202(key,r){
    if(key==='summary'){el('vv202Cards')?.scrollIntoView({behavior:'smooth'});return}
    const section={
      car:'car-toolbar',
      environment:'tab:environment',
      land:'tab:xray',
      credit:'tab:credit',
      map:'tab:map',
      docs:'tab:docs',
      reports:'tab:docs'
    }[key]||'car-toolbar';
    openWorkspace202(r).then(()=>setTimeout(()=>{
      openSection202('car',section);
      activateSideSection202(section);
    },80));
  }

  function linkBox202(r){
    const b=el('vv202LinkBox'), clients=arr(state?.clients);if(!b)return;
    b.classList.remove('hidden');
    b.innerHTML=clients.length?'<select id="vv202Client"><option value="">Selecione o cliente</option>'+clients.map(c=>'<option value="'+c.id+'">'+esc(c.name)+(c.cpf_cnpj?' / '+esc(c.cpf_cnpj):'')+'</option>').join('')+'</select><button id="vv202SaveLink">Vincular este CAR</button><small>O vínculo é confirmado pelo usuário; o aplicativo não deduz titularidade.</small>':'<p>Cadastre um cliente antes de vincular o CAR.</p>';
    if(el('vv202SaveLink'))el('vv202SaveLink').onclick=async()=>{
      const id=Number(el('vv202Client').value)||0;if(!id){toast('Selecione o cliente.',true);return}
      try{const p=await api().SaveAnalyzedCARToClient(id,r.car.car);await Promise.all([loadClients(),loadProperties(),loadDashboard()]);toast('Imóvel salvo e vinculado.');runCAR202(r.car.car,p.id,false)}catch(e){toast(String(e),true)}
    };
  }

  function src202(r,key){return arr(r.sources).find(x=>x.key===key)||{}}
  function status202(s){return {ok:'Sem ocorrência',hit:'Ocorrência encontrada',available:'Disponível',empty:'Sem dados no recorte',on_demand:'Sob demanda',unavailable:'Base indisponível',not_configured:'Não configurado',partial:'Consulta parcial',cached:'Cache local',not_found:'Não localizado',not_saved:'Não salvo'}[s]||'Não consultado'}
  function row202(r,key,label){
    const s=src202(r,key),n=Number(s.count)||0;
    const tone=s.status==='hit'?'hit':s.status==='ok'?'ok':['unavailable','not_configured'].includes(s.status)?'off':['cached','partial'].includes(s.status)?'warn':'neutral';
    return '<div class="vv202-row '+tone+'"><span class="dot"></span><strong>'+esc(label)+'</strong><span>'+esc(status202(s.status))+'</span><b>'+(n?n+' registro(s)':'-')+'</b></div>';
  }
  function environment202(r){return '<div class="vv202-list">'+row202(r,'ibama','IBAMA - Embargos')+row202(r,'mapbiomas','MapBiomas - Desmatamento')+row202(r,'inpe_fire','INPE - Focos de calor')+row202(r,'funai','FUNAI - Terras Indígenas')+row202(r,'icmbio','ICMBio - Unidades de Conservação')+row202(r,'mcr','MMA - MCR/PRODES')+'</div>'}
  function land202(r){
    const s=r.xray?.sigef||{};
    return '<div class="vv202-list"><div class="vv202-row '+(s.available?'ok':'off')+'"><span class="dot"></span><strong>SIGEF / INCRA</strong><span>'+(s.available?'Consultado':'Indisponível')+'</span><b>'+(Number(s.parcel_count)||0)+' parcela(s)</b></div><div class="vv202-row neutral"><span></span><strong>Sobreposição CAR x SIGEF</strong><span>melhor cobertura</span><b>'+(s.best_car_coverage_pct?num(s.best_car_coverage_pct,1)+'%':'-')+'</b></div><div class="vv202-row neutral"><span></span><strong>Matrículas / registros</strong><span>retornados</span><b>'+(Number(s.registry_count)||0)+'</b></div></div>';
  }
  function credit202(r){
    const s=r.xray?.sicor||{},ops=arr(s.operations);
    if(!ops.length)return '<div class="vv202-empty">'+esc(src202(r,'sicor').detail||'Nenhuma operação pública retornada.')+'</div>';
    return '<div class="vv202-credit-list">'+ops.slice(0,4).map(o=>'<div><span><strong>'+esc(o.purpose||o.activity||o.product||'Operação SICOR')+'</strong><small>'+esc([o.institution_name,o.program_name,o.year].filter(Boolean).join(' / '))+'</small></span><b>'+money(o.credit_value)+'</b></div>').join('')+'</div>';
  }

  function bcb202(r){
    const b=r.bcb||{}, market=b.market||{}, series=b.series||{}, inst=b.institutions||{}, ifd=b.ifdata||{}, bankRates=b.institution_rates||{};
    if(!b.generated_at && !b.available && !arr(b.sources).length)return '';
    const allProducts=arr(market.municipal_products), products=allProducts.slice(0,6);
    const programs=arr(market.state_programs).slice(0,5);
    const funding=arr(market.national_sources).slice(0,5);
    const institutions=arr(inst.institutions).slice(0,7);
    const ifdata=arr(ifd.institutions).slice(0,5);
    const rates=arr(series.metrics).filter(x=>x.measure==='Taxa de juros'&&x.status==='available').slice(0,6);
    const macro=arr(series.metrics).filter(x=>x.measure!=='Taxa de juros'&&x.status==='available').slice(0,6);
    const institutionRates=arr(bankRates.rates).slice(0,7);
    const marketContracts=allProducts.reduce((s,x)=>s+(Number(x.contracts)||0),0);
    const marketValue=allProducts.reduce((s,x)=>s+(Number(x.value)||0),0);
    return `
      <section class="vv202-bcb-wrap">
        <div class="vv202-bcb-title">
          <div><span>BANCO CENTRAL • DADOS ABERTOS</span><h3>Mercado e contexto oficial de crédito rural</h3><p>Dados agregados para conferência técnica. Não representam aprovação, limite, dívida ou risco individual do produtor.</p></div>
          <b class="${b.used_cache?'cache':b.available?'ok':'off'}">${b.used_cache?'CACHE 8H':b.available?'ATUALIZADO':'PARCIAL'}</b>
        </div>
        <div class="vv202-bcb-grid">
          <article class="vv202-panel vv202-bcb-card">
            <div class="vv202-panel-head"><div><span>MDCR / SICOR</span><h3>Mercado no município</h3></div><button data-bcb-source202="${esc(market.source_url||'https://dadosabertos.bcb.gov.br/dataset/matrizdadoscreditorural')}">${icon('database')}</button></div>
            <div class="vv202-bcb-metrics"><div><span>Contratos nos itens exibidos</span><strong>${marketContracts?num(marketContracts,0):'-'}</strong></div><div><span>Valor nos itens exibidos</span><strong>${marketValue?money(marketValue):'-'}</strong></div></div>
            <div class="vv202-bcb-list">${products.length?products.map(x=>'<div><span><strong>'+esc(x.label||x.kind||'Produto')+'</strong><small>'+esc([x.kind,x.year].filter(Boolean).join(' / '))+'</small></span><b>'+money(x.value)+'</b></div>').join(''):'<div class="vv202-empty">Sem produtos municipais retornados nesta consulta.</div>'}</div>
          </article>

          <article class="vv202-panel vv202-bcb-card">
            <div class="vv202-panel-head"><div><span>SGS • TAXAS RURAIS</span><h3>Taxas oficiais agregadas</h3></div><button data-bcb-source202="https://www.bcb.gov.br/estatisticas/">${icon('history')}</button></div>
            <div class="vv202-bcb-series">${rates.length?rates.map(seriesLine202).join(''):'<div class="vv202-empty">Séries de taxas indisponíveis nesta consulta.</div>'}</div>
          </article>

          <article class="vv202-panel vv202-bcb-card">
            <div class="vv202-panel-head"><div><span>SGS • CONTEXTO NACIONAL</span><h3>Saldo, concessões e inadimplência</h3></div><button data-bcb-source202="https://www.bcb.gov.br/estatisticas/">${icon('history')}</button></div>
            <div class="vv202-bcb-series">${macro.length?macro.map(seriesLine202).join(''):'<div class="vv202-empty">Séries macroeconômicas indisponíveis nesta consulta.</div>'}</div>
          </article>

          <article class="vv202-panel vv202-bcb-card">
            <div class="vv202-panel-head"><div><span>ENTIDADES SUPERVISIONADAS</span><h3>Instituições do contexto</h3></div><button data-bcb-source202="${esc(inst.source_url||'https://dadosabertos.bcb.gov.br/dataset/dados-cadastrais-de-entidades-autorizadas')}">${icon('bank')}</button></div>
            <div class="vv202-bcb-list">${institutions.length?institutions.map(x=>'<div><span><strong>'+esc(x.name)+'</strong><small>'+esc([x.type,x.situation,x.uf].filter(Boolean).join(' / '))+'</small></span><b>'+esc(x.code||'')+'</b></div>').join(''):'<div class="vv202-empty">'+esc(inst.message||'Sem correspondência segura nesta consulta.')+'</div>'}</div>
          </article>

          <article class="vv202-panel vv202-bcb-card">
            <div class="vv202-panel-head"><div><span>IFDATA</span><h3>Cadastro trimestral das instituições</h3></div><button data-bcb-source202="${esc(ifd.source_url||'https://dadosabertos.bcb.gov.br/dataset/ifdata---dados-selecionados-de-instituies-financeiras')}">${icon('bank')}</button></div>
            <div class="vv202-bcb-ref">Data-base: <strong>${esc(ifd.reference||'-')}</strong></div>
            <div class="vv202-bcb-list">${ifdata.length?ifdata.map(x=>'<div><span><strong>'+esc(x.name)+'</strong><small>'+esc([x.segment,x.activity,x.situation].filter(Boolean).join(' / '))+'</small></span><b>'+esc(x.uf||'')+'</b></div>').join(''):'<div class="vv202-empty">'+esc(ifd.message||'Sem instituição correspondente nesta data-base.')+'</div>'}</div>
          </article>

          <article class="vv202-panel vv202-bcb-card">
            <div class="vv202-panel-head"><div><span>PROGRAMAS E FONTES</span><h3>Como o crédito rural é distribuído</h3></div><button data-bcb-source202="https://dadosabertos.bcb.gov.br/dataset/matrizdadoscreditorural">${icon('database')}</button></div>
            <div class="vv202-bcb-list">${programs.length?programs.map(x=>'<div><span><strong>'+esc(x.label||'Programa')+'</strong><small>'+esc([x.detail,x.year].filter(Boolean).join(' / '))+'</small></span><b>'+money(x.value)+'</b></div>').join(''):'<div class="vv202-empty">Programas não retornados no recorte atual.</div>'}</div>
            <div class="vv202-bcb-subtitle">Fontes de recursos</div>
            <div class="vv202-bcb-list compact">${funding.length?funding.map(x=>'<div><span><strong>'+esc(x.label||'Fonte')+'</strong><small>'+esc(x.year||'')+'</small></span><b>'+money(x.value)+'</b></div>').join(''):'<div class="vv202-empty">Fontes de recursos não retornadas.</div>'}</div>
          </article>

          <article class="vv202-panel vv202-bcb-card">
            <div class="vv202-panel-head"><div><span>TAXAS POR INSTITUIÇÃO</span><h3>Médias publicadas pelo BCB</h3></div><button data-bcb-source202="${esc(bankRates.source_url||'https://dadosabertos.bcb.gov.br/dataset/taxas-de-juros-de-operacoes-de-credito')}">${icon('bank')}</button></div>
            <div class="vv202-bank-rates">${institutionRates.length?institutionRates.map(x=>'<div><span><strong>'+esc(x.institution||'Instituição')+'</strong><small>'+esc([x.segment,x.modality,x.end_date].filter(Boolean).join(' / '))+'</small></span><b>'+num(x.annual_rate,2)+'% a.a.</b></div>').join(''):'<div class="vv202-empty">'+esc(bankRates.message||'Nenhuma modalidade rural/agro retornada no recorte atual.')+'</div>'}</div>
            <div class="vv202-bcb-note">Taxa média observada nas operações publicadas pelo BCB. Não é oferta nem taxa garantida para o cliente.</div>
          </article>

          <article class="vv202-panel vv202-bcb-card">
            <div class="vv202-panel-head"><div><span>SCR.DATA</span><h3>Risco agregado do mercado</h3></div><button data-bcb-source202="https://dadosabertos.bcb.gov.br/dataset/scr_data">${icon('database')}</button></div>
            <div class="vv202-bcb-note standalone"><strong>Base oficial mensal por UF:</strong> carteira ativa, inadimplência e ativos problemáticos agregados. O ViaVerdeCAR mantém esta fonte identificada e usa as séries SGS leves no fluxo automático; os arquivos mensais completos do SCR não são baixados silenciosamente porque são volumosos e não servem para consultar dívida individual por CPF/CNPJ.</div>
          </article>
        </div>
      </section>`;
  }

  function seriesLine202(x){
    const trend=Number(x.change_pct)||0;
    const arrow=trend>0?'↑':trend<0?'↓':'→';
    return '<div class="vv202-series-line"><span><strong>'+esc(x.label)+'</strong><small>'+esc(x.latest_date||'')+' • SGS '+esc(x.sgs_code)+'</small></span><b>'+num(x.latest_value,2)+' '+esc(x.unit||'')+'</b><em class="'+(trend>0?'up':trend<0?'down':'flat')+'">'+arrow+' '+num(Math.abs(trend),1)+'%</em></div>';
  }

  function timeline202(r){
    const c=r.car||{},events=[];
    if(c.data_cadastro)events.push({d:c.data_cadastro,t:'Inscrição no CAR',s:c.status||''});
    arr(r.environmental?.alerts).forEach(a=>events.push({d:a.detected_at||a.published_at,t:'Alerta MapBiomas'+(a.alert_code?' '+a.alert_code:''),s:a.area_ha?num(a.area_ha)+' ha':''}));
    arr(r.xray?.sicor?.operations).forEach(o=>events.push({d:o.issue_date||String(o.year||''),t:'Operação de crédito (SICOR)',s:[o.purpose,o.credit_value?money(o.credit_value):''].filter(Boolean).join(' / ')}));
    const ib=r.environmental?.environment?.ibama_embargos||r.car?.environment?.ibama_embargos||[];
    arr(ib).forEach(x=>events.push({d:x.date,t:'Embargo / ocorrência IBAMA',s:x.area||x.status||''}));
    if(c.data_atualizacao)events.push({d:c.data_atualizacao,t:'Atualização do CAR',s:c.condition||''});
    events.sort((a,b)=>dateValue202(b.d)-dateValue202(a.d));
    return events.length?events.slice(0,7).map((x,i)=>'<div class="vv202-time t'+(i%4)+'"><i></i><b>'+esc(shortDate202(x.d))+'</b><span><strong>'+esc(x.t)+'</strong><small>'+esc(x.s||'')+'</small></span></div>').join(''):'<div class="vv202-empty">Nenhum evento com data disponível.</div>';
  }

  function drawMap202(r){
    const node=el('vv202MapCanvas');if(!node||!window.L)return;
    try{map202?.remove()}catch(_){};map202=null;
    map202=L.map(node,{zoomControl:true,attributionControl:true}).setView([-18.5,-44],5);
    const sat=L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',{maxZoom:19,attribution:'Esri'}).addTo(map202);
    const street=L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Street_Map/MapServer/tile/{z}/{y}/{x}',{maxZoom:19,attribution:'Esri'});
    const overlays={},fit=[];
    const add=(label,raw,style,visible=true)=>{
      if(!raw)return;try{const obj=typeof raw==='string'?JSON.parse(raw):raw;const layer=L.geoJSON(obj,{style});overlays[label]=layer;if(visible){layer.addTo(map202);fit.push(layer)}}catch(_){}
    };
    add('CAR (SICAR)',r.car?.geojson,{color:'#ef3131',weight:3,fillColor:'#ffffff',fillOpacity:.03},true);
    const live=r.environmental?.environment||{}, ce=r.car?.environment||{};
    const env=(live.ibama_checked||live.funai_checked||live.icmbio_checked||live.mcr_checked)?live:ce;
    arr(env.ibama_embargos).forEach((x,i)=>add('Embargo IBAMA '+(i+1),x.geojson,{color:'#dc2626',weight:3,fillColor:'#ef4444',fillOpacity:.24},true));
    arr(env.indigenous_findings).forEach((x,i)=>add('Terra Indígena '+(i+1),x.geojson,{color:'#f59e0b',weight:3,fillColor:'#fbbf24',fillOpacity:.18},true));
    arr(env.federal_uc_findings).forEach((x,i)=>add('UC Federal '+(i+1),x.geojson,{color:'#16a34a',weight:3,fillColor:'#22c55e',fillOpacity:.16},true));
    arr(r.environmental?.alerts).forEach((x,i)=>add('MapBiomas '+(x.alert_code||i+1),x.geometry_geojson,{color:'#f97316',weight:3,fillColor:'#fb923c',fillOpacity:.20},true));
    arr(r.xray?.sicor?.operations).forEach((o,oi)=>arr(o.glebas).forEach((g,gi)=>add('SICOR '+(oi+1)+'.'+(gi+1),g.geojson,{color:'#7c3aed',weight:2,dashArray:'7 5',fillColor:'#8b5cf6',fillOpacity:.10},false)));
    L.control.layers({'Satélite':sat,'Mapa':street},overlays,{collapsed:false,position:'topright'}).addTo(map202);
    if(fit.length){const g=L.featureGroup(fit),b=g.getBounds();if(b.isValid())map202.fitBounds(b.pad(.08),{maxZoom:16})}
  }

  function toneCAR202(c){if(c.lookup_status==='cached'||c.lookup_status==='partial')return'orange';if(c.lookup_status==='unavailable'||c.lookup_status==='not_found')return'red';return'blue'}
  function countAttention202(r){return arr(r.warnings).length+arr(r.sources).filter(s=>s.status==='hit').reduce((n,s)=>n+(Number(s.count)||1),0)+arr(r.sources).filter(s=>['unavailable','partial','cached'].includes(s.status)).length}
  function overall202(v){return {complete:'Completa',partial:'Parcial',not_found:'Não localizado',error:'Erro'}[v]||'Conferir'}
  function dateValue202(v){const d=new Date(v);if(!isNaN(d))return d.getTime();const y=Number(String(v||'').match(/20\d{2}/)?.[0]||0);return y?Date.UTC(y,0,1):0}
  function shortDate202(v){if(!v)return'-';const d=new Date(v);if(!isNaN(d))return d.toLocaleDateString('pt-BR',{month:'short',year:'numeric'});return String(v)}
  function date202(v){if(!v)return'-';const d=new Date(v);return isNaN(d)?String(v):d.toLocaleDateString('pt-BR')}

  document.addEventListener('DOMContentLoaded',()=>setTimeout(install202,140));
})();