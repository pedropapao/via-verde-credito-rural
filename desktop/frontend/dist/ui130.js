/* ViaVerdeCAR 1.3.0 — camada visual ESA WorldCover 2021 */
(function(){
  const byId=id=>document.getElementById(id);
  const s130={worldCover:null,visible:false};

  function waitForMap130(){
    if(state.map&&state.layerControl){prepareWorldCover130();return}
    setTimeout(waitForMap130,250);
  }

  function prepareWorldCover130(){
    if(s130.worldCover||!window.L||!state.map)return;
    try{
      s130.worldCover=L.tileLayer.wms('https://services.terrascope.be/wms/v2',{
        layers:'WORLDCOVER_2021_MAP',
        format:'image/png',
        transparent:true,
        opacity:.58,
        version:'1.3.0',
        attribution:'© ESA WorldCover 2021 / Contains modified Copernicus Sentinel data'
      });
      state.layerControl.addOverlay(s130.worldCover,'Cobertura do solo • ESA WorldCover 2021');
    }catch(e){return}

    const head=document.querySelector('.area-intelligence-head120');
    if(head&&!byId('worldCoverToggle130')){
      const tools=document.createElement('div');tools.className='landcover-tools130';
      tools.innerHTML='<button class="btn ghost" id="worldCoverToggle130">Mostrar cobertura do solo</button><span class="terrain-pill130">Relevo DEM 90 m</span>';
      head.appendChild(tools);
      byId('worldCoverToggle130').onclick=toggleWorldCover130;
    }

    const intelligence=byId('areaIntelligence120');
    if(intelligence&&!byId('worldCoverLegend130')){
      const legend=document.createElement('div');legend.id='worldCoverLegend130';legend.className='landcover-legend130 hidden';
      const items=[
        ['#006400','Árvores'],['#ffbb22','Arbustos'],['#ffff4c','Pastagem/gramínea'],
        ['#f096ff','Cultivo'],['#fa0000','Área construída'],['#b4b4b4','Solo exposto/esparso'],
        ['#f0f0f0','Neve/gelo'],['#0064c8','Água permanente'],['#0096a0','Área úmida herbácea'],
        ['#00cf75','Manguezal'],['#fae6a0','Musgos/líquens']
      ];
      legend.innerHTML=items.map(x=>'<div class="landcover-item130"><span class="landcover-swatch130" style="background:'+x[0]+'"></span>'+x[1]+'</div>').join('')+
        '<div class="landcover-note130"><strong>ESA WorldCover 2021, 10 m:</strong> camada visual global de cobertura do solo. Ela ajuda a interpretar o mapa, mas nesta versão não entra sozinha como conclusão técnica nem substitui vistoria, classificação agronômica ou os temas declarados do SICAR.</div>';
      intelligence.appendChild(legend);
    }
  }

  function toggleWorldCover130(){
    if(!s130.worldCover||!state.map)return;
    s130.visible=!s130.visible;
    if(s130.visible) s130.worldCover.addTo(state.map); else state.map.removeLayer(s130.worldCover);
    const btn=byId('worldCoverToggle130');if(btn)btn.textContent=s130.visible?'Ocultar cobertura do solo':'Mostrar cobertura do solo';
    byId('worldCoverLegend130')?.classList.toggle('hidden',!s130.visible);
  }

  document.addEventListener('DOMContentLoaded',()=>setTimeout(waitForMap130,500));
})();
