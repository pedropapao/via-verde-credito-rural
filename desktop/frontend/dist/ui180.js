/* ViaVerdeCAR 1.9.0 — Perfil Ambiental Automático */
(function(){
  const g=id=>document.getElementById(id);
  const s180={car:'',data:null,loading:false,layer:null};

  function api180(){return typeof api==='function'?api():window.go?.main?.App}
  function esc180(v){return typeof esc==='function'?esc(String(v??'')):String(v??'').replace(/[&<>"']/g,m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]))}
  function n180(v,d=2){try{return Number(v||0).toLocaleString('pt-BR',{minimumFractionDigits:d,maximumFractionDigits:d})}catch(_){return String(v||0)}}
  function date180(v){if(!v)return '—';const m=String(v).match(/^(\d{4})-(\d{2})-(\d{2})/);return m?m[3]+'/'+m[2]+'/'+m[1]:String(v)}
  function hasProperty180(){return Number(state?.selectedProperty?.id||0)>0}
  function arr180(v){return Array.isArray(v)?v:[]}
  function statusText180(s){if(!s)return 'Indisponível';if(s.available)return 'Consultado';if(s.applicable===false)return 'Não aplicável';return 'Indisponível'}

  function attach180(){
    const box=g('xrayWorkspace150');if(!box)return;
    const car=state?.car?.car||'';
    if(s180.car!==car){s180.car=car;s180.data=null;clearMap180()}
    if(g('environmentIntel180'))return;
    const hero=box.querySelector('.xray-hero150');if(!hero)return;
    const panel=document.createElement('article');
    panel.id='environmentIntel180';panel.className='panel environmental-intel180';
    const anchor=g('creditIntel170')||hero;
    anchor.after(panel);renderIdle180();
  }

  function renderIdle180(){
    const panel=g('environmentIntel180');if(!panel)return;
    if(!state?.car?.car){panel.innerHTML='<div class="environment-empty180">Consulte um CAR para habilitar a Inteligência Ambiental.</div>';return}
    if(s180.data){render180(s180.data);return}
    panel.innerHTML='<div class="panel-title"><div><span class="eyebrow">AMBIENTAL • PERFIL AUTOMÁTICO</span><h3>Raio X ambiental do imóvel</h3><p>Bioma, cobertura do solo e evidências territoriais — consulta automática.</p></div><span class="status-badge info">Preparando</span></div>'+
      '<div class="environment-intro180"><div class="environment-art180">🌎</div><div><strong>Perfil ambiental automático</strong><span>O Via Verde vai cruzar este CAR com serviços públicos e gratuitos. Nenhum download ou importação manual é necessário.</span></div></div>';
    setTimeout(()=>load180(false),0);
  }

  async function load180(force){
    if(s180.loading||!state?.car?.car)return;
    const panel=g('environmentIntel180');if(!panel)return;
    s180.loading=true;
    panel.innerHTML='<div class="panel-title"><div><span class="eyebrow">AMBIENTAL • PERFIL AUTOMÁTICO</span><h3>Montando perfil ambiental…</h3></div><span class="status-badge info">Consultando</span></div>'+
      '<div class="environment-loading180"><div>🌎</div><strong>Cruzando fontes ambientais públicas</strong><span>Bioma, cobertura do solo, MapBiomas, IBAMA, FUNAI, ICMBio e MMA/MCR.</span></div>';
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
    const s=r.summary||{},alerts=arr180(r.alerts),env=r.environment||{},mb=r.mapbiomas||{},warnings=arr180(r.warnings),p=r.profile||{};
    const badge=(s.high_attention_alerts||env.mcr_listed||env.ibama_embargo_count)?'Atenção técnica':alerts.length?alerts.length+' alerta(s)':'Perfil concluído';
    const hasMap=profileHasMap180(p)||alerts.some(a=>a.geometry_geojson);
    panel.innerHTML='<div class="panel-title"><div><span class="eyebrow">AMBIENTAL • PERFIL AUTOMÁTICO</span><h3>Raio X ambiental do imóvel</h3><p>CAR '+esc180(r.car||'')+' • '+esc180((r.municipality||'')+' / '+(r.uf||''))+'</p></div><span class="status-badge '+((s.high_attention_alerts||env.mcr_listed||env.ibama_embargo_count)?'warning':'ok')+'">'+esc180(badge)+'</span></div>'+
      profile180(r)+
      '<div class="environment-risk-title180"><strong>Evidências e restrições públicas</strong><span>Ocorrências localizadas nas fontes automáticas.</span></div>'+
      '<div class="environment-metrics180">'+
        metric180('Alertas MapBiomas',s.alerts||0,(mb.connected?'API conectada':'API não conectada'))+
        metric180('Área de alertas',n180(s.alert_area_in_car_ha)+' ha','estimada dentro do CAR')+
        metric180('Embargo IBAMA',env.ibama_embargo_count||0,'interseção(ões) no CAR')+
        metric180('Terra Indígena',env.indigenous_count||0,'interseção(ões) no CAR')+
        metric180('UC Federal',env.federal_uc_count||0,'interseção(ões) no CAR')+
        metric180('MMA / MCR',env.mcr_checked?(env.mcr_listed?'LISTADO':'Não listado'):'Indisponível','lista pública PRODES/MCR')+
      '</div>'+
      '<div class="environment-actions180">'+
        '<button class="btn primary" id="exportEnvironmentalReport180" '+(!hasProperty180()?'disabled':'')+'>Gerar laudo ambiental PDF</button>'+
        '<button class="btn ghost" id="exportEnvironmentalJSON180" '+(!hasProperty180()?'disabled':'')+'>Exportar evidências JSON</button>'+
        '<button class="btn ghost" id="showEnvironmentalMap180" '+(!hasMap?'disabled':'')+'>Mostrar dados no mapa</button>'+
        '<button class="btn ghost" id="refreshEnvironment180">Atualizar análise</button>'+
        '<button class="btn ghost" id="openMapBiomasProperty180">Abrir imóvel no MapBiomas</button>'+
      '</div>'+
      (!hasProperty180()?'<div class="environment-note180">Para gerar laudos, vincule o CAR a um imóvel salvo. O perfil pode ser consultado normalmente de forma avulsa.</div>':'')+
      evidenceMatrix180(r)+
      '<div class="environment-alerts180">'+(alerts.length?alerts.map((a,i)=>alert180(a,i)).join(''):'<div class="environment-empty180"><strong>Nenhum alerta MapBiomas retornado.</strong><br><small>Isso não é certificado de regularidade; o perfil acima continua trazendo informações ambientais do imóvel.</small></div>')+'</div>'+
      (warnings.length?'<details class="environment-warnings180"><summary>Limitações e ocorrências da consulta ('+warnings.length+')</summary>'+warnings.map(w=>'<div>• '+esc180(w)+'</div>').join('')+'</details>':'')+
      '<div class="environment-legal180"><strong>Interpretação técnica:</strong> '+esc180(r.interpretation||'Triagem auxiliar baseada em fontes públicas automáticas.')+'</div>';

    g('exportEnvironmentalReport180').onclick=exportReport180;
    g('exportEnvironmentalJSON180').onclick=exportJSON180;
    g('showEnvironmentalMap180').onclick=()=>showMap180(alerts,p);
    g('refreshEnvironment180').onclick=()=>load180(true);
    g('openMapBiomasProperty180').onclick=()=>openExternal('https://plataforma.alerta.mapbiomas.org/imovel/'+encodeURIComponent(r.car||''));
    panel.querySelectorAll('[data-alert-report180]').forEach(b=>b.onclick=()=>exportAlert180(b.dataset.alertReport180));
    panel.querySelectorAll('[data-alert-open180]').forEach(b=>b.onclick=()=>openExternal(b.dataset.alertOpen180));
    panel.querySelectorAll('[data-alert-map180]').forEach(b=>b.onclick=()=>showMap180([alerts[Number(b.dataset.alertMap180)]],{}));
  }

  function metric180(label,value,small){return '<div><span>'+esc180(label)+'</span><strong>'+esc180(String(value))+'</strong><small>'+esc180(small||'')+'</small></div>'}

  function profile180(r){
    const p=r.profile||{},classes=arr180(p.land_cover_classes);
    const biomeText=p.biome_available?(p.biome||'Identificado'):'Indisponível';
    const coverText=p.land_cover_available?(p.dominant_land_cover||'Classificado'):'Indisponível';
    const coverDetail=p.land_cover_available
      ?'ESA WorldCover '+(p.land_cover_year||2021)+' • '+(p.land_cover_samples||0)+' amostras dentro do CAR'
      :'ESA WorldCover não respondeu';

    let html='<section class="environment-profile180"><div class="environment-profile-head180"><div><span class="eyebrow">ETAPA 1 • PERFIL TERRITORIAL</span><h4>Bioma e cobertura do solo</h4></div><small>'+esc180(date180(r.generated_at))+'</small></div>'+
      '<div class="environment-profile-grid180 environment-profile-grid-stage1">'+
        profileCard180('🌿','Bioma',biomeText,p.biome_available?(p.biome_source||'MapBiomas Alerta'):'fonte não respondeu')+
        profileCard180('🛰️','Cobertura dominante',coverText,coverDetail)+
      '</div>'+
      fireCard180(p.fire||{})+
      landCoverDetails180(p,classes)+
      '<div class="environment-profile-note180">A cobertura do solo é uma estimativa amostral baseada no ESA WorldCover 2021 (10 m). Ela descreve a cobertura observada pelo produto de sensoriamento remoto e não substitui levantamento de campo, cadastro ambiental ou identificação da cultura atual.</div>'+
      '</section>';
    return html;
  }

  function profileCard180(icon,label,value,detail){return '<div class="environment-profile-card180"><div class="environment-profile-icon180">'+icon+'</div><div><span>'+esc180(label)+'</span><strong>'+esc180(value)+'</strong><small>'+esc180(detail||'')+'</small></div></div>'}

  function fireCard180(f){
    const status=String(f?.status||'consulta_nao_realizada');
    let value='Consulta não realizada',detail=f?.warning||'Programa Queimadas/INPE';
    if(status==='ocorrencia_encontrada'){
      const n=Number(f?.feature_count||0);
      value=n+' foco'+(n===1?' encontrado':'s encontrados');
      const parts=[];
      if(f?.last_detected_at)parts.push('Último '+date180(f.last_detected_at));
      if(arr180(f?.satellites).length)parts.push(arr180(f.satellites).slice(0,2).join(', '));
      detail=parts.join(' • ')||f?.window_label||'Programa Queimadas/INPE';
    }else if(status==='sem_ocorrencia'){
      value='0 focos encontrados';
      detail=f?.window_label||'Programa Queimadas/INPE consultado';
    }else if(status==='base_indisponivel'){
      value='Base indisponível';
      detail=f?.warning||'Programa Queimadas/INPE não respondeu';
    }
    return '<div class="environment-risk-title180"><strong>Focos de calor</strong><span>Programa Queimadas/INPE</span></div>'+
      '<div class="environment-profile-grid180 environment-profile-grid-stage1">'+
      profileCard180('🔥','Focos de calor',value,detail)+'</div>';
  }

  function landCoverDetails180(p,classes){
    if(!p?.land_cover_available||!classes.length)return '';
    return '<details class="environment-profile-details180" open><summary>Distribuição estimada da cobertura do solo</summary><div class="landcover-list190">'+classes.map(x=>{
      const pct=Math.max(0,Math.min(100,Number(x.percent||0)));
      return '<div class="landcover-row190"><div class="landcover-row-head190"><b>'+esc180(x.label||('Classe '+x.code))+'</b><span>'+n180(x.area_ha,2)+' ha • '+n180(pct,1)+'%</span></div><div class="landcover-track190"><div style="width:'+pct.toFixed(2)+'%"></div></div></div>';
    }).join('')+'</div></details>';
  }

  function evidenceMatrix180(r){
    const env=r.environment||{},mb=r.mapbiomas||{},p=r.profile||{};
    const row=(icon,name,status,detail,cls='')=>'<div class="environment-source180 '+cls+'"><span>'+icon+'</span><div><b>'+esc180(name)+'</b><small>'+esc180(detail||'')+'</small></div><strong>'+esc180(status)+'</strong></div>';
    const rows=[
      row('🌿','Bioma',p.biome_available?'Identificado':'Indisponível',p.biome_available?(p.biome+' • '+(p.biome_source||'MapBiomas')):'fonte não respondeu',!p.biome_available?'warn':''),
      row('🛰️','ESA WorldCover 2021',p.land_cover_available?'Consultado':'Indisponível',p.land_cover_available?((p.land_cover_samples||0)+' amostras • 10 m'):'serviço público não respondeu',!p.land_cover_available?'warn':''),
      row('🛰️','MapBiomas Alerta',mb.connected?(mb.total_alerts||0)+' alerta(s)':'Não conectado',mb.message||'API V2'),
      row('⛔','IBAMA / PAMGIA',env.ibama_checked?(env.ibama_embargo_count||0)+' ocorrência(s)':'Indisponível','embargos com interseção no CAR',env.ibama_embargo_count?'warn':''),
      row('🪶','FUNAI',env.funai_checked?(env.indigenous_count||0)+' ocorrência(s)':'Indisponível','Terras Indígenas com interseção no CAR',env.indigenous_count?'warn':''),
      row('🏞️','ICMBio',env.icmbio_checked?(env.federal_uc_count||0)+' ocorrência(s)':'Indisponível','Unidades de Conservação federais',env.federal_uc_count?'warn':''),
      row('📋','MMA / MCR-PRODES',env.mcr_checked?(env.mcr_listed?'LISTADO':'Não listado'):'Indisponível','lista pública vinculada ao Manual de Crédito Rural',env.mcr_listed?'warn':'')
    ];
    return '<details class="environment-matrix180"><summary>Matriz de fontes automáticas ('+rows.length+')</summary><div>'+rows.join('')+'</div></details>';
  }

  function alert180(a,i){
    const high=a.attention_level==='Alta prioridade de conferência',overlap=[];
    if(Number(a.mapbiomas_permanent_protected_area_ha)>0)overlap.push('APP/MapBiomas '+n180(a.mapbiomas_permanent_protected_area_ha,2)+' ha');
    if(Number(a.mapbiomas_legal_reserve_area_ha)>0)overlap.push('RL/MapBiomas '+n180(a.mapbiomas_legal_reserve_area_ha,2)+' ha');
    if(Number(a.ibama_overlap_ha)>0)overlap.push('IBAMA '+n180(a.ibama_overlap_ha,2)+' ha');
    if(Number(a.indigenous_overlap_ha)>0)overlap.push('TI '+n180(a.indigenous_overlap_ha,2)+' ha');
    if(Number(a.federal_uc_overlap_ha)>0)overlap.push('UC '+n180(a.federal_uc_overlap_ha,2)+' ha');
    return '<article class="environment-alert180 '+(high?'high':'')+'">'+
      '<div class="environment-alert-head180"><div><strong>Alerta '+esc180(a.alert_code||'—')+'</strong><small>'+date180(a.detected_at)+' • '+n180(a.area_ha,2)+' ha • '+esc180((a.sources||[]).join(', ')||'MapBiomas Alerta')+'</small></div><span class="status-badge '+(high?'warning':'info')+'">'+esc180(a.attention_level||'Conferir')+'</span></div>'+
      '<div class="environment-alert-grid180">'+
        metric180('Dentro do CAR',n180(a.alert_area_in_car_ha,2)+' ha',n180(a.alert_pct_of_car,2)+'% do imóvel')+
        metric180('APP / MapBiomas',n180(a.mapbiomas_permanent_protected_area_ha,2)+' ha','cruzamento informado pela API')+
        metric180('Reserva Legal / MapBiomas',n180(a.mapbiomas_legal_reserve_area_ha,2)+' ha','cruzamento informado pela API')+
        metric180('Imagem antes',date180(a.image_before_at),'data informada')+
        metric180('Imagem depois',date180(a.image_after_at),'data informada')+
        metric180('Status',a.status_name||'Publicado',date180(a.published_at))+
      '</div>'+
      (overlap.length?'<div class="environment-tags180">'+overlap.map(x=>'<span>'+esc180(x)+'</span>').join('')+'</div>':'')+
      ((a.attention_reasons||[]).length?'<div class="environment-reasons180">'+a.attention_reasons.map(x=>'<div>• '+esc180(x)+'</div>').join('')+'</div>':'')+
      '<div class="environment-actions180"><button class="btn ghost" data-alert-map180="'+i+'" '+(!a.geometry_geojson?'disabled':'')+'>Ver no mapa</button><button class="btn ghost" data-alert-open180="'+esc180(a.report_url||'')+'" '+(!a.report_url?'disabled':'')+'>Abrir MapBiomas</button><button class="btn primary" data-alert-report180="'+esc180(a.alert_code||'')+'" '+(!hasProperty180()?'disabled':'')+'>Gerar laudo deste alerta</button></div>'+
      '</article>';
  }

  async function exportReport180(){try{const p=await api180().ExportEnvironmentalTechnicalReport(state.selectedProperty?.id||0,false);toast('Laudo ambiental salvo em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}}
  async function exportJSON180(){try{const p=await api180().ExportEnvironmentalEvidenceJSON(state.selectedProperty?.id||0,false);toast('Evidências ambientais salvas em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}}
  async function exportAlert180(code){try{const p=await api180().ExportMapBiomasAlertTechnicalReport(state.selectedProperty?.id||0,code,false);toast('Laudo do alerta salvo em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}}

  function clearMap180(){if(s180.layer&&state?.map){try{state.map.removeLayer(s180.layer);state.layerControl?.removeLayer(s180.layer)}catch(_){}}s180.layer=null}
  function profileHasMap180(){return false}
  function addGeo180(group,raw,label,style,pointStyle){
    if(!raw)return 0;
    try{
      const layer=L.geoJSON(JSON.parse(raw),{style:()=>style||{},pointToLayer:pointStyle?((f,ll)=>L.circleMarker(ll,pointStyle)):undefined});
      layer.bindPopup('<strong>'+esc180(label)+'</strong>');layer.eachLayer(x=>x.addTo(group));return 1;
    }catch(_){return 0}
  }
  function showMap180(alerts){
    if(!state?.map)return;clearMap180();const group=L.featureGroup();let count=0;
    arr180(alerts).forEach(a=>{if(!a?.geometry_geojson)return;count+=addGeo180(group,a.geometry_geojson,'MapBiomas Alerta '+(a.alert_code||''),{weight:3,fillOpacity:.18})});
    if(!count){toast('Nenhuma geometria de alerta disponível para desenhar.',true);return}
    group.addTo(state.map);state.layerControl?.addOverlay(group,'Alertas ambientais');s180.layer=group;
    document.querySelector('[data-car-tab131="map"]')?.click();
    setTimeout(()=>{try{const b=group.getBounds();if(b.isValid())state.map.fitBounds(b.pad(.15),{maxZoom:16})}catch(_){}},100);
  }

  function watch180(){
    const install=()=>{try{attach180()}catch(e){try{console.error('ViaVerdeCAR 1.9.0 ambiental:',e)}catch(_){}}};
    install();const root=g('carTabXRay150')||document.body;new MutationObserver(()=>install()).observe(root,{childList:true,subtree:true});
  }
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',watch180);else watch180();
})();