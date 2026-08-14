-- VIA VERDE CRÉDITO RURAL - BANCO DE DADOS
-- Execute este arquivo UMA VEZ no SQL Editor do Supabase.
-- O backend usa a service_role e as tabelas ficam sem acesso público direto.

create extension if not exists pgcrypto;

create table if not exists public.users (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  username text not null unique,
  email text,
  password_hash text not null,
  role text not null default 'viewer' check (role in ('owner','viewer')),
  active boolean not null default true,
  created_at timestamptz not null default now(),
  last_login_at timestamptz
);

create table if not exists public.sessions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users(id) on delete cascade,
  token_hash text not null unique,
  csrf_token text not null,
  expires_at timestamptz not null,
  created_at timestamptz not null default now()
);
create index if not exists sessions_user_idx on public.sessions(user_id);
create index if not exists sessions_expires_idx on public.sessions(expires_at);

create table if not exists public.invites (
  id uuid primary key default gen_random_uuid(),
  token_hash text not null unique,
  name text,
  email text,
  role text not null default 'viewer' check (role='viewer'),
  expires_at timestamptz not null,
  used_at timestamptz,
  created_by uuid references public.users(id) on delete set null,
  created_at timestamptz not null default now()
);

create table if not exists public.clients (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  document text,
  phone text,
  email text,
  rba numeric(16,2) not null default 0,
  classification text,
  notes text,
  created_at timestamptz not null default now()
);
create index if not exists clients_name_idx on public.clients(name);

create table if not exists public.properties (
  id uuid primary key default gen_random_uuid(),
  client_id uuid not null references public.clients(id) on delete cascade,
  name text not null,
  city text,
  state text,
  area_ha numeric(14,4) not null default 0,
  registry text,
  car text,
  ccir text,
  itr text,
  tenure text,
  notes text,
  created_at timestamptz not null default now()
);
create index if not exists properties_client_idx on public.properties(client_id);

create table if not exists public.projects (
  id uuid primary key default gen_random_uuid(),
  client_id uuid not null references public.clients(id) on delete restrict,
  property_id uuid references public.properties(id) on delete set null,
  title text not null,
  modality text not null,
  activity text,
  bank text,
  program text,
  total_value numeric(16,2) not null default 0,
  financed_value numeric(16,2) not null default 0,
  interest_rate numeric(8,4) not null default 0,
  term_months integer not null default 0,
  status text not null default 'Em andamento',
  phase text,
  technical_summary text,
  alerts text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create index if not exists projects_client_idx on public.projects(client_id);
create index if not exists projects_status_idx on public.projects(status);
create index if not exists projects_updated_idx on public.projects(updated_at desc);

create table if not exists public.project_files (
  id uuid primary key default gen_random_uuid(),
  project_id uuid not null references public.projects(id) on delete cascade,
  name text not null,
  storage_path text not null unique,
  content_type text,
  size_bytes bigint not null default 0,
  uploaded_by uuid references public.users(id) on delete set null,
  created_at timestamptz not null default now()
);
create index if not exists project_files_project_idx on public.project_files(project_id);

create table if not exists public.daily_reports (
  id uuid primary key default gen_random_uuid(),
  number integer,
  date date not null default current_date,
  value numeric(16,2) not null default 0,
  bank text,
  referral text,
  producer text not null,
  project_type text,
  sent_date date,
  situation text,
  commission_rate numeric(8,4) not null default 0.5,
  commission numeric(16,4) not null default 0,
  pedro_percent numeric(8,4) not null default 0,
  paid_by text,
  notes text,
  created_at timestamptz not null default now()
);
create index if not exists daily_reports_date_idx on public.daily_reports(date desc);

create table if not exists public.rules (
  id uuid primary key default gen_random_uuid(),
  category text,
  title text not null,
  reference text,
  summary text,
  source_url text,
  verified_at date,
  active boolean not null default true,
  created_at timestamptz not null default now()
);

create table if not exists public.audit_logs (
  id uuid primary key default gen_random_uuid(),
  user_id uuid references public.users(id) on delete set null,
  action text not null,
  entity text,
  entity_id text,
  details text,
  created_at timestamptz not null default now()
);
create index if not exists audit_logs_created_idx on public.audit_logs(created_at desc);

-- Defesa em profundidade: o navegador não acessa as tabelas diretamente.
alter table public.users enable row level security;
alter table public.sessions enable row level security;
alter table public.invites enable row level security;
alter table public.clients enable row level security;
alter table public.properties enable row level security;
alter table public.projects enable row level security;
alter table public.project_files enable row level security;
alter table public.daily_reports enable row level security;
alter table public.rules enable row level security;
alter table public.audit_logs enable row level security;

-- Não são criadas policies para anon/authenticated. A service_role do backend ignora RLS.
