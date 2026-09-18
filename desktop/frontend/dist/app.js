const state = {
  clients: [], properties: [], selectedClient: null, selectedProperty: null,
  car: null, kml: null, comparison: null, map: null, carLayer: null, kmlLayer: null,
  update: null,
};

const $ = (id) => document.getElementById(id);
const fmt = (v, d=2) => (Number(v)||0).toLocaleString('pt-BR',{minimumFractionDigits:d,maximumFractionDigits:d});
const api = () => window.go?.main?.App;

function toast(msg, error=false){
  const el=$('toast'); el.textContent=msg; el.className='toast show'+(error?' error':'');
  clearTimeout(window.__toastTimer); window.__toastTimer=setTimeout(()=>el.className='toast',3200);
}

function setView(name){
  document.querySelectorAll('.view').forEach(v=>v.classList.remove('active'));
  document.querySelectorAll('.nav-item').forEach(v=>v.classList.toggle('active',v.dataset.view===name));
  $('view-'+name).classList.add('active');
  $('pageTitle').textContent={dashboard:'Visão geral',clients:'Clientes e imóveis',car:'CAR e mapa',settings:'Configurações'}[name]||'Via Verde CAR';
  if(name==='car') setTimeout(()=>state.map?.invalidateSize(),80);
}

function initNavigation(){
  document.querySelectorAll('.nav-item').forEach(b=>b.addEventListener('click',()=>setView(b.dataset.view)));
  $('goCarBtn').onclick=$('heroCarBtn').onclick=()=>setView('car');
}

function fillUF(){
  const ufs=['AC','AL','AP','AM','BA','CE','DF','ES','GO','MA','MT','MS','MG','PA','PB','PR','PE','PI','RJ','RN','RS','RO','RR','SC','SP','SE','TO'];
  $('propertyUF').innerHTML='<option value="">Selecione</option>'+ufs.map(v=>`<option>${v}</option>`).join('');
}

async function boot(){
  initNavigation(); fillUF(); bindForms(); initMap();
  window.addEventListener('online',updateOnline); window.addEventListener('offline',updateOnline); updateOnline();
  for(let i=0;i<50&&!api();i++) await new Promise(r=>setTimeout(r,100));
  if(!api()){toast('A ponte do aplicativo não foi carregada.',true);return;}
  try{
    const info=await api().GetAppInfo(); $('versionLabel').textContent='Versão '+info.version; $('settingsVersion').textContent=info.version; $('dataDir').textContent=info.data_dir;
    await Promise.all([loadDashboard(),loadClients(),loadProperties()]);
  }catch(e){toast(String(e),true)}
}

function updateOnline(){const online=navigator.onLine;$('connectionLabel').textContent=online?'Internet disponível':'Sem internet';document.querySelector('.online-dot').classList.toggle('offline',!online)}

async function loadDashboard(){const d=await api().GetDashboard();$('statClients').textContent=d.clients;$('statProperties').textContent=d.properties;$('statCAR').textContent=d.properties_with_car;$('statKML').textContent=d.properties_with_kml;$('statChecks').textContent=d.checks_last_30_days}

async function loadClients(){
  state.clients=await api().ListClients()||[]; renderClients(); populatePropertyClientContext();
}
function renderClients(){
  const q=($('clientSearch').value||'').toLowerCase(); const list=state.clients.filter(c=>(c.name+' '+c.cpf_cnpj).toLowerCase().includes(q));
  $('clientList').className='record-list'+(list.length?'':' empty-state');
  $('clientList').innerHTML=list.length?list.map(c=>`<div class="record-row ${state.selectedClient?.id===c.id?'selected':''}" data-client="${c.id}"><div><strong>${esc(c.name)}</strong><small>${esc(c.cpf_cnpj||'Sem CPF/CNPJ')} • ${esc(c.phone||'sem telefone')}</small></div><div class="row-actions"><button class="icon-btn edit-client" data-id="${c.id}" title="Editar">✎</button><button class="icon-btn delete-client" data-id="${c.id}" title="Excluir">×</button></div></div>`).join(''):'Nenhum cliente cadastrado.';
  document.querySelectorAll('[data-client]').forEach(el=>el.onclick=(e)=>{if(e.target.closest('button'))return;selectClient(Number(el.dataset.client))});
  document.querySelectorAll('.edit-client').forEach(b=>b.onclick=()=>editClient(Number(b.dataset.id)));
  document.querySelectorAll('.delete-client').forEach(b=>b.onclick=()=>deleteClient(Number(b.dataset.id)));
}
function selectClient(id){state.selectedClient=state.clients.find(c=>c.id===id)||null;$('selectedClientLabel').textContent=state.selectedClient?state.selectedClient.name:'Selecione um cliente acima.';$('newPropertyBtn').disabled=!state.selectedClient;renderClients();renderProperties()}
function editClient(id){const c=state.clients.find(x=>x.id===id);if(!c)return;$('clientId').value=c.id;$('clientName').value=c.name;$('clientDoc').value=c.cpf_cnpj;$('clientPhone').value=c.phone;$('clientNotes').value=c.notes;selectClient(id)}
function resetClientForm(){$('clientForm').reset();$('clientId').value=''}
async function deleteClient(id){if(!confirm('Excluir este cliente e todos os imóveis vinculados?'))return;try{await api().DeleteClient(id);if(state.selectedClient?.id===id)state.selectedClient=null;resetClientForm();await Promise.all([loadClients(),loadProperties(),loadDashboard()]);toast('Cliente excluído.')}catch(e){toast(String(e),true)}}

async function loadProperties(){state.properties=await api().ListProperties(0)||[];renderProperties();populateCarPropertySelect()}
function renderProperties(){
  const list=state.selectedClient?state.properties.filter(p=>p.client_id===state.selectedClient.id):[];
  $('propertyList').className='property-cards'+(list.length?'':' empty-state');
  $('propertyList').innerHTML=list.length?list.map(p=>`<div class="property-card" data-property-card="${p.id}"><span class="tag">${p.car_number?'CAR cadastrado':'sem CAR'}</span><h4>${esc(p.name)}</h4><p>${esc([p.municipality,p.uf].filter(Boolean).join(' / ')||'Município não informado')}</p><p>Matrícula: ${esc(p.registry||'—')} • Área: ${p.declared_area_ha?fmt(p.declared_area_ha,4)+' ha':'—'}</p><p>${p.kml_path?'✓ KML salvo':'○ Sem KML'}</p></div>`).join(''):(state.selectedClient?'Nenhum imóvel para este cliente.':'Cadastre ou selecione um cliente.');
  document.querySelectorAll('[data-property-card]').forEach(el=>el.onclick=()=>openProperty(Number(el.dataset.propertyCard)));
}
function populateCarPropertySelect(){
  $('carPropertySelect').innerHTML='<option value="">Consulta avulsa</option>'+state.properties.map(p=>`<option value="${p.id}">${esc(p.client_name)} — ${esc(p.name)}</option>`).join('');
}
function populatePropertyClientContext(){renderClients()}
function newProperty(){if(!state.selectedClient)return;$('propertyForm').classList.remove('hidden');$('propertyForm').reset();$('propertyId').value='';$('propertyUF').value='MG';$('propertyName').focus()}
function openProperty(id){const p=state.properties.find(x=>x.id===id);if(!p)return;state.selectedProperty=p;state.selectedClient=state.clients.find(c=>c.id===p.client_id)||state.selectedClient;$('propertyForm').classList.remove('hidden');$('propertyId').value=p.id;$('propertyName').value=p.name;$('propertyMunicipality').value=p.municipality;$('propertyUF').value=p.uf;$('propertyRegistry').value=p.registry;$('propertyCAR').value=p.car_number;$('propertyArea').value=p.declared_area_ha||'';selectClient(p.client_id)}

function bindForms(){
  $('newClientBtn').onclick=resetClientForm;$('clientSearch').oninput=renderClients;
  $('clientForm').onsubmit=async e=>{e.preventDefault();try{const c={id:Number($('clientId').value)||0,name:$('clientName').value,cpf_cnpj:$('clientDoc').value,phone:$('clientPhone').value,notes:$('clientNotes').value};const saved=await api().SaveClient(c);resetClientForm();await loadClients();selectClient(saved.id);await loadDashboard();toast('Cliente salvo.')}catch(err){toast(String(err),true)}};
  $('newPropertyBtn').onclick=newProperty;$('cancelPropertyBtn').onclick=()=>{$('propertyForm').classList.add('hidden');$('propertyForm').reset()};
  $('propertyForm').onsubmit=async e=>{e.preventDefault();try{const p={id:Number($('propertyId').value)||0,client_id:state.selectedClient?.id||0,name:$('propertyName').value,municipality:$('propertyMunicipality').value,uf:$('propertyUF').value,registry:$('propertyRegistry').value,car_number:$('propertyCAR').value,declared_area_ha:Number($('propertyArea').value)||0};const saved=await api().SaveProperty(p);$('propertyForm').classList.add('hidden');await Promise.all([loadProperties(),loadDashboard()]);state.selectedProperty=state.properties.find(x=>x.id===saved.id)||saved;toast('Imóvel salvo.')}catch(err){toast(String(err),true)}};
  $('carPropertySelect').onchange=()=>selectCarProperty(Number($('carPropertySelect').value)||0);
  $('lookupCarBtn').onclick=lookupCAR;$('attachKmlBtn').onclick=attachKML;$('loadKmlBtn').onclick=loadSavedKML;$('fitMapBtn').onclick=fitMap;
  $('exportKmlBtn').onclick=exportKML;$('reportBtn').onclick=exportReport;
  $('openOfficialBtn').onclick=()=>openExternal(state.car?.official_url);$('openMeuImovelBtn').onclick=()=>openExternal(state.car?.meu_imovel_url);$('openMapsBtn').onclick=()=>openExternal(state.car?.google_maps_url);
  $('backupBtn').onclick=$('settingsBackupBtn').onclick=backup;$('openDataBtn').onclick=async()=>{try{await api().OpenDataFolder()}catch(e){toast(String(e),true)}};
  $('checkUpdateBtn').onclick=checkUpdates;$('downloadUpdateBtn').onclick=()=>api().OpenUpdateDownload(state.update?.download_url||'');
  document.querySelectorAll('.external').forEach(b=>b.onclick=()=>openExternal(b.dataset.url));
}

async function selectCarProperty(id){
  state.selectedProperty=state.properties.find(p=>p.id===id)||null;
  $('attachKmlBtn').disabled=!state.selectedProperty;$('loadKmlBtn').disabled=!state.selectedProperty||!state.selectedProperty.kml_path;
  if(state.selectedProperty?.car_number){$('carInput').value=state.selectedProperty.car_number}else if(id){$('carInput').value=''}
  if(state.selectedProperty?.kml_path){try{await loadSavedKML()}catch(e){}}
}

function initMap(){
  if(!window.L){$('map').innerHTML='<div class="map-placeholder"><strong>Mapa online indisponível</strong><span>Verifique a conexão com a internet.</span></div>';return}
  $('map').innerHTML='';state.map=L.map('map',{zoomControl:true}).setView([-18.5,-44.0],5);
  const street=L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',{maxZoom:19,attribution:'© OpenStreetMap'}).addTo(state.map);
  const sat=L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',{maxZoom:19,attribution:'Tiles © Esri'});
  L.control.layers({'Mapa':street,'Satélite':sat},{}).addTo(state.map);
}
function drawGeoJSON(which,geojson){if(!state.map||!geojson)return;try{const obj=typeof geojson==='string'?JSON.parse(geojson):geojson;if(which==='car'&&state.carLayer)state.map.removeLayer(state.carLayer);if(which==='kml'&&state.kmlLayer)state.map.removeLayer(state.kmlLayer);const style=which==='car'?{color:'#2c7a49',weight:3,fillColor:'#4ca36b',fillOpacity:.16}:{color:'#d77922',weight:3,dashArray:'8 5',fillColor:'#e89a44',fillOpacity:.08};const layer=L.geoJSON(obj,{style}).addTo(state.map);if(which==='car')state.carLayer=layer;else state.kmlLayer=layer;fitMap()}catch(e){toast('Não foi possível desenhar a geometria.',true)}}
function fitMap(){if(!state.map)return;const layers=[state.carLayer,state.kmlLayer].filter(Boolean);if(!layers.length)return;const group=L.featureGroup(layers);const b=group.getBounds();if(b.isValid())state.map.fitBounds(b.pad(.08),{maxZoom:17})}

async function lookupCAR(){
  const number=$('carInput').value.trim();if(!number){toast('Informe o número completo do CAR.',true);return}
  $('lookupCarBtn').disabled=true;$('lookupCarBtn').textContent='Consultando...';
  try{
    const propertyID=state.selectedProperty?.id||0;const r=propertyID?await api().AnalyzePropertyCAR(propertyID,number):await api().LookupCAR(number);state.car=r;renderCAR(r);if(r.geojson)drawGeoJSON('car',r.geojson);if(propertyID){await Promise.all([loadProperties(),loadDashboard()]);state.selectedProperty=state.properties.find(p=>p.id===propertyID)||state.selectedProperty}
    if(state.kml)await compareGeometries();toast(r.found?'CAR consultado com sucesso.':'Código válido, mas não localizado na camada pública.',!r.found);
  }catch(e){toast(String(e),true);renderChecks([{level:'error',title:'Falha na consulta',detail:String(e)}])}
  finally{$('lookupCarBtn').disabled=false;$('lookupCarBtn').textContent='Consultar SICAR'}
}
function renderCAR(r){
  $('rCar').textContent=r.car||'—';$('rMunicipality').textContent=[r.municipality,r.uf].filter(Boolean).join(' / ')||'—';$('rArea').textContent=r.area_ha?fmt(r.area_ha,4)+' ha':'—';$('rGeoArea').textContent=r.geometry_area_ha?fmt(r.geometry_area_ha,4)+' ha':'—';$('rPerimeter').textContent=r.perimeter_m?fmt(r.perimeter_m/1000,3)+' km':'—';$('rCenter').textContent=(r.center_lat||r.center_lon)?`${Number(r.center_lat).toFixed(6)}, ${Number(r.center_lon).toFixed(6)}`:'—';$('rModules').textContent=r.fiscal_modules?fmt(r.fiscal_modules,2):'—';$('rCondition').textContent=r.condition||'—';
  const badge=$('carStatusBadge');badge.textContent=r.status||(!r.found?'Não localizado':'Localizado');badge.className='status-badge '+(r.status==='Ativo'?'ok':r.found?'warning':'error');
  $('openOfficialBtn').disabled=!r.official_url;$('openMeuImovelBtn').disabled=!r.meu_imovel_url;$('openMapsBtn').disabled=!r.google_maps_url;$('exportKmlBtn').disabled=!r.has_geometry;$('reportBtn').disabled=!state.selectedProperty||!r.found;renderChecks(r.checks||[])
}
function renderChecks(checks){const box=$('qualityChecks');box.innerHTML=checks.length?checks.map(c=>`<div class="quality-item ${c.level||'info'}"><span class="qicon">${c.level==='ok'?'✓':c.level==='warning'?'!':c.level==='error'?'×':'i'}</span><div><strong>${esc(c.title)}</strong><span>${esc(c.detail)}</span></div></div>`).join(''):'<div class="empty-state">Nenhuma análise realizada.</div>'}

async function attachKML(){if(!state.selectedProperty){toast('Selecione um imóvel salvo.',true);return}try{const r=await api().AttachKML(state.selectedProperty.id);state.kml=r;renderKML(r);drawGeoJSON('kml',r.geojson);await Promise.all([loadProperties(),loadDashboard()]);state.selectedProperty=state.properties.find(p=>p.id===state.selectedProperty.id)||state.selectedProperty;if(state.car)await compareGeometries();toast('KML importado e salvo no imóvel.')}catch(e){if(!String(e).includes('cancelado'))toast(String(e),true)}}
async function loadSavedKML(){if(!state.selectedProperty)return;try{const r=await api().LoadPropertyKML(state.selectedProperty.id);state.kml=r;renderKML(r);drawGeoJSON('kml',r.geojson);if(state.car)await compareGeometries()}catch(e){toast(String(e),true)}}
function renderKML(r){$('kArea').textContent=r.area_ha?fmt(r.area_ha,4)+' ha':'—';$('kPerimeter').textContent=r.perimeter_m?fmt(r.perimeter_m/1000,3)+' km':'—';$('kPoints').textContent=r.points||'—';$('kCenter').textContent=(r.center_lat||r.center_lon)?`${Number(r.center_lat).toFixed(6)}, ${Number(r.center_lon).toFixed(6)}`:'—';$('kmlBadge').textContent='Carregado';$('kmlBadge').className='status-badge ok'}
async function compareGeometries(){if(!state.car||!state.kml)return;state.comparison=await api().CompareKMLWithCAR(state.kml,state.car);const c=state.comparison;const box=$('comparisonBox');box.className='comparison-box '+c.level;box.innerHTML=`<strong>Comparação KML × CAR</strong><span>${esc(c.summary)}<br>Diferença de área: ${fmt(c.area_difference_ha,4)} ha (${fmt(c.area_difference_pct,2)}%). Distância entre centros: ${fmt(c.center_distance_m,0)} m.</span>`}

async function exportKML(){try{const p=await api().ExportCARKML(state.car.car);toast('KML salvo em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}}
async function exportReport(){if(!state.selectedProperty||!state.car)return;try{if(state.kml&&!state.comparison)await compareGeometries();const p=await api().ExportCARReport(state.selectedProperty.id,state.car,state.kml||{},state.comparison||{});toast('PDF salvo em '+p)}catch(e){if(!String(e).includes('cancelada'))toast(String(e),true)}}
async function backup(){try{const r=await api().BackupData();toast(r.message+' '+r.path)}catch(e){if(!String(e).includes('cancelado'))toast(String(e),true)}}
async function checkUpdates(){try{const u=await api().CheckUpdates();state.update=u;$('updateMessage').textContent=u.message;$('downloadUpdateBtn').classList.toggle('hidden',!u.available);toast(u.message)}catch(e){toast(String(e),true)}}
function openExternal(url){if(!url)return;if(window.runtime?.BrowserOpenURL)window.runtime.BrowserOpenURL(url);else window.open(url,'_blank')}
function esc(v){return String(v??'').replace(/[&<>'"]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]))}

document.addEventListener('DOMContentLoaded',boot);
