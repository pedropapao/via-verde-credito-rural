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
    panel.innerHTML='<div class="panel-title"><div><span class="eyebrow">AMBIENTAL • PERFIL AUTOMÁTICO</span><h3>Raio X ambiental do imóvel</h3><p>Relevo, bioma, água, PRODES, DETER, fogo, cobertura do solo e evidências territoriais — consulta automática.</p></div><span class="status-badge info">Preparando</span></div>'+
      '<div class="environment-intro180"><div class="environment-art180">🌎</div><div><strong>Perfil ambiental automático</strong><span>O Via Verde vai cruzar este CAR com serviços públicos e gratuitos. Nenhum download ou importação manual é necessário.</span></div></div>';
    setTimeout(()=>load180(false),0);
  }

  async function load180(force){
    if(s180.loading||!state?.car?.car)return;
    const panel=g('environmentIntel180');if(!panel)return;
    s180.loading=true;
    panel.innerHTML='<div class="panel-title"><div><span class="eyebrow">AMBIENTAL • PERFIL AUTOMÁTICO</span><h3>Montando perfil ambiental…</h3></div><span class="status-badge info">Consultando</span></div>'+
      '<div class="environment-loading180"><div>🌎</div><strong>Cruzando fontes ambientais públicas</strong><span>Relevo, bioma, hidrografia, PRODES, DETER, focos de fogo, WorldCover, MapBiomas, IBAMA, FUNAI, ICMBio e MMA/MCR.</span></div>';
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
    const p=r.profile||{},terrain=p.terrain||{},biome=p.biome||{},hydro=p.hydrology||{},prodes=p.prodes||{},deter=p.deter||{},fire=p.fire||{},lc=p.land_cover||{},near=p.nearby||{};
    const biomeText=biome.available?(biome.dominant||arr180(biome.items).map(x=>x.name).join(', ')):'Indisponível';
    const terrainText=terrain.available?n180(terrain.elevation_mean_m,0)+' m':'Indisponível';
    const slopeDetail=terrain.available?'declive médio '+n180(terrain.mean_slope_pct,1)+'% • relevo '+n180(terrain.relief_m,0)+' m':'fonte não respondeu';
    const hydroText=hydro.available?(hydro.river_reach_count||0)+' trecho(s)':'Indisponível';
    const hydroDetail=hydro.available?((hydro.named_rivers||[]).slice(0,2).join(', ')||((hydro.water_body_count||0)+' massa(s) d’água')):'ANA/SNIRH';
    const prodesText=prodes.available?n180(prodes.area_in_car_ha,2)+' ha':'Indisponível';
    const prodesDetail=prodes.available?(prodes.feature_count||0)+' polígono(s)'+(prodes.latest_year?' • último '+prodes.latest_year:''):'INPE';
    const deterText=deter.applicable===false?'Não se aplica':(deter.available?(deter.feature_count||0)+' aviso(s)':'Indisponível');
    const deterDetail=deter.applicable===false?'fora da cobertura pública operacional':(deter.latest_date?'último '+date180(deter.latest_date):'últimos 12 meses');
    const fireText=fire.available?(fire.feature_count||0)+' foco(s)':'Indisponível';
    const fireDetail=fire.available?(fire.last_detected_at?'último '+date180(fire.last_detected_at):fire.window_label||'camada recente'):'INPE Queimadas';
    const lcText=lc.available?(lc.dominant_class||'Classificado'):'Indisponível';
    const lcDetail=lc.available?n180(lc.dominant_pct,1)+'% das amostras • WorldCover '+(lc.reference_year||2021):'ESA WorldCover';
    const nearest=nearestText180(near);

    let html='<section class="environment-profile180"><div class="environment-profile-head180"><div><span class="eyebrow">PERFIL AMBIENTAL AUTOMÁTICO</span><h4>Características do imóvel</h4></div><small>'+esc180(date180(p.checked_at||r.generated_at))+'</small></div>'+
      '<div class="environment-profile-grid180">'+
        profileCard180('🌿','Bioma',biomeText,biome.available?arr180(biome.items).map(x=>x.name+' '+n180(x.car_pct,1)+'%').join(' • '):'IBGE')+
        profileCard180('⛰️','Altitude média',terrainText,slopeDetail)+
        profileCard180('💧','Hidrografia',hydroText,hydroDetail)+
        profileCard180('🛰️','PRODES',prodesText,prodesDetail)+
        profileCard180('⚡','DETER',deterText,deterDetail)+
        profileCard180('🔥','Fogo ativo',fireText,fireDetail)+
        profileCard180('🌾','Cobertura dominante',lcText,lcDetail)+
        profileCard180('📍','Contexto próximo',nearest.value,nearest.detail)+
      '</div>'+
      profileDetails180(p)+
      '</section>';
    return html;
  }

  function profileCard180(icon,label,value,detail){return '<div class="environment-profile-card180"><div class="environment-profile-icon180">'+icon+'</div><div><span>'+esc180(label)+'</span><strong>'+esc180(value)+'</strong><small>'+esc180(detail||'')+'</small></div></div>'}

  function nearestText180(n){
    if(!n?.available)return {value:'Indisponível',detail:'fontes territoriais'};
    const opts=[];
    if(n.embargo_checked&&n.embargo_found)opts.push({d:Number(n.nearest_embargo_km||0),t:'IBAMA'});
    if(n.indigenous_checked&&n.indigenous_found)opts.push({d:Number(n.nearest_indigenous_km||0),t:'TI'});
    if(n.uc_checked&&n.uc_found)opts.push({d:Number(n.nearest_uc_km||0),t:'UC'});
    opts.sort((a,b)=>a.d-b.d);
    const checked=[n.embargo_checked?'IBAMA':null,n.indigenous_checked?'FUNAI':null,n.uc_checked?'ICMBio':null].filter(Boolean);
    if(!opts.length){
      if(checked.length===3)return {value:'Nenhuma em '+n180(n.search_radius_km||50,0)+' km',detail:'IBAMA • FUNAI • ICMBio consultados'};
      return {value:'Nenhuma nas fontes consultadas',detail:checked.length?checked.join(' • ')+'; demais indisponíveis':'fontes indisponíveis'};
    }
    return {value:n180(opts[0].d,1)+' km',detail:'mais próxima: '+opts[0].t+(checked.length<3?' • consulta parcial':'')};
  }

  function profileDetails180(p){
    const terrain=p.terrain||{},biome=p.biome||{},h=p.hydrology||{},prodes=p.prodes||{},deter=p.deter||{},fire=p.fire||{},lc=p.land_cover||{},near=p.nearby||{};
    let blocks=[];
    if(terrain.available)blocks.push('<div class="environment-profile-detail180"><b>⛰️ Relevo</b><span>Altitude '+n180(terrain.elevation_min_m,0)+'–'+n180(terrain.elevation_max_m,0)+' m • média '+n180(terrain.elevation_mean_m,0)+' m • declive médio '+n180(terrain.mean_slope_pct,1)+'% • máximo '+n180(terrain.max_slope_pct,1)+'%</span><small>'+esc180(terrain.source||'Terrain Tiles')+' • confiança '+esc180(terrain.confidence||'—')+'</small></div>');
    if(h.available)blocks.push('<div class="environment-profile-detail180"><b>💧 Água</b><span>'+esc180((h.named_rivers||[]).join(', ')||'Nenhum curso nomeado interceptando o CAR')+' • '+(h.water_body_count||0)+' massa(s) d’água • '+n180(h.water_body_area_ha,2)+' ha</span><small>'+(h.nearest_station_name?'Estação telemétrica mais próxima: '+esc180(h.nearest_station_name)+' • '+n180(h.nearest_station_km,1)+' km':'ANA/SNIRH')+'</small></div>');
    if(prodes.available)blocks.push('<div class="environment-profile-detail180"><b>🛰️ Histórico PRODES</b><span>'+n180(prodes.area_in_car_ha,2)+' ha em '+(prodes.feature_count||0)+' polígono(s) interceptando o CAR</span><small>'+years180(prodes.years)+'</small></div>');
    if(deter.applicable!==false&&deter.available)blocks.push('<div class="environment-profile-detail180"><b>⚡ DETER — últimos 12 meses</b><span>'+(deter.feature_count||0)+' aviso(s) • '+n180(deter.area_in_car_ha,2)+' ha</span><small>'+(deter.latest_date?'Último aviso '+date180(deter.latest_date):'Nenhum aviso recente localizado')+'</small></div>');
    if(fire.available)blocks.push('<div class="environment-profile-detail180"><b>🔥 Focos de fogo</b><span>'+(fire.feature_count||0)+' foco(s) retornado(s) dentro do CAR'+(Number(fire.max_frp)>0?' • FRP máx. '+n180(fire.max_frp,1):'')+'</span><small>'+esc180(fire.window_label||'camada pública INPE')+'</small></div>');
    if(lc.available)blocks.push('<div class="environment-profile-detail180 environment-landcover180"><b>🌾 Cobertura do solo — amostragem WorldCover 2021</b><div>'+arr180(lc.classes).slice(0,6).map(x=>'<span><i style="width:'+Math.max(2,Math.min(100,Number(x.sample_pct||0)))+'%"></i><em>'+esc180(x.label)+' '+n180(x.sample_pct,1)+'%</em></span>').join('')+'</div><small>'+esc180(lc.warning||'Percentuais das amostras cartográficas.')+'</small></div>');
    if(near.available)blocks.push('<div class="environment-profile-detail180"><b>📍 Proximidade em até '+n180(near.search_radius_km||50,0)+' km</b><span>'+nearbyParts180(near).join(' • ')+'</span><small>Distâncias aproximadas do centro do CAR à geometria pública mais próxima.</small></div>');
    return blocks.length?'<details class="environment-profile-details180" open><summary>Detalhes do perfil automático</summary><div>'+blocks.join('')+'</div></details>':'';
  }

  function years180(years){const x=arr180(years).slice(0,6);return x.length?x.map(y=>y.year+': '+n180(y.area_ha,2)+' ha').join(' • '):'Sem ano identificado nos polígonos retornados'}
  function nearbyParts180(n){
    const a=[],radius=n180(n.search_radius_km||50,0);
    a.push(!n.embargo_checked?'IBAMA: indisponível':(n.embargo_found?'IBAMA '+n180(n.nearest_embargo_km,1)+' km':'IBAMA: nenhuma em '+radius+' km'));
    a.push(!n.indigenous_checked?'TI: indisponível':(n.indigenous_found?'TI '+n180(n.nearest_indigenous_km,1)+' km':'TI: nenhuma em '+radius+' km'));
    a.push(!n.uc_checked?'UC: indisponível':(n.uc_found?'UC '+n180(n.nearest_uc_km,1)+' km':'UC: nenhuma em '+radius+' km'));
    return a;
  }

  function evidenceMatrix180(r){
    const env=r.environment||{},mb=r.mapbiomas||{},p=r.profile||{};
    const row=(icon,name,status,detail,cls='')=>'<div class="environment-source180 '+cls+'"><span>'+icon+'</span><div><b>'+esc180(name)+'</b><small>'+esc180(detail||'')+'</small></div><strong>'+esc180(status)+'</strong></div>';
    let rows=[
      row('🛰️','MapBiomas Alerta',mb.connected?(mb.total_alerts||0)+' alerta(s)':'Não conectado',mb.message||'API V2'),
      row('⛔','IBAMA / PAMGIA',env.ibama_checked?(env.ibama_embargo_count||0)+' ocorrência(s)':'Indisponível','embargos com interseção no CAR',env.ibama_embargo_count?'warn':''),
      row('🪶','FUNAI',env.funai_checked?(env.indigenous_count||0)+' ocorrência(s)':'Indisponível','Terras Indígenas com interseção no CAR',env.indigenous_count?'warn':''),
      row('🏞️','ICMBio',env.icmbio_checked?(env.federal_uc_count||0)+' ocorrência(s)':'Indisponível','Unidades de Conservação federais',env.federal_uc_count?'warn':''),
      row('📋','MMA / MCR-PRODES',env.mcr_checked?(env.mcr_listed?'LISTADO':'Não listado'):'Indisponível','lista pública vinculada ao Manual de Crédito Rural',env.mcr_listed?'warn':'')
    ];
    const icons={terrain:'⛰️',biome:'🌿',hydrology:'💧',prodes:'🛰️',deter:'⚡',fire:'🔥',worldcover:'🌾',nearby:'📍'};
    arr180(p.sources).forEach(s=>rows.push(row(icons[s.key]||'•',s.label,statusText180(s),s.detail||s.source_url||'',!s.available&&s.applicable!==false?'warn':'')));
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
  function profileHasMap180(p){return !!(p?.prodes?.geojson||p?.deter?.geojson||p?.fire?.geojson||p?.hydrology?.river_geojson||p?.hydrology?.water_geojson||p?.biome?.geojson)}
  function addGeo180(group,raw,label,style,pointStyle){
    if(!raw)return 0;
    try{
      const layer=L.geoJSON(JSON.parse(raw),{style:()=>style||{},pointToLayer:pointStyle?((f,ll)=>L.circleMarker(ll,pointStyle)):undefined});
      layer.bindPopup('<strong>'+esc180(label)+'</strong>');layer.eachLayer(x=>x.addTo(group));return 1;
    }catch(_){return 0}
  }
  function showMap180(alerts,p){
    if(!state?.map)return;clearMap180();const group=L.featureGroup();let count=0;
    arr180(alerts).forEach(a=>{if(!a?.geometry_geojson)return;count+=addGeo180(group,a.geometry_geojson,'MapBiomas Alerta '+(a.alert_code||''),{color:'#b33a2b',weight:3,fillOpacity:.18})});
    count+=addGeo180(group,p?.prodes?.geojson,'INPE PRODES',{color:'#8b4513',weight:2,fillOpacity:.20});
    count+=addGeo180(group,p?.deter?.geojson,'INPE DETER',{color:'#d97706',weight:2,fillOpacity:.25});
    count+=addGeo180(group,p?.hydrology?.river_geojson,'ANA — Hidrografia',{color:'#2563eb',weight:2,opacity:.8});
    count+=addGeo180(group,p?.hydrology?.water_geojson,'ANA — Massa d’água',{color:'#0ea5e9',weight:1,fillOpacity:.25});
    count+=addGeo180(group,p?.biome?.geojson,'IBGE — Bioma',{color:'#6b7280',weight:1,fillOpacity:0});
    count+=addGeo180(group,p?.fire?.geojson,'INPE — Foco de fogo',null,{radius:5,weight:2,fillOpacity:.75,color:'#b91c1c'});
    if(!count){toast('Nenhuma geometria ambiental disponível para desenhar.',true);return}
    group.addTo(state.map);state.layerControl?.addOverlay(group,'Perfil ambiental automático');s180.layer=group;
    document.querySelector('[data-car-tab131="map"]')?.click();
    setTimeout(()=>{try{const b=group.getBounds();if(b.isValid())state.map.fitBounds(b.pad(.15),{maxZoom:16})}catch(_){}},100);
  }

  function watch180(){
    const install=()=>{try{attach180()}catch(e){try{console.error('ViaVerdeCAR 1.9.0 ambiental:',e)}catch(_){}}};
    install();const root=g('carTabXRay150')||document.body;new MutationObserver(()=>install()).observe(root,{childList:true,subtree:true});
  }
  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',watch180);else watch180();
})();