/* ViaVerdeCAR 1.2.0 — três alternativas de gleba com pontuação territorial preliminar. */
(function(){
  const el=id=>document.getElementById(id);
  const s120={result:null,layers:[],chosenLayer:null};
  const palette=['#226b45','#a36c1e','#496f9d'];

  function prepare120(){
    const box=el('autoAreaBox110'),btn=el('generateAutoArea110');
    if(box&&btn&&!el('areaIntelligence120')){
      btn.textContent='Analisar 3 opções';
      btn.onclick=generateAlternatives120;
      const hint=box.querySelector('.auto-area-hint110');
      if(hint)hint.textContent='O Via Verde gera até três alternativas compactas dentro do CAR e compara proximidade do acesso, formato, conflito com áreas já usadas e temas declarados do SICAR quando disponíveis.';
      const wrap=document.createElement('div');wrap.id='areaIntelligence120';wrap.className='area-intelligence120';
      wrap.innerHTML='<div class="area-intelligence-head120"><div><strong>Seleção inteligente de gleba</strong><span>As opções são uma triagem territorial preliminar. A pontuação não substitui avaliação agronômica, ambiental ou levantamento de campo.</span></div></div><div id="areaCandidates120"></div>';
      box.appendChild(wrap);
    }
  }

  function clearCandidateLayers120(){
    s120.layers.forEach(l=>{try{state.map?.removeLayer(l);state.layerControl?.removeLayer(l)}catch(e){}});s120.layers=[];
  }
  function clearChosenLayer120(){if(s120.chosenLayer){try{state.map?.removeLayer(s120.chosenLayer);state.layerControl?.removeLayer(s120.chosenLayer)}catch(e){}s120.chosenLayer=null}}

  async function generateAlternatives120(){
    if(!state.car?.geojson&&!state.selectedProperty){toast('Consulte um CAR antes de analisar as glebas.',true);return}
    const target=Number(el('autoAreaTarget110').value)||0;if(target<=0){toast('Informe a área necessária em hectares.',true);return}
    const btn=el('generateAutoArea110'),out=el('areaCandidates120');btn.disabled=true;btn.textContent='Analisando...';out.innerHTML='<div class="area-candidate-loading120">Gerando posições alternativas e comparando os critérios territoriais disponíveis...</div>';
    clearCandidateLayers120();
    try{
      const propertyID=state.selectedProperty?.id||0;
      const r=await api().GenerateProjectAreaAlternatives(propertyID,el('autoAreaName110').value,el('autoAreaPurpose110').value,target);
      s120.result=r;renderAlternatives120(r);drawAlternatives120(r.candidates||[]);
      toast((r.candidates||[]).length+' opção(ões) comparadas.');
    }catch(e){out.innerHTML='';toast(String(e),true)}
    finally{btn.disabled=false;btn.textContent='Analisar 3 opções'}
  }

  function metric120(label,value){return '<div class="area-metric120"><span>'+label+'</span><strong>'+value+'</strong></div>'}
  function renderAlternatives120(r){
    const out=el('areaCandidates120'),items=r.candidates||[];
    if(!items.length){out.innerHTML='<div class="empty-state">Nenhuma alternativa foi formada.</div>';return}
    const cards=items.map((x,i)=>{
      const access=r.access_used?metric120('Dist. acesso',x.distance_to_access_m>=1000?fmt(x.distance_to_access_m/1000,2)+' km':fmt(x.distance_to_access_m,0)+' m'):'';
      const conflicts=x.existing_overlap_pct>0.5?fmt(x.existing_overlap_pct,1)+'%':'0%';
      const flags=(x.flags||[]).length?(x.flags||[]).map(f=>'<div class="area-flag120">'+esc(f)+'</div>').join(''):'<div class="area-flag120 ok">Nenhum alerta relevante nos critérios disponíveis.</div>';
      return '<article class="area-candidate120 '+(i===0?'best':'')+'" data-alt="'+i+'"><div class="area-candidate-top120"><div><strong>'+esc(x.label)+(i===0?' • maior pontuação':'')+'</strong><div class="area-explanation120">'+esc(x.explanation||'')+'</div></div><div class="area-score120">'+fmt(x.score,0)+'/100</div></div><div class="area-metrics120">'+metric120('Área',fmt(x.area_ha,4)+' ha')+metric120('Dentro CAR',fmt(x.inside_car_pct,1)+'%')+metric120('Compacidade',fmt(x.compactness_pct,0)+'%')+access+metric120('Área já usada',conflicts)+'</div><div class="area-flags120">'+flags+'</div><div class="area-candidate-actions120"><button class="btn ghost" data-preview-alt="'+i+'">Ver no mapa</button><button class="btn primary" data-choose-alt="'+i+'">Usar esta área</button></div></article>';
    }).join('');
    const warnings=(r.warnings||[]).map(w=>esc(w)).join(' • ');
    out.innerHTML='<div class="area-candidates120">'+cards+'</div><div class="area-intelligence-note120">'+esc(r.method||'')+(warnings?'<br><strong>Aviso:</strong> '+warnings:'')+'</div>';
    document.querySelectorAll('[data-preview-alt]').forEach(b=>b.onclick=()=>previewAlternative120(Number(b.dataset.previewAlt)));
    document.querySelectorAll('[data-choose-alt]').forEach(b=>b.onclick=()=>chooseAlternative120(Number(b.dataset.chooseAlt),b));
  }

  function drawAlternatives120(items){
    clearCandidateLayers120();if(!state.map)return;
    items.forEach((x,i)=>{try{
      const layer=L.geoJSON(JSON.parse(x.geojson),{style:{color:palette[i%palette.length],weight:i===0?3:2,dashArray:i===0?'':'6 4',fillColor:palette[i%palette.length],fillOpacity:i===0?.16:.08}});
      layer.addTo(state.map);state.layerControl?.addOverlay(layer,x.label+' • '+fmt(x.score,0)+'/100');s120.layers.push(layer);
    }catch(e){}});
  }
  function previewAlternative120(i){
    const layer=s120.layers[i];if(!layer)return;const b=layer.getBounds();if(b.isValid())state.map.fitBounds(b.pad(.20),{maxZoom:18});
    document.querySelectorAll('.area-candidate120').forEach((c,idx)=>c.classList.toggle('best',idx===i));
  }

  async function chooseAlternative120(i,button){
    const alt=s120.result?.candidates?.[i];if(!alt)return;
    button.disabled=true;button.textContent='Salvando...';
    try{
      const propertyID=state.selectedProperty?.id||0;
      const x=await api().ChooseProjectAreaAlternative(propertyID,el('autoAreaName110').value,el('autoAreaPurpose110').value,alt);
      clearCandidateLayers120();showChosenArea120(x);
      if(propertyID){
        toast('Área escolhida salva no imóvel: '+fmt(x.area_ha,4)+' ha.');
        await selectCarProperty(propertyID);
      }else{
        toast('Área escolhida mantida temporariamente e pronta para exportar em KML.');
      }
      const out=el('areaCandidates120');if(out)out.innerHTML='<div class="area-chosen120"><strong>'+esc(x.name||alt.label)+'</strong> selecionada • '+fmt(x.area_ha,4)+' ha • '+fmt(x.inside_car_pct||alt.inside_car_pct,1)+'% dentro do CAR.</div>';
    }catch(e){toast(String(e),true);button.disabled=false;button.textContent='Usar esta área'}
  }

  function showChosenArea120(x){
    clearChosenLayer120();if(!state.map||!x?.geojson)return;
    try{const layer=L.geoJSON(JSON.parse(x.geojson),{style:{color:'#155f39',weight:4,fillColor:'#4f9b68',fillOpacity:.20}});layer.addTo(state.map);state.layerControl?.addOverlay(layer,'Gleba selecionada');s120.chosenLayer=layer;const b=layer.getBounds();if(b.isValid())state.map.fitBounds(b.pad(.18),{maxZoom:18})}catch(e){}
    if(!state.selectedProperty){
      const list=document.getElementById('areaList109');if(list)list.innerHTML='<div class="area-row109"><div><strong>'+esc(x.name||'Gleba selecionada')+'</strong><span>'+esc(x.purpose||'Consulta avulsa')+'</span><small>'+fmt(x.area_ha,4)+' ha • '+fmt((x.perimeter_m||0)/1000,3)+' km • '+fmt(x.inside_car_pct||0,1)+'% dentro do CAR • temporária</small></div><div class="row-actions"><button class="btn ghost" id="exportSelected120">Salvar KML</button></div></div>';
      el('exportSelected120')?.addEventListener('click',async()=>{try{const p=await api().ExportTemporaryProjectAreaKML();toast('KML salvo em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}});
    }
  }

  function reset120(){clearCandidateLayers120();clearChosenLayer120();s120.result=null;const out=el('areaCandidates120');if(out)out.innerHTML=''}
  document.addEventListener('DOMContentLoaded',()=>{prepare120();const oldReset=window.resetCARWorkspace;if(typeof oldReset==='function'){window.resetCARWorkspace=function(){oldReset();reset120()}}});
})();
