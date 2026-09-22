/* ViaVerdeCAR 1.4.6 — Crédito Rural sob demanda + MDCR automático com fallback oficial */
(function(){
  const g=id=>document.getElementById(id);
  const s140={overview:null,alertLayer:null,loadedCar:''};

  function prepare140(){
    const tabs=document.querySelector('.car-tabs131');
    const docsBtn=document.querySelector('[data-car-tab131="docs"]');
    if(tabs&&docsBtn&&!document.querySelector('[data-car-tab131="credit"]')){
      const b=document.createElement('button');b.className='car-tab131';b.dataset.carTab131='credit';b.textContent='Crédito Rural';
      tabs.insertBefore(b,docsBtn);
      const panel=document.createElement('section');panel.className='car-tab-panel131';panel.id='carTabCredit140';
      const docsPanel=g('carTabDocs131');docsPanel?.parentNode?.insertBefore(panel,docsPanel);
      panel.innerHTML='<div id="creditWorkspace140"></div>';
      b.onclick=()=>openCreditTab140();
    }
    prepareSettings140();
  }

  function openCreditTab140(){
    document.querySelectorAll('[data-car-tab131]').forEach(x=>x.classList.toggle('active',x.dataset.carTab131==='credit'));
    document.querySelectorAll('.car-tab-panel131').forEach(p=>p.classList.remove('active'));
    g('carTabCredit140')?.classList.add('active');
    renderCreditIdle140();
  }

  function renderCreditIdle140(){
    const box=g('creditWorkspace140');if(!box)return;
    if(!state.car?.car){
      box.innerHTML='<div class="credit-loading140">Consulte um CAR primeiro. A integração de Crédito Rural não é carregada durante a inicialização do aplicativo.</div>';
      return;
    }
    if(s140.loadedCar===state.car.car&&s140.overview){
      renderCredit140(s140.overview);
      return;
    }
    box.innerHTML='<article class="panel credit-hero140"><div class="panel-title"><div><span class="eyebrow">CRÉDITO RURAL • CAR '+esc(state.car.car)+'</span><h3>Consulta externa sob demanda</h3><p>Para proteger a estabilidade do Via Verde, Banco Central/SICOR e MapBiomas só serão consultados quando você mandar.</p></div><span class="status-badge neutral">Aguardando</span></div><div class="credit-actions140"><button class="btn primary" id="startCredit140">Consultar Crédito Rural agora</button><button class="btn ghost" id="openCreditMonitorDirect140">Copiar CAR e abrir Monitor</button></div><div class="credit-note140">Se uma fonte externa falhar, o erro ficará restrito a esta aba e o restante do aplicativo continuará funcionando.</div></article>';
    g('startCredit140').onclick=loadCredit140;
    g('openCreditMonitorDirect140').onclick=async()=>{await copyText(state.car.car,'CAR copiado.');openExternal('https://plataforma.creditorural.mapbiomas.org/')};
  }

  function prepareSettings140(){
    const grid=document.querySelector('#view-settings .settings-grid');
    if(!grid||g('mapBiomasSettings140'))return;
    const card=document.createElement('article');card.className='panel credit-settings140';card.id='mapBiomasSettings140';
    card.innerHTML='<span class="eyebrow">MAPBIOMAS ALERTA</span><h3>Conexão da API</h3><p>A API oficial do MapBiomas Alerta exige uma conta confirmada. A senha é usada somente para obter o token e não é armazenada pelo Via Verde.</p><div id="mapBiomasStatus140" class="credit-connected140">Conexão não verificada nesta sessão.</div><div class="mapbiomas-login140"><label>E-mail<input id="mapBiomasEmail140" type="email" autocomplete="username" placeholder="seu@email.com"></label><label>Senha<input id="mapBiomasPassword140" type="password" autocomplete="current-password" placeholder="••••••••"></label><div class="form-actions full"><button class="btn ghost" id="mapBiomasCheck140">Verificar conexão</button><button class="btn primary" id="mapBiomasConnect140">Conectar</button><button class="btn ghost" id="mapBiomasDisconnect140">Desconectar</button><button class="btn ghost" id="mapBiomasSignup140">Criar conta</button></div></div>';
    grid.appendChild(card);
    g('mapBiomasCheck140').onclick=loadMapBiomasStatus140;
    g('mapBiomasConnect140').onclick=connectMapBiomas140;
    g('mapBiomasDisconnect140').onclick=disconnectMapBiomas140;
    g('mapBiomasSignup140').onclick=()=>openExternal('https://plataforma.alerta.mapbiomas.org/sign-up');
  }

  async function loadMapBiomasStatus140(){
    try{
      const s=await api().GetMapBiomasAlertStatus();
      const box=g('mapBiomasStatus140');if(!box)return;
      box.textContent=s.connected?'Conectado: '+(s.email||'conta MapBiomas Alerta'):'Não conectado. Você ainda pode usar o Monitor do Crédito Rural e o contexto público do BCB.';
      box.classList.toggle('hidden',false);
      g('mapBiomasEmail140').value=s.email||'';
      g('mapBiomasDisconnect140').disabled=!s.connected;
    }catch(e){}
  }
  async function connectMapBiomas140(){
    const email=g('mapBiomasEmail140').value.trim(),password=g('mapBiomasPassword140').value;
    const b=g('mapBiomasConnect140');b.disabled=true;b.textContent='Conectando...';
    try{
      const s=await api().MapBiomasAlertLogin(email,password);
      g('mapBiomasPassword140').value='';toast(s.message||'MapBiomas Alerta conectado.');await loadMapBiomasStatus140();
      s140.loadedCar='';s140.overview=null
    }catch(e){toast(String(e),true)}
    finally{b.disabled=false;b.textContent='Conectar'}
  }
  async function disconnectMapBiomas140(){
    if(!confirm('Desconectar a conta MapBiomas Alerta deste computador?'))return;
    try{await api().DisconnectMapBiomasAlert();g('mapBiomasPassword140').value='';await loadMapBiomasStatus140();s140.loadedCar='';s140.overview=null;toast('MapBiomas Alerta desconectado.')}catch(e){toast(String(e),true)}
  }

  async function loadCredit140(){
    const box=g('creditWorkspace140');if(!box||!state.car?.car)return;
    box.innerHTML='<div class="credit-loading140">Consultando BCB/SICOR e MapBiomas para este CAR...</div>';
    try{
      const propertyID=state.selectedProperty?.id||0;
      const r=await api().GetRuralCreditOverview(propertyID);
      s140.overview=r;s140.loadedCar=r.car||state.car.car;renderCredit140(r);drawAlertMarkers140(r.alerts?.alerts||[]);
      setCreditCount140((r.alerts?.total_alerts||0)+(r.bcb?.available?1:0));
    }catch(e){box.innerHTML='<div class="credit-warning140">'+esc(String(e))+'</div>';setCreditCount140(0)}
  }

  function renderCredit140(r){
    const box=g('creditWorkspace140');if(!box)return;
    const alerts=r.alerts||{},bcb=r.bcb||{},warnings=r.warnings||[];
    const monitor='<article class="panel credit-hero140"><div class="panel-title"><div><span class="eyebrow">MAPBIOMAS • MONITOR DO CRÉDITO RURAL</span><h3>Operações e glebas públicas relacionadas ao CAR</h3><p>O Monitor do MapBiomas combina dados públicos do Sicor com CAR, glebas financiadas e bases territoriais. O Via Verde copia o CAR e abre a plataforma oficial para a consulta detalhada das operações públicas disponíveis.</p></div><span class="status-badge info">Monitor externo</span></div><div class="credit-summary140"><div><span>CAR pesquisado</span><strong>'+esc(r.car)+'</strong></div><div><span>Município</span><strong>'+esc(r.municipality||'—')+' / '+esc(r.uf||'—')+'</strong></div><div><span>Alertas MapBiomas</span><strong>'+((alerts.connected)?String(alerts.total_alerts||0):'Conexão opcional')+'</strong></div></div><div class="credit-actions140"><button class="btn primary" id="openCreditMonitor140">Copiar CAR e abrir Monitor</button><button class="btn ghost" id="openBCB140">Dados públicos BCB/SICOR</button><button class="btn ghost" id="refreshCredit140">Atualizar análise</button></div><div class="credit-note140">A ausência de operação na plataforma não prova inexistência de financiamento. Nem todas as operações possuem gleba/localização pública.</div></article>';

    let alertHtml='';
    if(!alerts.connected){
      alertHtml='<div class="credit-warning140"><strong>MapBiomas Alerta não conectado.</strong><br>Conecte sua conta em Configurações para o Via Verde consultar automaticamente alertas vinculados a este CAR. A API oficial exige autenticação.</div>';
    }else if(alerts.total_alerts){
      alertHtml='<div class="credit-summary140"><div><span>Alertas vinculados</span><strong>'+alerts.total_alerts+'</strong></div><div><span>Soma das áreas</span><strong>'+fmt(alerts.total_area_ha,2)+' ha</strong></div><div><span>CAR na base Alerta</span><strong>'+fmt(alerts.area_ha||0,2)+' ha</strong></div></div><div class="credit-alert-list140">'+(alerts.alerts||[]).map(a=>'<div class="credit-alert140"><div><strong>Alerta '+esc(a.alert_code)+'</strong><span>'+fmt(a.area_ha,2)+' ha • detectado em '+esc(a.detected_at||'—')+'</span><small>'+esc((a.sources||[]).join(', ')||'MapBiomas Alerta')+' • publicado '+esc(a.published_at||'—')+'</small></div><button class="btn ghost" data-open-alert140="'+esc(a.report_url)+'">Abrir laudo</button></div>').join('')+'</div><div class="credit-actions140"><button class="btn ghost" id="showAlertsMap140">Mostrar alertas no mapa</button><button class="btn ghost" id="openAlertProperty140">Abrir imóvel no MapBiomas Alerta</button></div>';
    }else{
      alertHtml='<div class="credit-warning140">'+esc(alerts.message||'Nenhum alerta retornado para este CAR.')+'</div><div class="credit-actions140"><button class="btn ghost" id="openAlertProperty140">Abrir imóvel no MapBiomas Alerta</button></div>';
    }

    const bcbRows=(bcb.rows||[]).map(x=>'<div class="credit-row140"><span>'+esc(x.kind||'—')+'</span><span>'+esc(x.year||'—')+'</span><strong>'+esc(x.product||'—')+'</strong><span>'+fmt(x.contracts||0,0)+'</span><span>R$ '+fmtMoney140(x.value||0)+'</span></div>').join('');
    const bcbHtml=bcb.available
      ? '<div class="credit-summary140"><div><span>Linhas exibidas</span><strong>'+String((bcb.rows||[]).length)+'</strong></div><div><span>Contratos somados*</span><strong>'+fmt(bcb.contracts||0,0)+'</strong></div><div><span>Valor somado*</span><strong>R$ '+fmtMoney140(bcb.value||0)+'</strong></div></div><div class="credit-table140"><div class="credit-row140 head"><span>Finalidade</span><span>Ano</span><span>Atividade</span><span>Contratos</span><span>Valor</span></div>'+bcbRows+'</div><div class="credit-note140">*Totais agregados do município. Não significa que esses contratos pertençam ao CAR consultado.</div>'
      : bcb.external_only
        ? '<div class="credit-warning140"><strong>A consulta automática foi tentada.</strong><br>'+esc(bcb.message||'A API pública não concluiu o filtro municipal nesta tentativa.')+'</div><div class="credit-actions140"><button class="btn ghost" id="openBCBStructured140">Abrir MDCR oficial</button></div>'
        : '<div class="credit-warning140">'+esc(bcb.message||'Contexto municipal BCB indisponível nesta consulta.')+'</div>';

    box.innerHTML=monitor+'<div class="credit-grid140"><div class="credit-stack140"><article class="panel"><div class="panel-title"><div><span class="eyebrow">MAPBIOMAS ALERTA</span><h3>Alertas vinculados ao CAR</h3><p>Consulta pela API oficial MapBiomas Alerta quando a conta estiver conectada.</p></div><span class="status-badge '+(alerts.connected?(alerts.total_alerts?'warning':'ok'):'neutral')+'">'+(alerts.connected?(alerts.total_alerts?'Atenção':'Consultado'):'Não conectado')+'</span></div>'+alertHtml+'<div class="credit-source140">Alertas são evidências territoriais a conferir; não determinam por si só ilegalidade, responsabilidade ou impedimento de crédito.</div></article></div><div class="credit-stack140"><article class="panel"><div class="panel-title"><div><span class="eyebrow">BANCO CENTRAL • SICOR / MDCR</span><h3>Contexto de crédito do município</h3><p>O Via Verde tenta consultar automaticamente o SICOR/MDCR para o município do CAR. A página oficial fica apenas como contingência quando a API pública não conclui o filtro.</p></div><span class="status-badge '+(bcb.available?'ok':bcb.external_only?'info':'warning')+'">'+(bcb.available?'Automático':bcb.external_only?'Contingência':'Indisponível')+'</span></div>'+bcbHtml+'</article></div></div>'+(warnings.length?'<div class="credit-warning140"><strong>Fontes com limitação nesta consulta:</strong><br>'+warnings.map(esc).join('<br>')+'</div>':'');

    g('openCreditMonitor140').onclick=async()=>{await copyText(r.car,'CAR copiado.');openExternal(r.mapbiomas_monitor_url)};
    g('openBCB140').onclick=()=>openExternal(r.bcb_source_url);
    g('openBCBStructured140')?.addEventListener('click',()=>openExternal(r.bcb_source_url));
    g('refreshCredit140').onclick=()=>{s140.loadedCar='';loadCredit140()};
    g('openAlertProperty140')?.addEventListener('click',()=>openExternal(r.mapbiomas_alert_property_url));
    g('showAlertsMap140')?.addEventListener('click',()=>showAlertsOnMap140());
    document.querySelectorAll('[data-open-alert140]').forEach(b=>b.onclick=()=>openExternal(b.dataset.openAlert140));
  }

  function fmtMoney140(v){
    try{return Number(v||0).toLocaleString('pt-BR',{minimumFractionDigits:2,maximumFractionDigits:2})}catch(e){return fmt(v,2)}
  }
  function setCreditCount140(count){
    const b=document.querySelector('[data-car-tab131="credit"]');if(!b)return;
    let s=b.querySelector('.tab-count131');
    if(count>0){if(!s){s=document.createElement('span');s.className='tab-count131';b.appendChild(s)}s.textContent=String(count)}
    else s?.remove();
  }

  function clearAlertMarkers140(){
    if(s140.alertLayer){try{state.map?.removeLayer(s140.alertLayer);state.layerControl?.removeLayer(s140.alertLayer)}catch(e){}s140.alertLayer=null}
  }
  function drawAlertMarkers140(alerts){
    clearAlertMarkers140();if(!state.map||!alerts?.length)return;
    const group=L.layerGroup();
    alerts.forEach(a=>{
      if(!a.latitude&&!a.longitude)return;
      const icon=L.divIcon({className:'',html:'<div class="alert-map-marker140"></div>',iconSize:[12,12],iconAnchor:[6,6]});
      const m=L.marker([a.latitude,a.longitude],{icon}).bindPopup('<strong>MapBiomas Alerta '+esc(a.alert_code)+'</strong><br>'+fmt(a.area_ha,2)+' ha<br>Detectado: '+esc(a.detected_at||'—'));
      m.addTo(group);
    });
    if(group.getLayers().length){state.layerControl?.addOverlay(group,'MapBiomas Alerta');s140.alertLayer=group}
  }
  function showAlertsOnMap140(){
    if(!s140.alertLayer){toast('Nenhum ponto de alerta disponível no mapa.',true);return}
    s140.alertLayer.addTo(state.map);
    document.querySelector('[data-car-tab131="map"]')?.click();
    setTimeout(()=>{try{const pts=s140.alertLayer.getLayers().map(x=>x.getLatLng()).filter(Boolean);if(pts.length)state.map.fitBounds(L.latLngBounds(pts).pad(.35),{maxZoom:15})}catch(e){}},100);
  }

  function safePrepare140(){
    try{prepare140();renderCreditIdle140()}catch(e){
      try{console.error('ViaVerdeCAR Crédito Rural isolado:',e)}catch(_){}
    }
  }

  document.addEventListener('DOMContentLoaded',safePrepare140);
})();
