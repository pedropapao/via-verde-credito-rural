/* ViaVerdeCAR 1.7.0 — inteligência de crédito rural */
(function(){
  const g=id=>document.getElementById(id);
  const s170={car:'',data:null,loading:false};

  function api170(){return typeof api==='function'?api():window.go?.main?.App}
  function esc170(v){return typeof esc==='function'?esc(String(v??'')):String(v??'').replace(/[&<>"']/g,m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]))}
  function money170(v){try{return Number(v||0).toLocaleString('pt-BR',{style:'currency',currency:'BRL'})}catch(e){return 'R$ '+Number(v||0).toFixed(2)}}
  function num170(v,d=2){try{return Number(v||0).toLocaleString('pt-BR',{minimumFractionDigits:d,maximumFractionDigits:d})}catch(e){return String(v||0)}}
  function pct170(v){return num170(v,2)+'%'}
  function has170(v){return v!==null&&v!==undefined&&String(v).trim()!==''&&Number(v)!==0}
  function dt170(v){if(!v)return '—';const s=String(v);const m=s.match(/^(\d{4})-(\d{2})-(\d{2})/);return m?m[3]+'/'+m[2]+'/'+m[1]:s}
  function statusClass170(v){const s=String(v||'').toUpperCase();if(/ATRAS|INADIM|PREJU|DESCLASS/.test(s))return 'warning';if(/LIQUID/.test(s))return 'ok';if(/RENEG|PRORROG/.test(s))return 'info';return 'neutral'}

  function attach170(){
    const box=g('xrayWorkspace150');if(!box)return;
    if(s170.car!==(state?.car?.car||'')){s170.car=state?.car?.car||'';s170.data=null}
    const hero=box.querySelector('.xray-hero150');
    if(!hero||g('creditIntel170'))return;
    const hasOps=(box.textContent||'').includes('Operações públicas')||box.querySelector('.xray-operation150');
    const panel=document.createElement('article');panel.id='creditIntel170';panel.className='panel credit-intel170';
    panel.innerHTML='<div class="panel-title"><div><span class="eyebrow">CRÉDITO RURAL • INTELIGÊNCIA FINANCEIRA</span><h3>Vida financeira das operações públicas</h3><p>Saldo e situação, liberações, cronograma, renegociação, desclassificação, Proagro, produção e enquadramento técnico.</p></div><span class="status-badge neutral" id="creditIntelBadge170">Sob demanda</span></div>'+
      '<div id="creditIntelBody170">'+
      '<div class="credit-intel-intro170"><div><strong>Consulta aprofundada</strong><span>Usa arquivos nacionais adicionais do SICOR. Na primeira execução pode exigir novos downloads e fica isolada do restante do Raio X.</span></div>'+
      '<div class="credit-intel-actions170"><button class="btn primary" id="loadCreditIntel170" '+(!hasOps?'disabled':'')+'>Carregar inteligência financeira</button><button class="btn ghost" id="openMCR170">Abrir MCR</button><button class="btn ghost" id="openZARC170">Abrir ZARC</button></div></div>'+
      '<div class="xray-source150">Os valores exibidos são somente os localizados nas bases públicas para o CAR. Não representam necessariamente a dívida total, todo o crédito privado nem toda a exposição financeira do produtor.</div>'+
      '</div>';
    hero.after(panel);
    g('loadCreditIntel170').onclick=()=>load170(false);
    g('openMCR170').onclick=()=>openExternal('https://www3.bcb.gov.br/mcr/completo');
    g('openZARC170').onclick=()=>openExternal('https://dados.agricultura.gov.br/dataset/tabua-de-risco-zoneamento-agricola-de-risco-climatico');
    if(s170.data)render170(s170.data);
  }

  async function load170(force){
    if(s170.loading||!state?.car?.car)return;
    const body=g('creditIntelBody170'),badge=g('creditIntelBadge170');if(!body)return;
    s170.loading=true;if(badge){badge.textContent='Consultando';badge.className='status-badge info'}
    body.innerHTML='<div class="credit-intel-loading170"><strong>Sincronizando dados financeiros oficiais…</strong><span>Consultando saldos, liberações, desembolsos, desclassificações, renegociações e Proagro vinculados às operações deste CAR.</span></div>';
    try{
      const r=await api170().GetCreditIntelligence(state.selectedProperty?.id||0,!!force);
      s170.data=r;s170.car=r.car||state.car.car;render170(r);
    }catch(e){
      body.innerHTML='<div class="xray-warning150"><strong>Inteligência financeira não concluída.</strong><br>'+esc170(String(e))+'</div><div class="credit-intel-actions170"><button class="btn primary" id="retryCreditIntel170">Tentar novamente</button></div>';
      g('retryCreditIntel170').onclick=()=>load170(true);
      if(badge){badge.textContent='Falhou';badge.className='status-badge warning'}
      try{toast(String(e),true)}catch(_){}
    }finally{s170.loading=false}
  }

  function render170(r){
    const body=g('creditIntelBody170'),badge=g('creditIntelBadge170');if(!body)return;
    const t=r.totals||{},ops=r.operations||[],warnings=r.warnings||[];
    if(badge){badge.textContent=ops.length?ops.length+' operação(ões)':'Sem operações';badge.className='status-badge '+(warnings.length?'info':'ok')}
    const metrics='<div class="credit-intel-metrics170">'+
      metric170('Contratado identificado',money170(t.contracted),'operações públicas localizadas')+
      metric170('Liberado identificado',money170(t.released),releaseNote170(t))+
      metric170('Últimos saldos',money170(t.latest_balances),(t.with_balance||0)+' operação(ões) com saldo localizado')+
      metric170('Recursos próprios',money170(t.own_resources),'informado nas operações')+
      metric170('Renegociação / prorrogação',t.with_renegotiation||0,'indício público localizado')+
      metric170('Proagro',t.with_proagro||0,'operações com eventos/dados localizados')+
      '</div>';
    const opsHtml=ops.length?'<div class="credit-op-list170">'+ops.map((op,i)=>operation170(op,i)).join('')+'</div>':'<div class="xray-empty150">Nenhuma operação pública foi localizada para a inteligência financeira.</div>';
    const warningHtml=warnings.length?'<details class="credit-warnings170"><summary>Limitações desta consulta ('+warnings.length+')</summary><div>'+warnings.map(x=>'<div>'+esc170(x)+'</div>').join('')+'</div></details>':'';
    body.innerHTML=metrics+
      '<div class="credit-intel-actions170"><button class="btn ghost" id="refreshCreditIntel170">Atualizar dados</button><button class="btn ghost" id="clearCreditIntel170">Limpar cache financeiro</button><button class="btn ghost" id="openBCB170">Microdados BCB</button><button class="btn ghost" id="openMCR170b">MCR</button><button class="btn ghost" id="openZARC170b">ZARC</button></div>'+
      '<div class="credit-intel-note170"><strong>Leitura correta:</strong> “identificado nas bases públicas”. Ausência de registro não prova ausência de financiamento, saldo, renegociação ou seguro.</div>'+
      opsHtml+warningHtml+
      '<div class="xray-source150">'+esc170(r.scope||'')+'</div>';
    g('refreshCreditIntel170').onclick=()=>load170(true);
    g('clearCreditIntel170').onclick=async()=>{try{await api170().ClearCreditIntelligenceCache();s170.data=null;toast('Cache financeiro limpo.');load170(true)}catch(e){toast(String(e),true)}};
    g('openBCB170').onclick=()=>openExternal(r.bcb_source_url||'https://www.bcb.gov.br/estabilidadefinanceira/tabelas-credito-rural-proagro');
    g('openMCR170b').onclick=()=>openExternal(r.mcr_source_url||'https://www3.bcb.gov.br/mcr/completo');
    g('openZARC170b').onclick=()=>openExternal(r.zarc_source_url||'https://dados.agricultura.gov.br/dataset/tabua-de-risco-zoneamento-agricola-de-risco-climatico');
    body.querySelectorAll('[data-credit-op170]').forEach(b=>b.onclick=()=>toggleOp170(b.dataset.creditOp170));
    body.querySelectorAll('[data-zarc-url170]').forEach(b=>b.onclick=()=>openExternal(b.dataset.zarcUrl170));
  }

  function releaseNote170(t){
    if(!Number(t.contracted))return 'sem base para percentual';
    const p=Number(t.released||0)/Number(t.contracted)*100;
    return num170(p,1)+'% do contratado identificado';
  }
  function metric170(label,value,small){return '<div class="credit-intel-metric170"><span>'+esc170(label)+'</span><strong>'+esc170(value)+'</strong><small>'+esc170(small||'')+'</small></div>'}

  function operation170(op,i){
    const balance=op.balance||{},pro=op.proagro||{},desc=op.declassification||{},ren=op.renegotiations||[];
    const releasedPct=Number(op.credit_value)>0?Math.min(999,Number(op.total_released||0)/Number(op.credit_value)*100):0;
    const title=(op.institution||'Instituição não identificada')+' • REF '+(op.ref_bacen||'—')+' / '+(op.order||'—');
    const status=balance.found?(balance.status||('Código '+(balance.status_code||'—'))):'Saldo não localizado';
    const top='<div class="credit-op-head170"><div><strong>'+esc170(title)+'</strong><small>'+esc170(op.program||op.purpose||'Operação SICOR')+' • emissão '+dt170(op.issue_date)+'</small></div><span class="status-badge '+statusClass170(status)+'">'+esc170(status)+'</span></div>';
    const quick='<div class="credit-op-quick170">'+
      mini170('Contratado',money170(op.credit_value))+
      mini170('Liberado',money170(op.total_released),op.total_released?num170(releasedPct,1)+'%':'sem liberação localizada')+
      mini170('Último saldo',balance.found?money170(balance.last_day):'—',balance.found?String(balance.month).padStart(2,'0')+'/'+balance.year:'sem saldo localizado')+
      mini170('Vencimento',dt170(op.due_date))+
      '</div>';
    const flags='<div class="credit-op-flags170">'+
      (ren.length?'<span>Renegociação '+ren.length+'</span>':'')+
      (desc.found?'<span class="warn">Desclassificação</span>':'')+
      ((pro.has_cop||pro.has_rcp||pro.has_judgment||pro.paid_value)?'<span>Proagro</span>':'')+
      (op.disbursements?.length?'<span>Cronograma '+op.disbursements.length+'</span>':'')+
      '</div>';
    return '<article class="credit-op170" id="creditOp170-'+i+'">'+top+quick+flags+
      '<button class="ui162-section-toggle credit-op-toggle170" type="button" data-credit-op170="'+i+'">Ver detalhes financeiros</button>'+
      '<div class="credit-op-details170">'+detailSections170(op)+'</div></article>';
  }
  function mini170(label,value,small){return '<div><span>'+esc170(label)+'</span><b>'+esc170(value)+'</b>'+(small?'<small>'+esc170(small)+'</small>':'')+'</div>'}
  function field170(label,value){if(value===undefined||value===null||value===''||value==='—')return '';return '<div><span>'+esc170(label)+'</span><b>'+esc170(value)+'</b></div>'}

  function detailSections170(op){
    const b=op.balance||{},d=op.declassification||{},p=op.proagro||{},ren=op.renegotiations||[];
    const financial='<section><h4>Financeiro</h4><div class="credit-fields170">'+
      field170('Recursos próprios',has170(op.own_resources)?money170(op.own_resources):'')+
      field170('Taxa de juros',has170(op.interest_rate_pct)?pct170(op.interest_rate_pct):'')+
      field170('Juros pós-fixados',has170(op.post_fixed_interest_pct)?pct170(op.post_fixed_interest_pct):'')+
      field170('Custo efetivo total',has170(op.effective_cost_pct)?pct170(op.effective_cost_pct):'')+
      field170('Prestação investimento',has170(op.investment_installment)?money170(op.investment_installment):'')+
      field170('Área financiada',has170(op.financed_area_ha)?num170(op.financed_area_ha)+' ha':'')+
      field170('Área informada',has170(op.informed_area_ha)?num170(op.informed_area_ha)+' ha':'')+
      field170('Instrumento',op.instrument||op.instrument_code)+
      field170('Categoria emitente',op.issuer_category||op.issuer_category_code)+
      field170('Seguro/garantia',op.insurance||op.insurance_code)+
      '</div></section>';

    const releases='<section><h4>Liberação e desembolso</h4>'+
      '<div class="credit-fields170">'+field170('Total liberado',money170(op.total_released||0))+field170('Última liberação',dt170(op.last_release_date))+field170('Cronograma previsto',money170(op.planned_disbursement||0))+'</div>'+
      timeline170('Liberações',op.releases,'date','value')+timeline170('Cronograma de desembolso',op.disbursements,'date','value')+'</section>';

    const situation='<section><h4>Situação e histórico</h4><div class="credit-fields170">'+
      field170('Situação SICOR',b.found?(b.status||b.status_code):'')+
      field170('Data-base saldo',b.found?String(b.month).padStart(2,'0')+'/'+b.year:'')+
      field170('Saldo último dia',b.found?money170(b.last_day):'')+
      field170('Saldo médio diário',b.found?money170(b.daily_average):'')+
      field170('Saldo médio vincendo',b.found?money170(b.due_daily_average):'')+
      (d.found?field170('Desclassificação',dt170(d.date)+' • '+(d.reason||d.reason_code||'')+' • '+money170(d.value)):'')+
      '</div>'+reneg170(ren)+'</section>';

    const production='<section><h4>Produção e sistema produtivo</h4><div class="credit-fields170">'+
      field170('Finalidade',op.purpose)+field170('Atividade',op.activity)+field170('Modalidade',op.modality)+field170('Produto',op.product)+field170('Variedade',op.variety)+
      field170('Previsão produção',has170(op.forecast_production)?num170(op.forecast_production):'')+
      field170('Quantidade',has170(op.quantity)?num170(op.quantity):'')+
      field170('Receita bruta esperada',has170(op.expected_revenue)?money170(op.expected_revenue):'')+
      field170('Produtividade obtida',has170(op.productivity_obtained)?num170(op.productivity_obtained):'')+
      field170('Irrigação',op.irrigation||op.irrigation_code)+field170('Agricultura',op.agriculture||op.agriculture_code)+field170('Cultivo',op.crop_type||op.crop_type_code)+field170('Integração/consórcio',op.integration||op.integration_code)+field170('Grão/semente',op.seed_type||op.seed_type_code)+field170('Fase produtiva',op.production_phase||op.production_phase_code)+
      '</div></section>';

    const z=op.zarc||{};
    const zPeriods=(z.periods||[]).map(x=>'<span class="'+(x.indicated?'ok':'warn')+'"><b>D'+esc170(x.decendio)+'</b>'+(x.indicated?(x.risk_pct?esc170(x.risk_pct)+'%':esc170(x.raw||'indicado')):'sem indicação')+'</span>').join('');
    const zarc='<section><h4>ZARC / plantio</h4><div class="credit-fields170">'+
      field170('Safra ZARC',z.safra)+field170('Resultado',z.status)+field170('Cultura cruzada',z.culture||op.product)+field170('Ciclo cultivar',z.cycle||op.cultivar_cycle||op.cultivar_cycle_code)+field170('Tipo de solo',z.soil||op.soil||op.soil_code)+field170('Manejo',z.management)+
      field170('Início plantio',dt170(op.planting_start))+field170('Fim plantio',dt170(op.planting_end))+field170('Portaria',z.portaria)+field170('Linhas compatíveis',z.candidates?String(z.candidates):'')+
      '</div>'+(zPeriods?'<div class="zarc-periods170">'+zPeriods+'</div>':'')+
      '<div class="credit-op-note170">'+esc170(z.message||'Sem cruzamento automático disponível para esta operação.')+' O resultado é uma conferência da Tábua de Risco pública e não substitui a Portaria ZARC nem a validação técnica do enquadramento.</div>'+
      (z.source_url?'<div class="credit-intel-actions170"><button class="btn ghost" data-zarc-url170="'+esc170(z.source_url)+'">Abrir fonte ZARC</button></div>':'')+'</section>';

    const proagro=proagro170(p,op.proagro_rate_pct);
    return financial+releases+situation+production+zarc+proagro+
      '<section><h4>Enquadramento MCR</h4><div class="credit-fields170">'+field170('Programa',op.program)+field170('Subprograma',op.subprogram)+field170('Fonte de recursos',op.resource)+'</div><div class="credit-op-note170">Taxa, limite, prazo e enquadramento devem ser confirmados no MCR vigente. O registro SICOR mostra como a operação foi informada, não substitui a regra normativa atual.</div></section>';
  }

  function timeline170(title,arr,dateKey,valueKey){
    if(!arr?.length)return '';
    return '<div class="credit-timeline170"><strong>'+esc170(title)+'</strong>'+arr.map(x=>'<span><b>'+dt170(x[dateKey])+'</b>'+money170(x[valueKey])+'</span>').join('')+'</div>';
  }
  function reneg170(arr){
    if(!arr?.length)return '';
    return '<div class="credit-reneg170"><strong>Renegociações / vínculos localizados</strong>'+arr.map(x=>'<div><span>'+esc170(x.related_ref?('REF relacionada '+x.related_ref+(x.related_order?' / '+x.related_order:'')):'Vínculo de renegociação')+'</span><b>'+esc170(x.value?money170(x.value):(x.legal_basis||x.legal_basis_code||'registro localizado'))+'</b></div>').join('')+'</div>';
  }
  function proagro170(p,rate){
    if(!(p?.has_cop||p?.has_rcp||p?.has_judgment||p?.paid_value||rate))return '';
    return '<section><h4>Proagro</h4><div class="credit-fields170">'+
      field170('Alíquota',has170(rate)?pct170(rate):'')+
      field170('COP',p.has_cop?(dt170(p.cop_date)+' • '+(p.cop_status||p.cop_status_code||'')):'')+
      field170('Evento',p.event||p.event_code)+field170('Solo COP',p.soil||p.soil_code)+field170('Ciclo COP',p.cycle||p.cycle_code)+
      field170('RCP',p.has_rcp?dt170(p.rcp_date):'')+field170('Área RCP',has170(p.rcp_area_ha)?num170(p.rcp_area_ha)+' ha':'')+
      field170('Julgamento',p.has_judgment?dt170(p.judgment_date):'')+field170('Cobertura crédito',has170(p.coverage_credit)?money170(p.coverage_credit):'')+field170('Cobertura recursos próprios',has170(p.coverage_own)?money170(p.coverage_own):'')+field170('Perdas não amparadas',has170(p.losses_uncovered)?money170(p.losses_uncovered):'')+field170('Parcelas pagas',has170(p.paid_value)?money170(p.paid_value):'')+
      '</div></section>';
  }

  function toggleOp170(i){
    const card=g('creditOp170-'+i);if(!card)return;
    card.classList.toggle('open');
    const b=card.querySelector('.credit-op-toggle170');if(b)b.textContent=card.classList.contains('open')?'Recolher detalhes':'Ver detalhes financeiros';
  }

  function watch170(){
    const install=()=>{try{attach170()}catch(e){try{console.error('ViaVerdeCAR 1.7.0 crédito:',e)}catch(_){}}};
    install();
    const root=g('carTabXRay150')||document.body;
    new MutationObserver(()=>install()).observe(root,{childList:true,subtree:true});
  }
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',watch170);else watch170();
})();