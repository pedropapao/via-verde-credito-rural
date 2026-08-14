-- VIA VERDE CRÉDITO RURAL v3.0 - UPGRADE DOSSIÊ 360
-- Execute UMA VEZ no SQL Editor do Supabase após 01_schema.sql e 02_seed_via_verde.sql.

create extension if not exists pgcrypto;

alter table public.properties add column if not exists latitude numeric(11,8);
alter table public.properties add column if not exists longitude numeric(11,8);
alter table public.properties add column if not exists water_info text;
alter table public.properties add column if not exists environmental_info text;
alter table public.properties add column if not exists access_info text;
alter table public.properties add column if not exists owner_name text;
alter table public.properties add column if not exists registry_date date;
alter table public.properties add column if not exists car_status text;

alter table public.projects add column if not exists own_resources numeric(16,2) not null default 0;
alter table public.projects add column if not exists grace_months integer not null default 0;
alter table public.projects add column if not exists payment_frequency text;
alter table public.projects add column if not exists sent_at date;
alter table public.projects add column if not exists contracted_at date;
alter table public.projects add column if not exists responsible text;
alter table public.projects add column if not exists sensitive_notes text;

create table if not exists public.project_budget_items (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects(id) on delete cascade,
  category text,
  item text not null,
  acquisition_start date,
  acquisition_end date,
  use_start date,
  use_end date,
  unit text,
  quantity numeric(16,4) not null default 0,
  unit_value numeric(16,4) not null default 0,
  total_value numeric(16,2) not null default 0,
  own_resources numeric(16,2) not null default 0,
  financed_value numeric(16,2) not null default 0,
  price_source text,
  notes text,
  created_at timestamptz not null default now()
);
create index if not exists project_budget_project_idx on public.project_budget_items(project_id);

create table if not exists public.project_revenues (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects(id) on delete cascade,
  product text not null,
  period text,
  unit text,
  quantity numeric(16,4) not null default 0,
  unit_price numeric(16,4) not null default 0,
  total_value numeric(16,2) not null default 0,
  source text,
  notes text,
  created_at timestamptz not null default now()
);
create index if not exists project_revenues_project_idx on public.project_revenues(project_id);

create table if not exists public.project_metrics (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null unique references public.projects(id) on delete cascade,
  activity_type text,
  area_ha numeric(14,4) not null default 0,
  productivity numeric(16,4) not null default 0,
  productivity_unit text,
  animal_count integer not null default 0,
  initial_weight_kg numeric(12,3) not null default 0,
  final_weight_kg numeric(12,3) not null default 0,
  gmd_kg_day numeric(10,4) not null default 0,
  cycle_days integer not null default 0,
  carcass_yield_pct numeric(8,4) not null default 0,
  mortality_pct numeric(8,4) not null default 0,
  stocking_rate numeric(10,4) not null default 0,
  other_debts numeric(16,2) not null default 0,
  other_income numeric(16,2) not null default 0,
  annual_payment numeric(16,2) not null default 0,
  notes text,
  updated_at timestamptz not null default now()
);

create table if not exists public.project_tasks (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects(id) on delete cascade,
  title text not null,
  category text,
  responsible text,
  requested_at date,
  due_at date,
  status text not null default 'Pendente',
  priority text not null default 'Normal',
  notes text,
  created_at timestamptz not null default now(),
  completed_at timestamptz
);
create index if not exists project_tasks_project_idx on public.project_tasks(project_id);
create index if not exists project_tasks_due_idx on public.project_tasks(due_at);

create table if not exists public.project_history (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects(id) on delete cascade,
  event_date timestamptz not null default now(),
  event_type text,
  title text not null,
  details text,
  user_id uuid references public.users(id) on delete set null
);
create index if not exists project_history_project_idx on public.project_history(project_id, event_date desc);

create table if not exists public.project_checklist (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects(id) on delete cascade,
  code text not null,
  document_name text not null,
  required boolean not null default true,
  status text not null default 'Pendente',
  valid_until date,
  notes text,
  source_rule text,
  created_at timestamptz not null default now(),
  unique(project_id, code)
);
create index if not exists project_checklist_project_idx on public.project_checklist(project_id);

create table if not exists public.property_files (
  id uuid primary key default gen_random_uuid(),
  property_id uuid not null references public.properties(id) on delete cascade,
  document_type text,
  name text not null,
  version_label text,
  valid_from date,
  valid_until date,
  storage_path text not null unique,
  content_type text,
  size_bytes bigint not null default 0,
  uploaded_by uuid references public.users(id) on delete set null,
  created_at timestamptz not null default now()
);
create index if not exists property_files_property_idx on public.property_files(property_id);

alter table public.project_files add column if not exists document_type text;
alter table public.project_files add column if not exists version_label text;
alter table public.project_files add column if not exists valid_until date;
alter table public.project_files add column if not exists is_current boolean not null default true;

create table if not exists public.project_sections (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects(id) on delete cascade,
  section_key text not null,
  section_title text not null,
  content text,
  status text not null default 'Pendente',
  updated_at timestamptz not null default now(),
  unique(project_id, section_key)
);

-- RLS: o navegador não acessa diretamente; o backend usa chave secreta.
alter table public.project_budget_items enable row level security;
alter table public.project_revenues enable row level security;
alter table public.project_metrics enable row level security;
alter table public.project_tasks enable row level security;
alter table public.project_history enable row level security;
alter table public.project_checklist enable row level security;
alter table public.property_files enable row level security;
alter table public.project_sections enable row level security;

-- Índices de pesquisa e operação
create index if not exists projects_bank_idx on public.projects(bank);
create index if not exists projects_modality_idx on public.projects(modality);
create index if not exists properties_city_idx on public.properties(city);

-- Função auxiliar para manter total do item coerente quando o backend não informar explicitamente.
create or replace function public.vv_budget_total()
returns trigger language plpgsql as $$
begin
  if new.total_value = 0 and new.quantity <> 0 and new.unit_value <> 0 then
    new.total_value := round((new.quantity * new.unit_value)::numeric, 2);
  end if;
  if new.financed_value = 0 and new.total_value > 0 then
    new.financed_value := greatest(new.total_value - new.own_resources, 0);
  end if;
  return new;
end $$;

drop trigger if exists trg_vv_budget_total on public.project_budget_items;
create trigger trg_vv_budget_total before insert or update on public.project_budget_items
for each row execute function public.vv_budget_total();
