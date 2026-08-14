-- CARGA INICIAL DA VIA VERDE
-- Baseada no controle diário disponibilizado em 13/08/2026.
-- Pode ser executada depois do 01_schema.sql. É idempotente pelos UUIDs fixos.

insert into public.clients (id,name,notes) values
('10000000-0000-0000-0000-000000000001','Adilson','Projeto de investimento pecuário leiteiro.'),
('10000000-0000-0000-0000-000000000002','Tofu','Custeio agrícola de soja.'),
('10000000-0000-0000-0000-000000000003','Carlos Alberto Ohara','Projetos pecuário e café.'),
('10000000-0000-0000-0000-000000000004','Antônio Alves Cordeiro','Investimento Moderfrota.'),
('10000000-0000-0000-0000-000000000005','Bruno / Lincoln','Aquisição, custeio agrícola e reforma.'),
('10000000-0000-0000-0000-000000000006','Daniel Lage','Restauração de pastagem.')
on conflict (id) do nothing;

insert into public.projects (id,client_id,title,modality,activity,bank,total_value,financed_value,status,phase,technical_summary,alerts) values
('20000000-0000-0000-0000-000000000001','10000000-0000-0000-0000-000000000001','Investimento pecuário leiteiro — Adilson','Investimento','Pecuária leiteira','CREDICOM',100000,100000,'Reprovado','Análise de crédito','Projeto registrado no controle da Via Verde.','Ainda não tive respostas.'),
('20000000-0000-0000-0000-000000000002','10000000-0000-0000-0000-000000000002','Custeio agrícola de soja — Tofu','Custeio agrícola','Soja','CRESOL',1500000,1500000,'Enviado ao banco','Acompanhamento bancário','Operação registrada no controle diário.','Acompanhamento junto ao banco.'),
('20000000-0000-0000-0000-000000000003','10000000-0000-0000-0000-000000000003','Investimento / custeio pecuário — Carlos Ohara','Custeio pecuário','Bovinocultura','CRESOL',370000,370000,'Enviado ao banco','Contratação / documentação','Operação pecuária registrada no controle diário.','Preciso enviar o PDF do inventário do vendedor.'),
('20000000-0000-0000-0000-000000000004','10000000-0000-0000-0000-000000000003','Custeio agrícola de café — Carlos Ohara','Custeio agrícola','Café','SICOOB',54000,54000,'Enviado ao banco','Finalização de contrato','Custeio agrícola de café registrado no controle diário.','Sicoob ainda finalizando o contrato.'),
('20000000-0000-0000-0000-000000000005','10000000-0000-0000-0000-000000000004','Investimento Moderfrota — Antônio Cordeiro','Investimento','Máquinas / implementos','CREDICOM',121500,121500,'Enviado ao banco','Aguardando retorno','Projeto Moderfrota registrado no controle diário.','Aguardando respostas.'),
('20000000-0000-0000-0000-000000000006','10000000-0000-0000-0000-000000000005','Aquisição de bovinos — Bruno / Lincoln','Custeio pecuário','Aquisição de bovinos','CREDICOM',1000000,1000000,'Em andamento','Coleta de informações','Operação de aquisição registrada no controle diário.','Não me mandou as informações ainda.'),
('20000000-0000-0000-0000-000000000007','10000000-0000-0000-0000-000000000005','Custeio agrícola — Bruno / Lincoln','Custeio agrícola','Milho / sorgo','CREDICOM',200000,200000,'Em andamento','Coleta de informações','Operação agrícola registrada no controle diário.','Não me mandou as informações ainda.'),
('20000000-0000-0000-0000-000000000008','10000000-0000-0000-0000-000000000005','Reforma — Bruno / Lincoln','Investimento','Reforma / ampliação','CREDICOM',200000,200000,'Em andamento','Coleta de informações','Projeto de reforma registrado no controle diário.','Não me mandou as informações ainda.'),
('20000000-0000-0000-0000-000000000009','10000000-0000-0000-0000-000000000006','Restauração de pastagem — Daniel Lage','Investimento','Restauração de pastagem','BB',248261.50,248261.50,'Em andamento','Ajuste de projeto e contrato','Projeto registrado no controle diário.','Hoje foi mexendo no projeto dia inteiro tentando ajustar o contrato com o projeto.')
on conflict (id) do nothing;

insert into public.daily_reports (id,number,date,value,bank,referral,producer,project_type,sent_date,situation,commission_rate,commission,notes) values
('30000000-0000-0000-0000-000000000001',1,'2026-07-16',100000,'CREDICOM','GABRIEL','ADILSON','INVESTIMENTO PECUÁRIO LEITE','2026-07-28','REPROVADO',0.5,500,'Ainda não tive respostas'),
('30000000-0000-0000-0000-000000000002',2,'2026-07-01',1500000,'CRESOL','CARLOS','TOFU','CUSTEIO AGRÍCOLA SOJA','2026-08-05','ENVIADO AO BANCO',0.5,7500,'Enviei mensagem para o sicoob para liberar o projeto no sisbr do Ronan'),
('30000000-0000-0000-0000-000000000003',3,'2026-07-01',370000,'CRESOL','CARLOS','CARLOS ALBERTO OHARA','INVESTIMENTO/CUSTEIO PECUÁRIO','2026-07-30','ENVIADO AO BANCO',0.5,1850,'Preciso enviar o pdf do inventário do vendedor'),
('30000000-0000-0000-0000-000000000005',5,'2026-07-13',54000,'SICOOB','CARLOS','CARLOS ALBERTO OHARA','CUSTEIO AGRÍCOLA CAFÉ','2026-07-27','ENVIADO AO BANCO',0.5,270,'Sicoob ainda finalizando o contrato'),
('30000000-0000-0000-0000-000000000006',6,'2026-08-03',121500,'CREDICOM','GABRIEL','Antônio Alves Cordeiro','INVESTIMENTO MORDEN FROTA','2026-08-05','ENVIADO AO BANCO',0.5,607.5,'Aguardando respostas'),
('30000000-0000-0000-0000-000000000007',7,'2026-07-15',1000000,'CREDICOM','GABRIEL','BRUNO/LINCOLN','CUSTEIO PECUÁRIO AQUISIÇÃO',null,'EM ANDAMENTO',0.5,5000,'Não me mandou as informações ainda'),
('30000000-0000-0000-0000-000000000008',8,'2026-07-15',200000,'CREDICOM','GABRIEL','BRUNO/LINCOLN','CUSTEIO AGRÍCOLA',null,'EM ANDAMENTO',0.5,1000,'Não me mandou as informações ainda'),
('30000000-0000-0000-0000-000000000009',9,'2026-07-15',200000,'CREDICOM','GABRIEL','BRUNO/LINCOLN','REFORMA',null,'EM ANDAMENTO',0.5,1000,'Não me mandou as informações ainda'),
('30000000-0000-0000-0000-000000000010',10,'2026-08-12',248261.50,'BB','GABRIEL','Daniel Lage','RESTAURAÇÃO DE PASTAGEM',null,'EM ANDAMENTO',0.5,1241.3075,'Hoje foi mexendo no projeto dia inteiro tentando ajustar o contrato com o projeto')
on conflict (id) do nothing;

insert into public.rules (id,category,title,reference,summary,source_url,verified_at,active) values
('40000000-0000-0000-0000-000000000001','MCR','Manual de Crédito Rural — consulta oficial','MCR','Antes de informar taxas, limites, enquadramento ou regra financeira, conferir a versão vigente no Manual de Crédito Rural do Banco Central.','https://manuais.bcb.gov.br/app/manual/mcr/publico','2026-08-14',true),
('40000000-0000-0000-0000-000000000002','Conformidade','Zoneamento, coordenadas e CAR','MCR 2-1','Nas operações vinculadas a área delimitada, conferir ZARC quando aplicável, coordenadas geodésicas e requisitos ambientais/CAR conforme o caso.','https://manuais.bcb.gov.br/app/manual/mcr/publico','2026-08-14',true),
('40000000-0000-0000-0000-000000000003','Projeto','Orçamento, plano ou projeto','MCR 2-2','O orçamento deve discriminar espécie, valor e época das despesas e considerar os recursos financiados e próprios.','https://manuais.bcb.gov.br/app/manual/mcr/publico','2026-08-14',true),
('40000000-0000-0000-0000-000000000004','Pecuária','Documentos de bovinos e bubalinos','MCR 2-1','Conferir exigências vigentes de nota fiscal, GTA e ficha sanitária/documento equivalente conforme a natureza da operação.','https://manuais.bcb.gov.br/app/manual/mcr/publico','2026-08-14',true)
on conflict (id) do nothing;
