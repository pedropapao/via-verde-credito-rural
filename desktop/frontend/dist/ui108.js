/* ViaVerdeCAR 1.0.8 — camada aditiva de usabilidade, sem alterar contratos do backend. */
(function(){
  const byId=id=>document.getElementById(id);
  const addState=(id,state)=>{const el=byId(id);if(!el)return;el.classList.remove('ok','warning','info','error');el.classList.add(state)};

  function prepareUI108(){
    const strip=document.querySelector('.professional-strip');
    if(strip){strip.id='professionalStrip';strip.classList.add('no-kml');const cards=strip.querySelectorAll('.metric-card');cards.forEach((c,i)=>c.classList.add(i<3?'comparison-metric-card':'history-metric-card'))}
    const main=document.querySelector('.professional-main');if(main?.firstElementChild)main.firstElementChild.classList.add('professional-copy');
    const actions=document.querySelector('.professional-actions');
    if(actions&&!byId('quickExportKmlBtn')){
      const k=document.createElement('button');k.className='btn ghost quick-action';k.id='quickExportKmlBtn';k.disabled=true;k.textContent='Gerar KML';k.onclick=()=>exportKML();
      const r=document.createElement('button');r.className='btn ghost quick-action';r.id='quickReportBtn';r.disabled=true;r.textContent='Gerar relatório';r.onclick=()=>exportReport();
      actions.prepend(r);actions.prepend(k);
    }
    const kmlCard=document.querySelectorAll('.analysis-column .result-card')[1];if(kmlCard){kmlCard.id='kmlAnalysisCard';kmlCard.classList.add('hidden')}
    const envCards=document.querySelectorAll('.environment-grid .environment-card');['envIbamaCard','envFunaiCard','envICMBioCard','envMCRCard'].forEach((id,i)=>{if(envCards[i])envCards[i].id=id});
    const themesGrid=document.querySelector('.themes-grid');if(themesGrid)themesGrid.id='themesGrid';
    const warnings=byId('themesWarnings');
    if(themesGrid&&warnings&&!byId('themesUnavailable')){
      const box=document.createElement('div');box.id='themesUnavailable';box.className='themes-unavailable hidden';box.innerHTML='<div><strong>Temas ambientais do SICAR temporariamente indisponíveis</strong><span id="themesUnavailableSummary">APP, Reserva Legal, Vegetação Nativa, Área Consolidada, Uso Restrito e Servidão não puderam ser obtidos nesta consulta.</span></div><div class="themes-unavailable-actions"><button class="btn ghost" id="retryThemesBtn">Tentar novamente</button><button class="text-btn" id="toggleThemeDetailsBtn">Ver detalhes técnicos</button></div>';
      themesGrid.after(box);box.after(warnings);byId('retryThemesBtn').onclick=retryThemes108;byId('toggleThemeDetailsBtn').onclick=()=>{const hidden=warnings.classList.toggle('hidden');byId('toggleThemeDetailsBtn').textContent=hidden?'Ver detalhes técnicos':'Ocultar detalhes técnicos'};
    }
    const bottom=document.querySelector('.professional-bottom');const conference=bottom?.querySelector('article:first-child');
    if(conference){conference.classList.add('conference-panel');const eye=conference.querySelector('.eyebrow');const h=conference.querySelector('h3');if(eye)eye.textContent='CONFERÊNCIA AUTOMÁTICA';if(h)h.textContent='Resumo das validações';const p=document.createElement('p');p.textContent='Resultados compactos para leitura rápida. Ocorrências e indisponibilidades continuam detalhadas nas seções acima.';conference.querySelector('.panel-title>div')?.appendChild(p);byId('qualityChecks')?.classList.add('quality-list-compact')}
  }

  async function retryThemes108(){
    if(!state.car?.found||!state.car?.geojson){toast('Consulte um CAR com geometria antes de tentar novamente.',true);return}
    const btn=byId('retryThemesBtn');if(btn){btn.disabled=true;btn.textContent='Consultando...'}
    const summary=byId('themesUnavailableSummary');
    if(summary)summary.textContent='Tentando novamente apenas os temas do SICAR. Em alguns municípios o serviço público pode levar alguns minutos; o restante do aplicativo continua preservado.';
    try{
      const propertyID=state.selectedProperty?.id||0;
      const themes=await api().RetrySICARThemes(propertyID);
      state.car.themes=themes||{};
      renderThemes(state.car.themes);
      const available=Object.values(state.car.themes?.themes||{}).filter(x=>x?.available).length;
      toast(available?available+' tema(s) SICAR carregado(s).':'O serviço público do SICAR continuou indisponível nesta tentativa.',!available);
    }catch(e){
      toast(String(e),true);
      if(summary)summary.textContent='A nova tentativa não conseguiu obter os pacotes públicos. O CAR, mapa e demais análises continuam válidos.';
    }finally{
      if(btn){btn.disabled=false;btn.textContent='Tentar novamente'}
    }
  }

  function installOverrides(){
    const originalRenderCAR=renderCAR, originalRenderChecks=renderChecks, originalRenderKML=renderKML, originalReset=resetCARWorkspace, originalProfessional=updateProfessional, originalThemes=renderThemes, originalEnvironment=renderEnvironment;

    renderChecks=function(checks){const list=[...(checks||[])],theme=list.filter(c=>(c.title||'').trim().toLowerCase()==='temas sicar'),other=list.filter(c=>(c.title||'').trim().toLowerCase()!=='temas sicar');if(theme.length>1){const failed=theme.filter(c=>c.level!=='ok'),ok=theme.length-failed.length;other.push({level:failed.length?'info':'ok',title:'Temas SICAR',detail:failed.length===theme.length?`${theme.length} temas não puderam ser obtidos nesta consulta. Veja os detalhes na seção Temas declarados do SICAR.`:`${ok} tema(s) consultado(s) e ${failed.length} indisponível(is). Veja os detalhes na seção Temas declarados do SICAR.`})}else other.push(...theme);originalRenderChecks(other)};

    renderCAR=function(r){originalRenderCAR(r);if(!r.property_name&&!state.selectedProperty?.name)byId('rPropertyName').textContent='Não informado pela fonte pública';if(!r.owner_data_access)byId('rOwnerAccess').textContent='Não disponibilizado na consulta pública';if(byId('quickExportKmlBtn'))byId('quickExportKmlBtn').disabled=!r.has_geometry;if(byId('quickReportBtn'))byId('quickReportBtn').disabled=!state.selectedProperty||!r.found};

    renderKML=function(r){byId('kmlAnalysisCard')?.classList.remove('hidden');byId('professionalStrip')?.classList.remove('no-kml');originalRenderKML(r)};

    resetCARWorkspace=function(){originalReset();byId('kmlAnalysisCard')?.classList.add('hidden');byId('professionalStrip')?.classList.add('no-kml');if(byId('quickExportKmlBtn'))byId('quickExportKmlBtn').disabled=true;if(byId('quickReportBtn'))byId('quickReportBtn').disabled=true};

    updateProfessional=function(){originalProfessional();byId('professionalStrip')?.classList.toggle('no-kml',!state.kml);if(state.car?.found&&!state.kml){byId('professionalSummary').textContent=state.car.auto_kml_path?'O perímetro público foi convertido para KML e salvo. O KML externo é opcional e só aparece quando houver uma comparação independente.':'A consulta pública retornou a geometria. Você já pode gerar o KML do SICAR; carregue um KML externo apenas se quiser comparar as geometrias.'}};

    renderThemes=function(themes){originalThemes(themes);const grid=byId('themesGrid'),unavailable=byId('themesUnavailable'),warnings=byId('themesWarnings'),toggle=byId('toggleThemeDetailsBtn');if(!grid||!unavailable||!warnings)return;if(!themes?.checked_at){grid.classList.remove('hidden');unavailable.classList.add('hidden');warnings.classList.add('hidden');if(toggle)toggle.textContent='Ver detalhes técnicos';return}const map=themes.themes||{},available=Object.values(map).filter(x=>x?.available).length,raw=themes.warnings||[];byId('themesBadge').className='status-badge '+(available>=4?'ok':available?'warning':'info');if(!available){grid.classList.add('hidden');unavailable.classList.remove('hidden');warnings.classList.add('hidden');if(toggle)toggle.textContent='Ver detalhes técnicos';byId('themesUnavailableSummary').textContent='APP, Reserva Legal, Vegetação Nativa, Área Consolidada, Uso Restrito e Servidão não puderam ser obtidos na tentativa rápida. Clique em Tentar novamente para uma consulta dedicada, que pode levar alguns minutos. O restante da análise do CAR continua válido.'}else{grid.classList.remove('hidden');unavailable.classList.add('hidden');warnings.classList.toggle('hidden',!raw.length)}};

    renderEnvironment=function(env){originalEnvironment(env);if(!env?.checked_at){['envIbamaCard','envFunaiCard','envICMBioCard','envMCRCard'].forEach(id=>addState(id,'info'));return}const ibama=Number(env.ibama_embargo_count||0),funai=Number(env.indigenous_count||0),uc=Number(env.federal_uc_count||0);if(env.ibama_checked){byId('envIbamaDetail').textContent=ibama?'Interseção encontrada':'✓ Consultado • nenhuma interseção encontrada';addState('envIbamaCard',ibama?'warning':'ok')}else addState('envIbamaCard','info');if(env.funai_checked){byId('envFunaiDetail').textContent=funai?'Interseção com Terra Indígena':'✓ Consultado • nenhuma interseção encontrada';addState('envFunaiCard',funai?'warning':'ok')}else addState('envFunaiCard','info');if(env.icmbio_checked){byId('envICMBioDetail').textContent=uc?'Interseção com UC federal':'✓ Consultado • nenhuma interseção encontrada';addState('envICMBioCard',uc?'warning':'ok')}else addState('envICMBioCard','info');if(env.mcr_checked){byId('envMCRDetail').textContent=env.mcr_listed?'CAR localizado na lista pública MMA/MCR-PRODES':'✓ Consultado • CAR não localizado na lista pública';addState('envMCRCard',env.mcr_listed?'warning':'ok')}else addState('envMCRCard','info')};
  }

  document.addEventListener('DOMContentLoaded',()=>{prepareUI108();installOverrides();});
})();
