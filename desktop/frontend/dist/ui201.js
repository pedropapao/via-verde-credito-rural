/* ViaVerdeCAR 2.0.1 — workspace visual inspirado no mockup aprovado */
(()=>{
  const E=id=>document.getElementById(id);
  const A=v=>Array.isArray(v)?v:[];
  const H=v=>String(v??'').replace(/[&<>"']/g,m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]));
  const N=(v,d=2)=>(Number(v)||0).toLocaleString('pt-BR',{minimumFractionDigits:d,maximumFractionDigits:d});
  const MONEY=v=>(Number(v)||0).toLocaleString('pt-BR',{style:'currency',currency:'BRL'});
  let current=null;
  let overviewMap=null;
  let searchTimer=null;

  function install(){
    if(E('vvWorkspace201')) return;
    rebuildSidebar201();
    rebuildTopbar201();

    const dashboard=E('view-dashboard');
    if(!dashboard) return;
    dashboard.insertAdjacentHTML('afterbegin', workspaceHTML201());

    dashboard.querySelectorAll(':scope > .hero-card,:scope > .stats-grid,:scope > .dashboard-grid,#vvCentral200').forEach(x=>x.classList.add('vv-hide201'));

    E('vvSearch201').addEventListener('input',()=>{
      clearTimeout(searchTimer);
      searchTimer=setTimeout(()=>search201(false),320);
    });
    E('vvSearch201').addEventListener('keydown',ev=>{
      if(ev.key==='Enter'){ev.preventDefault();search201(true)}
    });
    E('vvAnalyze201').onclick=()=>search201(true);
    E('vvClear201').onclick=()=>{
      E('vvSearch201').value='';
      renderWelcome201();
      E('vvSearch201').focus();
    };

    document.addEventListener('keydown',ev=>{
      if((ev.ctrlKey||ev.metaKey)&&ev.key.toLowerCase()==='k'){
        ev.preventDefault();
        setView('dashboard');
        setTimeout(()=>E('vvSearch201')?.focus(),50);
      }
    });

    waitForData201();
  }

  function rebuildSidebar201(){
    const side=document.querySelector('.sidebar');
    if(!side||side.dataset.vv201==='1')return;
    side.dataset.vv201='1';
    const brand=side.querySelector('.brand');
    if(brand){
      brand.innerHTML='<div class="vv-logo201">◆</div><div><strong>ViaVerdeCAR</strong><span>CONSULTA E ANÁLISE RURAL</span></div><b class="vv-version-chip201">2.0.1</b>';
    }
    const nav=side.querySelector('nav');
    if(nav){
      nav.innerHTML=[
        ['dashboard','⌂','Início',''],
        ['clients','♙','Clientes',''],
        ['clients','⌑','Imóveis','property-panel'],
        ['car','⌕','Consulta CAR','car-toolbar'],
        ['car','♧','Raio X Ambiental','environment-panel'],
        ['car','▱','Raio X Fundiário',''],
        ['car','▣','Crédito Rural',''],
        ['car','⌖','Mapa','map-layout'],
        ['car','▤','Relatórios','report-panel'],
        ['car','□','Dossiês','report-panel'],
      ].map((x,i)=>'<button class="nav-item '+(i===0?'active':'')+'" data-vv-view201="'+x[0]+'" data-vv-section201="'+x[3]+'"><span>'+x[1]+'</span> '+x[2]+'</button>').join('');
      nav.querySelectorAll('[data-vv-view201]').forEach(b=>b.onclick=()=>{
        const view=b.dataset.vvView201;
        document.querySelectorAll('.sidebar .nav-item').forEach(x=>x.classList.remove('active'));b.classList.add('active');
        setView(view);
        const cls=b.dataset.vvSection201;
        if(cls)setTimeout(()=>document.querySelector('#view-'+view+' .'+cls)?.scrollIntoView({behavior:'smooth',block:'start'}),120);
      });
    }
    const foot=side.querySelector('.sidebar-footer');
    if(foot){
      foot.insertAdjacentHTML('beforebegin','<div class="vv-side-tools201"><button data-vv-tools201="sources">● <span>Fontes de Dados</span></button><button data-vv-tools201="backup">▱ <span>Backup / Banco</span></button><button data-vv-tools201="settings">ⓘ <span>Configurações</span></button></div>');
      side.querySelector('[data-vv-tools201="backup"]').onclick=()=>E('backupBtn')?.click();
      side.querySelector('[data-vv-tools201="settings"]').onclick=()=>setView('settings');
      side.querySelector('[data-vv-tools201="sources"]').onclick=()=>{setView('dashboard');setTimeout(()=>E('vvSources201')?.scrollIntoView({behavior:'smooth'}),80)};
    }
  }

  function rebuildTopbar201(){
    const top=document.querySelector('.topbar');
    if(!top||top.dataset.vv201==='1')return;
    top.dataset.vv201='1';
    top.classList.add('vv-topbar201');
    top.innerHTML='<div class="vv-top-search201"><span>⌕</span><input id="vvTopSearch201" placeholder="Digite o CAR, CPF, CNPJ, nome do imóvel, matrícula ou município..." /><button id="vvTopAnalyze201">ANALISAR</button></div><div class="vv-top-actions201"><button data-vv-top201="history">↶<span>Histórico</span></button><button data-vv-top201="settings">⚙<span>Configurações</span></button><button data-vv-top201="help">?<span>Ajuda</span></button></div><h1 id="pageTitle" class="vv-hidden-title201">Visão geral</h1><button id="backupBtn" class="vv-hidden-title201">Backup</button><button id="goCarBtn" class="vv-hidden-title201">CAR</button>';
    const input=E('vvTopSearch201');
    const btn=E('vvTopAnalyze201');
    input.addEventListener('input',()=>{if(E('vvSearch201'))E('vvSearch201').value=input.value});
    input.addEventListener('keydown',ev=>{if(ev.key==='Enter'){ev.preventDefault(); if(E('vvSearch201'))E('vvSearch201').value=input.value; search201(true)}});
    btn.onclick=()=>{if(E('vvSearch201'))E('vvSearch201').value=input.value;search201(true)};
    top.querySelector('[data-vv-top201="settings"]').onclick=()=>setView('settings');
    top.querySelector('[data-vv-top201="history"]').onclick=()=>{setView('car');setTimeout(()=>document.querySelector('.history-panel')?.scrollIntoView({behavior:'smooth'}),100)};
    top.querySelector('[data-vv-top201="help"]').onclick=()=>toast('Digite CAR para o Raio X automático; CPF/CNPJ consulta vínculos locais e mostra caminhos oficiais disponíveis.');
  }

  function workspaceHTML201(){
    return `
      <div id="vvWorkspace201" class="vv-workspace201">
        <section class="vv-search-hero201">
          <div class="vv-search-copy201"><span class="eyebrow">CENTRAL AUTOMÁTICA DO IMÓVEL RURAL</span><h2>CAR → mapa → ambiente → fundiário → crédito → dossiê</h2><p>Uma entrada para concentrar o trabalho repetitivo. O sistema mantém separado o que foi encontrado, o que não ocorreu e o que ficou indisponível.</p></div>
          <div class="vv-search-box201"><span>⌕</span><input id="vvSearch201" placeholder="CAR, CPF, CNPJ, cliente, imóvel, matrícula ou município" /><button id="vvAnalyze201">ANALISAR</button><button id="vvClear201" title="Limpar">×</button></div>
          <div class="vv-search-help201"><span>CAR executa Raio X completo</span><span>CPF/CNPJ cruza vínculos locais</span><span>Ctrl+K abre a busca</span></div>
        </section>
        <section id="vvDocResearch201" class="vv-doc201 hidden"></section>
        <section id="vvOverview201"></section>
      </div>`;
  }

  function waitForData201(){
    let tries=0;
    const tick=setInterval(()=>{
      tries++;
      if(typeof state!=='undefined' && Array.isArray(state.properties)){
        clearInterval(tick);
        renderWelcome201();
      }else if(tries>50)clearInterval(tick);
    },120);
  }

  function renderWelcome201(){
    current=null;
    E('vvDocResearch201').classList.add('hidden');
    E('vvOverview201').innerHTML=`
      <div class="vv-welcome201">
        <div class="vv-welcome-main201"><span class="vv-welcome-icon201">⌖</span><div><span class="eyebrow">PRONTO PARA ANALISAR</span><h3>Informe um CAR para montar a visão geral automaticamente</h3><p>O ViaVerdeCAR cruza o perímetro com as fontes configuradas e monta o painel com mapa, alertas, SIGEF, SICOR e ações do imóvel.</p></div></div>
        <div id="vvRecentGrid201" class="vv-recent-grid201"></div>
      </div>`;
    renderRecent201();
  }

  function renderRecent201(){
    const box=E('vvRecentGrid201'); if(!box)return;
    const list=A(state?.properties).slice().sort((a,b)=>String(b.updated_at||'').localeCompare(String(a.updated_at||''))).slice(0,6);
    if(!list.length){box.innerHTML='<div class="vv-empty201">Nenhum imóvel cadastrado ainda.</div>';return}
    box.innerHTML=list.map((p,i)=>`<button class="vv-recent-card201" data-vv-recent201="${i}"><span>${p.car_number?'CAR VINCULADO':'CADASTRO LOCAL'}</span><strong>${H(p.name||'Imóvel')}</strong><small>${H([p.client_name,p.municipality,p.uf].filter(Boolean).join(' • '))}</small><b>${p.declared_area_ha?N(p.declared_area_ha)+' ha':'Área não informada'}</b></button>`).join('');
    box.querySelectorAll('[data-vv-recent201]').forEach(b=>b.onclick=async()=>{
      const p=list[Number(b.dataset.vvRecent201)]; if(!p)return;
      if(p.car_number){E('vvSearch201').value=p.car_number;E('vvTopSearch201').value=p.car_number;await runCAR201(p.car_number,p.id,true)}
      else{setView('clients');selectClient(p.client_id)}
    });
  }

  async function search201(run){
    const q=(E('vvSearch201')?.value||E('vvTopSearch201')?.value||'').trim();
    if(!q){renderWelcome201();return}
    E('vvSearch201').value=q; E('vvTopSearch201').value=q;
    try{
      const r=await api().SearchEverything(q);
      if(run && r.can_analyze_car && r.valid){
        const p=A(r.hits).find(x=>x.property_id&&x.car);
        return runCAR201(r.normalized,p?.property_id||0,true);
      }
      if(r.mode==='cpf'||r.mode==='cnpj')return renderDocumentResearch201(r);
      renderSearchResults201(r);
    }catch(err){toast(String(err),true)}
  }

  function renderSearchResults201(r){
    E('vvDocResearch201').classList.add('hidden');
    const hits=A(r.hits);
    E('vvOverview201').innerHTML=`<section class="vv-search-results201"><div class="vv-section-title201"><div><span class="eyebrow">RESULTADOS</span><h3>${H(r.message||'Busca')}</h3></div><b>${hits.length} resultado(s)</b></div><div class="vv-result-list201">${hits.length?hits.map((h,i)=>`<article><div><span>${H(h.kind||'registro')}</span><strong>${H(h.title)}</strong><small>${H(h.subtitle||h.car||h.cpf_cnpj||'')}</small></div><div class="vv-row-actions201">${h.car?'<button data-vv-an201="'+i+'">Analisar CAR</button>':''}<button data-vv-open201="${i}">Abrir</button></div></article>`).join(''):'<div class="vv-empty201">Nenhum cadastro local encontrado.</div>'}</div></section>`;
    document.querySelectorAll('[data-vv-an201]').forEach(b=>b.onclick=()=>{const h=hits[Number(b.dataset.vvAn201)];runCAR201(h.car,h.property_id||0,true)});
    document.querySelectorAll('[data-vv-open201]').forEach(b=>b.onclick=async()=>openHit201(hits[Number(b.dataset.vvOpen201)]));
  }

  function renderDocumentResearch201(r){
    const box=E('vvDocResearch201'); box.classList.remove('hidden');
    const hits=A(r.hits), opts=A(r.official_options);
    box.innerHTML=`
      <div class="vv-doc-head201"><div><span class="eyebrow">PESQUISA POR ${H(String(r.mode||'').toUpperCase())}</span><h3>${H(r.message)}</h3><p>${H(r.privacy_notice||'')}</p></div><span class="vv-doc-number201">${H(r.normalized||r.query)}</span></div>
      <div class="vv-doc-grid201">
        <div class="vv-doc-local201"><h4>Vínculos já conhecidos pelo ViaVerdeCAR</h4>${hits.length?hits.map((h,i)=>`<button data-vv-doc-hit201="${i}"><div><strong>${H(h.title)}</strong><span>${H(h.subtitle||'')}</span><small>${H(h.car||h.cpf_cnpj||'')}</small></div><b>→</b></button>`).join(''):'<div class="vv-empty201">Nenhum cliente/imóvel local vinculado a este documento.</div>'}</div>
        <div class="vv-doc-official201"><h4>Caminhos oficiais</h4>${opts.map((o,i)=>`<article class="status-${H(o.status)}"><div><span>${o.automatic?'AUTOMÁTICO':'OFICIAL'}</span><strong>${H(o.label)}</strong></div><p>${H(o.detail)}</p>${o.url?'<button data-vv-source-link201="'+i+'">Abrir fonte oficial</button>':''}</article>`).join('')}</div>
      </div>
      <div class="vv-doc-note201"><b>Importante:</b> o ViaVerdeCAR não transforma coincidência de nome ou cadastro local em prova de titularidade do CAR. A relação só é afirmada quando já foi vinculada pelo usuário ou comprovada por fonte/documento apropriado.</div>`;
    hits.forEach((_,i)=>{const b=box.querySelector('[data-vv-doc-hit201="'+i+'"]');if(b)b.onclick=()=>openHit201(hits[i])});
    opts.forEach((o,i)=>{const b=box.querySelector('[data-vv-source-link201="'+i+'"]');if(b)b.onclick=()=>openExternal(o.url)});
    E('vvOverview201').innerHTML='';
  }

  async function openHit201(h){
    if(!h)return;
    if(h.car)return runCAR201(h.car,h.property_id||0,true);
    if(h.client_id){setView('clients');selectClient(h.client_id)}
  }

  async function runCAR201(car,propertyID,force){
    const box=E('vvOverview201');
    E('vvDocResearch201').classList.add('hidden');
    box.innerHTML=loading201(car);
    box.scrollIntoView({behavior:'smooth',block:'start'});
    const btn=E('vvAnalyze201'); if(btn){btn.disabled=true;btn.textContent='ANALISANDO…'}
    try{
      const r=await api().RunCARAutomation(car,Number(propertyID)||0,!!force);
      current=r; window.__vv201LastResult=r;
      renderOverview201(r);
      state.car=r.car;
      if(r.car?.car){
        renderCAR(r.car);
        if(r.car.geojson)drawGeoJSON('car',r.car.geojson);
        E('carInput').value=r.car.car;
      }
      await Promise.all([loadProperties(),loadDashboard()]);
      if(r.property_id)state.selectedProperty=state.properties.find(p=>p.id===r.property_id)||state.selectedProperty;
    }catch(err){
      box.innerHTML='<div class="vv-fatal201"><strong>Não foi possível concluir o Raio X.</strong><p>'+H(String(err))+'</p><span>Falha de fonte não é tratada como ausência de ocorrência.</span></div>';
      toast(String(err),true);
    }finally{
      if(btn){btn.disabled=false;btn.textContent='ANALISAR'}
    }
  }

  function loading201(car){
    return `<div class="vv-loading201"><div class="vv-loader201"></div><div><span class="eyebrow">RAIO X AUTOMÁTICO</span><h3>Consultando o imóvel e cruzando as bases públicas…</h3><p>${H(car)}</p></div><div class="vv-load-track201"><i></i></div><div class="vv-load-steps201"><span>CAR</span><span>Mapa</span><span>Ambiental</span><span>Fundiário</span><span>Crédito</span><span>Resumo</span></div></div>`;
  }

  function renderOverview201(r){
    const car=r.car||{}, env=r.environmental||{}, x=r.xray||{}, sigef=x.sigef||{}, sicor=x.sicor||{};
    const area=Number(car.area_ha||car.geometry_area_ha)||0;
    const envCount=(Number(env.environment?.ibama_embargo_count)||0)+(Number(env.environment?.indigenous_count)||0)+(Number(env.environment?.federal_uc_count)||0)+(env.environment?.mcr_listed?1:0)+(Number(env.mapbiomas?.total_alerts)||0)+(Number(env.profile?.fire?.feature_count)||0);
    const affected=Number(env.summary?.alert_area_in_car_ha)||0;
    const attention=countAttention201(r);
    const carLabel=car.lookup_status==='cached'?'Cache local':car.lookup_status==='partial'?'Ficha parcial':car.status||((car.found||car.has_geometry)?'Localizado':'Não localizado');
    const lookupClass=car.lookup_status==='cached'||car.lookup_status==='partial'?'warn':car.public_confirmed===false?'warn':'ok';

    E('vvOverview201').innerHTML=`
      <section class="vv-summary-cards201" id="vvSources201">
        ${summaryCard201('⌂','CAR',carLabel,'Área',area?N(area)+' ha':'—','Município',H([car.municipality,car.uf].filter(Boolean).join(' / ')||'—'),lookupClass)}
        ${summaryCard201('◆','Ambiental',String(envCount),'Ocorrências/alertas',envCount+' registro(s)','Área alertada',affected?N(affected)+' ha'+(area?' ('+N(affected/area*100,1)+'%)':''):'—',envCount?'warn':'ok')}
        ${summaryCard201('▱','Fundiário',String(Number(sigef.parcel_count)||0),'SIGEF',(Number(sigef.parcel_count)||0)+' parcela(s)','Melhor cobertura',sigef.best_car_coverage_pct?N(sigef.best_car_coverage_pct,1)+'%':'—',(Number(sigef.parcel_count)||0)?'warn':'ok')}
        ${summaryCard201('▣','Crédito Rural',String(Number(sicor.operation_count)||0),'Operações',(Number(sicor.operation_count)||0)+' registro(s)','Valor total',sicor.total_credit_value?MONEY(sicor.total_credit_value):'—',(Number(sicor.operation_count)||0)?'info':'ok')}
        ${summaryCard201('!','Atenções',String(attention),'Pontos de verificação',attention+' item(ns)','Status',H(overallLabel201(r.overall_status)),attention?'danger':'ok')}
      </section>

      <section class="vv-main-grid201">
        <article class="vv-panel201 vv-property201">
          <div class="vv-panel-head201"><div><span class="eyebrow">RESUMO DO IMÓVEL</span><h3>${H(car.property_name||state?.selectedProperty?.name||'Imóvel rural')}</h3></div><button id="vvOpenCar201" title="Abrir CAR">⌖</button></div>
          <dl>
            <div><dt>CAR</dt><dd class="mono">${H(car.car||'—')}</dd></div>
            <div><dt>Município/UF</dt><dd>${H([car.municipality,car.uf].filter(Boolean).join(' / ')||'—')}</dd></div>
            <div><dt>Área</dt><dd>${area?N(area)+' ha':'—'}</dd></div>
            <div><dt>Módulos fiscais</dt><dd>${car.fiscal_modules?N(car.fiscal_modules,2):'—'}</dd></div>
            <div><dt>Situação</dt><dd><span class="vv-tag201 ${lookupClass}">${H(carLabel)}</span></dd></div>
            <div><dt>Condição</dt><dd>${H(car.condition||'—')}</dd></div>
            <div><dt>Inscrição</dt><dd>${H(date201(car.data_cadastro))}</dd></div>
            <div><dt>Atualização</dt><dd>${H(date201(car.data_atualizacao))}</dd></div>
          </dl>
          ${car.lookup_detail?'<div class="vv-car-source201 status-'+H(car.lookup_status||'')+'"><b>SICAR:</b> '+H(car.lookup_detail)+'</div>':''}
          <div class="vv-property-actions201"><button id="vvKml201">Gerar KML (SICAR)</button><button id="vvMapAction201">Abrir no mapa</button>${!r.property_id&&car.found?'<button id="vvLink201">Salvar / Vincular</button>':''}</div>
          <div id="vvLinkBox201" class="vv-linkbox201 hidden"></div>
        </article>

        <article class="vv-panel201 vv-map-panel201">
          <div class="vv-panel-head201"><div><span class="eyebrow">MAPA</span><h3>Imóvel e ocorrências</h3></div><span class="vv-live201">camadas reais</span></div>
          <div id="vvMap201"></div>
          <div class="vv-map-caption201">CAR, alertas e camadas públicas que retornaram geometria nesta análise.</div>
        </article>

        <article class="vv-panel201 vv-timeline-panel201">
          <div class="vv-panel-head201"><div><span class="eyebrow">RASTREABILIDADE</span><h3>Linha do tempo</h3></div></div>
          <div class="vv-timeline201">${timelineHTML201(r)}</div>
        </article>
      </section>

      <nav class="vv-tabs201">
        <button class="active" data-vv-tab201="summary">▣ Resumo</button><button data-vv-tab201="car">◇ CAR</button><button data-vv-tab201="environment">◇ Ambiental</button><button data-vv-tab201="land">▱ Fundiário</button><button data-vv-tab201="credit">▣ Crédito Rural</button><button data-vv-tab201="map">⌖ Mapa</button><button data-vv-tab201="docs">□ Documentos</button><button data-vv-tab201="reports">▤ Relatórios</button>
      </nav>

      <section class="vv-bottom-grid201">
        <article class="vv-panel201"><div class="vv-panel-head201"><div><span class="eyebrow">RAIO X AMBIENTAL</span><h3>Ocorrências e bases</h3></div><span class="vv-count201">${envCount}</span></div>${environmentHTML201(r)}</article>
        <article class="vv-panel201"><div class="vv-panel-head201"><div><span class="eyebrow">RAIO X FUNDIÁRIO</span><h3>SIGEF / INCRA</h3></div><span class="vv-count201 warn">${Number(sigef.parcel_count)||0}</span></div>${landHTML201(r)}</article>
        <article class="vv-panel201"><div class="vv-panel-head201"><div><span class="eyebrow">CRÉDITO RURAL</span><h3>SICOR</h3></div><span class="vv-count201 info">${Number(sicor.operation_count)||0}</span></div>${creditHTML201(r)}</article>
        <article class="vv-panel201 vv-actions201"><div class="vv-panel-head201"><div><span class="eyebrow">AÇÕES RÁPIDAS</span><h3>Próximo passo</h3></div></div>
          <button class="primary" id="vvDossier201" ${r.property_id?'':'disabled'}>▤ <span><b>Gerar dossiê completo</b><small>PDF + KML + mapas + evidências</small></span></button>
          <button class="blue" id="vvRefresh201">↻ <span><b>Atualizar tudo</b><small>Executa novamente as análises</small></span></button>
          <button id="vvExport201" ${car.has_geometry?'':'disabled'}>⇩ <span><b>Exportar KML</b><small>Perímetro público SICAR</small></span></button>
          <button id="vvReport201" ${r.property_id?'':'disabled'}>▤ <span><b>Abrir relatórios</b><small>Conferência técnica do imóvel</small></span></button>
        </article>
      </section>
      ${A(r.warnings).length?'<details class="vv-warnings201"><summary>Avisos e limitações desta análise ('+A(r.warnings).length+')</summary>'+A(r.warnings).map(x=>'<p>• '+H(x)+'</p>').join('')+'</details>':''}
    `;

    bindOverview201(r);
    setTimeout(()=>drawOverviewMap201(r),80);
  }

  function summaryCard201(icon,title,badge,l1,v1,l2,v2,kind){
    return `<article class="vv-summary-card201 ${kind}"><div class="vv-summary-head201"><span class="vv-summary-icon201">${icon}</span><strong>${H(title)}</strong><b>${H(badge)}</b></div><div class="vv-summary-pairs201"><div><span>${H(l1)}</span><strong>${v1}</strong></div><div><span>${H(l2)}</span><strong>${v2}</strong></div></div></article>`;
  }

  function bindOverview201(r){
    E('vvOpenCar201').onclick=()=>openCarWorkspace201(r);
    E('vvMapAction201').onclick=()=>openCarWorkspace201(r,'map');
    E('vvKml201').onclick=()=>{openCarWorkspace201(r);setTimeout(()=>E('exportKmlBtn')?.click(),150)};
    if(E('vvLink201'))E('vvLink201').onclick=()=>renderLinkBox201(r);
    E('vvRefresh201').onclick=()=>runCAR201(r.car.car,r.property_id||0,true);
    E('vvExport201').onclick=()=>{openCarWorkspace201(r);setTimeout(()=>E('exportKmlBtn')?.click(),150)};
    E('vvDossier201').onclick=async()=>{
      if(!r.property_id)return;
      await openCarWorkspace201(r);
      setTimeout(()=>E('packageBtn')?.click(),180);
    };
    E('vvReport201').onclick=async()=>{await openCarWorkspace201(r);setTimeout(()=>document.querySelector('.report-panel')?.scrollIntoView({behavior:'smooth'}),120)};
    document.querySelectorAll('[data-vv-tab201]').forEach(b=>b.onclick=()=>tab201(b.dataset.vvTab201,r));
  }

  async function openCarWorkspace201(r,section){
    setView('car');
    if(r.property_id){
      if(E('carPropertySelect'))E('carPropertySelect').value=String(r.property_id);
      await selectCarProperty(r.property_id);
    }else{
      state.car=r.car;renderCAR(r.car);if(r.car.geojson)drawGeoJSON('car',r.car.geojson);E('carInput').value=r.car.car||'';
    }
    if(section==='map')setTimeout(()=>document.querySelector('.map-layout')?.scrollIntoView({behavior:'smooth'}),100);
  }

  function tab201(key,r){
    const map={car:'car-toolbar',environment:'environment-panel',map:'map-layout',reports:'report-panel'};
    if(key==='summary')return E('vvSources201')?.scrollIntoView({behavior:'smooth'});
    if(key==='credit'||key==='land'||key==='docs')return openCarWorkspace201(r);
    openCarWorkspace201(r).then(()=>setTimeout(()=>document.querySelector('.'+(map[key]||'car-toolbar'))?.scrollIntoView({behavior:'smooth'}),100));
  }

  function renderLinkBox201(r){
    const box=E('vvLinkBox201');if(!box)return;
    const clients=A(state?.clients);
    box.classList.remove('hidden');
    box.innerHTML=clients.length?`<select id="vvLinkClient201"><option value="">Selecione o cliente…</option>${clients.map(c=>'<option value="'+Number(c.id)+'">'+H(c.name)+(c.cpf_cnpj?' • '+H(c.cpf_cnpj):'')+'</option>').join('')}</select><button id="vvLinkSave201">Vincular este CAR</button><small>O vínculo é uma decisão do usuário; o app não deduz titularidade.</small>`:'<p>Cadastre um cliente antes de vincular este CAR.</p><button id="vvGoClient201">Cadastrar cliente</button>';
    if(E('vvGoClient201'))E('vvGoClient201').onclick=()=>setView('clients');
    if(E('vvLinkSave201'))E('vvLinkSave201').onclick=async()=>{
      const id=Number(E('vvLinkClient201').value)||0;if(!id){toast('Selecione o cliente.',true);return}
      try{
        const p=await api().SaveAnalyzedCARToClient(id,r.car.car);
        await Promise.all([loadClients(),loadProperties(),loadDashboard()]);
        toast('Imóvel vinculado e salvo.');
        await runCAR201(r.car.car,p.id,false);
      }catch(err){toast(String(err),true)}
    };
  }

  function source201(r,key){
    return A(r.sources).find(x=>x.key===key)||{};
  }
  function statusText201(s){
    return {ok:'Sem ocorrência',hit:'Ocorrência',unavailable:'Base indisponível',not_configured:'Não configurado',partial:'Parcial',cached:'Cache local',not_found:'Não localizado',not_saved:'Não salvo'}[s]||s||'Não consultado';
  }
  function sourceRow201(r,key,label){
    const s=source201(r,key), count=Number(s.count)||0;
    return `<div class="vv-data-row201 status-${H(s.status||'none')}"><span class="vv-dot201"></span><strong>${H(label)}</strong><span>${H(statusText201(s.status))}</span><b>${count?count+' registro(s)':'—'}</b></div>`;
  }
  function environmentHTML201(r){
    return '<div class="vv-data-list201">'+
      sourceRow201(r,'ibama','IBAMA • Embargos')+
      sourceRow201(r,'mapbiomas','MapBiomas • Desmatamento')+
      sourceRow201(r,'inpe_fire','INPE • Focos de calor')+
      sourceRow201(r,'funai','FUNAI • Terras Indígenas')+
      sourceRow201(r,'icmbio','ICMBio • Unidades de Conservação')+
      sourceRow201(r,'mcr','MMA • MCR/PRODES')+
      '</div>';
  }
  function landHTML201(r){
    const s=r.xray?.sigef||{};
    return `<div class="vv-data-list201"><div class="vv-data-row201"><span class="vv-dot201 warn"></span><strong>SIGEF / INCRA</strong><span>${s.available?'Consultado':'Indisponível'}</span><b>${Number(s.parcel_count)||0} parcela(s)</b></div><div class="vv-data-row201"><span></span><strong>Cobertura CAR</strong><span>melhor correspondência</span><b>${s.best_car_coverage_pct?N(s.best_car_coverage_pct,1)+'%':'—'}</b></div><div class="vv-data-row201"><span></span><strong>Matrículas/registros</strong><span>retornados</span><b>${Number(s.registry_count)||0}</b></div><div class="vv-data-row201"><span></span><strong>Fonte</strong><span>${H(s.message||'Consulta espacial pública')}</span><b></b></div></div>`;
  }
  function creditHTML201(r){
    const s=r.xray?.sicor||{}, ops=A(s.operations);
    if(!ops.length)return '<div class="vv-empty201">'+H(source201(r,'sicor').detail||'Nenhuma operação pública retornada para o CAR.')+'</div>';
    return '<div class="vv-data-list201">'+ops.slice(0,4).map(o=>`<div class="vv-credit-row201"><div><strong>${H(o.purpose||o.activity||o.product||'Operação SICOR')}</strong><span>${H([o.institution_name,o.program_name,o.year].filter(Boolean).join(' • '))}</span></div><b>${MONEY(o.credit_value)}</b></div>`).join('')+'</div>';
  }

  function timelineHTML201(r){
    const car=r.car||{}, events=[];
    if(car.data_cadastro)events.push({d:car.data_cadastro,t:'Inscrição/registro do CAR',s:car.status||''});
    A(r.environmental?.alerts).forEach(a=>events.push({d:a.detected_at||a.published_at,t:'Alerta MapBiomas '+(a.alert_code||''),s:a.area_ha?N(a.area_ha)+' ha':''}));
    A(r.xray?.sicor?.operations).forEach(o=>events.push({d:o.issue_date||String(o.year||''),t:'Operação de crédito (SICOR)',s:[o.purpose,o.credit_value?MONEY(o.credit_value):''].filter(Boolean).join(' • ')}));
    if(car.data_atualizacao)events.push({d:car.data_atualizacao,t:'Atualização do CAR',s:car.condition||''});
    events.sort((a,b)=>dateValue201(b.d)-dateValue201(a.d));
    if(!events.length)return '<div class="vv-empty201">Nenhum evento com data disponível.</div>';
    return events.slice(0,7).map((x,i)=>`<div class="vv-time-item201 c${i%4}"><i></i><div><b>${H(shortDate201(x.d))}</b><strong>${H(x.t)}</strong><span>${H(x.s||'')}</span></div></div>`).join('');
  }

  function dateValue201(v){const d=new Date(v);if(!isNaN(d))return d.getTime();const y=Number(String(v||'').match(/20\d{2}/)?.[0]||0);return y?Date.UTC(y,0,1):0}
  function date201(v){if(!v)return '—';const d=new Date(v);return isNaN(d)?String(v):d.toLocaleDateString('pt-BR')}
  function shortDate201(v){if(!v)return '—';const d=new Date(v);if(!isNaN(d))return d.toLocaleDateString('pt-BR',{month:'short',year:'numeric'});return String(v)}
  function overallLabel201(v){return {complete:'Completa',partial:'Parcial',not_found:'Não localizado',error:'Erro'}[v]||'Conferir'}
  function countAttention201(r){
    const warningCount=A(r.warnings).length;
    const hits=A(r.sources).filter(s=>s.status==='hit').reduce((n,s)=>n+(Number(s.count)||1),0);
    const unavailable=A(r.sources).filter(s=>['unavailable','partial','cached'].includes(s.status)).length;
    return warningCount+hits+unavailable;
  }

  function drawOverviewMap201(r){
    const el=E('vvMap201');if(!el||!window.L)return;
    if(overviewMap){try{overviewMap.remove()}catch(_){}overviewMap=null}
    overviewMap=L.map(el,{zoomControl:true,attributionControl:true}).setView([-18.5,-44],5);
    const sat=L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',{maxZoom:19,attribution:'Tiles © Esri'}).addTo(overviewMap);
    const street=L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Street_Map/MapServer/tile/{z}/{y}/{x}',{maxZoom:19,attribution:'Tiles © Esri'});
    const base={'Satélite':sat,'Mapa':street}, overlays={}; const fit=[];
    const add=(label,raw,style,show=true)=>{
      if(!raw)return;
      try{
        const obj=typeof raw==='string'?JSON.parse(raw):raw;
        const layer=L.geoJSON(obj,{style});
        overlays[label]=layer;if(show){layer.addTo(overviewMap);fit.push(layer)}
      }catch(_){}
    };
    add('CAR (SICAR)',r.car?.geojson,{color:'#ff3b30',weight:3,fillColor:'#ffffff',fillOpacity:.03},true);
    const env=r.environmental?.environment||{};
    A(env.ibama_embargos).forEach((x,i)=>add('Embargo IBAMA '+(i+1),x.geojson,{color:'#e11d48',weight:3,fillColor:'#ef4444',fillOpacity:.25},true));
    A(env.indigenous_findings).forEach((x,i)=>add('Terra Indígena '+(i+1),x.geojson,{color:'#f59e0b',weight:3,fillColor:'#fbbf24',fillOpacity:.20},true));
    A(env.federal_uc_findings).forEach((x,i)=>add('UC Federal '+(i+1),x.geojson,{color:'#16a34a',weight:3,fillColor:'#22c55e',fillOpacity:.18},true));
    A(r.environmental?.alerts).forEach((x,i)=>{if(x.geometry_geojson)add('MapBiomas '+(x.alert_code||i+1),x.geometry_geojson,{color:'#f97316',weight:3,fillColor:'#fb923c',fillOpacity:.23},true)});
    A(r.xray?.sicor?.operations).forEach((o,oi)=>A(o.glebas).forEach((g,gi)=>add('SICOR '+(oi+1)+'.'+(gi+1),g.geojson,{color:'#7c3aed',weight:2,dashArray:'7 5',fillColor:'#8b5cf6',fillOpacity:.10},false)));
    L.control.layers(base,overlays,{collapsed:false,position:'topright'}).addTo(overviewMap);
    if(fit.length){const group=L.featureGroup(fit);const b=group.getBounds();if(b.isValid())overviewMap.fitBounds(b.pad(.08),{maxZoom:16})}
  }

  document.addEventListener('DOMContentLoaded',()=>setTimeout(install,40));
})();
