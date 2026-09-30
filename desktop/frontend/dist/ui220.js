/* ViaVerdeCAR 2.2.1 — Projeto Inteligente */
(function(){
  'use strict';
  var projects=[], filterProperty=0, editing=null, areas=[];

  function E(id){return document.getElementById(id)}
  function A(v){return Array.isArray(v)?v:[]}
  function H(v){return String(v==null?'':v).replace(/[&<>"']/g,function(m){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]})}
  function money(v){return (Number(v)||0).toLocaleString('pt-BR',{style:'currency',currency:'BRL'})}
  function n(v,d){return (Number(v)||0).toLocaleString('pt-BR',{minimumFractionDigits:d||0,maximumFractionDigits:d||0})}
  function appAPI(){return typeof api==='function'?api():window.go&&window.go.main&&window.go.main.App}
  function statusLabel(v){return {draft:'Em elaboração',review:'Em revisão',sent:'Enviado ao banco',bank_pending:'Pendência do banco',approved:'Aprovado',contracted:'Contratado',released:'Liberado',rejected:'Reprovado',cancelled:'Cancelado'}[v]||'Em elaboração'}
  function statusTone(v){return {draft:'neutral',review:'warn',sent:'blue',bank_pending:'warn',approved:'ok',contracted:'ok',released:'ok',rejected:'danger',cancelled:'neutral'}[v]||'neutral'}
  function opLabel(v){return {custeio:'Custeio',investimento:'Investimento',aquisicao:'Aquisição',renegociacao:'Renegociação',outro:'Outro'}[v]||'Não informado'}

  function install(){
    if(E('view-projects'))return;
    var main=document.querySelector('.main-area'); if(!main)return;
    var section=document.createElement('section');
    section.className='view'; section.id='view-projects';
    section.innerHTML='<div class="vv220-root"><div class="vv220-loading">Carregando Central de Projetos...</div></div>';
    main.appendChild(section);
    installTopButton();
  }

  function installTopButton(){
    var actions=document.querySelector('.vv204-top-actions');
    if(!actions||E('vv220ProjectsBtn'))return;
    var b=document.createElement('button'); b.id='vv220ProjectsBtn';
    b.innerHTML='<svg class="vv204-icon" viewBox="0 0 24 24"><path d="M3 7h7l2 2h9v10H3V7Z M8 13h8 M12 10v6"/></svg><span>Projetos</span>';
    b.onclick=openProjects;
    actions.insertBefore(b,actions.firstChild);
  }

  async function openProjects(propertyID){
    if(typeof setView==='function')setView('projects');
    filterProperty=Number(propertyID)||Number(state&&state.selectedProperty&&state.selectedProperty.id)||0;
    await refresh();
  }

  async function refresh(){
    var root=document.querySelector('#view-projects .vv220-root'); if(!root)return;
    root.innerHTML='<div class="vv220-loading">Carregando projetos...</div>';
    try{
      if(typeof loadProperties==='function' && (!state||!A(state.properties).length))await loadProperties();
      projects=A(await appAPI().ListRuralProjects(filterProperty||0));
      render();
    }catch(e){
      root.innerHTML='<div class="vv220-error">Não foi possível carregar a Central de Projetos: '+H(String(e))+'</div>';
    }
  }

  function propertyOptions(selected){
    var opts='<option value="0">Todos os imóveis</option>';
    A(state&&state.properties).forEach(function(p){
      opts+='<option value="'+p.id+'" '+(Number(selected)===Number(p.id)?'selected':'')+'>'+H((p.client_name?p.client_name+' — ':'')+p.name+(p.municipality?' • '+p.municipality+'/'+p.uf:''))+'</option>';
    });
    return opts;
  }

  function render(){
    var root=document.querySelector('#view-projects .vv220-root'); if(!root)return;
    var total=projects.length, value=projects.reduce(function(s,p){return s+(Number(p.requested_amount)||0)},0);
    var pending=projects.filter(function(p){return p.status==='bank_pending'||p.status==='review'}).length;
    root.innerHTML=
      '<div class="vv220-head"><div><span>PROJETO INTELIGENTE • 2.2.1</span><h2>Projetos e operações do imóvel</h2><p>O projeto reutiliza CAR, áreas, documentos, análises e crédito rural já existentes no ViaVerdeCAR.</p></div><button id="vv220New" class="primary">+ Novo projeto</button></div>'+
      '<div class="vv220-toolbar"><select id="vv220PropertyFilter">'+propertyOptions(filterProperty)+'</select><button id="vv220Refresh">Atualizar lista</button></div>'+
      '<div class="vv220-stats"><article><span>Projetos</span><strong>'+total+'</strong><small>cadastrados no filtro atual</small></article><article><span>Valor solicitado</span><strong>'+money(value)+'</strong><small>soma dos projetos exibidos</small></article><article><span>Em revisão / pendência</span><strong>'+pending+'</strong><small>exigem acompanhamento</small></article></div>'+
      '<div class="vv220-list">'+(projects.length?projects.map(projectCard).join(''):'<div class="vv220-empty"><strong>Nenhum projeto cadastrado.</strong><span>Crie o primeiro projeto e reaproveite os dados que o ViaVerdeCAR já conhece do imóvel.</span></div>')+'</div>';

    E('vv220New').onclick=function(){openForm(null)};
    E('vv220Refresh').onclick=refresh;
    E('vv220PropertyFilter').onchange=function(){filterProperty=Number(this.value)||0;refresh()};
    root.querySelectorAll('[data-edit-project]').forEach(function(b){b.onclick=function(){var p=projects.find(function(x){return Number(x.id)===Number(b.dataset.editProject)});openForm(p)}});
    root.querySelectorAll('[data-prepare-project]').forEach(function(b){b.onclick=function(){prepareProject(Number(b.dataset.prepareProject))}});
    root.querySelectorAll('[data-delete-project]').forEach(function(b){b.onclick=function(){deleteProject(Number(b.dataset.deleteProject))}});
    root.querySelectorAll('[data-open-property]').forEach(function(b){b.onclick=function(){openPropertyCAR(Number(b.dataset.openProperty))}});
  }

  function projectCard(p){
    var meta=[p.bank,p.credit_line,opLabel(p.operation_type),p.activity].filter(Boolean).join(' • ');
    return '<article class="vv220-card">'+
      '<div class="vv220-card-top"><div><span>'+H(p.client_name)+' • '+H(p.property_name)+'</span><h3>'+H(p.name)+'</h3><p>'+H(meta||'Complete banco, linha, operação e atividade.')+'</p></div><b class="vv220-badge '+statusTone(p.status)+'">'+H(statusLabel(p.status))+'</b></div>'+
      '<div class="vv220-card-data"><div><small>Valor solicitado</small><strong>'+(p.requested_amount?money(p.requested_amount):'—')+'</strong></div><div><small>Área do projeto</small><strong>'+(p.area_ha?n(p.area_ha,2)+' ha':(p.area_id?'Gleba vinculada':'—'))+'</strong></div><div><small>Prazo</small><strong>'+(p.term_months?p.term_months+' meses':'—')+'</strong></div><div><small>Taxa</small><strong>'+(p.interest_rate_pct?n(p.interest_rate_pct,2)+'% a.a.':'—')+'</strong></div></div>'+
      '<div class="vv220-card-actions"><button class="primary" data-prepare-project="'+p.id+'">Preparar projeto</button><button data-open-property="'+p.property_id+'">Abrir imóvel</button><button data-edit-project="'+p.id+'">Editar</button><button class="danger" data-delete-project="'+p.id+'">Excluir projeto</button></div>'+
    '</article>';
  }

  async function openForm(p){
    editing=p||null;
    var props=A(state&&state.properties);
    if(!props.length){try{await loadProperties();props=A(state.properties)}catch(_e){}}
    var propertyID=Number(p&&p.property_id)||filterProperty||Number(state&&state.selectedProperty&&state.selectedProperty.id)||0;
    await loadAreas(propertyID);
    var modal=document.createElement('div'); modal.id='vv220Modal'; modal.className='vv220-modal';
    modal.innerHTML='<div class="vv220-modal-card"><div class="vv220-modal-head"><div><span>'+(p?'EDITAR PROJETO':'NOVO PROJETO')+'</span><h3>'+(p?H(p.name):'Criar projeto rural')+'</h3></div><button id="vv220Close">×</button></div>'+
      '<div class="vv220-form">'+
      '<label class="wide">Imóvel<select id="vv220FProperty">'+projectPropertyOptions(propertyID)+'</select></label>'+
      '<label class="wide">Nome do projeto<input id="vv220FName" value="'+H(p&&p.name||'')+'" placeholder="Ex.: Custeio de Café 2026/27"></label>'+
      '<label>Banco / cooperativa<input id="vv220FBank" value="'+H(p&&p.bank||'')+'" placeholder="Ex.: Banco do Brasil"></label>'+
      '<label>Linha / programa<input id="vv220FLine" value="'+H(p&&p.credit_line||'')+'" placeholder="Ex.: Pronaf Custeio"></label>'+
      '<label>Tipo de operação<select id="vv220FOperation">'+operationOptions(p&&p.operation_type)+'</select></label>'+
      '<label>Atividade<input id="vv220FActivity" list="vv221ActivityOptions" value="'+H(p&&p.activity||'')+'" placeholder="Ex.: Café, Pecuária, Irrigação"><datalist id="vv221ActivityOptions"><option value="Café"><option value="Agricultura"><option value="Pecuária"><option value="Irrigação"><option value="Soja"><option value="Milho"><option value="Misto"></datalist></label>'+
      '<label>Valor solicitado (R$)<input id="vv220FAmount" type="number" min="0" step="0.01" value="'+Number(p&&p.requested_amount||0)+'"></label>'+
      '<label>Prazo (meses)<input id="vv220FTerm" type="number" min="0" step="1" value="'+Number(p&&p.term_months||0)+'"></label>'+
      '<label>Taxa (% a.a.)<input id="vv220FRate" type="number" min="0" step="0.01" value="'+Number(p&&p.interest_rate_pct||0)+'"></label>'+
      '<label>Área informada (ha)<input id="vv220FArea" type="number" min="0" step="0.0001" value="'+Number(p&&p.area_ha||0)+'"></label>'+
      '<label class="wide">Gleba / área já cadastrada<select id="vv220FAreaID">'+areaOptions(p&&p.area_id)+'</select></label>'+
      '<label>Status<select id="vv220FStatus">'+statusOptions(p&&p.status||'draft')+'</select></label>'+
      '<label class="wide">Observações<textarea id="vv220FNotes" rows="4" placeholder="Pendências, condição do banco, garantias ou observações operacionais">'+H(p&&p.notes||'')+'</textarea></label>'+
      '</div><div class="vv220-modal-actions"><button id="vv220Cancel">Cancelar</button><button id="vv220Save" class="primary">Salvar projeto</button></div></div>';
    document.body.appendChild(modal);
    E('vv220Close').onclick=closeModal; E('vv220Cancel').onclick=closeModal;
    E('vv220FProperty').onchange=async function(){await loadAreas(Number(this.value)||0);E('vv220FAreaID').innerHTML=areaOptions(0)};
    E('vv220Save').onclick=saveForm;
  }

  function projectPropertyOptions(selected){
    return A(state&&state.properties).map(function(p){return '<option value="'+p.id+'" '+(Number(selected)===Number(p.id)?'selected':'')+'>'+H((p.client_name?p.client_name+' — ':'')+p.name)+'</option>'}).join('');
  }
  function operationOptions(v){
    return [['','Selecione'],['custeio','Custeio'],['investimento','Investimento'],['aquisicao','Aquisição'],['renegociacao','Renegociação'],['outro','Outro']].map(function(x){return '<option value="'+x[0]+'" '+(x[0]===v?'selected':'')+'>'+x[1]+'</option>'}).join('');
  }
  function statusOptions(v){
    return [['draft','Em elaboração'],['review','Em revisão'],['sent','Enviado ao banco'],['bank_pending','Pendência do banco'],['approved','Aprovado'],['contracted','Contratado'],['released','Liberado'],['rejected','Reprovado'],['cancelled','Cancelado']].map(function(x){return '<option value="'+x[0]+'" '+(x[0]===v?'selected':'')+'>'+x[1]+'</option>'}).join('');
  }
  function areaOptions(selected){
    return '<option value="0">Sem gleba vinculada</option>'+areas.map(function(a){return '<option value="'+a.id+'" '+(Number(selected)===Number(a.id)?'selected':'')+'>'+H(a.name)+' • '+n(a.area_ha,2)+' ha</option>'}).join('');
  }
  async function loadAreas(propertyID){
    areas=[];
    if(!propertyID)return;
    try{areas=A(await appAPI().ListProjectAreas(propertyID))}catch(_e){areas=[]}
  }

  async function saveForm(){
    try{
      var obj={
        id:Number(editing&&editing.id)||0,
        property_id:Number(E('vv220FProperty').value)||0,
        name:E('vv220FName').value,
        bank:E('vv220FBank').value,
        credit_line:E('vv220FLine').value,
        operation_type:E('vv220FOperation').value,
        activity:E('vv220FActivity').value,
        requested_amount:Number(E('vv220FAmount').value)||0,
        term_months:Number(E('vv220FTerm').value)||0,
        interest_rate_pct:Number(E('vv220FRate').value)||0,
        area_ha:Number(E('vv220FArea').value)||0,
        area_id:Number(E('vv220FAreaID').value)||0,
        status:E('vv220FStatus').value,
        notes:E('vv220FNotes').value
      };
      var saved=await appAPI().SaveRuralProject(obj);
      closeModal(); filterProperty=filterProperty||Number(saved.property_id)||0;
      if(typeof toast==='function')toast('Projeto salvo.');
      await refresh();
    }catch(e){if(typeof toast==='function')toast(String(e),true);else alert(String(e))}
  }

  async function deleteProject(id){
    if(!confirm('Excluir somente este projeto? O imóvel, CAR, documentos, áreas e análises serão preservados.'))return;
    try{await appAPI().DeleteRuralProject(id);if(typeof toast==='function')toast('Projeto excluído. Dados do imóvel preservados.');await refresh()}catch(e){if(typeof toast==='function')toast(String(e),true)}
  }

  async function prepareProject(id){
    var modal=document.createElement('div');modal.id='vv220Modal';modal.className='vv220-modal';
    modal.innerHTML='<div class="vv220-modal-card vv220-prep-card"><div class="vv220-loading">Preparando diagnóstico operacional do projeto...</div></div>';
    document.body.appendChild(modal);
    try{
      var r=await appAPI().PrepareRuralProject(id), p=r.project||{};
      var checks=A(r.checks), profile=r.smart_profile||{};
      var smart=checks.filter(function(c){return c.group==='smart'}), base=checks.filter(function(c){return c.group!=='smart'});
      var rules=A(profile.rules);
      modal.innerHTML='<div class="vv220-modal-card vv220-prep-card"><div class="vv220-modal-head"><div><span>PROJETO INTELIGENTE • 2.2.1</span><h3>'+H(p.name||'Projeto')+'</h3></div><button id="vv220Close">×</button></div>'+
        '<div class="vv221-profile '+(profile.ambiguous?'warn':'ok')+'"><div><span>PERFIL DETECTADO</span><strong>'+H(profile.activity_label||'Atividade não classificada')+' • '+H(profile.operation_label||'Operação não informada')+'</strong><p>'+H(profile.message||'')+'</p></div>'+(rules.length?'<div class="vv221-rules">'+rules.map(function(x){return '<b>'+H(x)+'</b>'}).join('')+'</div>':'')+'</div>'+
        '<div class="vv220-readiness"><div><strong>'+Number(r.readiness_pct||0)+'%</strong><span>prontidão operacional</span></div><div><b>'+Number(r.ready||0)+'</b><small>pronto(s)</small></div><div><b>'+Number(r.pending||0)+'</b><small>pendente(s)</small></div><div><b>'+Number(r.review||0)+'</b><small>conferir</small></div></div>'+
        (smart.length?'<div class="vv221-check-title"><strong>Exigências detectadas para este projeto</strong><span>Aplicadas automaticamente conforme atividade e operação.</span></div><div class="vv220-checks vv221-smart-checks">'+smart.map(renderPrepCheck).join('')+'</div>':'')+
        '<div class="vv221-check-title"><strong>Conferências gerais do imóvel</strong><span>Dados reaproveitados da estrutura existente.</span></div><div class="vv220-checks">'+base.map(renderPrepCheck).join('')+'</div>'+
        '<div class="vv220-scope">'+H(r.scope||'')+'</div>'+
        '<div class="vv220-modal-actions"><button id="vv220OpenProperty">Abrir imóvel e análises</button><button id="vv220PrepClose" class="primary">Fechar</button></div></div>';
      E('vv220Close').onclick=closeModal;E('vv220PrepClose').onclick=closeModal;
      E('vv220OpenProperty').onclick=function(){closeModal();openPropertyCAR(Number(p.property_id)||0)};
    }catch(e){modal.innerHTML='<div class="vv220-modal-card"><div class="vv220-error">'+H(String(e))+'</div><div class="vv220-modal-actions"><button id="vv220PrepClose">Fechar</button></div></div>';E('vv220PrepClose').onclick=closeModal}
  }

  function renderPrepCheck(c){
    return '<article class="'+H(c.status)+' '+(c.group==='smart'?'smart':'')+'"><i></i><div><strong>'+H(c.label)+'</strong><p>'+H(c.detail)+'</p><small>'+H(c.source||'')+'</small></div></article>';
  }

  function openPropertyCAR(propertyID){
    var p=A(state&&state.properties).find(function(x){return Number(x.id)===Number(propertyID)});
    if(!p){if(typeof toast==='function')toast('Imóvel não localizado na base local.',true);return}
    if(p.car_number && E('vv204Search')){
      if(typeof setView==='function')setView('dashboard');
      E('vv204Search').value=p.car_number;
      E('vv204Analyze')&&E('vv204Analyze').click();
    } else {
      if(typeof setView==='function')setView('clients');
      if(typeof selectClient==='function')selectClient(p.client_id);
    }
  }

  function closeModal(){var m=E('vv220Modal');if(m)m.remove()}

  window.ViaVerdeProjects={open:openProjects,refresh:refresh};
  document.addEventListener('DOMContentLoaded',function(){setTimeout(install,380);setTimeout(installTopButton,900)});
})();