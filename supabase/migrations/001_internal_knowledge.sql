-- B-stage internal knowledge base for Supabase PostgreSQL + pgvector.
-- Raw objects belong in a private Supabase Storage bucket named internal-raw.
create extension if not exists pgcrypto;
create extension if not exists vector;

create table if not exists documents (
  id uuid primary key default gen_random_uuid(),
  title text not null,
  document_type text not null,
  source text not null default 'internal_upload',
  storage_path text not null,
  original_name text not null,
  author text,
  department text,
  product_area text,
  version text not null default '1',
  status text not null default 'active' check (status in ('active', 'deprecated', 'historical', 'draft')),
  access_level text not null default 'team',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists document_chunks (
  id uuid primary key default gen_random_uuid(),
  document_id uuid not null references documents(id) on delete cascade,
  chunk_index integer not null,
  section_title text,
  content text not null,
  token_count integer,
  embedding vector,
  embedding_model text,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now(),
  unique (document_id, chunk_index)
);

create table if not exists support_tickets (
  id uuid primary key default gen_random_uuid(),
  document_id uuid references documents(id) on delete set null,
  external_ticket_id text,
  user_message text not null,
  category text,
  sub_category text,
  sentiment text,
  product_area text,
  resolution text,
  occurred_at timestamptz,
  embedding vector,
  created_at timestamptz not null default now()
);

create table if not exists interviews (
  id uuid primary key default gen_random_uuid(),
  document_id uuid references documents(id) on delete cascade,
  persona text,
  scenario text,
  transcript text not null,
  occurred_at timestamptz,
  created_at timestamptz not null default now()
);

create table if not exists insights (
  id uuid primary key default gen_random_uuid(),
  interview_id uuid references interviews(id) on delete cascade,
  pain_points jsonb not null default '[]'::jsonb,
  behaviors jsonb not null default '[]'::jsonb,
  needs jsonb not null default '[]'::jsonb,
  evidence_chunk_ids uuid[] not null default '{}',
  confidence numeric(3,2),
  created_at timestamptz not null default now()
);

create table if not exists research_notes (
  id uuid primary key default gen_random_uuid(),
  document_id uuid references documents(id) on delete set null,
  title text not null,
  topic text,
  key_findings jsonb not null default '[]'::jsonb,
  open_questions jsonb not null default '[]'::jsonb,
  confidence numeric(3,2),
  created_at timestamptz not null default now()
);

create table if not exists structured_datasets (
  id uuid primary key default gen_random_uuid(),
  document_id uuid references documents(id) on delete set null,
  name text not null,
  schema jsonb not null default '{}'::jsonb,
  row_count integer not null default 0,
  freshness text,
  created_at timestamptz not null default now()
);
create table if not exists product_requirements (
  id uuid primary key default gen_random_uuid(),
  document_id uuid not null references documents(id) on delete cascade,
  product_area text,
  version text,
  status text not null default 'active',
  summary text,
  created_at timestamptz not null default now()
);

create index if not exists documents_metadata_idx on documents (document_type, department, status, updated_at desc);
create index if not exists document_chunks_document_idx on document_chunks (document_id, chunk_index);
create index if not exists document_chunks_fts_idx on document_chunks using gin (to_tsvector('simple', content));
create index if not exists support_tickets_metadata_idx on support_tickets (category, product_area, occurred_at desc);
create index if not exists research_notes_document_idx on research_notes (document_id, created_at desc);
create index if not exists structured_datasets_document_idx on structured_datasets (document_id, created_at desc);

-- Add an HNSW index after choosing a fixed embedding dimension and model, e.g.:
-- alter table document_chunks alter column embedding type vector(1536);
-- create index document_chunks_embedding_hnsw on document_chunks using hnsw (embedding vector_cosine_ops);
-- Product delivery workflow: immutable review history with editable draft versions.
create table if not exists product_initiatives (
  id uuid primary key default gen_random_uuid(),
  initiative_key text not null unique,
  title text not null,
  status text not null default 'draft' check (status in ('draft', 'in_review', 'approved', 'rejected', 'archived')),
  owner text not null default '',
  current_version text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists prd_versions (
  id uuid primary key default gen_random_uuid(),
  initiative_id uuid not null references product_initiatives(id) on delete cascade,
  version text not null,
  status text not null default 'draft' check (status in ('draft', 'in_review', 'approved', 'rejected', 'archived')),
  snapshot jsonb not null default '{}'::jsonb,
  change_summary text not null default '',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (initiative_id, version)
);

create table if not exists prd_requirements (
  id uuid primary key default gen_random_uuid(),
  prd_version_id uuid not null references prd_versions(id) on delete cascade,
  requirement_key text not null,
  title text not null,
  priority text not null,
  user_story text not null,
  preconditions jsonb not null default '[]'::jsonb,
  main_flow jsonb not null default '[]'::jsonb,
  business_rules jsonb not null default '[]'::jsonb,
  exceptions jsonb not null default '[]'::jsonb,
  permissions jsonb not null default '[]'::jsonb,
  acceptance jsonb not null default '[]'::jsonb,
  tracking jsonb not null default '[]'::jsonb,
  status text not null default 'draft',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  unique (prd_version_id, requirement_key)
);

create table if not exists evidence_links (
  id uuid primary key default gen_random_uuid(),
  prd_requirement_id uuid not null references prd_requirements(id) on delete cascade,
  source_type text not null,
  source_id text not null default '',
  relation text not null default 'supports' check (relation in ('supports', 'conflicts', 'gap')),
  note text not null default '',
  created_at timestamptz not null default now(),
  unique (prd_requirement_id, source_type, source_id, relation)
);

create table if not exists prd_reviews (
  id uuid primary key default gen_random_uuid(),
  prd_version_id uuid not null references prd_versions(id) on delete cascade,
  reviewer_type text not null,
  reviewer_name text not null default '',
  readiness_score integer not null check (readiness_score between 0 and 100),
  verdict text not null,
  status text not null check (status in ('not_ready', 'ready_for_review', 'passed', 'failed')),
  checks jsonb not null default '[]'::jsonb,
  created_at timestamptz not null default now()
);

create table if not exists prd_approvals (
  id uuid primary key default gen_random_uuid(),
  prd_version_id uuid not null references prd_versions(id) on delete cascade,
  role text not null,
  reviewer text not null default '',
  status text not null default 'pending' check (status in ('pending', 'approved', 'rejected')),
  comment text not null default '',
  decided_at timestamptz,
  created_at timestamptz not null default now(),
  unique (prd_version_id, role)
);

create index if not exists product_initiatives_status_idx on product_initiatives (status, updated_at desc);
create index if not exists prd_versions_initiative_idx on prd_versions (initiative_id, updated_at desc);
create index if not exists prd_requirements_version_idx on prd_requirements (prd_version_id, priority);
create index if not exists evidence_links_requirement_idx on evidence_links (prd_requirement_id);
create index if not exists prd_reviews_version_idx on prd_reviews (prd_version_id, created_at desc);
create index if not exists prd_approvals_version_idx on prd_approvals (prd_version_id, status);
-- Agent operational memory. This is not an evidence store: summaries are only for context budgeting.
create table if not exists agent_runs (
  id text primary key,
  question text not null,
  status text not null,
  result_summary text not null default '',
  started_at timestamptz not null,
  completed_at timestamptz not null,
  created_at timestamptz not null default now()
);

create table if not exists agent_events (
  id text primary key,
  run_id text not null references agent_runs(id) on delete cascade,
  sequence integer not null,
  category text not null,
  tool text not null,
  input_summary text not null default '',
  output_summary text not null default '',
  importance integer not null default 50 check (importance between 0 and 100),
  created_at timestamptz not null,
  unique (run_id, sequence)
);

create table if not exists agent_memory_summaries (
  scope text primary key,
  summary text not null,
  covered_event_count integer not null default 0,
  updated_at timestamptz not null default now()
);

create index if not exists agent_runs_completed_idx on agent_runs (completed_at desc);
create index if not exists agent_events_recent_idx on agent_events (created_at desc, importance desc);