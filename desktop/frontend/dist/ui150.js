/* ViaVerdeCAR 1.6.0 — Raio X completo + fundiário SIGEF/INCRA */
(function(){
  const q=id=>document.getElementById(id);
  const s150={car:'',result:null,glebaLayer:null,sigefLayer:null};

  function prepare150(){
    const tabs=document.querySelector('.car-tabs131');
    if(!tabs||document.querySelector('[data-car-tab131="xray"]'))return;
    const summary=document.querySelector('[data-car-tab131="summary"]');
    const b=document.createElement('button');b.className='car-tab131';b.dataset.carTab131='xray';b.textContent='Raio X';
    summary?.after(b);
    const panel=document.createElement('section');panel.className='car-tab-panel131';panel.id='carTabXRay150';
    const summaryPanel=q('carTabSummary131');summaryPanel?.after(panel);
    panel.innerHTML='<div id="xrayWorkspace150" class="xray150"></div>';
    b.onclick=openXRay150;
  }

  function openXRay150(){
    document.querySelectorAll('[data-car-tab131]').forEach(x=>x.classList.toggle('active',x.dataset.carTab131==='xray'));
    document.querySelectorAll('.car-tab-panel131').forEach(p=>p.classList.remove('active'));
    q('carTabXRay150')?.classList.add('active');
    if(s150.car!==state.car?.car){s150.car=state.car?.car||'';s150.result=null;clearGlebas150();clearSIGEF150()}
    renderIdle150();
  }

  function renderIdle150(){
    const box=q('xrayWorkspace150');if(!box)return;
    if(!state.car?.car){
      box.innerHTML='<div class="xray-empty150"><strong>Consulte um CAR primeiro.</strong><br>O Raio X será montado a partir do CAR, microdados públicos do SICOR, glebas financiadas disponíveis e demais fontes já usadas pelo Via Verde.</div>';return;
    }
    if(s150.result){renderXRay150(s150.result);return}
    box.innerHTML='<article class="panel xray-hero150"><span class="eyebrow">RAIO X DO IMÓVEL</span><h3>'+esc(state.car.car)+'</h3><p>Busca referências públicas do CAR nos microdados oficiais do SICOR desde 2013, localiza operações, banco, programa, fonte de recursos, finalidade, atividade, produto e glebas financiadas quando publicadas. Depois cruza as glebas antigas com o CAR e com as áreas do projeto atual.</p><div class="xray-scope150"><span>BCB/SICOR desde 2013</span><span>Glebas públicas</span><span>SIGEF/INCRA</span><span>MapBiomas Alerta</span><span>Ambiental</span><span>Áreas do projeto</span></div><div class="xray-actions150"><button class="btn primary" id="buildXRay150">Montar Raio X completo</button><button class="btn ghost" id="openMonitor150">Abrir Monitor MapBiomas</button></div><div class="xray-source150"><strong>Primeira sincronização:</strong> os microdados oficiais do Banco Central são arquivos nacionais grandes. A primeira análise pode levar alguns minutos. Depois o resultado deste CAR fica em cache local por 14 dias. Não feche o aplicativo durante a primeira sincronização.</div></article>';
    q('buildXRay150').onclick=()=>buildXRay150(false);
    q('openMonitor150').onclick=async()=>{await copyText(state.car.car,'CAR copiado.');openExternal('https://plataforma.creditorural.mapbiomas.org/')};
  }

  async function buildXRay150(force){
    const box=q('xrayWorkspace150');if(!box||!state.car?.car)return;
    clearGlebas150();clearSIGEF150();
    box.innerHTML='<div class="xray-loading150"><strong>Montando o Raio X completo…</strong><span>Sincronizando índice de propriedades do SICOR e procurando operações vinculadas ao CAR. Na primeira vez isso pode levar alguns minutos porque os arquivos oficiais são nacionais.</span></div>';
    try{
      const r=await api().BuildPropertyXRay(state.selectedProperty?.id||0,!!force);
      s150.result=r;s150.car=r.car?.car||state.car.car;renderXRay150(r);setXRayCount150(r.summary?.public_credit_operations||0);
    }catch(e){
      box.innerHTML='<div class="xray-warning150"><strong>Raio X não concluído.</strong><br>'+esc(String(e))+'</div><div class="xray-actions150"><button class="btn primary" id="retryXRay150">Tentar novamente</button></div>';
      q('retryXRay150').onclick=()=>buildXRay150(true);toast(String(e),true)
    }
  }

  function renderXRay150(r){
    const box=q('xrayWorkspace150');if(!box)return;
    const s=r.summary||{},sicor=r.sicor||{},sigef=r.sigef||{},mb=r.mapbiomas||{},car=r.car||{},env=car.environment||{},themes=car.themes?.themes||{};
    const operations=sicor.operations||[];
    const sigefAvailable=sigef.available===true;
    const mapbiomasAvailable=mb.available===true;
    const themeAvailable=Object.values(themes).filter(x=>x?.available).length;
    const warningCount=(r.warnings||[]).length;
    const hero='<article class="panel xray-hero150"><div class="panel-title"><div><span class="eyebrow">RAIO X • '+esc(car.municipality||'')+' / '+esc(car.uf||'')+'</span><h3>'+esc(car.car||'')+'</h3><p>'+esc(sicor.scope||'Microdados públicos do SICOR e bases territoriais públicas.')+'</p></div><span class="status-badge '+(operations.length?'ok':'info')+'">'+(sicor.used_cache?'Cache local':'Atualizado')+'</span></div><div class="xray-metrics150">'+
      metric150('Operações públicas',s.public_credit_operations||0,(sicor.destination_count||0)+' destinação(ões) vinculada(s)')+
      metric150('Crédito identificado','R$ '+money150(s.public_credit_value||0),'soma das destinações localizadas')+
      metric150('Glebas financiadas',s.financed_glebas||0,'geometria pública disponível')+
      metric150('Parcelas SIGEF',sigefAvailable?(s.sigef_parcels||0):'Indisponível',sigefAvailable?((s.sigef_registries||0)+' registro(s) publicado(s)'):'fonte pública sem resposta')+
      metric150('Conflitos c/ projeto',s.glebas_with_project_hit||0,'sobreposição > 0,5%')+
      metric150('Alertas MapBiomas',mb.connected?(mapbiomasAvailable?(s.mapbiomas_alerts||0):'Indisponível'):'Não conectado',mb.connected?(mapbiomasAvailable?'consulta concluída':'fonte sem resposta'):'conta não conectada')+
      metric150('Ocorrências ambientais',s.environmental_hits||0,'IBAMA/FUNAI/ICMBio/MCR')+
      '</div><div class="xray-actions150"><button class="btn primary" id="showSicorGlebas150" '+(!s.financed_glebas?'disabled':'')+'>Mostrar glebas financiadas</button><button class="btn primary" id="showSIGEF150" '+(!(sigef.parcels||[]).length?'disabled':'')+'>Mostrar SIGEF no mapa</button><button class="btn ghost" id="exportSicor150" '+(!operations.length?'disabled':'')+'>Exportar operações CSV</button><button class="btn ghost" id="refreshXRay150">Recalcular análise</button><button class="btn ghost" id="openMonitor150">Abrir Monitor MapBiomas</button></div><div class="xray-source150">Fonte principal do histórico individual: microdados públicos do SICOR/BCB. A ausência de operação ou gleba pública não prova inexistência de financiamento. O Monitor do Crédito Rural também informa que sua base representa o universo publicamente rastreável, não todo o crédito rural.</div></article>';

    const sigefParcels=sigef.parcels||[];
    const sigefCards=sigefParcels.length?sigefParcels.map((p,i)=>sigefCard150(p,i)).join(''):'<div class="xray-empty150">'+esc(sigef.message||'Nenhuma parcela SIGEF pública localizada por interseção espacial. Isso não prova ausência de matrícula ou registro imobiliário.')+'</div>';
    const sigefHtml='<article class="panel xray-panel150 sigef-panel150"><div class="panel-title"><div><span class="eyebrow">FUNDIÁRIO • SIGEF/INCRA</span><h3>Parcelas certificadas que cruzam o CAR</h3><p>Consulta espacial automática na camada pública do SIGEF. Registro/matrícula só é exibido quando publicado pela própria fonte.</p></div><span class="status-badge '+(sigef.available?(sigefParcels.length?'ok':'info'):'warning')+'">'+(sigef.available?sigefParcels.length+' parcela(s)':'Indisponível')+'</span></div><div class="sigef-summary150">'+
      envCard150('Parcelas encontradas',sigefAvailable?(sigef.parcel_count||0):'Indisponível',sigefAvailable?'interseção espacial com o CAR':'fonte pública sem resposta')+
      envCard150('Registros publicados',sigefAvailable?(sigef.registry_count||0):'Indisponível',sigefAvailable?'campo registro_m da fonte':'fonte pública sem resposta')+
      envCard150('Melhor cobertura do CAR',sigefAvailable?((sigef.best_car_coverage_pct||0)?fmt(sigef.best_car_coverage_pct,1)+'%':'—'):'Indisponível',sigefAvailable?'percentual do CAR dentro da parcela':'fonte pública sem resposta')+
      envCard150('Diferença de área',sigefAvailable?((sigef.parcel_count||0)?fmt(sigef.best_area_difference_ha||0,2)+' ha':'—'):'Indisponível',sigefAvailable?'melhor coincidência espacial':'fonte pública sem resposta')+
      '</div><div class="sigef-list150">'+sigefCards+'</div><div class="xray-source150">'+esc(sigef.message||'A consulta fundiária é auxiliar e não substitui matrícula/certidão do Registro de Imóveis.')+' A ausência de parcela SIGEF não prova ausência de matrícula.</div></article>';

    const operationHtml=operations.length?operations.map((op,i)=>operationCard150(op,i)).join(''):'<div class="xray-empty150">Nenhuma operação pública foi localizada para este CAR nos arquivos processados. Isso não significa que o imóvel nunca tenha recebido financiamento.</div>';
    const envHtml='<article class="panel xray-panel150"><div class="panel-title"><div><span class="eyebrow">SOCIOAMBIENTAL</span><h3>Camadas e ocorrências</h3></div><span class="status-badge info">'+(s.environmental_hits||0)+' ocorrência(s)</span></div><div class="xray-env-grid150">'+
      envCard150('IBAMA • Embargos',env.ibama_checked?(env.ibama_embargo_count||0):'Indisponível',env.ibama_checked?'consulta pública':'fonte sem resposta')+
      envCard150('FUNAI • Terras indígenas',env.funai_checked?(env.indigenous_count||0):'Indisponível',env.funai_checked?'consulta pública':'fonte sem resposta')+
      envCard150('ICMBio • UCs',env.icmbio_checked?(env.federal_uc_count||0):'Indisponível',env.icmbio_checked?'consulta pública':'fonte sem resposta')+
      envCard150('MCR / Prodes',env.mcr_checked?(env.mcr_listed?'LISTADO':'NÃO LISTADO'):'Indisponível',env.mcr_checked?'lista pública consultada':'fonte sem resposta')+
      envCard150('Temas SICAR',themeAvailable+'/6','APP, RL, vegetação e demais temas declarados')+
      envCard150('MapBiomas Alerta',mb.connected?(mapbiomasAvailable?(mb.total_alerts||0):'Indisponível'):'Não conectado',mb.message||'API MapBiomas Alerta')+
      '</div></article>';

    const localHtml='<article class="panel xray-panel150"><div class="panel-title"><div><span class="eyebrow">PROJETO ATUAL</span><h3>Comparação com o que você está projetando</h3></div><span class="status-badge neutral">'+((r.areas||[]).length)+' área(s)</span></div><div class="xray-env-grid150">'+
      envCard150('Área do CAR',fmt(car.geometry_area_ha||car.area_ha||0,2)+' ha','geometria pública')+
      envCard150('Áreas do projeto',(r.areas||[]).length,(r.areas||[]).map(a=>a.name+' '+fmt(a.area_ha,2)+' ha').join(' • ')||'nenhuma área criada')+
      envCard150('Acesso',r.route?.route_distance_km?fmt(r.route.route_distance_km,2)+' km':'—',r.route?.reference_label||'roteiro não disponível')+
      envCard150('Glebas antigas em conflito',s.glebas_with_project_hit||0,'cruzamento espacial com áreas do Via Verde')+
      '</div></article>';

    const warnings=[...(r.warnings||[])];
    if(sicor.unresolved_references)warnings.push(sicor.unresolved_references+' referência(s) do CAR no índice SICOR ficaram sem operação anual localizada.');
    const warningsHtml='<article class="panel xray-panel150"><div class="panel-title"><div><span class="eyebrow">LIMITAÇÕES E CONFERÊNCIA</span><h3>O que merece atenção</h3></div><span class="status-badge '+(warningCount?'warning':'ok')+'">'+warnings.length+'</span></div><div class="xray-warnings150">'+(warnings.length?warnings.map(x=>'<div class="xray-warning150">'+esc(x)+'</div>').join(''):'<div class="xray-warning150">Nenhuma limitação adicional registrada nesta execução.</div>')+'</div></article>';

    box.innerHTML=hero+sigefHtml+'<div class="xray-grid150"><div class="xray-stack150"><article class="panel xray-panel150"><div class="panel-title"><div><span class="eyebrow">HISTÓRICO PÚBLICO SICOR</span><h3>Operações e destinações vinculadas ao CAR</h3><p>REF BACEN identifica a operação; NU_ORDEM identifica cada destinação. A lista mostra instituição, programa, fonte, finalidade, atividade, produto, valores e glebas publicadas.</p></div><span class="status-badge '+(operations.length?'ok':'info')+'">'+(sicor.operation_count||0)+' operação(ões) • '+(sicor.destination_count||0)+' destinação(ões)</span></div><div class="xray-operation-list150">'+operationHtml+'</div></article></div><aside class="xray-stack150">'+localHtml+envHtml+warningsHtml+'</aside></div>';

    q('showSicorGlebas150').onclick=()=>showAllSicorGlebas150(r);
    q('showSIGEF150').onclick=()=>showAllSIGEF150(r);
    q('exportSicor150').onclick=()=>exportSicor150(car.car);
    q('refreshXRay150').onclick=()=>buildXRay150(true);
    q('openMonitor150').onclick=async()=>{await copyText(car.car,'CAR copiado.');openExternal('https://plataforma.creditorural.mapbiomas.org/')};
    document.querySelectorAll('[data-xray-op150]').forEach(b=>b.onclick=()=>focusOperation150(Number(b.dataset.xrayOp150)));
    document.querySelectorAll('[data-sigef-parcel150]').forEach(b=>b.onclick=()=>focusSIGEF150(Number(b.dataset.sigefParcel150)));
    setXRayCount150(operations.length);
  }

  function sigefCard150(p,i){
    const registry=p.registry||'Não publicado';
    const high=p.comparison_level==='high';
    const partial=p.comparison_level==='partial';
    const cls=high?'high':partial?'partial':'intersection';
    return '<div class="sigef-card150 '+cls+'"><div class="sigef-top150"><div><strong>'+esc(p.area_name||p.parcel_code||'Parcela SIGEF')+'</strong><small>Parcela '+esc(p.parcel_code||'—')+'</small></div><span class="status-badge '+(high?'ok':partial?'info':'warning')+'">'+esc(p.comparison_summary||'Interseção espacial')+'</span></div><div class="xray-fields150">'+
      field150('Registro / matrícula',registry)+
      field150('Código do imóvel',p.property_code||'—')+
      field150('Situação',p.property_situation||p.status||'—')+
      field150('Área SIGEF',p.area_ha?fmt(p.area_ha,2)+' ha':'—')+
      field150('CAR dentro da parcela',fmt(p.car_inside_parcel_pct||0,1)+'%')+
      field150('Parcela dentro do CAR',fmt(p.parcel_inside_car_pct||0,1)+'%')+
      field150('Interseção',fmt(p.intersection_area_ha||0,2)+' ha')+
      field150('Diferença de área',fmt(p.area_difference_ha||0,2)+' ha')+
      field150('Aprovação',p.approval_date||'—')+
      field150('Registro',p.registry_date||'—')+
      field150('ART',p.art||'—')+
      field150('RT publicado',p.technical_rt||'—')+
      '</div><div class="xray-actions150"><button class="btn ghost" data-sigef-parcel150="'+i+'">Ver parcela no mapa</button></div></div>';
  }

  function operationCard150(op,i){
    const glebas=op.glebas||[];
    const conflict=glebas.some(g=>(g.project_overlap_pct||0)>.5);
    const tags=glebas.map(g=>'<span class="xray-gleba150 '+((g.project_overlap_pct||0)>.5?'conflict':'')+'">Gleba '+esc(g.index||'')+' • '+fmt(g.area_ha||0,2)+' ha'+((g.project_overlap_pct||0)>.5?' • '+fmt(g.project_overlap_pct,1)+'% no projeto':'')+'</span>').join('');
    return '<div class="xray-operation150 '+(conflict?'conflict':'')+'"><div class="xray-operation-top150"><div><strong>'+esc(op.institution_name||op.institution_code||'Instituição não identificada')+'</strong><small>REF BACEN '+esc(op.ref_bacen)+' • ordem '+esc(op.order)+' • '+esc(op.issue_date||String(op.year||''))+'</small></div><div class="xray-value150">R$ '+money150(op.credit_value||0)+'</div></div><div class="xray-fields150">'+
      field150('Programa',op.program_name||op.program_code||'—')+
      field150('Fonte',op.resource_name||op.resource_code||'—')+
      field150('Finalidade',op.purpose||'—')+
      field150('Atividade',op.activity||'—')+
      field150('Modalidade',op.modality||'—')+
      field150('Produto',op.product||op.enterprise_code||'—')+
      field150('Área financiada',op.financed_area_ha?fmt(op.financed_area_ha,2)+' ha':'—')+
      field150('Vencimento',op.due_date||'—')+
      '</div>'+(glebas.length?'<div class="xray-glebas150">'+tags+'</div><div class="xray-actions150"><button class="btn ghost" data-xray-op150="'+i+'">Ver gleba no mapa</button></div>':'<div class="xray-source150">Esta operação foi localizada pelo CAR, mas não possui gleba WKT pública encontrada para o ano processado.</div>')+'</div>';
  }

  function metric150(label,value,small){return '<div class="xray-metric150"><span>'+esc(label)+'</span><strong>'+value+'</strong><small>'+esc(small||'')+'</small></div>'}
  function field150(label,value){return '<div class="xray-field150"><span>'+esc(label)+'</span><b>'+esc(String(value||'—'))+'</b></div>'}
  function envCard150(label,value,small){return '<div class="xray-env150"><span>'+esc(label)+'</span><strong>'+esc(String(value))+'</strong><small>'+esc(small||'')+'</small></div>'}
  function money150(v){try{return Number(v||0).toLocaleString('pt-BR',{minimumFractionDigits:2,maximumFractionDigits:2})}catch(e){return fmt(v,2)}}

  function clearSIGEF150(){
    if(s150.sigefLayer){try{state.map?.removeLayer(s150.sigefLayer);state.layerControl?.removeLayer(s150.sigefLayer)}catch(e){}s150.sigefLayer=null}
  }
  function showAllSIGEF150(r){
    clearSIGEF150();if(!state.map)return;
    const group=L.featureGroup();let count=0;
    (r.sigef?.parcels||[]).forEach((p,i)=>{if(!p.geojson)return;try{
      const layer=L.geoJSON(JSON.parse(p.geojson),{style:{color:'#6f4f9b',weight:3,dashArray:'6 4',fillOpacity:.08}});
      layer.bindPopup('<strong>SIGEF/INCRA</strong><br>'+esc(p.area_name||p.parcel_code||'Parcela')+'<br>Registro: '+esc(p.registry||'não publicado')+'<br>Área: '+fmt(p.area_ha||0,2)+' ha<br>CAR dentro da parcela: '+fmt(p.car_inside_parcel_pct||0,1)+'%');
      layer.eachLayer(x=>x.addTo(group));count++;
    }catch(e){}});
    if(!count){toast('Nenhuma geometria SIGEF pública disponível.',true);return}
    group.addTo(state.map);state.layerControl?.addOverlay(group,'SIGEF/INCRA • parcelas públicas');s150.sigefLayer=group;
    document.querySelector('[data-car-tab131="map"]')?.click();
    setTimeout(()=>{try{const b=group.getBounds();if(b.isValid())state.map.fitBounds(b.pad(.18),{maxZoom:17})}catch(e){}},100);
  }
  function focusSIGEF150(i){
    const p=s150.result?.sigef?.parcels?.[i];if(!p?.geojson||!state.map)return;
    clearSIGEF150();const group=L.featureGroup();
    try{
      const layer=L.geoJSON(JSON.parse(p.geojson),{style:{color:'#6f4f9b',weight:4,dashArray:'6 4',fillOpacity:.10}});
      layer.bindPopup('<strong>SIGEF/INCRA</strong><br>'+esc(p.area_name||p.parcel_code||'Parcela')+'<br>Registro: '+esc(p.registry||'não publicado')+'<br>Código do imóvel: '+esc(p.property_code||'—')+'<br>Área: '+fmt(p.area_ha||0,2)+' ha<br>CAR dentro da parcela: '+fmt(p.car_inside_parcel_pct||0,1)+'%');
      layer.eachLayer(x=>x.addTo(group));
    }catch(e){toast('Não foi possível desenhar a parcela SIGEF.',true);return}
    group.addTo(state.map);state.layerControl?.addOverlay(group,'SIGEF/INCRA • '+(p.parcel_code||'parcela'));s150.sigefLayer=group;
    document.querySelector('[data-car-tab131="map"]')?.click();
    setTimeout(()=>{try{const b=group.getBounds();if(b.isValid())state.map.fitBounds(b.pad(.2),{maxZoom:18})}catch(e){}},80);
  }

  function clearGlebas150(){
    if(s150.glebaLayer){try{state.map?.removeLayer(s150.glebaLayer);state.layerControl?.removeLayer(s150.glebaLayer)}catch(e){}s150.glebaLayer=null}
  }
  function showAllSicorGlebas150(r){
    clearGlebas150();if(!state.map)return;
    const group=L.featureGroup();let count=0;
    (r.sicor?.operations||[]).forEach(op=>(op.glebas||[]).forEach(g=>{if(!g.geojson)return;try{
      const conflict=(g.project_overlap_pct||0)>.5;
      const layer=L.geoJSON(JSON.parse(g.geojson),{style:{color:conflict?'#9b6518':'#346b9a',weight:3,fillOpacity:.15}});
      layer.bindPopup('<strong>SICOR • '+esc(op.ref_bacen)+'</strong><br>'+esc(op.institution_name||'')+'<br>Gleba '+esc(g.index||'')+' • '+fmt(g.area_ha,2)+' ha'+(conflict?'<br><b>Sobreposição com projeto: '+fmt(g.project_overlap_pct,1)+'%</b>':''));
      layer.eachLayer(x=>x.addTo(group));count++;
    }catch(e){}}));
    if(!count){toast('Nenhuma geometria de gleba pública disponível.',true);return}
    group.addTo(state.map);state.layerControl?.addOverlay(group,'SICOR • glebas financiadas');s150.glebaLayer=group;
    document.querySelector('[data-car-tab131="map"]')?.click();
    setTimeout(()=>{try{const b=group.getBounds();if(b.isValid())state.map.fitBounds(b.pad(.18),{maxZoom:17})}catch(e){}},100);
  }
  function focusOperation150(i){
    const op=s150.result?.sicor?.operations?.[i];if(!op?.glebas?.length)return;
    clearGlebas150();const group=L.featureGroup();
    op.glebas.forEach(g=>{try{const conflict=(g.project_overlap_pct||0)>.5;const layer=L.geoJSON(JSON.parse(g.geojson),{style:{color:conflict?'#9b6518':'#346b9a',weight:4,fillOpacity:.20}});layer.eachLayer(x=>x.addTo(group))}catch(e){}});
    group.addTo(state.map);state.layerControl?.addOverlay(group,'SICOR • '+op.ref_bacen);s150.glebaLayer=group;
    document.querySelector('[data-car-tab131="map"]')?.click();setTimeout(()=>{try{const b=group.getBounds();if(b.isValid())state.map.fitBounds(b.pad(.2),{maxZoom:18})}catch(e){}},80)
  }
  async function exportSicor150(car){try{const p=await api().ExportSICORXRayCSV(car);toast('CSV salvo em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}}
  function setXRayCount150(n){
    const b=document.querySelector('[data-car-tab131="xray"]');if(!b)return;let s=b.querySelector('.tab-count131');
    if(n>0){if(!s){s=document.createElement('span');s.className='tab-count131';b.appendChild(s)}s.textContent=String(n)}else s?.remove()
  }

  document.addEventListener('DOMContentLoaded',prepare150);
})();
