/* ViaVerdeCAR 2.1.1 - ficha completa de operacao de credito rural */
(function(){
  'use strict';

  var S={car:'',data:null,loading:false,timer:null,poll:null,started:0,map:null,target:''};
  var steps=[
    ['operations','Operações SICOR'],['domains','Tabelas e domínios'],['details','Detalhes financeiros'],
    ['releases','Liberações'],['disbursements','Desembolsos'],['declassification','Desclassificações'],
    ['renegotiation','Renegociações'],['balances','Saldos e situação'],['proagro','Proagro'],
    ['zarc','ZARC'],['market','Contexto de mercado']
  ];

  function E(id){return document.getElementById(id)}
  function A(v){return Array.isArray(v)?v:[]}
  function H(v){return String(v==null?'':v).replace(/[&<>"']/g,function(m){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]})}
  function MONEY(v){return (Number(v)||0).toLocaleString('pt-BR',{style:'currency',currency:'BRL'})}
  function N(v,d){return (Number(v)||0).toLocaleString('pt-BR',{minimumFractionDigits:d==null?2:d,maximumFractionDigits:d==null?2:d})}
  function DT(v){if(!v)return '—';var d=new Date(v);return isNaN(d)?String(v):d.toLocaleDateString('pt-BR')}
  function api211(){return typeof api==='function'?api():window.go&&window.go.main&&window.go.main.App}
  function result211(){return window.__vv204LastResult||null}
  function key211(ref,order){return String(ref||'').trim()+'|'+String(order||'').trim()}
  function statusTone211(v){var s=String(v||'').toUpperCase();if(/INADIM|ATRAS|PREJU|DESCLASS/.test(s))return'danger';if(/LIQUID/.test(s))return'ok';if(/RENEG|PRORROG/.test(s))return'warn';return'neutral'}
  function icon211(name){
    var d={database:'M4 6c0-2 3.6-3 8-3s8 1 8 3-3.6 3-8 3-8-1-8-3Z M4 6v6c0 2 3.6 3 8 3s8-1 8-3V6 M4 12v6c0 2 3.6 3 8 3s8-1 8-3v-6',
      file:'M6 3h8l4 4v14H6V3Z M14 3v5h5 M9 12h6 M9 16h6',
      close:'M6 6l12 12 M18 6 6 18',
      refresh:'M20 7v5h-5 M4 17v-5h5 M6.1 8.2A7 7 0 0 1 18.6 9 M5.4 15a7 7 0 0 0 12.5 1.8',
      alert:'M12 3 22 20H2L12 3Z M12 9v4 M12 17h.01'}[name]||'';
    return '<svg class="vv211-icon" viewBox="0 0 24 24" aria-hidden="true"><path d="'+d+'"/></svg>';
  }

  function currentCreditTab(){
    var pane=E('vv204Pane');if(!pane)return null;
    var title=pane.querySelector('.vv204-tab-title span');
    return title&&String(title.textContent||'').trim().toUpperCase()==='CRÉDITO RURAL'?pane:null;
  }

  function summaryHTML(r){
    var t=r&&r.totals||{};
    return '<div class="vv211-summary">'+
      '<div><span>Contratado</span><strong>'+MONEY(t.contracted)+'</strong></div>'+
      '<div><span>Liberado</span><strong>'+MONEY(t.released)+'</strong></div>'+
      '<div><span>Últimos saldos</span><strong>'+MONEY(t.latest_balances)+'</strong></div>'+
      '<div><span>Recursos próprios</span><strong>'+MONEY(t.own_resources)+'</strong></div>'+
      '<div><span>Reneg./prorrog.</span><strong>'+Number(t.with_renegotiation||0)+'</strong></div>'+
      '<div><span>Proagro</span><strong>'+Number(t.with_proagro||0)+'</strong></div>'+
      '<small>'+(r&&r.used_cache?'Cache local recente • ':'')+'Atualizado '+H(DT(r&&r.generated_at))+'</small>'+
    '</div>';
  }

  function opRow211(o,i,intel){
    var detailed=A(intel&&intel.operations).find(function(x){return key211(x.ref_bacen,x.order)===key211(o.ref_bacen,o.order)});
    var b=detailed&&detailed.balance||{},status=b.found?(b.status||('Código '+(b.status_code||''))):'';
    var sub=[o.institution_name,o.program_name,o.subprogram_name,o.year].filter(Boolean).join(' • ');
    var extra=detailed?(detailed.total_released?('Liberado '+MONEY(detailed.total_released)):(b.found?'Saldo '+MONEY(b.last_day):'Detalhes carregados')):'';
    return '<article class="vv211-op-row">'+
      '<div class="vv211-op-index">'+String(i+1).padStart(2,'0')+'</div>'+
      '<div class="vv211-op-main"><span>'+H(o.purpose||o.activity||o.modality||'Operação SICOR')+'</span><strong>'+H(o.product||o.variety||'Crédito rural')+'</strong><small>'+H(sub)+'</small>'+
      '<div class="vv211-op-ref">REF '+H(o.ref_bacen||'—')+' • ordem '+H(o.order||'—')+(extra?' • '+H(extra):'')+'</div></div>'+
      '<div class="vv211-op-value"><strong>'+MONEY(o.credit_value)+'</strong>'+(status?'<span class="'+statusTone211(status)+'">'+H(status)+'</span>':'<small>valor contratado</small>')+'</div>'+
      '<button data-vv211-op="'+H(key211(o.ref_bacen,o.order))+'">'+icon211('file')+'Ver operação completa</button>'+
    '</article>';
  }

  function install211(){
    var pane=currentCreditTab(),r=result211();if(!pane||!r)return;
    var car=String(r.car&&r.car.car||'');
    if(S.car&&S.car!==car){stopProgress211();S.data=null;S.car=car}
    if(!S.car)S.car=car;
    if(E('vv211CreditPanel'))return;

    var grid=pane.querySelector('.vv204-credit-grid');
    if(!grid)return;
    var sic=r.xray&&r.xray.sicor||{},ops=A(sic.operations);

    var panel=document.createElement('article');
    panel.id='vv211CreditPanel';panel.className='vv204-panel vv211-panel';
    panel.innerHTML='<div class="vv211-panel-head"><div><span>FICHA COMPLETA • SICOR/BCB</span><h3>Detalhamento das operações vinculadas ao CAR</h3><p>Taxas, liberações, saldos, situação, produção, renegociação, Proagro, ZARC e glebas quando publicados pelo Banco Central.</p></div>'+
      '<button id="vv211Load" '+(ops.length?'':'disabled')+'>'+icon211('database')+'<span>'+(S.data?'Atualizar inteligência':'Carregar detalhes')+'</span></button></div>'+
      '<div class="vv211-panel-note"><strong>Leitura correta:</strong> são registros públicos identificados no SICOR/BCB. Ausência de registro não prova ausência de financiamento, dívida, seguro ou renegociação.</div>'+
      (S.data?summaryHTML(S.data):'<div class="vv211-panel-state">A consulta detalhada é executada somente quando solicitada para preservar a velocidade do Raio X inicial.</div>');
    pane.insertBefore(panel,grid);

    var firstList=pane.querySelector('.vv204-operation-list');
    if(firstList){
      var parent=firstList.closest('.vv204-panel');
      if(parent)parent.classList.add('vv211-ops-panel','vv204-credit-wide');
      firstList.className='vv211-op-list';
      firstList.innerHTML=ops.length?ops.map(function(o,i){return opRow211(o,i,S.data)}).join(''):'<div class="vv204-empty">Nenhuma operação pública vinculada foi localizada.</div>';
    }

    if(E('vv211Load'))E('vv211Load').onclick=function(){load211(r,true,'')};
    pane.querySelectorAll('[data-vv211-op]').forEach(function(b){
      b.onclick=function(){
        var k=b.getAttribute('data-vv211-op')||'';
        if(S.data&&S.car===car)openOperation211(k);
        else load211(r,false,k);
      };
    });
  }

  function stopProgress211(){
    if(S.timer){clearInterval(S.timer);S.timer=null}
    if(S.poll){clearInterval(S.poll);S.poll=null}
  }
  function closeModal211(){
    stopProgress211();
    if(S.map){try{S.map.remove()}catch(_e){}S.map=null}
    var m=E('vv211Modal');if(m)m.remove();
  }
  function progressHTML211(){
    return '<div class="vv211-progress"><div class="vv211-progress-head"><span>'+icon211('database')+'</span><div><small>INTELIGÊNCIA FINANCEIRA</small><h3 id="vv211PTitle">Preparando consulta detalhada</h3><p id="vv211PDetail">Localizando as operações públicas vinculadas ao CAR.</p></div><b id="vv211PTime">00:00</b></div>'+
      '<div class="vv211-progress-bar"><i id="vv211PBar"></i></div><div class="vv211-progress-steps">'+
      steps.map(function(x,i){return '<div data-vv211-step="'+x[0]+'" class="'+(i===0?'current':'waiting')+'"><i></i><span><b>'+H(x[1])+'</b><small>'+(i===0?'Em andamento':'Aguardando')+'</small></span></div>'}).join('')+
      '</div><div class="vv211-progress-note">Na primeira consulta, alguns arquivos nacionais do SICOR podem precisar ser baixados e processados. As próximas consultas usam cache local recente.</div></div>';
  }
  function showProgress211(){
    closeModal211();
    var m=document.createElement('div');m.id='vv211Modal';m.className='vv211-modal';
    m.innerHTML='<div class="vv211-modal-card progress"><div class="vv211-modal-top"><span></span><button id="vv211Close">'+icon211('close')+'</button></div>'+progressHTML211()+'</div>';
    document.body.appendChild(m);E('vv211Close').onclick=closeModal211;
  }
  function updateProgress211(p){
    var step=Math.min(steps.length,Math.max(1,Number(p&&p.step)||1));
    if(E('vv211PTitle'))E('vv211PTitle').textContent=p&&p.label||steps[step-1][1];
    if(E('vv211PDetail'))E('vv211PDetail').textContent=p&&p.detail||'Consultando dados oficiais.';
    if(E('vv211PBar'))E('vv211PBar').style.width=((p&&p.done?steps.length:Math.max(0,step-1))/steps.length*100)+'%';
    steps.forEach(function(x,i){
      var el=document.querySelector('[data-vv211-step="'+x[0]+'"]');if(!el)return;
      var done=!!(p&&p.done)||i<step-1,current=!(p&&p.done)&&i===step-1;
      el.className=done?'done':current?'current':'waiting';
      var sm=el.querySelector('small');if(sm)sm.textContent=done?'Concluído':current?'Em andamento':'Aguardando';
    });
  }
  function startProgress211(){
    stopProgress211();S.started=Date.now();
    S.timer=setInterval(function(){
      var sec=Math.floor((Date.now()-S.started)/1000),m=Math.floor(sec/60),s=sec%60;
      if(E('vv211PTime'))E('vv211PTime').textContent=String(m).padStart(2,'0')+':'+String(s).padStart(2,'0');
    },1000);
    var poll=async function(){if(!S.loading)return;try{updateProgress211(await api211().GetCreditIntelligenceProgress())}catch(_e){}};
    poll();S.poll=setInterval(poll,850);
  }
  async function load211(r,force,target){
    if(S.loading)return;
    S.loading=true;S.car=String(r.car&&r.car.car||'');S.target=target||'';
    showProgress211();startProgress211();
    try{
      var data=await api211().GetCreditIntelligence(Number(r.property_id)||0,!!force);
      S.data=data;stopProgress211();
      if(target)openOperation211(target);
      else{closeModal211();var p=currentCreditTab();if(p){var old=E('vv211CreditPanel');if(old)old.remove();install211()}try{toast((data.used_cache?'Cache financeiro carregado. ':'Inteligência financeira atualizada. ')+Number(A(data.operations).length)+' operação(ões).')}catch(_e){}}
    }catch(e){
      stopProgress211();
      var card=E('vv211Modal')&&E('vv211Modal').querySelector('.vv211-modal-card');
      if(card)card.innerHTML='<div class="vv211-error">'+icon211('alert')+'<h3>Consulta detalhada não concluída</h3><p>'+H(String(e))+'</p><div><button id="vv211Retry">Tentar novamente</button><button id="vv211Cancel">Fechar</button></div></div>';
      if(E('vv211Retry'))E('vv211Retry').onclick=function(){closeModal211();load211(r,true,target)};
      if(E('vv211Cancel'))E('vv211Cancel').onclick=closeModal211;
    }finally{S.loading=false}
  }

  function field211(label,value){if(value===undefined||value===null||value===''||value==='—')return'';return '<div><span>'+H(label)+'</span><strong>'+H(value)+'</strong></div>'}
  function num211(label,value,suffix){var n=Number(value)||0;return n?field211(label,N(n,2)+(suffix||'')):''}
  function money211(label,value){var n=Number(value)||0;return n?field211(label,MONEY(n)):''}
  function fields211(arr){return '<div class="vv211-fields">'+arr.join('')+'</div>'}
  function section211(title,html){return String(html||'').trim()?'<section class="vv211-section"><h4>'+H(title)+'</h4>'+html+'</section>':''}
  function timeline211(title,rows){rows=A(rows);return rows.length?'<div class="vv211-timeline"><strong>'+H(title)+'</strong>'+rows.map(function(x){return '<div><span>'+H(DT(x.date))+'</span><b>'+MONEY(x.value)+'</b></div>'}).join('')+'</div>':''}
  function reneg211(rows){rows=A(rows);return rows.length?'<div class="vv211-reneg"><strong>Renegociações / prorrogações localizadas</strong>'+rows.map(function(x){return '<div><span>'+H(x.related_ref?('REF relacionada '+x.related_ref+(x.related_order?' / '+x.related_order:'')):'Vínculo SICOR')+'</span><b>'+H(x.legal_basis||x.legal_basis_code||(x.value?MONEY(x.value):'registro localizado'))+'</b></div>'}).join('')+'</div>':''}

  function detailHTML211(op,r){
    var b=op.balance||{},d=op.declassification||{},p=op.proagro||{},z=op.zarc||{},ren=A(op.renegotiations),glebas=A(op.glebas);
    var status=b.found?(b.status||('Código '+(b.status_code||'—'))):'Saldo/situação não localizado';
    var releasedPct=Number(op.credit_value)>0?Number(op.total_released||0)/Number(op.credit_value)*100:0;
    var identification=fields211([
      field211('REF BACEN',op.ref_bacen),field211('REF BACEN efetivo',op.effective_ref_bacen),field211('Ordem / destinação',op.order),field211('Ano',op.year),
      field211('Instituição',op.institution),field211('Agência IF',op.agency_code),field211('Município IBGE da operação',op.municipality_code),
      field211('Programa',op.program),field211('Subprograma',op.subprogram),field211('Fonte de recursos',op.resource),
      field211('Instrumento',op.instrument||op.instrument_code),field211('Categoria emitente',op.issuer_category||op.issuer_category_code)
    ]);
    var financial=fields211([
      money211('Valor contratado',op.credit_value),money211('Recursos próprios',op.own_resources),money211('Parcela de crédito',op.credit_installment),
      money211('Prestação de investimento',op.investment_installment),num211('Taxa de juros',op.interest_rate_pct,'%'),num211('Juros pós-fixados',op.post_fixed_interest_pct,'%'),
      num211('Custo efetivo total',op.effective_cost_pct,'%'),num211('Risco STN',op.stn_risk_pct,'%'),num211('Risco fundo constitucional',op.constitutional_fund_risk_pct,'%'),
      money211('Garantia de renda mínima',op.minimum_income_guarantee),field211('Contrato STN',op.stn_contract_code),field211('IF cadastrante',op.registering_institution),
      field211('Emissão',DT(op.issue_date)),field211('Vencimento',DT(op.due_date))
    ]);
    var situation=fields211([
      field211('Situação SICOR',status),field211('Data-base do saldo',b.found?String(b.month).padStart(2,'0')+'/'+b.year:''),
      money211('Saldo no último dia',b.last_day),money211('Saldo médio diário',b.daily_average),money211('Saldo médio vincendo',b.due_daily_average),
      d.found?field211('Desclassificação',DT(d.date)+' • '+(d.reason||d.reason_code||'')+' • '+MONEY(d.value)):''
    ])+reneg211(ren);
    var production=fields211([
      field211('Finalidade',op.purpose),field211('Atividade',op.activity),field211('Modalidade',op.modality),field211('Produto',op.product),field211('Variedade',op.variety),
      num211('Área financiada',op.financed_area_ha,' ha'),num211('Área informada',op.informed_area_ha,' ha'),num211('Previsão de produção',op.forecast_production),
      num211('Quantidade',op.quantity),money211('Receita bruta esperada',op.expected_revenue),num211('Produtividade obtida',op.productivity_obtained),
      field211('Irrigação',op.irrigation||op.irrigation_code),field211('Agricultura',op.agriculture||op.agriculture_code),field211('Cultivo',op.crop_type||op.crop_type_code),
      field211('Integração / consórcio',op.integration||op.integration_code),field211('Grão / semente',op.seed_type||op.seed_type_code),field211('Fase produtiva',op.production_phase||op.production_phase_code),
      field211('Ciclo cultivar',op.cultivar_cycle||op.cultivar_cycle_code),field211('Solo',op.soil||op.soil_code),
      field211('Início plantio',DT(op.planting_start)),field211('Fim plantio',DT(op.planting_end)),field211('Início colheita',DT(op.harvest_start)),field211('Fim colheita',DT(op.harvest_end))
    ]);
    var proagro=(p.has_cop||p.has_rcp||p.has_judgment||Number(p.paid_value)||Number(op.proagro_rate_pct))?fields211([
      num211('Alíquota Proagro',op.proagro_rate_pct,'%'),field211('COP',p.has_cop?(DT(p.cop_date)+' • '+(p.cop_status||p.cop_status_code||'')):''),
      field211('Evento',p.event||p.event_code),field211('Ciclo COP',p.cycle||p.cycle_code),field211('Solo COP',p.soil||p.soil_code),
      field211('RCP',p.has_rcp?DT(p.rcp_date):''),num211('Área RCP',p.rcp_area_ha,' ha'),num211('Produção prevista RCP',p.rcp_forecast_prod),
      money211('Receita prevista RCP',p.rcp_forecast_revenue),field211('Julgamento',p.has_judgment?DT(p.judgment_date):''),
      money211('Cobertura crédito',p.coverage_credit),money211('Cobertura recursos próprios',p.coverage_own),money211('Perdas não amparadas',p.losses_uncovered),money211('Valor pago',p.paid_value)
    ]):'';
    var zp=A(z.periods).map(function(x){return '<span class="'+(x.indicated?'ok':'warn')+'"><b>D'+H(x.decendio)+'</b>'+(x.indicated?(x.risk_pct?H(x.risk_pct)+'%':H(x.raw||'indicado')):'sem indicação')+'</span>'}).join('');
    var zarc=fields211([field211('Resultado',z.status),field211('Safra',z.safra),field211('Cultura',z.culture||op.product),field211('Ciclo',z.cycle||op.cultivar_cycle),field211('Solo',z.soil||op.soil),field211('Manejo',z.management),field211('Portaria',z.portaria)])+
      (zp?'<div class="vv211-zarc">'+zp+'</div>':'')+(z.message?'<p class="vv211-note">'+H(z.message)+'</p>':'');
    var glebaRows=glebas.length?'<div class="vv211-glebas">'+glebas.map(function(g,i){return '<div><span><b>Gleba '+H(g.index||i+1)+'</b><small>'+N(g.area_ha,2)+' ha</small></span><em>'+N(g.inside_car_pct,1)+'% no CAR'+(g.project_overlap_name?' • '+H(g.project_overlap_name)+' '+N(g.project_overlap_pct,1)+'%':'')+'</em></div>'}).join('')+'</div>':'<p class="vv211-note">Nenhuma geometria de gleba foi publicada/localizada para esta destinação.</p>';
    var warnings=A(r.warnings).length?'<details class="vv211-warnings"><summary>Limitações e avisos desta consulta ('+A(r.warnings).length+')</summary>'+A(r.warnings).map(function(x){return '<p>'+H(x)+'</p>'}).join('')+'</details>':'';
    return '<div class="vv211-hero"><div><span>REF '+H(op.ref_bacen||'—')+' • ORDEM '+H(op.order||'—')+'</span><h2>'+H(op.product||op.purpose||'Operação de crédito rural')+'</h2><p>'+H([op.institution,op.program,op.subprogram].filter(Boolean).join(' • '))+'</p></div><div><small>Contratado</small><strong>'+MONEY(op.credit_value)+'</strong><em class="'+statusTone211(status)+'">'+H(status)+'</em></div></div>'+
      '<div class="vv211-kpis"><div><span>Liberado</span><strong>'+MONEY(op.total_released)+'</strong><small>'+N(releasedPct,1)+'% do contratado</small></div><div><span>Último saldo</span><strong>'+(b.found?MONEY(b.last_day):'—')+'</strong><small>'+(b.found?String(b.month).padStart(2,'0')+'/'+b.year:'não localizado')+'</small></div><div><span>Vencimento</span><strong>'+H(DT(op.due_date))+'</strong><small>'+H(op.modality||op.activity||'SICOR')+'</small></div><div><span>Glebas</span><strong>'+glebas.length+'</strong><small>'+N(glebas.reduce(function(s,g){return s+(Number(g.area_ha)||0)},0),2)+' ha publicados</small></div></div>'+
      section211('Identificação e enquadramento',identification)+section211('Financeiro',financial)+
      section211('Liberações e desembolso',timeline211('Liberações efetivas',op.releases)+timeline211('Cronograma previsto',op.disbursements))+
      section211('Situação e histórico',situation)+section211('Produção e sistema produtivo',production)+section211('Proagro',proagro)+section211('ZARC / plantio',zarc)+
      section211('Área financiada / glebas',glebaRows+(glebas.length?'<div id="vv211CreditMap" class="vv211-map"></div>':''))+
      '<div class="vv211-sources"><strong>Fontes oficiais e rastreabilidade</strong><span>'+H(r.scope||'')+'</span><div><button data-vv211-source="'+H(r.bcb_source_url||'')+'">Microdados BCB</button><button data-vv211-source="'+H(r.mcr_source_url||'')+'">MCR</button><button data-vv211-source="'+H(r.zarc_source_url||'')+'">ZARC</button></div></div>'+warnings;
  }

  function openOperation211(k){
    var r=S.data;if(!r)return;
    var op=A(r.operations).find(function(x){return key211(x.ref_bacen,x.order)===k});
    if(!op){try{toast('A operação detalhada não foi localizada na resposta do SICOR.',true)}catch(_e){}return}
    closeModal211();
    var m=document.createElement('div');m.id='vv211Modal';m.className='vv211-modal';
    m.innerHTML='<div class="vv211-modal-card"><div class="vv211-modal-top"><div><span>FICHA COMPLETA DA OPERAÇÃO</span><small>'+(r.used_cache?'cache local • ':'')+H(DT(r.generated_at))+'</small></div><div><button id="vv211Refresh">'+icon211('refresh')+'Atualizar</button><button id="vv211Close">'+icon211('close')+'</button></div></div><div class="vv211-modal-body">'+detailHTML211(op,r)+'</div></div>';
    document.body.appendChild(m);
    E('vv211Close').onclick=closeModal211;
    E('vv211Refresh').onclick=function(){var rr=result211();closeModal211();if(rr)load211(rr,true,k)};
    m.querySelectorAll('[data-vv211-source]').forEach(function(b){b.onclick=function(){var u=b.getAttribute('data-vv211-source');if(u)openExternal(u)}});
    m.onclick=function(e){if(e.target===m)closeModal211()};
    setTimeout(function(){drawMap211(op)},80);
  }

  function drawMap211(op){
    var node=E('vv211CreditMap'),glebas=A(op.glebas);if(!node||!window.L||!glebas.length)return;
    if(S.map){try{S.map.remove()}catch(_e){}S.map=null}
    var m=L.map(node,{zoomControl:true,attributionControl:true}).setView([-18.5,-44],5);S.map=m;
    L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',{maxZoom:19,attribution:'Esri'}).addTo(m);
    var fit=[],r=result211();
    if(r&&r.car&&r.car.geojson){try{var car=L.geoJSON(JSON.parse(r.car.geojson),{style:{color:'#dc2626',weight:3,fillOpacity:.02}}).addTo(m);fit.push(car)}catch(_e){}}
    glebas.forEach(function(g,i){if(!g.geojson)return;try{var x=L.geoJSON(JSON.parse(g.geojson),{style:{color:'#7c3aed',weight:3,fillColor:'#8b5cf6',fillOpacity:.18}}).addTo(m);x.bindPopup('Gleba '+H(g.index||i+1)+' • '+N(g.area_ha,2)+' ha');fit.push(x)}catch(_e){}});
    if(fit.length){var b=L.featureGroup(fit).getBounds();if(b.isValid())m.fitBounds(b.pad(.08),{maxZoom:17})}
  }

  function watch211(){
    var run=function(){try{install211()}catch(e){try{console.error('ViaVerdeCAR 2.1.1 crédito:',e)}catch(_e){}}};
    run();
    new MutationObserver(run).observe(document.body,{childList:true,subtree:true});
  }
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',watch211);else watch211();
})();