/* ViaVerdeCAR 2.0 — Central automática */
(()=>{
  const g=id=>document.getElementById(id);
  const e=v=>String(v??'').replace(/[&<>"']/g,m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]));
  const arr=v=>Array.isArray(v)?v:[];
  const n=(v,d=2)=>(Number(v)||0).toLocaleString('pt-BR',{minimumFractionDigits:d,maximumFractionDigits:d});
  let timer=null,lastSearch=null,running=false,lastAutomation=null;

  function install(){
    const view=g('view-dashboard');
    if(!view||g('vvCentral200'))return;
    view.insertAdjacentHTML('afterbegin',`
      <section id="vvCentral200" class="vv-central200">
        <div class="vv-central-head200">
          <div>
            <span class="eyebrow">VIA VERDE CAR 2.0 • CENTRAL AUTOMÁTICA</span>
            <h2>Um dado de entrada. O imóvel inteiro em análise.</h2>
            <p>Pesquise por CAR, CPF/CNPJ cadastrado, cliente, imóvel, matrícula ou município. Com um CAR válido, o ViaVerdeCAR executa automaticamente os módulos territorial, ambiental, fundiário e de crédito.</p>
          </div>
          <div class="vv-central-badge200"><b>CAR → KML → RAIO X</b><span>fluxo integrado</span></div>
        </div>
        <div class="vv-command200">
          <div class="vv-command-input200">
            <span>⌕</span>
            <input id="vvSearch200" autocomplete="off" placeholder="CAR, CPF, CNPJ, cliente, imóvel, matrícula ou município" />
          </div>
          <button class="btn primary vv-run200" id="vvSearchBtn200">Pesquisar / analisar</button>
        </div>
        <div class="vv-helper200">
          <span>Ex.: MG-0000000-...</span><span>CPF/CNPJ busca primeiro na base local</span><span>Falha de uma fonte não interrompe as demais</span>
        </div>
        <div id="vvSearchPanel200" class="vv-search-panel200 hidden"></div>
        <div id="vvAutomation200" class="vv-automation200 hidden"></div>
      </section>
    `);

    const oldHero=view.querySelector('.hero-card');
    if(oldHero)oldHero.classList.add('vv-legacy-hero200');

    g('vvSearch200').addEventListener('input',()=>{
      clearTimeout(timer);
      timer=setTimeout(()=>search(false),280);
    });
    g('vvSearch200').addEventListener('keydown',ev=>{
      if(ev.key==='Enter'){ev.preventDefault();search(true)}
    });
    g('vvSearchBtn200').onclick=()=>search(true);

    installCarAction();
  }

  function installCarAction(){
    const bar=document.querySelector('#view-car .car-toolbar');
    if(!bar||g('vvAnalyzeAll200'))return;
    const wrap=bar.querySelector('.toolbar-actions')||bar;
    const b=document.createElement('button');
    b.id='vvAnalyzeAll200';b.className='btn primary';b.textContent='Analisar tudo';
    b.onclick=()=>{
      const car=(g('carInput')?.value||state?.selectedProperty?.car_number||'').trim();
      if(!car){toast('Informe um CAR para executar a análise completa.',true);return}
      run(car,state?.selectedProperty?.id||0,true);
    };
    wrap.prepend(b);
  }

  async function search(runDirect){
    const q=(g('vvSearch200')?.value||'').trim();
    const panel=g('vvSearchPanel200');
    if(!q){
      panel.className='vv-search-panel200 hidden';panel.innerHTML='';return;
    }
    try{
      const r=await api().SearchEverything(q);lastSearch=r;
      if(runDirect&&r.can_analyze_car&&r.valid){
        const local=arr(r.hits).find(x=>x.property_id&&x.car);
        return run(r.normalized,local?.property_id||0,true);
      }
      renderSearch(r);
    }catch(err){toast(String(err),true)}
  }

  function renderSearch(r){
    const panel=g('vvSearchPanel200');
    const hits=arr(r.hits);
    const notice=r.privacy_notice?`<div class="vv-privacy200">🔒 ${e(r.privacy_notice)}</div>`:'';
    panel.className='vv-search-panel200';
    panel.innerHTML=`
      <div class="vv-search-summary200">
        <div><b>${e(r.message||'Resultado da busca')}</b><span>${hits.length} resultado(s) local(is)</span></div>
        ${r.can_analyze_car?`<button class="btn primary" id="vvDirectCAR200">Executar análise completa do CAR</button>`:''}
      </div>
      ${notice}
      <div class="vv-hit-list200">
        ${hits.length?hits.map((h,i)=>hitHTML(h,i)).join(''):'<div class="vv-empty200">Nenhum cadastro local corresponde à busca.</div>'}
      </div>`;
    if(r.can_analyze_car){
      g('vvDirectCAR200').onclick=()=>{
        const local=hits.find(x=>x.property_id&&x.car);
        run(r.normalized,local?.property_id||0,true);
      };
    }
    panel.querySelectorAll('[data-open200]').forEach(b=>b.onclick=()=>openHit(hits[Number(b.dataset.open200)]));
    panel.querySelectorAll('[data-analyze200]').forEach(b=>b.onclick=()=>{
      const h=hits[Number(b.dataset.analyze200)];
      if(h?.car)run(h.car,h.property_id||0,true);
    });
  }

  function hitHTML(h,i){
    const meta=[h.client_name,h.municipality&&h.uf?`${h.municipality} / ${h.uf}`:h.municipality,h.registry?`Mat. ${h.registry}`:'',h.area_ha?`${n(h.area_ha,2)} ha`:''].filter(Boolean).join(' • ');
    return `<article class="vv-hit200">
      <div class="vv-hit-icon200">${h.kind==='client'?'👤':h.kind==='car_public'?'⌖':'▱'}</div>
      <div class="vv-hit-main200"><strong>${e(h.title)}</strong><span>${e(meta||h.subtitle||'')}</span><small>${e(h.car||h.cpf_cnpj||'')}</small></div>
      <div class="vv-hit-actions200">
        ${h.kind!=='car_public'?'<button class="btn ghost" data-open200="'+i+'">Abrir</button>':''}
        ${h.car?'<button class="btn primary" data-analyze200="'+i+'">Analisar tudo</button>':''}
      </div>
    </article>`;
  }

  async function openHit(h){
    if(!h)return;
    if(h.kind==='client'&&h.client_id){
      setView('clients');selectClient(h.client_id);return;
    }
    if(h.property_id){
      setView('car');
      if(g('carPropertySelect'))g('carPropertySelect').value=String(h.property_id);
      await selectCarProperty(h.property_id);
    }
  }

  function loadingHTML(car){
    return `<div class="vv-running200">
      <div class="vv-spinner200"></div>
      <div><span class="eyebrow">ANÁLISE AUTOMÁTICA EM ANDAMENTO</span><h3 id="vvRunningTitle200">Consultando SICAR e geometria…</h3><p>CAR ${e(car)}</p></div>
      <div class="vv-progress-track200"><div id="vvProgressBar200"></div></div>
      <div class="vv-progress-steps200">
        <span class="active">CAR</span><span>KML</span><span>Ambiental</span><span>Fundiário</span><span>Crédito</span><span>Resumo</span>
      </div>
    </div>`;
  }

  async function run(car,propertyID,force){
    if(running)return;
    running=true;
    const box=g('vvAutomation200');
    box.className='vv-automation200';
    box.innerHTML=loadingHTML(car);
    g('vvSearchPanel200')?.classList.add('hidden');
    box.scrollIntoView({behavior:'smooth',block:'center'});

    const phases=[
      ['Consultando SICAR e geometria…',12],['Gerando e vinculando KML…',28],
      ['Cruzando bases ambientais…',48],['Analisando SIGEF / INCRA…',65],
      ['Conferindo crédito rural / SICOR…',82],['Consolidando evidências…',94]
    ];
    let p=0;
    const tick=setInterval(()=>{
      p=Math.min(p+1,phases.length-1);
      const title=g('vvRunningTitle200'),bar=g('vvProgressBar200');
      if(title)title.textContent=phases[p][0];if(bar)bar.style.width=phases[p][1]+'%';
      document.querySelectorAll('.vv-progress-steps200 span').forEach((s,i)=>s.classList.toggle('active',i<=p));
    },4200);

    try{
      const r=await api().RunCARAutomation(car,Number(propertyID)||0,!!force);
      lastAutomation=r;
      clearInterval(tick);
      renderAutomation(r);
      await Promise.all([loadProperties(),loadDashboard()]);
      if(r?.car?.car){
        state.car=r.car;renderCAR(r.car);
        if(r.car.geojson)drawGeoJSON('car',r.car.geojson);
        if(g('carInput'))g('carInput').value=r.car.car;
      }
    }catch(err){
      clearInterval(tick);
      box.innerHTML=`<div class="vv-error200"><strong>A análise automática não foi concluída.</strong><span>${e(String(err))}</span><small>As fontes que não responderam não são tratadas como “zero ocorrência”.</small></div>`;
      toast(String(err),true);
    }finally{running=false}
  }

  function statusLabel(s){
    return {ok:'Sem ocorrência',hit:'Ocorrência/registro',unavailable:'Base indisponível',not_configured:'Não configurado',not_saved:'Não salvo',not_found:'Não localizado'}[s]||s||'Não consultado';
  }

  function renderAutomation(r){
    const box=g('vvAutomation200');
    const src=arr(r.sources),warn=arr(r.warnings);
    const car=r.car||{};
    const overall=r.overall_status==='complete'?'Análise concluída':r.overall_status==='partial'?'Análise parcial':r.overall_status==='not_found'?'CAR não localizado':'Conferir';
    box.innerHTML=`
      <div class="vv-result-head200">
        <div><span class="eyebrow">RESUMO EXECUTIVO AUTOMÁTICO</span><h3>${e(car.property_name||'Imóvel rural')} <span class="vv-pill200 ${e(r.overall_status)}">${e(overall)}</span></h3>
          <p>${e(r.executive_summary||'')}</p></div>
        <div class="vv-result-actions200"><button class="btn ghost" id="vvNewSearch200">Nova busca</button><button class="btn ghost" id="vvDossier200" ${!r.property_id?'disabled':''}>Gerar dossiê</button><button class="btn primary" id="vvOpenMap200">Abrir imóvel e mapa</button></div>
      </div>
      <div class="vv-property-strip200">
        <div><span>CAR</span><strong>${e(car.car||'—')}</strong></div>
        <div><span>Município</span><strong>${e([car.municipality,car.uf].filter(Boolean).join(' / ')||'—')}</strong></div>
        <div><span>Área SICAR</span><strong>${car.area_ha?n(car.area_ha,2)+' ha':'—'}</strong></div>
        <div><span>Situação</span><strong>${e(car.status||'—')}</strong></div>
        <div><span>KML</span><strong>${car.auto_kml_path?'Gerado':'Geometria pronta'}</strong></div>
      </div>
      <div class="vv-source-grid200">${src.map(sourceHTML).join('')}</div>
      ${warn.length?`<details class="vv-warnings200"><summary>Avisos e limitações (${warn.length})</summary>${warn.map(x=>'<div>• '+e(x)+'</div>').join('')}</details>`:''}
      <div class="vv-evidence-note200">“Sem ocorrência” só é exibido quando a respectiva fonte foi efetivamente consultada. Base indisponível, não configurada ou não consultada permanecem identificadas separadamente.</div>
    `;
    g('vvNewSearch200').onclick=()=>{g('vvSearch200').value='';g('vvSearch200').focus();box.classList.add('hidden')};
    g('vvDossier200').onclick=async()=>{
      if(!r.property_id)return;
      try{
        setView('car');
        if(g('carPropertySelect'))g('carPropertySelect').value=String(r.property_id);
        await selectCarProperty(r.property_id);
        if(r.car?.car){state.car=r.car;renderCAR(r.car);if(r.car.geojson)drawGeoJSON('car',r.car.geojson)}
        await exportPackage();
      }catch(err){toast(String(err),true)}
    };
    g('vvOpenMap200').onclick=async()=>{
      setView('car');
      if(r.property_id){
        if(g('carPropertySelect'))g('carPropertySelect').value=String(r.property_id);
        await selectCarProperty(r.property_id);
      }else{
        state.car=car;renderCAR(car);if(car.geojson)drawGeoJSON('car',car.geojson);if(g('carInput'))g('carInput').value=car.car||'';
      }
    };
  }

  function sourceHTML(s){
    const icon={sicar:'⌖',kml:'◇',ibama:'⛔',funai:'◈',icmbio:'▰',mcr:'▤',mapbiomas:'🛰',inpe_fire:'🔥',sigef:'▱',sicor:'R$'}[s.key]||'•';
    return `<article class="vv-source200 status-${e(s.status)}">
      <div class="vv-source-top200"><span class="vv-source-icon200">${icon}</span><span class="vv-source-status200">${e(statusLabel(s.status))}</span></div>
      <strong>${e(s.label)}</strong><p>${e(s.detail||'')}</p>${Number(s.count)>0?'<b>'+Number(s.count)+' registro(s)</b>':''}
    </article>`;
  }

  function watch(){
    install();
    new MutationObserver(()=>{install();installCarAction()}).observe(document.body,{childList:true,subtree:true});
  }
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',watch);else watch();
})();
