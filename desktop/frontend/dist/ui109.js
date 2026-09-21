/* ViaVerdeCAR 1.0.9 — carteira, conferência, glebas, roteiro e Temas SICAR resilientes. */
(function(){
  const g=id=>document.getElementById(id);
  const s109={areas:[],areaLayers:{},drawMode:'',drawPoints:[],drawLayer:null,route:null,routeMarkers:[]};

  function prepare109(){
    const stats=document.querySelector('#view-dashboard .stats-grid');
    if(stats&&!g('portfolioPanel109')){
      const panel=document.createElement('article');panel.id='portfolioPanel109';panel.className='panel portfolio-panel';
      panel.innerHTML='<div class="portfolio-head"><div><span class="eyebrow">MONITORAMENTO DA CARTEIRA</span><h3>Verificar todos os CARs cadastrados</h3><p>Rápida: cadastro/geometria e análise completa apenas quando houver mudança. Completa: refaz também as triagens ambientais de todos os imóveis.</p></div><div class="portfolio-actions"><button class="btn ghost" id="portfolioQuick109">Verificação rápida</button><button class="btn primary" id="portfolioFull109">Verificação completa</button></div></div><div id="portfolioSummary109" class="portfolio-summary hidden"></div><div id="portfolioResults109" class="portfolio-results"></div>';
      stats.after(panel);
      g('portfolioQuick109').onclick=()=>auditPortfolio109(false);
      g('portfolioFull109').onclick=()=>auditPortfolio109(true);
    }

    const conference=document.querySelector('.conference-panel');
    if(conference&&!g('conferenceSummary109')){
      const box=document.createElement('div');box.id='conferenceSummary109';box.innerHTML='<div class="empty-state">Selecione um imóvel salvo para montar a conferência consolidada.</div>';
      const title=conference.querySelector('.panel-title');title?.after(box);
    }

    const themes=document.querySelector('.themes-panel');
    if(themes&&!g('themeTools109')){
      const tools=document.createElement('div');tools.id='themeTools109';tools.className='theme-tools109';
      tools.innerHTML='<select id="themeSelect109"><option value="APP">APP</option><option value="RESERVA_LEGAL">Reserva Legal</option><option value="VEGETACAO_NATIVA">Vegetação Nativa</option><option value="AREA_CONSOLIDADA">Área Consolidada</option><option value="USO_RESTRITO">Uso Restrito</option><option value="SERVIDAO_ADMINISTRATIVA">Servidão Administrativa</option></select><button class="btn ghost" id="importTheme109">Importar pacote oficial ZIP</button><span id="themeCache109" class="theme-cache109"></span>';
      themes.querySelector('.panel-title>div')?.appendChild(tools);
      g('importTheme109').onclick=importTheme109;
    }

    const bottom=document.querySelector('.professional-bottom');
    if(bottom&&!g('feature109Grid')){
      const grid=document.createElement('div');grid.id='feature109Grid';grid.className='feature109-grid';
      grid.innerHTML=`
      <article class="panel project-areas-panel">
        <div class="panel-title"><div><span class="eyebrow">ÁREAS DO PROJETO</span><h3>Glebas e áreas financiadas</h3><p>Desenhe no mapa ou importe um KML. O Via Verde calcula área, perímetro, centro e quanto da gleba está dentro do CAR.</p></div></div>
        <div class="compact-form109">
          <label>Nome da área<input id="areaName109" placeholder="Ex.: RenovAgro 25 ha"></label>
          <label>Finalidade<input id="areaPurpose109" placeholder="Ex.: recuperação de pastagem"></label>
        </div>
        <div class="area-actions109"><button class="btn primary" id="drawArea109">Desenhar no mapa</button><button class="btn ghost" id="saveDraw109" disabled>Salvar desenho</button><button class="btn ghost" id="cancelDraw109" disabled>Cancelar</button><button class="btn ghost" id="importArea109">Importar KML</button></div>
        <div id="drawingStatus109" class="drawing-status109 hidden"></div>
        <div id="areaList109" class="area-list109"><div class="empty-state">Selecione um imóvel.</div></div>
      </article>
      <article class="panel access-route-panel">
        <div class="panel-title"><div><span class="eyebrow">ROTEIRO DE ACESSO</span><h3>Entrada, sede e referência</h3><p>Marque os pontos no mapa. Distâncias de acesso podem ser informadas manualmente; quando ausentes, o app usa distância em linha reta e deixa isso explícito.</p></div></div>
        <div class="compact-form109">
          <label class="full">Referência de partida<input id="routeReference109" placeholder="Ex.: Praça central de Jacuí / trevo da rodovia"></label>
          <label>Referência → entrada (km)<input id="routeDist1109" type="number" min="0" step="0.01"></label>
          <label>Entrada → sede (km)<input id="routeDist2109" type="number" min="0" step="0.01"></label>
          <label class="full">Observações<textarea id="routeNotes109" rows="2" placeholder="Estrada, porteira, ponto de referência..."></textarea></label>
        </div>
        <div class="route-actions109"><button class="btn ghost" data-route-point="reference">Marcar referência</button><button class="btn ghost" data-route-point="entrance">Marcar entrada</button><button class="btn ghost" data-route-point="headquarters">Marcar sede</button><button class="btn primary" id="saveRoute109">Salvar roteiro</button></div>
        <div class="route-coords109"><div class="route-point109"><span>Referência</span><strong id="routeRefCoord109">—</strong></div><div class="route-point109"><span>Entrada</span><strong id="routeEntranceCoord109">—</strong></div><div class="route-point109"><span>Sede</span><strong id="routeHQCoord109">—</strong></div></div>
        <div id="routeText109" class="route-text109">Nenhum roteiro salvo.</div>
        <div class="route-actions109" style="margin-top:8px"><button class="btn ghost" id="copyRoute109">Copiar roteiro</button><button class="btn ghost" id="exportRoute109">Exportar TXT</button><button class="btn ghost" id="openRouteMaps109">Abrir no mapa</button></div>
      </article>`;
      bottom.before(grid);
      g('drawArea109').onclick=startAreaDraw109;g('saveDraw109').onclick=saveDrawnArea109;g('cancelDraw109').onclick=cancelDraw109;g('importArea109').onclick=importArea109;
      document.querySelectorAll('[data-route-point]').forEach(b=>b.onclick=()=>startRoutePoint109(b.dataset.routePoint));
      g('saveRoute109').onclick=saveRoute109;g('copyRoute109').onclick=()=>copyText(s109.route?.text||'','Roteiro copiado.');g('exportRoute109').onclick=exportRoute109;g('openRouteMaps109').onclick=()=>openExternal(s109.route?.google_maps_url);
    }
    state.map?.on('click',handleMapClick109);
  }

  async function auditPortfolio109(full){
    const a=g('portfolioQuick109'),b=g('portfolioFull109');a.disabled=b.disabled=true;(full?b:a).textContent=full?'Verificando tudo...':'Verificando...';
    try{
      const r=await api().AuditCARPortfolio(full);renderPortfolio109(r);await Promise.all([loadDashboard(),loadProperties()]);toast(r.checked+' CAR(s) verificados.');
    }catch(e){toast(String(e),true)}finally{a.disabled=b.disabled=false;a.textContent='Verificação rápida';b.textContent='Verificação completa'}
  }
  function renderPortfolio109(r){
    const box=g('portfolioSummary109');box.classList.remove('hidden');box.innerHTML=`<div><span>Verificados</span><strong>${r.checked||0}</strong></div><div><span>Sem alteração</span><strong>${r.unchanged||0}</strong></div><div><span>Alterados</span><strong>${r.changed||0}</strong></div><div><span>Requerem revisão</span><strong>${r.needs_review||0}</strong></div><div><span>Falhas</span><strong>${r.failed||0}</strong></div>`;
    const list=g('portfolioResults109');const items=r.items||[];list.innerHTML=items.length?items.map(x=>`<div class="portfolio-row ${x.error?'failed':x.changed?'changed':''}"><div><strong>${esc(x.client_name)} — ${esc(x.property_name)}</strong><small>${esc(x.car)}</small></div><div><span>${esc((x.changes||[]).join(' • ')||x.status||'Sem alteração material')}</span></div><div class="result">${esc(x.error||x.result)}</div></div>`).join(''):'<div class="empty-state">Nenhum imóvel com CAR cadastrado.</div>';
  }

  async function loadConference109(){
    if(!state.selectedProperty){g('conferenceSummary109').innerHTML='<div class="empty-state">Selecione um imóvel salvo para montar a conferência consolidada.</div>';return}
    try{const r=await api().GetPropertyConference(state.selectedProperty.id);renderConference109(r)}catch(e){g('conferenceSummary109').innerHTML='<div class="empty-state">Conferência ainda não disponível para este imóvel.</div>'}
  }
  function renderConference109(r){
    const actions=(r.manual_actions||[]).map(x=>'<li>'+esc(x)+'</li>').join('');
    g('conferenceSummary109').innerHTML=`<div class="conference-overview109"><div class="conference-state ${r.status}"><span>Resultado consolidado</span><strong>${esc(r.label)}</strong></div><div><span>OK</span><strong>${r.ok_count||0}</strong></div><div><span>Atenção</span><strong>${r.attention_count||0}</strong></div><div><span>Indisponível</span><strong>${r.unavailable_count||0}</strong></div></div>${actions?'<ul class="manual-actions109">'+actions+'</ul>':''}`;
  }

  async function importTheme109(){
    if(!state.selectedProperty){toast('Selecione um imóvel salvo.',true);return}
    try{const r=await api().ImportSICARThemeZIP(state.selectedProperty.id,g('themeSelect109').value);toast(r.message);if(state.car?.car)await lookupCAR()}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}
  }
  function showThemeCache109(themes){
    const metrics=Object.values(themes?.themes||{}),stale=metrics.filter(x=>x.cache_status==='cache_stale'),manual=metrics.filter(x=>x.cache_status==='manual');
    const parts=[];if(stale.length)parts.push(stale.length+' tema(s) usando último pacote válido em cache');if(manual.length)parts.push(manual.length+' tema(s) usando pacote importado manualmente');
    g('themeCache109').textContent=parts.join(' • ');
  }

  async function loadAreas109(){
    clearAreaLayers109();
    if(!state.selectedProperty){s109.areas=[];g('areaList109').innerHTML='<div class="empty-state">Selecione um imóvel.</div>';return}
    try{s109.areas=await api().ListProjectAreas(state.selectedProperty.id)||[];renderAreas109();drawAreas109()}catch(e){s109.areas=[];g('areaList109').innerHTML='<div class="empty-state">Não foi possível carregar as áreas.</div>'}
  }
  function renderAreas109(){
    const box=g('areaList109');box.innerHTML=s109.areas.length?s109.areas.map(x=>{const inside=x.inside_car_pct||0;return `<div class="area-row109"><div><strong>${esc(x.name)}</strong><span>${esc(x.purpose||'Sem finalidade informada')}</span><small>${fmt(x.area_ha,4)} ha • ${fmt(x.perimeter_m/1000,3)} km • <b class="${inside>=98?'inside-ok':'inside-warning'}">${fmt(inside,1)}% dentro do CAR</b></small></div><div class="row-actions"><button class="icon-btn" data-area-fit="${x.id}" title="Enquadrar">⌖</button><button class="icon-btn" data-area-export="${x.id}" title="Exportar KML">KML</button><button class="icon-btn" data-area-delete="${x.id}" title="Excluir">×</button></div></div>`}).join(''):'<div class="empty-state">Nenhuma área de projeto cadastrada.</div>';
    document.querySelectorAll('[data-area-fit]').forEach(b=>b.onclick=()=>fitArea109(Number(b.dataset.areaFit)));document.querySelectorAll('[data-area-export]').forEach(b=>b.onclick=()=>exportArea109(Number(b.dataset.areaExport)));document.querySelectorAll('[data-area-delete]').forEach(b=>b.onclick=()=>deleteArea109(Number(b.dataset.areaDelete)));
  }
  function clearAreaLayers109(){Object.values(s109.areaLayers).forEach(l=>{try{state.map?.removeLayer(l);state.layerControl?.removeLayer(l)}catch(e){}});s109.areaLayers={}}
  function drawAreas109(){if(!state.map)return;s109.areas.forEach((x,i)=>{try{const l=L.geoJSON(JSON.parse(x.geojson),{style:{color:'#7a5a18',weight:2,dashArray:'4 3',fillColor:'#d7b866',fillOpacity:.12}});l.addTo(state.map);state.layerControl?.addOverlay(l,'Área do projeto • '+x.name);s109.areaLayers[x.id]=l}catch(e){}})}
  function fitArea109(id){const l=s109.areaLayers[id];if(l){const b=l.getBounds();if(b.isValid())state.map.fitBounds(b.pad(.15),{maxZoom:18})}}
  async function exportArea109(id){try{const p=await api().ExportProjectAreaKML(id);toast('KML salvo em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}}
  async function deleteArea109(id){if(!confirm('Excluir esta área do projeto?'))return;try{await api().DeleteProjectArea(id);await loadAreas109();await loadConference109();toast('Área excluída.')}catch(e){toast(String(e),true)}}
  function startAreaDraw109(){if(!state.selectedProperty){toast('Selecione um imóvel salvo.',true);return}s109.drawMode='area';s109.drawPoints=[];clearDrawLayer109();setDrawUI109(true,'Clique no mapa para marcar os vértices da gleba. Use pelo menos 3 pontos e depois clique em Salvar desenho.')}
  function cancelDraw109(){s109.drawMode='';s109.drawPoints=[];clearDrawLayer109();setDrawUI109(false,'')}
  function setDrawUI109(active,msg){g('saveDraw109').disabled=!active;g('cancelDraw109').disabled=!active;g('drawingStatus109').classList.toggle('hidden',!active);g('drawingStatus109').textContent=msg;g('map')?.classList.toggle('map-drawing109',active||s109.drawMode.startsWith('route:'))}
  function clearDrawLayer109(){if(s109.drawLayer){try{state.map?.removeLayer(s109.drawLayer)}catch(e){}s109.drawLayer=null}}
  function redrawDraft109(){clearDrawLayer109();if(!state.map||!s109.drawPoints.length)return;const latlngs=s109.drawPoints.map(p=>[p.lat,p.lng]);s109.drawLayer=s109.drawPoints.length>=3?L.polygon(latlngs,{color:'#7a5a18',weight:2,fillOpacity:.12}).addTo(state.map):L.polyline(latlngs,{color:'#7a5a18',weight:2,dashArray:'4 3'}).addTo(state.map);g('drawingStatus109').textContent=s109.drawPoints.length+' ponto(s) marcado(s). '+(s109.drawPoints.length>=3?'Pronto para salvar.':'Marque pelo menos 3 pontos.')}
  async function saveDrawnArea109(){if(s109.drawPoints.length<3){toast('Marque pelo menos 3 pontos.',true);return}const name=g('areaName109').value.trim();if(!name){toast('Informe o nome da área.',true);return}const c=s109.drawPoints.map(p=>[p.lng,p.lat]);c.push(c[0]);const geo=JSON.stringify({type:'Feature',properties:{source:'Via Verde CAR - desenho'},geometry:{type:'Polygon',coordinates:[c]}});try{await api().SaveProjectArea({property_id:state.selectedProperty.id,name,purpose:g('areaPurpose109').value,geojson:geo});cancelDraw109();await loadAreas109();await loadConference109();toast('Área do projeto salva.')}catch(e){toast(String(e),true)}}
  async function importArea109(){if(!state.selectedProperty){toast('Selecione um imóvel salvo.',true);return}try{await api().ImportProjectAreaKML(state.selectedProperty.id,g('areaName109').value,g('areaPurpose109').value);await loadAreas109();await loadConference109();toast('KML da área importado.')}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}}

  function startRoutePoint109(kind){if(!state.selectedProperty){toast('Selecione um imóvel salvo.',true);return}s109.drawMode='route:'+kind;g('map')?.classList.add('map-drawing109');toast('Clique no mapa para marcar '+({reference:'a referência',entrance:'a entrada',headquarters:'a sede'}[kind])+'.')}
  function handleMapClick109(e){if(s109.drawMode==='area'){s109.drawPoints.push(e.latlng);redrawDraft109();return}if(!s109.drawMode.startsWith('route:'))return;const kind=s109.drawMode.split(':')[1];s109.route=s109.route||{property_id:state.selectedProperty?.id||0};s109.route[kind+'_lat']=e.latlng.lat;s109.route[kind+'_lon']=e.latlng.lng;s109.drawMode='';g('map')?.classList.remove('map-drawing109');renderRoute109();drawRouteMarkers109();toast('Ponto marcado.')}
  async function loadRoute109(){clearRouteMarkers109();if(!state.selectedProperty){s109.route=null;renderRoute109();return}try{s109.route=await api().GetAccessRoute(state.selectedProperty.id);renderRoute109();drawRouteMarkers109()}catch(e){s109.route={property_id:state.selectedProperty.id};renderRoute109()}}
  function coordText109(lat,lon){return (lat||lon)?Number(lat).toFixed(6)+', '+Number(lon).toFixed(6):'—'}
  function renderRoute109(){const r=s109.route||{};g('routeReference109').value=r.reference_label||'';g('routeDist1109').value=r.reference_to_entrance_km||'';g('routeDist2109').value=r.entrance_to_headquarters_km||'';g('routeNotes109').value=r.notes||'';g('routeRefCoord109').textContent=coordText109(r.reference_lat,r.reference_lon);g('routeEntranceCoord109').textContent=coordText109(r.entrance_lat,r.entrance_lon);g('routeHQCoord109').textContent=coordText109(r.headquarters_lat,r.headquarters_lon);g('routeText109').textContent=r.text||'Nenhum roteiro salvo.';g('copyRoute109').disabled=!r.text;g('exportRoute109').disabled=!r.text;g('openRouteMaps109').disabled=!r.google_maps_url}
  function clearRouteMarkers109(){s109.routeMarkers.forEach(m=>{try{state.map?.removeLayer(m)}catch(e){}});s109.routeMarkers=[]}
  function drawRouteMarkers109(){clearRouteMarkers109();if(!state.map||!s109.route)return;[['reference','Referência'],['entrance','Entrada'],['headquarters','Sede']].forEach(([k,label])=>{const lat=s109.route[k+'_lat'],lon=s109.route[k+'_lon'];if(lat||lon){const m=L.marker([lat,lon]).addTo(state.map).bindTooltip(label);s109.routeMarkers.push(m)}})}
  async function saveRoute109(){if(!state.selectedProperty){toast('Selecione um imóvel salvo.',true);return}const r=s109.route||{};const payload={property_id:state.selectedProperty.id,reference_label:g('routeReference109').value,reference_lat:Number(r.reference_lat)||0,reference_lon:Number(r.reference_lon)||0,entrance_lat:Number(r.entrance_lat)||0,entrance_lon:Number(r.entrance_lon)||0,headquarters_lat:Number(r.headquarters_lat)||0,headquarters_lon:Number(r.headquarters_lon)||0,reference_to_entrance_km:Number(g('routeDist1109').value)||0,entrance_to_headquarters_km:Number(g('routeDist2109').value)||0,notes:g('routeNotes109').value};try{s109.route=await api().SaveAccessRoute(payload);renderRoute109();drawRouteMarkers109();await loadConference109();toast('Roteiro salvo.')}catch(e){toast(String(e),true)}}
  async function exportRoute109(){if(!state.selectedProperty)return;try{const p=await api().ExportAccessRouteTXT(state.selectedProperty.id);toast('Roteiro salvo em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}}

  async function loadFeaturesForProperty109(){await Promise.all([loadAreas109(),loadRoute109(),loadConference109()])}
  function reset109(){cancelDraw109();clearAreaLayers109();clearRouteMarkers109();s109.areas=[];s109.route=null;renderRoute109();g('areaList109').innerHTML='<div class="empty-state">Selecione um imóvel.</div>';g('conferenceSummary109').innerHTML='<div class="empty-state">Selecione um imóvel salvo para montar a conferência consolidada.</div>';g('themeCache109').textContent=''}

  function installOverrides109(){
    const oldSelect=selectCarProperty,oldRenderCAR=renderCAR,oldRenderThemes=renderThemes,oldReset=resetCARWorkspace,oldCompare=compareGeometries;
    selectCarProperty=async function(id){await oldSelect(id);await loadFeaturesForProperty109()};
    renderCAR=function(r){oldRenderCAR(r);showThemeCache109(r.themes||{});setTimeout(loadConference109,0)};
    renderThemes=function(t){oldRenderThemes(t);showThemeCache109(t)};
    resetCARWorkspace=function(){oldReset();reset109()};
    compareGeometries=async function(){const r=await oldCompare();setTimeout(loadConference109,0);return r};
  }

  document.addEventListener('DOMContentLoaded',()=>{prepare109();installOverrides109()});
})();
