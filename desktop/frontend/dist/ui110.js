/* ViaVerdeCAR 1.1.0 — automação total do mapa, acesso e área do projeto. */
(function(){
  const q=id=>document.getElementById(id);
  const s110={route:null,routeLayer:null};

  function prepare110(){
    const areaPanel=document.querySelector('.project-areas-panel');
    if(areaPanel&&!q('autoAreaBox110')){
      const box=document.createElement('div');box.id='autoAreaBox110';box.className='auto-area-box110';
      box.innerHTML='<div class="auto-area-grid110"><label>Nome da área<input id="autoAreaName110" placeholder="Ex.: Pastagem RenovAgro"></label><label>Finalidade<input id="autoAreaPurpose110" placeholder="Ex.: recuperação de pastagem"></label><label>Área necessária (ha)<input id="autoAreaTarget110" type="number" min="0.01" step="0.01" placeholder="5,00"></label><button class="btn primary" id="generateAutoArea110">Delimitar automaticamente</button></div><div class="auto-area-hint110">O Via Verde procura uma gleba compacta totalmente dentro do CAR. Se houver acesso automático, começa próximo à entrada estimada; caso contrário, usa a região central do imóvel. A área manual continua disponível como alternativa.</div><button class="text-btn manual-area-toggle110" id="toggleManualArea110">Mostrar desenho/importação manual</button>';
      const form=areaPanel.querySelector('.compact-form109');form?.before(box);
      const actions=areaPanel.querySelector('.area-actions109');if(actions){actions.classList.add('manual-area-actions110','hidden-manual110')}
      const legacyForm=areaPanel.querySelector('.compact-form109');if(legacyForm)legacyForm.classList.add('hidden-manual110');
      q('generateAutoArea110').onclick=generateAutoArea110;
      q('toggleManualArea110').onclick=()=>{const hidden=actions?.classList.toggle('hidden-manual110');legacyForm?.classList.toggle('hidden-manual110',hidden);q('toggleManualArea110').textContent=hidden?'Mostrar desenho/importação manual':'Ocultar opções manuais'};
      const title=areaPanel.querySelector('h3');if(title)title.textContent='Delimitação automática de glebas';
      const p=areaPanel.querySelector('.panel-title p');if(p)p.textContent='Informe apenas a área necessária. O aplicativo cria a delimitação dentro do CAR, calcula as métricas e salva o KML.';
    }

    const routePanel=document.querySelector('.access-route-panel');
    if(routePanel&&!q('autoRouteCard110')){
      const title=routePanel.querySelector('h3');if(title)title.textContent='Roteiro automático até o imóvel';
      const p=routePanel.querySelector('.panel-title p');if(p)p.textContent='O aplicativo identifica uma cidade/núcleo urbano de referência, procura o melhor ponto de acesso viário no perímetro e calcula a rota automaticamente.';
      routePanel.querySelectorAll('.route-actions109').forEach((el,i)=>{if(i===0)el.dataset.legacyRoute='1'});
      const card=document.createElement('div');card.id='autoRouteCard110';card.className='auto-route-card110';
      card.innerHTML='<div class="empty-state">Consulte um CAR para calcular automaticamente o acesso.</div>';
      routePanel.querySelector('.panel-title')?.after(card);
      const manualButtons=routePanel.querySelectorAll('[data-route-point],#saveRoute109');
      manualButtons.forEach(b=>b.closest('.route-actions109')?.classList.add('hidden'));
      const copy=q('copyRoute109'),exp=q('exportRoute109'),maps=q('openRouteMaps109');if(copy)copy.classList.add('hidden');if(exp)exp.classList.add('hidden');if(maps)maps.classList.add('hidden');
    }
  }

  async function generateAutoArea110(){
    if(!state.selectedProperty){toast('Selecione um imóvel salvo para gerar a gleba.',true);return}
    const target=Number(q('autoAreaTarget110').value)||0;if(target<=0){toast('Informe a área necessária em hectares.',true);return}
    const btn=q('generateAutoArea110');btn.disabled=true;btn.textContent='Delimitando...';
    try{
      const x=await api().GenerateAutomaticProjectArea(state.selectedProperty.id,q('autoAreaName110').value,q('autoAreaPurpose110').value,target);
      toast('Gleba automática criada: '+fmt(x.area_ha,4)+' ha.');
      await selectCarProperty(state.selectedProperty.id);
      setTimeout(()=>{const layer=[...document.querySelectorAll('[data-area-fit]')].find(b=>Number(b.dataset.areaFit)===x.id);layer?.click()},250);
    }catch(e){toast(String(e),true)}
    finally{btn.disabled=false;btn.textContent='Delimitar automaticamente'}
  }

  function clearRouteLayer110(){if(s110.routeLayer){try{state.map?.removeLayer(s110.routeLayer);state.layerControl?.removeLayer(s110.routeLayer)}catch(e){}s110.routeLayer=null}}
  function drawRoute110(r){
    clearRouteLayer110();if(!state.map||!r?.route_geojson)return;
    try{
      const obj=JSON.parse(r.route_geojson);const layer=L.geoJSON(obj,{style:{color:'#2563a6',weight:4,opacity:.85}});
      layer.addTo(state.map);state.layerControl?.addOverlay(layer,'Rota automática');s110.routeLayer=layer;
      if(r.reference_lat||r.reference_lon)L.circleMarker([r.reference_lat,r.reference_lon],{radius:6,weight:2}).addTo(layer).bindTooltip('Cidade de referência');
      if(r.entrance_lat||r.entrance_lon)L.circleMarker([r.entrance_lat,r.entrance_lon],{radius:7,weight:2}).addTo(layer).bindTooltip('Acesso estimado');
    }catch(e){}
  }
  function coord(lat,lon){return(lat||lon)?Number(lat).toFixed(6)+', '+Number(lon).toFixed(6):'—'}
  function renderAutoRoute110(r){
    s110.route=r||null;const box=q('autoRouteCard110');if(!box)return;
    if(!r||!r.entrance_lat){box.innerHTML='<div class="empty-state">Rota automática ainda não disponível. Consulte novamente o CAR ou use Recalcular quando houver conexão.</div>';clearRouteLayer110();return}
    const steps=(r.steps||[]).map((x,i)=>'<div class="auto-route-step110"><b>'+(i+1)+'</b><span>'+esc(x.instruction)+(x.road?' — <strong>'+esc(x.road)+'</strong>':'')+'</span><small>'+(x.distance_km?fmt(x.distance_km,2)+' km':'')+'</small></div>').join('');
    box.innerHTML='<div class="auto-route-head110"><div><strong>'+esc(r.reference_label||'Cidade de referência')+' → imóvel</strong><span>Melhor acesso viário estimado automaticamente</span></div><span class="status-badge ok">Automático</span></div><div class="auto-route-metrics110"><div><span>Cidade / referência</span><strong>'+coord(r.reference_lat,r.reference_lon)+'</strong></div><div><span>Acesso estimado</span><strong>'+coord(r.entrance_lat,r.entrance_lon)+'</strong></div><div><span>Distância pela rota</span><strong>'+fmt(r.route_distance_km,2)+' km</strong></div><div><span>Tempo estimado</span><strong>'+fmt(r.route_duration_min,0)+' min</strong></div></div><div class="auto-route-text110">'+esc(r.text||'')+'</div><div class="auto-route-actions110"><button class="btn primary" id="openAutoMaps110">Abrir rota no Google Maps</button><button class="btn ghost" id="copyAutoRoute110">Copiar roteiro completo</button><button class="btn ghost" id="exportAutoRoute110">Exportar TXT</button><button class="btn ghost" id="regenAutoRoute110">Recalcular automaticamente</button></div>'+(steps?'<div class="auto-route-steps110">'+steps+'</div>':'')+'<div class="route-source110">'+esc(r.route_source||'')+' • A rota e o acesso são estimativas sobre base viária pública; confirme porteira/estrada particular em campo.</div>';
    q('openAutoMaps110').onclick=()=>openExternal(r.google_maps_url);q('copyAutoRoute110').onclick=()=>copyText(fullRouteText110(r),'Roteiro copiado.');q('exportAutoRoute110').onclick=exportAutoRoute110;q('regenAutoRoute110').onclick=regenAutoRoute110;drawRoute110(r);
  }
  function fullRouteText110(r){let t=r.text||'';if((r.steps||[]).length)t+='\n\nINSTRUÇÕES DA ROTA\n'+r.steps.map((x,i)=>(i+1)+'. '+x.instruction+(x.road?' — '+x.road:'')+(x.distance_km?' ('+fmt(x.distance_km,2)+' km)':'')).join('\n');return t}
  async function exportAutoRoute110(){if(!state.selectedProperty){toast('Selecione o imóvel salvo.',true);return}try{const p=await api().ExportAccessRouteTXT(state.selectedProperty.id);toast('Roteiro salvo em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}}
  async function regenAutoRoute110(){if(!state.selectedProperty){toast('Selecione o imóvel salvo para recalcular.',true);return}const b=q('regenAutoRoute110');b.disabled=true;b.textContent='Calculando...';try{const r=await api().RegenerateAutomaticAccessRoute(state.selectedProperty.id);renderAutoRoute110(r);toast('Rota recalculada.')}catch(e){toast(String(e),true)}finally{b.disabled=false;b.textContent='Recalcular automaticamente'}}

  async function loadAutoRouteForProperty110(){
    if(!state.selectedProperty){renderAutoRoute110(state.car?.auto_route);return}
    try{const r=await api().GetAccessRoute(state.selectedProperty.id);renderAutoRoute110(r)}catch(e){renderAutoRoute110(state.car?.auto_route)}
  }

  async function restoreTemporarySession110(){
    for(let i=0;i<40&&!api();i++)await new Promise(r=>setTimeout(r,100));
    if(!api()||state.car)return;
    try{
      const c=await api().GetLastCARSession();const r=c?.result;if(!r?.car)return;
      const match=(state.properties||[]).find(p=>String(p.car_number||'').toUpperCase()===String(r.car).toUpperCase());
      if(match){q('carPropertySelect').value=String(match.id);await selectCarProperty(match.id)}
      else{state.car=r;q('carInput').value=r.car;renderCAR(r);if(r.geojson)drawGeoJSON('car',r.geojson);renderAutoRoute110(r.auto_route)}
      const title=q('professionalTitle');if(title&&!q('sessionBadge110')){const b=document.createElement('span');b.id='sessionBadge110';b.className='session-badge110';b.textContent='Mapa temporário restaurado';title.after(b)}
    }catch(e){}
  }

  function installOverrides110(){
    const oldSelect=selectCarProperty,oldRenderCAR=renderCAR,oldReset=resetCARWorkspace;
    selectCarProperty=async function(id){await oldSelect(id);await loadAutoRouteForProperty110()};
    renderCAR=function(r){oldRenderCAR(r);if(r?.auto_route?.entrance_lat)renderAutoRoute110(r.auto_route);else if(state.selectedProperty)setTimeout(loadAutoRouteForProperty110,0)};
    resetCARWorkspace=function(){oldReset();clearRouteLayer110();renderAutoRoute110(null);q('sessionBadge110')?.remove()};
  }

  document.addEventListener('DOMContentLoaded',()=>{prepare110();installOverrides110();setTimeout(restoreTemporarySession110,1400)});
})();
