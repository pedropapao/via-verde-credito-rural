/* ViaVerdeCAR 1.8.3 — Inteligência Ambiental & importação oficial SICAR */
(function(){
  const g=id=>document.getElementById(id);
  const s180={car:'',data:null,loading:false,layer:null};

  function api180(){return typeof api==='function'?api():window.go?.main?.App}
  function esc180(v){return typeof esc==='function'?esc(String(v??'')):String(v??'').replace(/[&<>"']/g,m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]))}
  function n180(v,d=2){try{return Number(v||0).toLocaleString('pt-BR',{minimumFractionDigits:d,maximumFractionDigits:d})}catch(_){return String(v||0)}}
  function date180(v){if(!v)return '—';const m=String(v).match(/^(\d{4})-(\d{2})-(\d{2})/);return m?m[3]+'/'+m[2]+'/'+m[1]:String(v)}
  function hasProperty180(){return Number(state?.selectedProperty?.id||0)>0}

  function attach180(){
    const box=g('xrayWorkspace150');if(!box)return;
    const car=state?.car?.car||'';
    if(s180.car!==car){s180.car=car;s180.data=null;clearMap180()}
    if(g('environmentIntel180'))return;
    const hero=box.querySelector('.xray-hero150');if(!hero)return;
    const panel=document.createElement('article');
    panel.id='environmentIntel180';panel.className='panel environmental-intel180';
    const anchor=g('creditIntel170')||hero;
    anchor.after(panel);
    renderIdle180();
  }

  function renderIdle180(){
    const panel=g('environmentIntel180');if(!panel)return;
    if(!state?.car?.car){
      panel.innerHTML='<div class="environment-empty180">Consulte um CAR para habilitar a Inteligência Ambiental.</div>';
      return;
    }
    if(s180.data){render180(s180.data);return}
    panel.innerHTML='<div class="panel-title"><div><span class="eyebrow">AMBIENTAL • INTELIGÊNCIA E LAUDOS</span><h3>Triagem técnica profissional do imóvel</h3><p>MapBiomas Alerta + CAR + APP + Reserva Legal + IBAMA + FUNAI + ICMBio + MMA/MCR, com cruzamentos espaciais e laudos rastreáveis.</p></div><span class="status-badge neutral">Sob demanda</span></div>'+
      '<div class="environment-intro180"><div class="environment-art180">🌿</div><div><strong>Análise ambiental aprofundada</strong><span>O Via Verde organiza evidências públicas, calcula sobreposições e gera relatório técnico. Alerta não é tratado automaticamente como infração.</span></div></div>'+
      '<div class="environment-actions180"><button class="btn primary" id="loadEnvironment180">Analisar ambiente</button><button class="btn ghost" id="openMapBiomasMethod180">Metodologia MapBiomas</button></div>'+
      '<div class="environment-note180">Para o detalhamento completo dos alertas, mantenha a conta MapBiomas Alerta conectada em Configurações.</div>';
    g('loadEnvironment180').onclick=()=>load180(false);
    g('openMapBiomasMethod180').onclick=()=>openExternal('https://alerta.mapbiomas.org/metodo-mapbiomas-alerta/');
  }

  async function load180(force){
    if(s180.loading||!state?.car?.car)return;
    const panel=g('environmentIntel180');if(!panel)return;
    s180.loading=true;
    panel.innerHTML='<div class="panel-title"><div><span class="eyebrow">AMBIENTAL • INTELIGÊNCIA E LAUDOS</span><h3>Construindo diagnóstico ambiental…</h3></div><span class="status-badge info">Consultando</span></div>'+
      '<div class="environment-loading180"><div>🌎</div><strong>Cruzando evidências ambientais</strong><span>Consultando MapBiomas Alerta e organizando APP, Reserva Legal, embargos, Terras Indígenas, UCs e lista MMA/MCR.</span></div>';
    try{
      const r=await api180().GetEnvironmentalIntelligence(state.selectedProperty?.id||0,!!force);
      s180.data=r;s180.car=r.car||state.car.car;render180(r);
    }catch(e){
      panel.innerHTML='<div class="panel-title"><div><span class="eyebrow">AMBIENTAL</span><h3>Análise não concluída</h3></div><span class="status-badge warning">Falhou</span></div><div class="xray-warning150">'+esc180(String(e))+'</div><div class="environment-actions180"><button class="btn primary" id="retryEnvironment180">Tentar novamente</button></div>';
      g('retryEnvironment180').onclick=()=>load180(true);
      try{toast(String(e),true)}catch(_){}
    }finally{s180.loading=false}
  }

  function render180(r){
    const panel=g('environmentIntel180');if(!panel)return;
    const s=r.summary||{},alerts=r.alerts||[],env=r.environment||{},mb=r.mapbiomas||{},warnings=r.warnings||[],themes=r.themes||{};
    const appOK=themeAvailable180(themes,'APP'),rlOK=themeAvailable180(themes,'RESERVA_LEGAL');
    const badge=alerts.length?(s.high_attention_alerts?'Atenção técnica':alerts.length+' alerta(s)'):'Sem alertas retornados';
    panel.innerHTML='<div class="panel-title"><div><span class="eyebrow">AMBIENTAL • INTELIGÊNCIA E LAUDOS</span><h3>Diagnóstico técnico de evidências públicas</h3><p>Resultado específico do CAR '+esc180(r.car||'')+' • '+esc180((r.municipality||'')+' / '+(r.uf||''))+'</p></div><span class="status-badge '+(s.high_attention_alerts?'warning':alerts.length?'info':'ok')+'">'+esc180(badge)+'</span></div>'+
      '<div class="environment-metrics180">'+
        metric180('Alertas MapBiomas',s.alerts||0,(mb.connected?'API conectada':'API não conectada'))+
        metric180('Área de alertas no CAR',n180(s.alert_area_in_car_ha)+' ha','soma estimada das ocorrências')+
        metric180('Sobre APP',appOK?n180(s.app_overlap_ha)+' ha':'Indisponível',appOK?(s.alerts_over_app||0)+' alerta(s)':'tema SICAR não obtido')+
        metric180('Sobre Reserva Legal',rlOK?n180(s.rl_overlap_ha)+' ha':'Indisponível',rlOK?(s.alerts_over_rl||0)+' alerta(s)':'tema SICAR não obtido')+
        metric180('Embargo IBAMA',s.alerts_over_ibama||0,'alerta(s) com interseção')+
        metric180('Alta prioridade',s.high_attention_alerts||0,'regra objetiva de conferência')+
      '</div>'+
      '<div class="environment-actions180">'+
        '<button class="btn primary" id="exportEnvironmentalReport180" '+(!hasProperty180()?'disabled':'')+'>Gerar laudo ambiental PDF</button>'+
        '<button class="btn ghost" id="exportEnvironmentalJSON180" '+(!hasProperty180()?'disabled':'')+'>Exportar evidências JSON</button>'+
        '<button class="btn ghost" id="showEnvironmentalMap180" '+(!alerts.some(a=>a.geometry_geojson)?'disabled':'')+'>Mostrar alertas no mapa</button>'+
        '<button class="btn ghost" id="refreshEnvironment180">Atualizar análise</button>'+
        '<button class="btn ghost" id="openMapBiomasProperty180">Abrir imóvel no MapBiomas</button>'+
      '</div>'+
      (!hasProperty180()?'<div class="environment-note180">Para gerar laudos, vincule o CAR a um imóvel salvo. A análise pode ser visualizada normalmente em consulta avulsa.</div>':'')+
      ((!appOK||!rlOK)?'<div class="environment-note180"><strong>Camadas SICAR incompletas.</strong> “Indisponível” significa que a fonte não foi obtida nesta execução; o ViaVerdeCAR não assume 0 ha.</div>':'')+
      manualSICARControls180(r)+
      evidenceMatrix180(r)+
      '<div class="environment-alerts180">'+(alerts.length?alerts.map((a,i)=>alert180(a,i,r)).join(''):'<div class="environment-empty180">'+esc180(mb.message||'Nenhum alerta MapBiomas foi retornado nesta consulta.')+'<br><small>Ausência de alerta não equivale a certificado de regularidade ambiental.</small></div>')+'</div>'+
      (warnings.length?'<details class="environment-warnings180"><summary>Limitações e ocorrências da consulta ('+warnings.length+')</summary>'+warnings.map(w=>'<div>• '+esc180(w)+'</div>').join('')+'</details>':'')+
      '<div class="environment-legal180"><strong>Interpretação técnica:</strong> '+esc180(r.interpretation||'Triagem auxiliar baseada em fontes públicas.')+'</div>';

    g('exportEnvironmentalReport180').onclick=exportReport180;
    g('exportEnvironmentalJSON180').onclick=exportJSON180;
    g('showEnvironmentalMap180').onclick=()=>showMap180(alerts);
    g('refreshEnvironment180').onclick=()=>load180(true);
    g('openMapBiomasProperty180').onclick=()=>openExternal('https://plataforma.alerta.mapbiomas.org/imovel/'+encodeURIComponent(r.car||''));
    const openSicar=g('openSICARDownloads180');if(openSicar)openSicar.onclick=()=>openExternal('https://consulta.car.gov.br/');
    panel.querySelectorAll('[data-import-sicar-theme180]').forEach(b=>b.onclick=()=>importSICARTheme180(b.dataset.importSicarTheme180));
    panel.querySelectorAll('[data-alert-report180]').forEach(b=>b.onclick=()=>exportAlert180(b.dataset.alertReport180));
    panel.querySelectorAll('[data-alert-open180]').forEach(b=>b.onclick=()=>openExternal(b.dataset.alertOpen180));
    panel.querySelectorAll('[data-alert-map180]').forEach(b=>b.onclick=()=>showMap180([alerts[Number(b.dataset.alertMap180)]]));
  }

  function metric180(label,value,small){return '<div><span>'+esc180(label)+'</span><strong>'+esc180(String(value))+'</strong><small>'+esc180(small||'')+'</small></div>'}
  function themeMetric180(themes,code){return themes?.themes?.[code]||null}
  function themeAvailable180(themes,code){return !!themeMetric180(themes,code)?.available}
  function themeStatus180(themes,code,label){
    const m=themeMetric180(themes,code);
    if(m?.status==='manual_required')return label+': download manual';
    if(!m||!m.available)return label+': indisponível';
    if(m.cache_status==='cache_stale')return label+': cache anterior';
    if(m.cache_status==='manual')return label+': importado';
    if(m.cache_status==='cache_fresh')return label+': cache recente';
    return label+': consultado';
  }

  function manualSICARControls180(r){
    const themes=r?.themes||{};
    const defs=[
      ['APP','APP'],
      ['RESERVA_LEGAL','Reserva Legal'],
      ['VEGETACAO_NATIVA','Vegetação Nativa'],
      ['AREA_CONSOLIDADA','Área Consolidada'],
      ['USO_RESTRITO','Uso Restrito'],
      ['SERVIDAO_ADMINISTRATIVA','Servidão Administrativa']
    ];
    const missing=defs.filter(([code])=>!themeAvailable180(themes,code));
    if(!missing.length)return '';
    const manual=missing.some(([code])=>themeMetric180(themes,code)?.status==='manual_required');
    if(!manual)return '';
    return '<section class="sicar-manual180">'+
      '<div class="sicar-manual-head180"><div><strong>📦 Temas detalhados do SICAR exigem validação humana</strong><span>A Base de Downloads está devolvendo a página de validação/CAPTCHA em vez do ZIP. Faça o download oficial no portal e importe cada tema abaixo. O ViaVerdeCAR cruza o arquivo com este CAR automaticamente.</span></div><button class="btn ghost" id="openSICARDownloads180">Abrir Base oficial</button></div>'+
      '<div class="sicar-manual-themes180">'+missing.map(([code,label])=>'<button class="btn ghost" data-import-sicar-theme180="'+code+'">Importar '+esc180(label)+'</button>').join('')+'</div>'+
      '<small>O aplicativo não tenta contornar o CAPTCHA. O ZIP importado fica identificado como “pacote oficial importado” e pode ser usado nos laudos.</small>'+
      '</section>';
  }

  async function importSICARTheme180(code){
    if(s180.loading)return;
    try{
      s180.loading=true;
      const result=await api180().ImportSICARThemeZIP(state.selectedProperty?.id||0,code);
      toast(result?.message||'Tema SICAR importado e cruzado com o CAR.');
      s180.data=null;
    }catch(e){
      if(!String(e).toLowerCase().includes('cancelada'))toast(String(e),true);
      return;
    }finally{s180.loading=false}
    await load180(false);
  }

  function evidenceMatrix180(r){
    const env=r.environment||{},themes=r.themes||{},mb=r.mapbiomas||{};
    const available=Object.values(themes.themes||{}).filter(x=>x?.available).length;
    const themeDetail=[
      themeStatus180(themes,'APP','APP'),
      themeStatus180(themes,'RESERVA_LEGAL','RL'),
      themeStatus180(themes,'VEGETACAO_NATIVA','Vegetação'),
      themeStatus180(themes,'AREA_CONSOLIDADA','Consolidada'),
      themeStatus180(themes,'USO_RESTRITO','Uso restrito'),
      themeStatus180(themes,'SERVIDAO_ADMINISTRATIVA','Servidão')
    ].join(' • ');
    const row=(icon,name,status,detail,cls='')=>'<div class="environment-source180 '+cls+'"><span>'+icon+'</span><div><b>'+esc180(name)+'</b><small>'+esc180(detail)+'</small></div><strong>'+esc180(status)+'</strong></div>';
    return '<details class="environment-matrix180" open><summary>Matriz de evidências e fontes</summary><div>'+
      row('🛰️','MapBiomas Alerta',mb.connected?(mb.total_alerts||0)+' alerta(s)':'Não conectado',mb.message||'API V2')+
      row('🌱','SICAR — temas',available+'/6',themeDetail,available<6?'warn':'')+
      row('⛔','IBAMA / PAMGIA',env.ibama_checked?(env.ibama_embargo_count||0)+' ocorrência(s)':'Indisponível','embargos com interseção no CAR',env.ibama_embargo_count?'warn':'')+
      row('🪶','FUNAI',env.funai_checked?(env.indigenous_count||0)+' ocorrência(s)':'Indisponível','Terras Indígenas com interseção no CAR',env.indigenous_count?'warn':'')+
      row('🏞️','ICMBio',env.icmbio_checked?(env.federal_uc_count||0)+' ocorrência(s)':'Indisponível','Unidades de Conservação federais',env.federal_uc_count?'warn':'')+
      row('📋','MMA / MCR-PRODES',env.mcr_checked?(env.mcr_listed?'LISTADO':'Não listado'):'Indisponível','lista pública vinculada ao Manual de Crédito Rural',env.mcr_listed?'warn':'')+
      '</div></details>';
  }

  function alert180(a,i,r){
    const high=a.attention_level==='Alta prioridade de conferência';
    const themes=r?.themes||{},env=r?.environment||{};
    const appOK=themeAvailable180(themes,'APP'),rlOK=themeAvailable180(themes,'RESERVA_LEGAL');
    const overlap=[];
    if(Number(a.app_overlap_ha)>0)overlap.push('APP '+n180(a.app_overlap_ha,2)+' ha');
    if(Number(a.rl_overlap_ha)>0)overlap.push('RL '+n180(a.rl_overlap_ha,2)+' ha');
    if(Number(a.ibama_overlap_ha)>0)overlap.push('IBAMA '+n180(a.ibama_overlap_ha,2)+' ha');
    if(Number(a.indigenous_overlap_ha)>0)overlap.push('TI '+n180(a.indigenous_overlap_ha,2)+' ha');
    if(Number(a.federal_uc_overlap_ha)>0)overlap.push('UC '+n180(a.federal_uc_overlap_ha,2)+' ha');
    return '<article class="environment-alert180 '+(high?'high':'')+'">'+
      '<div class="environment-alert-head180"><div><strong>Alerta '+esc180(a.alert_code||'—')+'</strong><small>'+date180(a.detected_at)+' • '+n180(a.area_ha,2)+' ha • '+esc180((a.sources||[]).join(', ')||'MapBiomas Alerta')+'</small></div><span class="status-badge '+(high?'warning':'info')+'">'+esc180(a.attention_level||'Conferir')+'</span></div>'+
      '<div class="environment-alert-grid180">'+
        metric180('Dentro do CAR',n180(a.alert_area_in_car_ha,2)+' ha',n180(a.alert_pct_of_car,2)+'% do imóvel')+
        metric180('APP',appOK?n180(a.app_overlap_ha,2)+' ha':'Indisponível',appOK?'interseção calculada':'tema SICAR não obtido')+
        metric180('Reserva Legal',rlOK?n180(a.rl_overlap_ha,2)+' ha':'Indisponível',rlOK?'interseção calculada':'tema SICAR não obtido')+
        metric180('Imagem antes',date180(a.image_before_at),'data informada')+
        metric180('Imagem depois',date180(a.image_after_at),'data informada')+
        metric180('Status',a.status_name||'Publicado',date180(a.published_at))+
      '</div>'+
      (overlap.length?'<div class="environment-tags180">'+overlap.map(x=>'<span>'+esc180(x)+'</span>').join('')+'</div>':'')+
      ((a.attention_reasons||[]).length?'<div class="environment-reasons180">'+a.attention_reasons.map(x=>'<div>• '+esc180(x)+'</div>').join('')+'</div>':'')+
      '<div class="environment-actions180"><button class="btn ghost" data-alert-map180="'+i+'" '+(!a.geometry_geojson?'disabled':'')+'>Ver no mapa</button><button class="btn ghost" data-alert-open180="'+esc180(a.report_url||'')+'" '+(!a.report_url?'disabled':'')+'>Abrir MapBiomas</button><button class="btn primary" data-alert-report180="'+esc180(a.alert_code||'')+'" '+(!hasProperty180()?'disabled':'')+'>Gerar laudo deste alerta</button></div>'+
      '</article>';
  }

  async function exportReport180(){
    try{const p=await api180().ExportEnvironmentalTechnicalReport(state.selectedProperty?.id||0,false);toast('Laudo ambiental salvo em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}
  }
  async function exportJSON180(){
    try{const p=await api180().ExportEnvironmentalEvidenceJSON(state.selectedProperty?.id||0,false);toast('Evidências ambientais salvas em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}
  }
  async function exportAlert180(code){
    try{const p=await api180().ExportMapBiomasAlertTechnicalReport(state.selectedProperty?.id||0,code,false);toast('Laudo do alerta salvo em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}
  }

  function clearMap180(){
    if(s180.layer&&state?.map){try{state.map.removeLayer(s180.layer);state.layerControl?.removeLayer(s180.layer)}catch(_){}}
    s180.layer=null;
  }
  function showMap180(alerts){
    if(!state?.map||!Array.isArray(alerts))return;
    clearMap180();
    const group=L.featureGroup();let count=0;
    alerts.forEach(a=>{if(!a?.geometry_geojson)return;try{
      const layer=L.geoJSON(JSON.parse(a.geometry_geojson),{style:{color:'#b33a2b',weight:3,fillOpacity:.18}});
      layer.bindPopup('<strong>MapBiomas Alerta '+esc180(a.alert_code)+'</strong><br>'+n180(a.alert_area_in_car_ha,2)+' ha no CAR<br>Detectado: '+date180(a.detected_at));
      layer.eachLayer(x=>x.addTo(group));count++;
    }catch(_){}});
    if(!count){toast('Nenhuma geometria de alerta disponível para desenhar.',true);return}
    group.addTo(state.map);state.layerControl?.addOverlay(group,'MapBiomas • alertas ambientais');s180.layer=group;
    document.querySelector('[data-car-tab131="map"]')?.click();
    setTimeout(()=>{try{const b=group.getBounds();if(b.isValid())state.map.fitBounds(b.pad(.2),{maxZoom:17})}catch(_){}},100);
  }

  function watch180(){
    const install=()=>{try{attach180()}catch(e){try{console.error('ViaVerdeCAR 1.8.3 ambiental:',e)}catch(_){}}};
    install();
    const root=g('carTabXRay150')||document.body;
    new MutationObserver(()=>install()).observe(root,{childList:true,subtree:true});
  }
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',watch180);else watch180();
})();