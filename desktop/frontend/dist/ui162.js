/* ViaVerdeCAR 1.6.2 — arquitetura de informação essencial primeiro */
(function(){
  const g=id=>document.getElementById(id);

  function clickTab(name){
    const b=document.querySelector('[data-car-tab131="'+name+'"]');
    if(b)b.click();
  }

  function renamePrimaryTabs162(){
    const labels={summary:'Visão geral',xray:'Raio X',map:'Mapa',planning:'Projeto',docs:'Documentos'};
    Object.entries(labels).forEach(([key,label])=>{
      const b=document.querySelector('[data-car-tab131="'+key+'"]');
      if(!b||b.dataset.ui162Label==='1')return;
      b.dataset.ui162Label='1';
      const icon=b.querySelector('.car-tab161-icon');
      const count=b.querySelector('.tab-count131');
      [...b.childNodes].forEach(n=>{if(n.nodeType===3)n.remove()});
      if(icon)icon.after(document.createTextNode(' '+label));
      else b.prepend(document.createTextNode(label));
      if(count&&!b.contains(count))b.appendChild(count);
    });
  }

  function simplifyDashboard162(){
    const view=g('view-dashboard'),grid=view?.querySelector('.dashboard-grid');
    if(!view||!grid||g('dashboardHelp162'))return;
    const d=document.createElement('details');
    d.id='dashboardHelp162';d.className='dashboard-help162';
    const s=document.createElement('summary');
    s.textContent='Como usar e informações do aplicativo';
    grid.parentNode.insertBefore(d,grid);
    d.appendChild(s);d.appendChild(grid);
  }

  function buildEssential162(){
    const summary=g('carTabSummary131');if(!summary||g('essential162'))return;
    const box=document.createElement('section');box.id='essential162';box.className='essential162';
    box.innerHTML=
      '<article class="essential-main162"><div><span class="eyebrow">O QUE IMPORTA AGORA</span><h3 id="essentialTitle162">Consulte um CAR para começar</h3><p id="essentialMessage162">O Via Verde vai destacar somente situação, área, pendências e os próximos passos.</p></div><div class="essential-actions162"><button class="btn primary" id="essentialXray162">Abrir Raio X</button><button class="btn ghost" id="essentialMap162">Ver mapa</button><button class="btn ghost" id="essentialProject162">Projeto</button></div></article>'+
      '<article class="essential-card162"><span>Situação</span><strong id="essentialStatus162">—</strong><small id="essentialStatusNote162">Aguardando CAR</small></article>'+
      '<article class="essential-card162"><span>Área</span><strong id="essentialArea162">—</strong><small id="essentialAreaNote162">Área geométrica / SICAR</small></article>'+
      '<article class="essential-card162"><span>Atenções</span><strong id="essentialAttention162">—</strong><small id="essentialAttentionNote162">Alertas e conferências</small></article>';
    const grid=summary.querySelector('.summary-grid131');
    summary.insertBefore(box,grid||summary.firstChild);
    g('essentialXray162').onclick=()=>clickTab('xray');
    g('essentialMap162').onclick=()=>clickTab('map');
    g('essentialProject162').onclick=()=>clickTab('planning');
  }

  function moveTechnicalSummary162(){
    const summary=g('carTabSummary131');if(!summary||g('technicalDetails162'))return;
    const strip=summary.querySelector('.professional-strip');
    const conference=summary.querySelector('.conference-panel');
    if(!strip&&!conference)return;
    const d=document.createElement('details');d.id='technicalDetails162';d.className='technical-details162';
    const sm=document.createElement('summary');sm.textContent='Conferência técnica, KML e validações';
    const c=document.createElement('div');c.className='technical-content162';
    d.append(sm,c);
    if(strip)c.appendChild(strip);
    if(conference)c.appendChild(conference);
    summary.appendChild(d);
  }

  function textOf(id){return (g(id)?.textContent||'').trim()}
  function setText162(id,value){const el=g(id);if(el&&el.textContent!==String(value))el.textContent=String(value)}
  function updateEssential162(){
    if(!g('essential162'))return;
    const has=!!state?.car?.found;
    const status=textOf('carStatusBadge')||'—';
    const area=textOf('rGeoArea')&&textOf('rGeoArea')!=='—'?textOf('rGeoArea'):textOf('rArea');
    const env=state?.car?.environment||{};
    const qualityWarnings=document.querySelectorAll('#qualityChecks .quality-item.warning,#qualityChecks .quality-item.error').length;
    const environmentalHits=(Number(env.ibama_embargo_count)||0)+(Number(env.indigenous_count)||0)+(Number(env.federal_uc_count)||0)+(env.mcr_listed?1:0);
    const total=qualityWarnings+environmentalHits;

    setText162('essentialStatus162',has?status:'—');
    setText162('essentialStatusNote162',has?(textOf('rMunicipality')||'CAR consultado'):'Aguardando CAR');
    setText162('essentialArea162',has?(area||'—'):'—');
    setText162('essentialAreaNote162',has?'Área geométrica / declarada':'Área geométrica / SICAR');
    setText162('essentialAttention162',has?String(total):'—');
    setText162('essentialAttentionNote162',has?(total?'itens para revisar':'nenhum alerta direto nesta tela'):'Alertas e conferências');
    setText162('essentialTitle162',has?(state.car.property_name||state.car.car||'Imóvel consultado'):'Consulte um CAR para começar');
    setText162('essentialMessage162',!has
      ?'O Via Verde vai destacar somente situação, área, pendências e os próximos passos.'
      :total
        ?'Há '+total+' ponto(s) que merecem conferência. Abra o Raio X para ver apenas o que exige atenção.'
        :'Consulta carregada. Use o Raio X para crédito, SIGEF e ambiental; mapa e projeto ficam separados.');
  }

  function addSecondaryEntry162(){
    const xray=g('xrayWorkspace150');if(!xray)return;
    const hero=xray.querySelector('.xray-hero150');if(!hero||hero.querySelector('.xray-secondary162'))return;
    const actions=hero.querySelector('.xray-actions150');
    const row=document.createElement('div');row.className='essential-actions162 xray-secondary162';
    row.innerHTML='<button class="btn ghost" data-open-secondary162="environment">Ambiental detalhado</button><button class="btn ghost" data-open-secondary162="credit">Crédito detalhado</button>';
    (actions||hero).appendChild(row);
    row.querySelectorAll('[data-open-secondary162]').forEach(b=>b.onclick=()=>openSecondary162(b.dataset.openSecondary162));
  }

  function openSecondary162(name){
    const panel=name==='environment'?g('carTabEnvironment131'):g('carTabCredit140');
    const btn=document.querySelector('[data-car-tab131="'+name+'"]');
    if(btn)btn.click();
    else if(panel){
      document.querySelectorAll('.car-tab-panel131').forEach(p=>p.classList.remove('active'));
      panel.classList.add('active');
    }
    installSecondaryHead162(panel,name);
    window.scrollTo({top:0,behavior:'smooth'});
  }

  function installSecondaryHead162(panel,name){
    if(!panel||panel.querySelector('.secondary-head162'))return;
    const head=document.createElement('div');head.className='secondary-head162';
    head.innerHTML='<div><strong>'+(name==='environment'?'Ambiental detalhado':'Crédito detalhado')+'</strong><span>Informações avançadas; o resumo principal permanece no Raio X.</span></div><button class="btn ghost">← Voltar ao Raio X</button>';
    head.querySelector('button').onclick=()=>clickTab('xray');
    panel.prepend(head);
  }

  function metricValue162(label){
    const cards=[...document.querySelectorAll('#xrayWorkspace150 .xray-metric150')];
    const c=cards.find(x=>(x.querySelector('span')?.textContent||'').trim()===label);
    if(!c)return 0;
    const t=(c.querySelector('strong')?.textContent||'0').replace(/[^0-9,.-]/g,'').replace(',','.');
    return Number(t)||0;
  }

  function addXRayAttention162(){
    const box=g('xrayWorkspace150');if(!box)return;
    const hero=box.querySelector('.xray-hero150');if(!hero||box.querySelector('.xray-attention162'))return;
    const conflicts=metricValue162('Conflitos c/ projeto');
    const alerts=metricValue162('Alertas MapBiomas');
    const envHits=metricValue162('Ocorrências ambientais');
    const total=conflicts+alerts+envHits;
    const bar=document.createElement('div');
    bar.className='xray-attention162 '+(total?'warning':'ok');
    bar.innerHTML='<div><strong>'+(total?'Há itens que merecem revisão':'Nenhum alerta direto nos dados carregados')+'</strong><span>'+(total?('Conflitos: '+conflicts+' • Alertas: '+alerts+' • Ambiental: '+envHits):'Abra os blocos abaixo somente se precisar conferir detalhes.')+'</span></div><div class="essential-actions162"><button class="btn ghost" data-open-secondary162="environment">Ambiental</button><button class="btn ghost" data-open-secondary162="credit">Crédito</button></div>';
    hero.after(bar);
    bar.querySelectorAll('[data-open-secondary162]').forEach(b=>b.onclick=()=>openSecondary162(b.dataset.openSecondary162));
  }

  function sectionTitle162(panel){
    return (panel.querySelector('.eyebrow')?.textContent||panel.querySelector('h3')?.textContent||'').trim();
  }

  function makeSectionCollapsible162(panel,forceOpen){
    if(!panel||panel.dataset.ui162Section==='1')return;
    const title=panel.querySelector('.panel-title');if(!title)return;
    panel.dataset.ui162Section='1';panel.classList.add('section-card162');
    const b=document.createElement('button');b.className='ui162-section-toggle';b.type='button';
    const collapsed=!forceOpen;
    panel.classList.toggle('ui162-section-collapsed',collapsed);
    b.textContent=collapsed?'Ver detalhes':'Recolher';
    b.onclick=()=>{
      const next=!panel.classList.contains('ui162-section-collapsed');
      panel.classList.toggle('ui162-section-collapsed',next);
      b.textContent=next?'Ver detalhes':'Recolher';
    };
    title.appendChild(b);
  }

  function simplifyXRay162(){
    const box=g('xrayWorkspace150');if(!box)return;
    addSecondaryEntry162();
    addXRayAttention162();
    const panels=[...box.querySelectorAll('.xray-panel150,.sigef-panel150')];
    panels.forEach(p=>{
      const label=sectionTitle162(p).toUpperCase();
      const open=/LIMITAÇÕES/.test(label)&&!/0$/.test((p.querySelector('.status-badge')?.textContent||'').trim());
      makeSectionCollapsible162(p,open);
    });
  }

  function prepare162(){
    document.body.classList.add('vv-ui162');
    renamePrimaryTabs162();
    simplifyDashboard162();
    buildEssential162();
    moveTechnicalSummary162();
    updateEssential162();
    simplifyXRay162();

    ['carStatusBadge','rMunicipality','rGeoArea','rArea','qualityChecks'].forEach(id=>{
      const el=g(id);if(el)new MutationObserver(()=>updateEssential162()).observe(el,{childList:true,subtree:true,characterData:true});
    });
    const xray=g('xrayWorkspace150');
    if(xray)new MutationObserver(()=>simplifyXRay162()).observe(xray,{childList:true,subtree:true});
  }

  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',prepare162);
  else prepare162();
})();