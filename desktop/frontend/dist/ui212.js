/* ViaVerdeCAR 2.1.2 — interface escura operacional */
(function(){
  'use strict';

  var activeKey='home';
  var summaryLoadToken=0;

  function E(id){return document.getElementById(id)}
  function A(v){return Array.isArray(v)?v:[]}
  function H(v){return String(v==null?'':v).replace(/[&<>"']/g,function(m){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]})}
  function MONEY(v){return (Number(v)||0).toLocaleString('pt-BR',{style:'currency',currency:'BRL'})}
  function api212(){return typeof api==='function'?api():window.go&&window.go.main&&window.go.main.App}
  function result212(){return window.__vv204LastResult||null}

  function svg(path){
    return '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="'+path+'"/></svg>';
  }
  var icons={
    home:'M3.5 11.2 12 4l8.5 7.2v8.3h-6v-5h-5v5h-6v-8.3Z',
    properties:'M3.5 5.5 9 3l6 2.5L20.5 3v15.5L15 21l-6-2.5L3.5 21V5.5Z M9 3v15.5 M15 5.5V21',
    car:'M6 3h8l4 4v14H6V3Z M14 3v5h5 M9 12h6 M9 16h6',
    credit:'M4 6c0-2 3.6-3 8-3s8 1 8 3-3.6 3-8 3-8-1-8-3Z M4 6v6c0 2 3.6 3 8 3s8-1 8-3V6 M4 12v6c0 2 3.6 3 8 3s8-1 8-3v-6',
    docs:'M6 3h8l4 4v14H6V3Z M14 3v5h5 M9 12h6 M9 16h6',
    pending:'M12 3 22 20H2L12 3Z M12 9v4 M12 17h.01',
    analyses:'M4 20V10 M10 20V4 M16 20v-7 M22 20H2',
    map:'M3.5 5.5 9 3l6 2.5L20.5 3v15.5L15 21l-6-2.5L3.5 21V5.5Z M9 3v15.5 M15 5.5V21',
    dossier:'M6 3h8l4 4v14H6V3Z M14 3v5h5 M9 12h6 M9 16h6',
    settings:'M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7Z M19 13.5l2-1.5-2-1.5-.5-2.1.8-2.3-2.4-1.4-1.7 1.7-2.2-.5L12 3.5l-2 2.4-2.2.5-1.7-1.7L3.7 6l.8 2.4L4 10.5 2 12l2 1.5.5 2.1-.8 2.4 2.4 1.4 1.7-1.7 2.2.5 2 2.3 2-2.3 2.2-.5 1.7 1.7 2.4-1.4-.8-2.4.5-2.1Z',
    leaf:'M19.5 3.5C13 3.8 7.2 6.7 4.8 11.7c-1.9 4 .2 7.6 3.7 8.2 4.1.7 7.4-2.3 8.3-6.5.7-3.5 1.4-6.4 2.7-9.9Z M7 18c2.4-4.4 5.3-7.2 9-9.4',
    arrow:'M5 12h14 M14 7l5 5-5 5'
  };

  function navButton(key,label,icon){
    return '<button class="vv212-nav-item" data-vv212-nav="'+key+'"><span>'+svg(icons[icon])+'</span><b>'+H(label)+'</b></button>';
  }

  function buildSidebar(){
    var side=document.querySelector('.sidebar');if(!side)return;
    if(side.dataset.vv212==='1')return;
    side.dataset.vv212='1';
    side.innerHTML=
      '<button class="vv212-side-brand" id="vv212Brand"><span>'+svg(icons.leaf)+'</span><div><strong>ViaVerdeCAR</strong><small>Crédito Rural e Análises Ambientais</small></div></button>'+
      '<nav class="vv212-nav">'+
        navButton('home','Início','home')+
        navButton('properties','Imóveis','properties')+
        navButton('car','CAR','car')+
        navButton('credit','Crédito Rural','credit')+
        navButton('docs','Documentos','docs')+
        navButton('pending','Pendências','pending')+
        navButton('analyses','Análises','analyses')+
        navButton('map','Mapa','map')+
        navButton('dossier','Dossiê','dossier')+
      '</nav>'+
      '<div class="vv212-side-bottom">'+navButton('settings','Configurações','settings')+'<small>ViaVerdeCAR 2.1.2</small><div class="vv212-system-meta"><span id="connectionLabel">Internet</span><span id="versionLabel">Via Verde CAR</span></div></div>';

    E('vv212Brand').onclick=function(){goNav('home')};
    side.querySelectorAll('[data-vv212-nav]').forEach(function(b){
      b.onclick=function(){goNav(b.getAttribute('data-vv212-nav'))};
    });
    setActive(activeKey);
  }

  function setActive(key){
    activeKey=key||'home';
    document.querySelectorAll('[data-vv212-nav]').forEach(function(b){
      b.classList.toggle('active',b.getAttribute('data-vv212-nav')===activeKey);
    });
  }

  function setViewSafe(view){
    try{if(typeof setView==='function'){setView(view);return true}}catch(_e){}
    var original=document.querySelector('.nav-item[data-view="'+view+'"]');
    if(original){original.click();return true}
    return false;
  }

  function clickTab(tab){
    var b=document.querySelector('[data-vv204-tab="'+tab+'"]');
    if(b){b.click();return true}
    return false;
  }

  function goNav(key){
    setActive(key);
    if(key==='properties'){setViewSafe('clients');return}
    if(key==='settings'){setViewSafe('settings');return}
    if(key==='home'){
      setViewSafe('dashboard');
      setTimeout(function(){clickTab('summary')},30);
      return;
    }
    if(key==='car'){
      setViewSafe('dashboard');
      setTimeout(function(){
        if(!clickTab('car'))setViewSafe('car');
      },30);
      return;
    }
    var tab={credit:'credit',docs:'docs',pending:'pending',analyses:'environment',map:'map',dossier:'reports'}[key];
    setViewSafe('dashboard');
    setTimeout(function(){
      if(!clickTab(tab)){
        try{toast('Analise um CAR para abrir '+({'credit':'Crédito Rural','docs':'Documentos','pending':'Pendências','analyses':'Análises','map':'Mapa','dossier':'Dossiê'}[key]||'esta área')+'.')}catch(_e){}
      }
    },30);
  }

  function syncActive(){
    var clients=E('view-clients'),settings=E('view-settings'),carView=E('view-car');
    if(clients&&clients.classList.contains('active'))return setActive('properties');
    if(settings&&settings.classList.contains('active'))return setActive('settings');
    if(carView&&carView.classList.contains('active'))return setActive('car');
    var active=document.querySelector('#vv204Tabs [data-vv204-tab].active');
    if(active){
      var tab=active.getAttribute('data-vv204-tab');
      var key={summary:'home',car:'car',environment:'analyses',land:'analyses',credit:'credit',map:'map',docs:'docs',pending:'pending',reports:'dossier'}[tab]||'home';
      setActive(key);
    }
  }

  function envHitCount(r){
    var keys=['ibama','mapbiomas','inpe_fire','funai','icmbio','mcr'],env=r&&r.environmental||{},sources=env.sources||{};
    return keys.reduce(function(n,k){
      var s=sources[k]||env[k]||{};
      return n+(s.status==='hit'?(Number(s.count)||1):0);
    },0);
  }

  function sideCard(tone,label,value,detail,key){
    return '<button class="vv212-summary-card '+tone+'" data-vv212-side="'+key+'"><span>'+svg(icons[key==='credit'?'credit':key==='docs'?'docs':'pending'])+'</span><div><small>'+H(label)+'</small><strong>'+H(value)+'</strong><em>'+H(detail||'')+'</em></div>'+svg(icons.arrow)+'</button>';
  }

  function buildSummarySide(r){
    var pane=E('vv204Pane');if(!pane)return;
    var sic=r&&r.xray&&r.xray.sicor||{},hits=envHitCount(r),pid=Number(r&&r.property_id)||0;
    var signature=[String(r&&r.car&&r.car.car||''),pid,Number(sic.operation_count)||0,Number(sic.total_credit_value)||0,hits].join('|');
    var old=E('vv212SummarySide');
    if(old&&old.dataset.signature===signature)return;
    if(old)old.remove();
    var side=document.createElement('aside');
    side.id='vv212SummarySide';side.className='vv212-summary-side';side.dataset.signature=signature;
    side.innerHTML=
      '<div class="vv212-summary-side-title"><span>'+svg(icons.analyses)+'</span><strong>Resumo do Imóvel</strong></div>'+
      sideCard('green','Crédito Rural',(Number(sic.operation_count)||0)+' operação(ões)',sic.total_credit_value?MONEY(sic.total_credit_value)+' contratados':'Dados públicos SICOR','credit')+
      sideCard('blue','Documentos',pid?'Carregando...':'Vincule o imóvel',pid?'Consultando central documental':'Necessário para armazenar arquivos','docs')+
      sideCard('red','Pendências',pid?'Carregando...':'Vincule o imóvel',pid?'Verificando checklist':'Necessário para checklist persistente','pending')+
      '<div class="vv212-last-analysis"><div><span>'+svg(icons.analyses)+'</span><strong>Últimas análises</strong></div>'+
        '<p><i class="'+(hits?'warn':'ok')+'"></i><span>Ambiental</span><b>'+(hits?hits+' ocorrência(s)':'Sem ocorrência nas bases respondidas')+'</b></p>'+
        '<p><i class="ok"></i><span>Crédito Rural</span><b>'+(Number(sic.operation_count)||0)+' operação(ões)</b></p>'+
        '<p><i class="'+(r&&r.car&&r.car.has_geometry?'ok':'neutral')+'"></i><span>Geometria CAR</span><b>'+(r&&r.car&&r.car.has_geometry?'Disponível':'Não disponível')+'</b></p>'+
      '</div>'+
      '<button class="vv212-dossier" data-vv212-side="dossier">'+svg(icons.dossier)+'<strong>Gerar Dossiê</strong>'+svg(icons.arrow)+'</button>';

    pane.appendChild(side);
    side.querySelectorAll('[data-vv212-side]').forEach(function(b){b.onclick=function(){goNav(b.getAttribute('data-vv212-side'))}});
    loadDocumentSummary(r,side);
  }

  async function loadDocumentSummary(r,side){
    var pid=Number(r&&r.property_id)||0;if(!pid||!side)return;
    var token=++summaryLoadToken;
    try{
      var d=await api212().GetPropertyDocumentCenter(pid);
      if(token!==summaryLoadToken||!document.body.contains(side))return;
      var s=d&&d.summary||{},received=Number(s.received)||0,pending=(Number(s.pending)||0)+(Number(s.expired)||0),review=Number(s.review)||0;
      var docs=side.querySelector('[data-vv212-side="docs"]');
      var pend=side.querySelector('[data-vv212-side="pending"]');
      if(docs){
        docs.querySelector('strong').textContent=received+' recebido(s)';
        docs.querySelector('em').textContent=(review?review+' para conferir':'Checklist documental');
      }
      if(pend){
        pend.querySelector('strong').textContent=pending+' pendência(s)';
        pend.querySelector('em').textContent=review?review+' item(ns) para conferir':'Checklist atualizado';
        pend.classList.toggle('red',pending>0);
        pend.classList.toggle('green',pending===0);
      }
    }catch(_e){
      var docs2=side.querySelector('[data-vv212-side="docs"]');
      var pend2=side.querySelector('[data-vv212-side="pending"]');
      if(docs2){docs2.querySelector('strong').textContent='Fonte local indisponível';docs2.querySelector('em').textContent='Tente novamente'}
      if(pend2){pend2.querySelector('strong').textContent='Não verificado';pend2.querySelector('em').textContent='Checklist não carregado'}
    }
  }

  function enhanceSummary(){
    var pane=E('vv204Pane');if(!pane)return;
    var grid=pane.querySelector('.vv204-summary-grid');
    if(!grid){pane.classList.remove('vv212-summary-mode');return}
    pane.classList.add('vv212-summary-mode');
    var property=grid.querySelector('.vv204-property-card');
    if(property)property.classList.add('vv212-property-hero');
    var map=grid.querySelector('.vv204-map-card');
    if(map)map.classList.add('vv212-main-map');
    var r=result212();if(r)buildSummarySide(r);
  }

  function simplifyTopbar(){
    var search=E('vv204Search');if(search)search.placeholder='Buscar CAR, imóvel, CPF/CNPJ ou município...';
  }

  function install(){
    document.body.classList.add('vv212-dark');
    buildSidebar();
    simplifyTopbar();
    syncActive();
    enhanceSummary();
  }

  function watch(){
    install();
    var mo=new MutationObserver(function(){
      try{buildSidebar();syncActive();enhanceSummary()}catch(_e){}
    });
    mo.observe(document.body,{childList:true,subtree:true,attributes:true,attributeFilter:['class']});
  }

  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',watch);else watch();
})();