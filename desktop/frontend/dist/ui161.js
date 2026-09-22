/* ViaVerdeCAR 1.6.1 — refinamentos de experiência sem alterar regras de negócio */
(function(){
  const iconByTab={
    summary:'▤',
    xray:'◎',
    map:'⌖',
    planning:'◫',
    environment:'♧',
    credit:'R$',
    docs:'▧'
  };

  function addTabIcons(){
    document.querySelectorAll('.car-tab131').forEach(btn=>{
      if(btn.querySelector('.car-tab161-icon'))return;
      const key=btn.dataset.carTab131||'';
      const icon=iconByTab[key];
      if(!icon)return;
      const span=document.createElement('span');
      span.className='car-tab161-icon';
      span.textContent=icon;
      btn.prepend(span);
    });
  }

  function makeCollapsible(card,topSelector,index){
    if(!card||card.dataset.ui161Enhanced==='1')return;
    const top=card.querySelector(topSelector);
    if(!top)return;
    card.dataset.ui161Enhanced='1';
    const button=document.createElement('button');
    button.type='button';
    button.className='ui161-detail-toggle';
    const collapsed=index>0;
    card.classList.toggle('ui161-collapsed',collapsed);
    button.textContent=collapsed?'Detalhes':'Recolher';
    button.setAttribute('aria-expanded',collapsed?'false':'true');
    button.addEventListener('click',ev=>{
      ev.stopPropagation();
      const willCollapse=!card.classList.contains('ui161-collapsed');
      card.classList.toggle('ui161-collapsed',willCollapse);
      button.textContent=willCollapse?'Detalhes':'Recolher';
      button.setAttribute('aria-expanded',willCollapse?'false':'true');
    });
    top.appendChild(button);
  }

  function enhanceXRay(){
    document.querySelectorAll('.xray-operation150').forEach((card,i)=>makeCollapsible(card,'.xray-operation-top150',i));
    document.querySelectorAll('.sigef-card150').forEach((card,i)=>makeCollapsible(card,'.sigef-top150',i));
  }

  function prepare(){
    document.body.classList.add('vv-ui161');
    addTabIcons();
    enhanceXRay();

    const workspace=document.getElementById('xrayWorkspace150');
    if(workspace){
      new MutationObserver(()=>enhanceXRay()).observe(workspace,{childList:true,subtree:true});
    }

    const tabs=document.querySelector('.car-tabs131');
    if(tabs){
      new MutationObserver(()=>addTabIcons()).observe(tabs,{childList:true,subtree:true});
    }
  }

  if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',prepare);
  else prepare();
})();