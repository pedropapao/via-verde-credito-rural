/* ViaVerdeCAR 2.3.0 — Automação Técnica do Projeto */
(function(){
  'use strict';

  function E(id){return document.getElementById(id)}
  function A(v){return Array.isArray(v)?v:[]}
  function H(v){return String(v==null?'':v).replace(/[&<>"']/g,function(m){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]})}
  function appAPI(){return typeof api==='function'?api():window.go&&window.go.main&&window.go.main.App}
  function n(v,d){return (Number(v)||0).toLocaleString('pt-BR',{minimumFractionDigits:d||0,maximumFractionDigits:d||0})}
  function statusMeta(v){
    return {
      ready:['Concluído','ok'],
      pending:['Pendente','danger'],
      review:['Conferir','warn'],
      source_unavailable:['Fonte indisponível','muted'],
      not_configured:['Não configurado','muted'],
      not_applicable:['Não se aplica','neutral'],
      not_run:['Não consultado','neutral'],
      error:['Erro','danger']
    }[v]||['Conferir','warn'];
  }

  function installSettings(){
    var grid=document.querySelector('#view-settings .settings-grid');
    if(!grid||E('vv230IntegrationSettings'))return;
    var card=document.createElement('article');
    card.id='vv230IntegrationSettings'; card.className='panel vv230-settings';
    card.innerHTML=
      '<span class="eyebrow">AUTOMAÇÃO TÉCNICA • 2.3.0</span><h3>Credenciais das fontes</h3>'+
      '<p>Agritec e SATVeg usam o mesmo token da AgroAPI/Embrapa. O HidroWebService da ANA exige credencial própria. Sem credencial, o ViaVerdeCAR continua funcionando e marca a fonte como não configurada.</p>'+
      '<div class="vv230-config-state"><span>Embrapa <b id="vv230EmbrapaState">—</b></span><span>ANA <b id="vv230ANAState">—</b></span></div>'+
      '<div class="vv230-settings-form">'+
        '<label>Token AgroAPI / Embrapa<input id="vv230EmbrapaToken" type="password" autocomplete="off" placeholder="Cole um novo token para configurar ou substituir"></label>'+
        '<label>Identificador ANA<input id="vv230ANAUser" autocomplete="off" placeholder="CPF/CNPJ ou identificador liberado pela ANA"></label>'+
        '<label>Senha ANA<input id="vv230ANAPass" type="password" autocomplete="off" placeholder="Digite apenas para configurar ou substituir"></label>'+
      '</div>'+
      '<div class="form-actions"><button class="btn primary" id="vv230SaveSettings">Salvar integrações</button><button class="btn ghost" id="vv230ClearEmbrapa">Limpar Embrapa</button><button class="btn ghost" id="vv230ClearANA">Limpar ANA</button></div>'+
      '<small class="vv230-settings-note">As credenciais ficam somente na base local do ViaVerdeCAR deste computador. Elas não são enviadas para o GitHub nem incluídas no código-fonte.</small>';
    grid.appendChild(card);
    E('vv230SaveSettings').onclick=saveSettings;
    E('vv230ClearEmbrapa').onclick=function(){saveSettings(true,false)};
    E('vv230ClearANA').onclick=function(){saveSettings(false,true)};
    loadSettings();
  }

  async function loadSettings(){
    if(!E('vv230IntegrationSettings'))return;
    try{
      var s=await appAPI().GetProjectIntegrationSettings();
      E('vv230EmbrapaState').textContent=s&&s.embrapa_configured?'Configurada':'Não configurada';
      E('vv230ANAState').textContent=s&&s.ana_configured?'Configurada':'Não configurada';
      if(E('vv230ANAUser')&&!E('vv230ANAUser').value)E('vv230ANAUser').value=s&&s.ana_username||'';
    }catch(e){
      E('vv230EmbrapaState').textContent='Indisponível';
      E('vv230ANAState').textContent='Indisponível';
    }
  }

  async function saveSettings(clearEmbrapa,clearANA){
    try{
      var input={
        embrapa_token:E('vv230EmbrapaToken')?E('vv230EmbrapaToken').value:'',
        ana_username:E('vv230ANAUser')?E('vv230ANAUser').value:'',
        ana_password:E('vv230ANAPass')?E('vv230ANAPass').value:'',
        clear_embrapa:!!clearEmbrapa,
        clear_ana:!!clearANA
      };
      await appAPI().SaveProjectIntegrationSettings(input);
      if(E('vv230EmbrapaToken'))E('vv230EmbrapaToken').value='';
      if(E('vv230ANAPass'))E('vv230ANAPass').value='';
      if(typeof toast==='function')toast('Configuração das integrações atualizada.');
      await loadSettings();
    }catch(e){if(typeof toast==='function')toast(String(e),true);else alert(String(e))}
  }

  function sourceCard(title,status,detail,extra){
    var meta=statusMeta(status);
    return '<article class="vv230-source-card '+H(meta[1])+'"><div class="vv230-source-head"><strong>'+H(title)+'</strong><b>'+H(meta[0])+'</b></div><p>'+H(detail||'Sem detalhe.')+'</p>'+(extra||'')+'</article>';
  }

  function satvegChart(result){
    var vals=A(result&&result.values);
    if(vals.length<2)return '';
    var width=700,height=150,pad=12;
    var min=Math.min.apply(null,vals),max=Math.max.apply(null,vals);
    if(!isFinite(min)||!isFinite(max))return '';
    if(max===min){max=min+0.01}
    var step=(width-pad*2)/(vals.length-1);
    var pts=vals.map(function(v,i){
      var x=pad+i*step, y=height-pad-((Number(v)-min)/(max-min))*(height-pad*2);
      return x.toFixed(1)+','+y.toFixed(1);
    }).join(' ');
    return '<div class="vv230-chart"><svg viewBox="0 0 '+width+' '+height+'" preserveAspectRatio="none"><polyline points="'+pts+'" fill="none" stroke="currentColor" stroke-width="2"/></svg><div><span>NDVI mín. '+n(min,3)+'</span><span>NDVI máx. '+n(max,3)+'</span><span>'+H(result.first_date||'')+' → '+H(result.last_date||'')+'</span></div></div>';
  }

  function agritecExtra(r){
    var parts=[];
    if(r&&r.culture&&r.culture.id){
      parts.push('<div class="vv230-mini"><b>Cultura Agritec</b><span>'+H(r.culture.id+' • '+(r.culture.full_name||r.culture.name||''))+'</span></div>');
    }
    var cvs=A(r&&r.cultivars);
    if(cvs.length){
      parts.push('<div class="vv230-mini"><b>Cultivares localizadas</b><span>'+cvs.slice(0,8).map(H).join(' • ')+'</span></div>');
    }
    if(r&&r.productivity_ran){
      parts.push('<div class="vv230-mini"><b>Produtividade Agritec</b><span>Último valor '+n(r.productivity_latest,3)+' t/ha • média '+n(r.productivity_mean,3)+' t/ha</span></div>');
    }
    return parts.join('');
  }

  function anaExtra(r){
    var stations=A(r&&r.stations),html='';
    if(stations.length){
      html+='<div class="vv230-stations">'+stations.slice(0,5).map(function(s){return '<span><b>'+H(s.name||s.code)+'</b> '+n(s.distance_km,1)+' km • '+H(s.type||'')+'</span>'}).join('')+'</div>';
    }
    if(r&&r.rain_value_count){
      html+='<div class="vv230-mini"><b>Chuva</b><span>'+H(r.period_start)+' a '+H(r.period_end)+' • '+r.rain_value_count+' valores • soma bruta '+n(r.rain_total_mm,1)+' mm</span></div>';
    }
    if(r&&r.flow_value_count){
      html+='<div class="vv230-mini"><b>Vazão</b><span>'+H(r.period_start)+' a '+H(r.period_end)+' • '+r.flow_value_count+' valores • média '+n(r.flow_mean_m3s,2)+' m³/s</span></div>';
    }
    return html;
  }

  function pendingHTML(items){
    items=A(items);
    if(!items.length)return '<div class="vv230-none">Nenhuma pendência operacional detectada nas verificações contabilizadas.</div>';
    return '<div class="vv230-pending">'+items.map(function(p){
      return '<article class="'+H(p.severity||'attention')+'"><i></i><div><strong>'+H(p.label)+'</strong><p>'+H(p.detail)+'</p><small>'+H(p.source||'')+'</small></div></article>';
    }).join('')+'</div>';
  }

  async function runAutomation(projectID,force){
    var modal=document.createElement('div');modal.id='vv230AutomationModal';modal.className='vv220-modal';
    modal.innerHTML='<div class="vv220-modal-card vv230-auto-card"><div class="vv220-loading">Executando CAR, ZARC, Agritec, ANA, SATVeg e pendências do projeto...</div></div>';
    document.body.appendChild(modal);
    try{
      var r=await appAPI().RunRuralProjectAutomation(projectID,!!force);
      var p=r.preparation&&r.preparation.project||{},prep=r.preparation||{};
      modal.innerHTML='<div class="vv220-modal-card vv230-auto-card">'+
        '<div class="vv220-modal-head"><div><span>AUTOMAÇÃO TÉCNICA • 2.3.0</span><h3>'+H(p.name||'Projeto')+'</h3><p>'+H(r.used_cache?'Resultado recente reutilizado do cache local.':'Fontes executadas nesta preparação conforme aplicabilidade e configuração.')+'</p></div><button id="vv230AutoClose">×</button></div>'+
        '<div class="vv220-readiness"><div><strong>'+Number(prep.readiness_pct||0)+'%</strong><span>prontidão operacional</span></div><div><b>'+Number(prep.ready||0)+'</b><small>pronto(s)</small></div><div><b>'+Number(prep.pending||0)+'</b><small>pendente(s)</small></div><div><b>'+Number(prep.review||0)+'</b><small>conferir</small></div></div>'+
        '<div class="vv230-source-grid">'+
          sourceCard('ZARC • MAPA',r.zarc&&r.zarc.status,r.zarc&&r.zarc.detail,'')+
          sourceCard('Agritec • Embrapa',r.agritec&&r.agritec.status,r.agritec&&r.agritec.detail,agritecExtra(r.agritec))+
          sourceCard('ANA • HidroWebService',r.ana&&r.ana.status,r.ana&&r.ana.detail,anaExtra(r.ana))+
          sourceCard('SATVeg • NDVI',r.satveg&&r.satveg.status,r.satveg&&r.satveg.detail,satvegChart(r.satveg))+
        '</div>'+
        '<div class="vv221-check-title"><strong>Pendências e próximas ações</strong><span>Geradas a partir do projeto, documentos e análises executadas.</span></div>'+
        pendingHTML(r.pending)+
        '<div class="vv220-scope">'+H(r.scope||'')+'</div>'+
        '<div class="vv220-modal-actions"><button id="vv230SettingsBtn">Configurar fontes</button><button id="vv230DossierBtn">Gerar dossiê</button><button id="vv230ForceBtn">Atualizar fontes</button><button id="vv230AutoDone" class="primary">Fechar</button></div>'+
      '</div>';
      E('vv230AutoClose').onclick=closeAutomation;
      E('vv230AutoDone').onclick=closeAutomation;
      E('vv230ForceBtn').onclick=function(){closeAutomation();runAutomation(projectID,true)};
      E('vv230DossierBtn').onclick=function(){dossier(projectID)};
      E('vv230SettingsBtn').onclick=function(){closeAutomation();if(typeof setView==='function')setView('settings');setTimeout(installSettings,100)};
    }catch(e){
      modal.innerHTML='<div class="vv220-modal-card"><div class="vv220-error">Não foi possível executar a automação técnica: '+H(String(e))+'</div><div class="vv220-modal-actions"><button id="vv230AutoDone">Fechar</button></div></div>';
      E('vv230AutoDone').onclick=closeAutomation;
    }
  }

  function closeAutomation(){var m=E('vv230AutomationModal');if(m)m.remove()}

  async function dossier(projectID){
    try{
      var path=await appAPI().ExportRuralProjectDossierPDF(projectID,false);
      if(typeof toast==='function')toast('Dossiê salvo em '+path);
    }catch(e){if(!String(e).toLowerCase().includes('cancelada')){if(typeof toast==='function')toast(String(e),true);else alert(String(e))}}
  }

  window.ViaVerdeTechnicalAutomation={run:runAutomation,dossier:dossier,settings:loadSettings};

  document.addEventListener('DOMContentLoaded',function(){
    setTimeout(installSettings,700);
  });
  var mo=new MutationObserver(function(){
    if(document.querySelector('#view-settings.active')||document.querySelector('#view-settings .settings-grid'))installSettings();
  });
  setTimeout(function(){mo.observe(document.body,{childList:true,subtree:true})},400);
})();