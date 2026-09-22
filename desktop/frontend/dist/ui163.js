/* ViaVerdeCAR 1.6.3 — inteligência financeira de crédito rural sob demanda */
(function(){
  const g=id=>document.getElementById(id);
  const fin={car:'',result:null,loading:false};

  function install163(){
    const box=g('xrayWorkspace150'); if(!box)return;
    const hero=box.querySelector('.xray-hero150'); if(!hero)return;
    const actions=hero.querySelector('.xray-actions150'); if(!actions)return;
    if(!g('creditFull163')){
      const b=document.createElement('button');
      b.id='creditFull163'; b.className='btn primary'; b.textContent='Financeiro completo';
      actions.insertBefore(b,actions.firstChild);
      b.onclick=()=>load163(false);
    }
    if(fin.result&&fin.car===state?.car?.car&&!g('creditIntelligence163')) render163(fin.result);
  }

  async function load163(force){
    if(fin.loading||!state?.car?.car)return;
    fin.loading=true;
    const b=g('creditFull163');
    if(b){b.disabled=true;b.textContent='Carregando financeiro…'}
    ensurePanel163('<div class="credit163-loading"><strong>Montando histórico financeiro completo…</strong><span>Saldo, liberações, cronograma, renegociação, desclassificação, Proagro e conferência ZARC usam bases oficiais. A primeira sincronização pode demorar.</span></div>');
    try{
      const r=await api().BuildSICORFinancialIntelligence(state.selectedProperty?.id||0,!!force);
      fin.result=r; fin.car=r.car||state.car.car; render163(r);
    }catch(e){
      ensurePanel163('<div class="xray-warning150"><strong>Financeiro completo não concluído.</strong><br>'+esc(String(e))+'</div><div class="xray-actions150"><button class="btn primary" id="retryCredit163">Tentar novamente</button></div>');
      g('retryCredit163')?.addEventListener('click',()=>load163(true));
      toast(String(e),true);
    }finally{
      fin.loading=false;
      if(b){b.disabled=false;b.textContent='Financeiro completo'}
    }
  }

  function ensurePanel163(html){
    const hero=document.querySelector('#xrayWorkspace150 .xray-hero150'); if(!hero)return;
    let p=g('creditIntelligence163');
    if(!p){p=document.createElement('section');p.id='creditIntelligence163';p.className='panel credit163';hero.after(p)}
    p.innerHTML=html;
  }

  function render163(r){
    const ops=r.operations||[];
    const warnings=r.warnings||[];
    const metrics='<div class="credit163-metrics">'+
      metric163('Saldo último dia','R$ '+money163(r.latest_balance_total||0),'soma dos últimos saldos públicos localizados')+
      metric163('Liberado identificado','R$ '+money163(r.released_total||0),'liberações públicas vinculadas às destinações')+
      metric163('Renegociações',r.renegotiated_operations||0,'operações com registro público relacionado')+
      metric163('Proagro pago','R$ '+money163(r.proagro_paid_total||0),'parcelas públicas com valor pago')+
      '</div>';
    const body=ops.length?ops.map(operation163).join(''):'<div class="xray-empty150">Nenhuma operação SICOR pública disponível para detalhamento financeiro.</div>';
    const warn=warnings.length?'<details class="credit163-warnings"><summary>Limitações e fontes indisponíveis ('+warnings.length+')</summary>'+warnings.map(x=>'<div class="xray-warning150">'+esc(x)+'</div>').join('')+'</details>':'';
    ensurePanel163(
      '<div class="panel-title"><div><span class="eyebrow">CRÉDITO RURAL • FINANCEIRO COMPLETO</span><h3>Saldo, liberações, renegociação e Proagro</h3><p>Os valores abaixo são somente os registros públicos encontrados pelo ViaVerdeCAR. Não representam, por si, a dívida total do produtor.</p></div><span class="status-badge '+(r.used_cache?'info':'ok')+'">'+(r.used_cache?'Cache 7 dias':'Atualizado')+'</span></div>'+
      metrics+
      '<div class="credit163-actions"><button class="btn ghost" id="refreshCredit163">Atualizar financeiro</button><button class="btn ghost" id="openMCR163">Abrir MCR oficial</button><button class="btn ghost" id="openZARC163">Abrir ZARC oficial</button><button class="btn ghost" id="openZARCDataset163">Base ZARC</button></div>'+
      '<div class="credit163-list">'+body+'</div>'+warn+
      '<div class="xray-source150">Crédito contratado no SICOR pode incluir operação ainda sem liberação de recursos. Saldo, situação, liberações e Proagro são exibidos conforme o último registro público localizado; ausência de registro não equivale a inexistência.</div>'
    );
    g('refreshCredit163')?.addEventListener('click',()=>load163(true));
    g('openMCR163')?.addEventListener('click',()=>openExternal(r.mcr_source_url||'https://www3.bcb.gov.br/mcr/completo'));
    g('openZARC163')?.addEventListener('click',()=>openExternal(r.zarc_source_url||'https://www.gov.br/agricultura/pt-br/assuntos/riscos-seguro/programa-nacional-de-zoneamento-agricola-de-risco-climatico'));
    g('openZARCDataset163')?.addEventListener('click',()=>openExternal(r.zarc_dataset_url||'https://dados.agricultura.gov.br/dataset/tabua-de-risco-zoneamento-agricola-de-risco-climatico'));
    document.querySelectorAll('[data-mcr163]').forEach(b=>b.onclick=()=>openExternal(b.dataset.mcr163));
    document.querySelectorAll('[data-zarc163]').forEach(b=>b.onclick=()=>openExternal(b.dataset.zarc163));
  }

  function operation163(op){
    const x=op.intelligence||{},bal=x.latest_balance||{},zc=x.zarc_check||{};
    const base=(bal.base_year&&bal.base_month)?String(bal.base_month).padStart(2,'0')+'/'+bal.base_year:'—';
    const status=bal.situation_name||bal.situation_code||'Sem saldo público localizado';
    const financial=[
      field163('Taxa registrada',op.interest_rate_pct?fmt(op.interest_rate_pct,4)+'% a.a.':'—'),
      field163('Encargo pós-fixado',op.post_fixed_interest_pct?fmt(op.post_fixed_interest_pct,4)+'%':'—'),
      field163('CET registrado',op.effective_cost_pct?fmt(op.effective_cost_pct,4)+'%':'—'),
      field163('Recursos próprios',op.own_resources?'R$ '+money163(op.own_resources):'—'),
      field163('Prestação investimento',op.investment_installment?'R$ '+money163(op.investment_installment):'—'),
      field163('Seguro / garantia',op.insurance_name||op.insurance_code||'—'),
      field163('Instrumento',op.instrument_name||op.instrument_code||'—'),
      field163('Área informada',op.informed_area_ha?fmt(op.informed_area_ha,2)+' ha':'—'),
      field163('Produção prevista',num163(op.expected_production)),
      field163('Produtividade obtida',num163(op.obtained_productivity)),
      field163('Receita bruta esperada',op.expected_gross_revenue?'R$ '+money163(op.expected_gross_revenue):'—'),
      field163('Alíquota Proagro',op.proagro_rate_pct?fmt(op.proagro_rate_pct,4)+'%':'—'),
      field163('Risco STN',op.stn_risk_pct?fmt(op.stn_risk_pct,4)+'%':'—'),
      field163('Risco fundo constitucional',op.fund_risk_pct?fmt(op.fund_risk_pct,4)+'%':'—'),
      field163('Recursos próprios serviço',op.service_own_resources?'R$ '+money163(op.service_own_resources):'—'),
      field163('Bônus CAR',op.bonus_car_pct?fmt(op.bonus_car_pct,4)+'%':'—'),
      field163('Contrato STN',op.contract_stn||'—'),
      field163('CNPJ cadastrante',op.registrant_cnpj||'—')
    ].join('');

    const production=[
      field163('Irrigação',op.irrigation_name||op.irrigation_code||'—'),
      field163('Agricultura',op.agriculture_name||op.agriculture_code||'—'),
      field163('Cultivo',op.cultivation_name||op.cultivation_code||'—'),
      field163('Integração/consórcio',op.integration_name||op.integration_code||'—'),
      field163('Grão/semente',op.grain_seed_name||op.grain_seed_code||'—'),
      field163('Fase produtiva',op.production_phase_name||op.production_phase_code||'—'),
      field163('Solo da operação',op.soil_name||op.soil_code||'—'),
      field163('Ciclo/cultivar',op.cycle_name||op.cycle_code||'—'),
      field163('Quantidade',num163(op.quantity)),
      field163('Plantio',dateRange163(op.planting_start,op.planting_end)),
      field163('Colheita',dateRange163(op.harvest_start,op.harvest_end))
    ].join('');

    const releases=(x.releases||[]);
    const releaseHtml=releases.length?'<div class="credit163-table">'+releases.map(v=>row163(v.date,'R$ '+money163(v.value))).join('')+'</div>':'<div class="credit163-empty">Nenhuma liberação pública localizada.</div>';
    const schedule=(x.disbursements||[]);
    const scheduleHtml=schedule.length?'<div class="credit163-table">'+schedule.map(v=>row163(v.expected_date,'R$ '+money163(v.value))).join('')+'</div>':'<div class="credit163-empty">Nenhum cronograma público localizado.</div>';

    const disq=(x.disqualifications||[]);
    const disqHtml=disq.length?disq.map(v=>'<div class="credit163-event warning"><strong>'+esc(v.date||'Desclassificação')+'</strong><span>'+esc(v.reason_name||v.reason_code||'Motivo não descrito')+(v.value?' • R$ '+money163(v.value):'')+(v.type?' • '+esc(v.type):'')+'</span></div>').join(''):'<div class="credit163-empty">Nenhuma desclassificação pública localizada.</div>';

    const reneg=(x.renegotiations||[]);
    const renegHtml=reneg.length?reneg.map(generic163).join(''):'<div class="credit163-empty">Nenhum registro público de renegociação vinculado a esta operação.</div>';
    const src=(x.source_changes||[]);
    const srcHtml=src.length?src.map(generic163).join(''):'<div class="credit163-empty">Nenhuma alteração pública de fonte localizada.</div>';

    const cop=x.proagro_cop||[],rcp=x.proagro_rcp||[],jud=x.proagro_judgments||[],pay=x.proagro_payments||[];
    const proagro=(cop.length||rcp.length||jud.length||pay.length)
      ?'<div class="credit163-proagro">'+
        cop.map(v=>'<div class="credit163-event"><strong>COP '+esc(v.status_name||v.status_code||'')+'</strong><span>'+(v.event_name||v.event_code?esc(v.event_name||v.event_code):'')+(v.communication_date?' • comunicação '+esc(v.communication_date):'')+'</span></div>').join('')+
        rcp.map(v=>'<div class="credit163-event"><strong>RCP '+esc(v.status_code||'')+'</strong><span>'+(v.delivery_date?'entrega '+esc(v.delivery_date):'')+(v.area_ha?' • '+fmt(v.area_ha,2)+' ha':'')+'</span></div>').join('')+
        jud.map(v=>'<div class="credit163-event"><strong>Julgamento '+esc(v.decision_code||'')+'</strong><span>'+(v.decision_date?esc(v.decision_date):'')+(v.coverage_credit?' • cobertura crédito R$ '+money163(v.coverage_credit):'')+(v.uncovered_losses?' • perdas não amparadas R$ '+money163(v.uncovered_losses):'')+'</span></div>').join('')+
        pay.map(v=>'<div class="credit163-event"><strong>Parcela Proagro</strong><span>'+(v.paid_date?esc(v.paid_date)+' • ':'')+'pago R$ '+money163(v.paid||0)+'</span></div>').join('')+
        '</div>'
      :'<div class="credit163-empty">Nenhum evento público do Proagro localizado para esta destinação.</div>';

    const released=x.released_total||0, credit=op.credit_value||0;
    const diff=Math.max(0,credit-released);
    return '<article class="credit163-operation">'+
      '<div class="credit163-ophead"><div><strong>'+esc(op.institution_name||op.institution_code||'Instituição não identificada')+'</strong><small>REF BACEN '+esc(op.ref_bacen)+' • ordem '+esc(op.order)+' • '+esc(op.program_name||op.program_code||'sem programa')+'</small></div><div><b>R$ '+money163(credit)+'</b><span>'+esc(status)+'</span></div></div>'+
      '<div class="credit163-summary">'+
        metric163('Saldo público','R$ '+money163(bal.last_day_balance||0),'data-base '+base)+
        metric163('Liberado','R$ '+money163(released),releases.length+' liberação(ões)')+
        metric163('Dif. contratado × liberado','R$ '+money163(diff),'diferença matemática; não significa parcela obrigatoriamente pendente')+
        metric163('Desembolso previsto','R$ '+money163(x.scheduled_total||0),schedule.length+' parcela(s) do cronograma')+
      '</div>'+
      '<details><summary>Condições e produção registradas</summary><div class="credit163-fields">'+financial+production+'</div></details>'+
      '<details><summary>Saldo, situação e liberações</summary><div class="credit163-fields">'+field163('Situação',status)+field163('Data-base do saldo',base)+field163('Saldo médio diário',bal.average_daily?'R$ '+money163(bal.average_daily):'—')+field163('Saldo vincendo médio',bal.average_due?'R$ '+money163(bal.average_due):'—')+'</div><h5>Liberações</h5>'+releaseHtml+'<h5>Cronograma de desembolso</h5>'+scheduleHtml+'</details>'+
      '<details><summary>Renegociação, fonte e desclassificação</summary><h5>Renegociações</h5>'+renegHtml+'<h5>Alterações de fonte</h5>'+srcHtml+'<h5>Desclassificações</h5>'+disqHtml+'</details>'+
      '<details><summary>Proagro</summary>'+proagro+'</details>'+
      '<details><summary>Enquadramento MCR e ZARC</summary><div class="credit163-fields">'+
        field163('Programa',x.mcr?.program||op.program_name||'—')+
        field163('Subprograma',x.mcr?.subprogram||op.subprogram_name||'—')+
        field163('Taxa registrada',x.mcr?.registered_rate_pct?fmt(x.mcr.registered_rate_pct,4)+'% a.a.':'—')+
        field163('Produto ZARC',x.zarc?.product||op.product||'—')+
        field163('Plantio informado',dateRange163(x.zarc?.planting_start,x.zarc?.planting_end))+
        field163('Manejo / irrigação',x.zarc?.irrigation||'—')+
        field163('Safra ZARC',zc.safra||'—')+
        field163('Solo ZARC/SICOR',zc.soil_name||zc.soil_code||'—')+
        field163('Ciclo/grupo',zc.cycle_name||zc.cycle_code||'—')+
        field163('Decêndios do plantio',(zc.planting_decendios||[]).join(', ')||'—')+
        field163('Riscos localizados',(zc.risk_levels||[]).length?(zc.risk_levels||[]).map(v=>v+'%').join(', '):'—')+
      '</div>'+zarcCheck163(zc)+'<div class="credit163-actions"><button class="btn ghost" data-mcr163="'+escAttr163(x.mcr?.source_url||'https://www3.bcb.gov.br/mcr/completo')+'">MCR oficial</button><button class="btn ghost" data-zarc163="'+escAttr163(zc.dataset_url||x.zarc?.source_url||'https://www.gov.br/agricultura/pt-br/assuntos/riscos-seguro/programa-nacional-de-zoneamento-agricola-de-risco-climatico')+'">Fonte ZARC</button></div><div class="xray-source150">'+esc(x.mcr?.message||'Confirme as regras vigentes no MCR oficial.')+'</div></details>'+
    '</article>';
  }

  function zarcCheck163(z){
    if(!z||!z.attempted)return '<div class="credit163-event"><strong>ZARC automático</strong><span>Sem dados suficientes ou análise não aplicável a esta destinação.</span></div>';
    const cls=z.matched?' ok':(z.available?' warning':'');
    const label=z.matched?'Correspondência ZARC localizada':(z.available?'Conferir ZARC':'ZARC automático sem conclusão');
    const extra=[
      (z.portarias||[]).length?'Portaria(s): '+(z.portarias||[]).join(' | '):'',
      (z.manejos||[]).length?'Manejo(s): '+(z.manejos||[]).join(' | '):''
    ].filter(Boolean).join(' • ');
    return '<div class="credit163-event'+cls+'"><strong>'+esc(label)+'</strong><span>'+esc(z.message||'')+(extra?'<br>'+esc(extra):'')+'</span></div>';
  }

  function generic163(rec){
    const fields=rec?.fields||{};
    const keys=Object.keys(fields).sort();
    return '<div class="credit163-event"><strong>Registro público relacionado</strong><span>'+keys.map(k=>esc(friendly163(k))+': '+esc(fields[k])).join(' • ')+'</span></div>';
  }
  function friendly163(k){
    const m={REF_BACEN:'REF BACEN',REF_BACEN_ORIGEM:'REF origem',REF_BACEN_RENEGOCIADA:'REF renegociada',NU_ORDEM:'Ordem',DT_RENEGOCIACAO:'Data renegociação',VL_RENEGOCIADO:'Valor renegociado',CD_BASE_LEGAL:'Base legal',CD_FONTE_RECURSO:'Fonte',VL_PARC_CREDITO:'Valor'};
    return m[k]||k.replaceAll('_',' ');
  }
  function metric163(label,value,small){return '<div class="credit163-metric"><span>'+esc(label)+'</span><strong>'+esc(String(value))+'</strong><small>'+esc(small||'')+'</small></div>'}
  function field163(label,value){return '<div class="credit163-field"><span>'+esc(label)+'</span><b>'+esc(String(value||'—'))+'</b></div>'}
  function row163(a,b){return '<div><span>'+esc(a||'—')+'</span><strong>'+esc(b||'—')+'</strong></div>'}
  function money163(v){try{return Number(v||0).toLocaleString('pt-BR',{minimumFractionDigits:2,maximumFractionDigits:2})}catch(e){return String(v||0)}}
  function num163(v){return Number(v||0)?Number(v).toLocaleString('pt-BR',{maximumFractionDigits:4}):'—'}
  function dateRange163(a,b){if(!a&&!b)return '—';if(a&&b&&a!==b)return a+' a '+b;return a||b||'—'}
  function escAttr163(v){return String(v||'').replaceAll('&','&amp;').replaceAll('"','&quot;').replaceAll('<','&lt;').replaceAll('>','&gt;')}

  function resetIfCarChanged(){
    const car=state?.car?.car||'';
    if(fin.car&&fin.car!==car){fin.car='';fin.result=null;g('creditIntelligence163')?.remove()}
    install163();
  }

  const observer=new MutationObserver(()=>resetIfCarChanged());
  function start163(){
    const box=g('xrayWorkspace150');
    if(box)observer.observe(box,{childList:true,subtree:true});
    resetIfCarChanged();
  }
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',start163); else start163();
})();