-- Local development seed: two resources for squad "payments" and one for "cards".
-- Apply after migrations: psql "$DATABASE_URL" -f db/seeds/dev.sql
INSERT INTO tracked_resource (id, name, kind, provider, environment, owner, region, claim_namespace, claim_name, claim_kind,
  claim_api_version, repository, git_path, resource_state, current_revision, composition_name, correlation_id, ever_operational,
  created_at, updated_at)
VALUES
 ('11111111-1111-1111-1111-111111111111', 'pgsql-payments-dev', 'PostgreSQLInstance', 'azure', 'dev', 'payments', 'brazilsouth',
  'team-payments', 'pgsql-payments-dev', 'PostgreSQLInstance', 'database.crosswatc.io/v1alpha1', 'crosswatc-org/argocd',
  'claims/dev/pgsql-payments.yaml', 'OPERATIONAL', 1, 'postgresql-azure-v3', 'crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T', true,
  now() - interval '2 days', now()),
 ('22222222-2222-2222-2222-222222222222', 'vm-payments-batch', 'VirtualMachine', 'azure', 'dev', 'payments', 'brazilsouth',
  'team-payments', 'vm-payments-batch', 'VirtualMachine', 'compute.crosswatc.io/v1alpha1', 'crosswatc-org/argocd',
  'claims/dev/vm-payments-batch.yaml', 'PROVISIONING', 1, 'vm-azure-v2', 'crosswatc-req-01JCAB0000000000000000000A', false,
  now() - interval '1 hour', now()),
 ('33333333-3333-3333-3333-333333333333', 'storage-cards', 'StorageAccount', 'azure', 'dev', 'cards', 'brazilsouth',
  'team-cards', 'storage-cards', 'StorageAccount', 'storage.crosswatc.io/v1alpha1', 'crosswatc-org/argocd',
  'claims/dev/storage-cards.yaml', 'OPERATIONAL', 1, 'storage-azure-v1', 'crosswatc-req-01JCAB0000000000000000000B', true,
  now() - interval '5 days', now())
ON CONFLICT DO NOTHING;

INSERT INTO revision (id, resource_id, rev, kind, state, commit_sha, pr_number, pr_url, started_at, completed_at, state_changed_at) VALUES
 ('aaaaaaaa-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 1, 'CREATE', 'SUCCEEDED', 'f00d', 171,
  'https://github.com/crosswatc-org/argocd/pull/171', now() - interval '2 days', now() - interval '2 days' + interval '9 minutes', now()),
 ('aaaaaaaa-0000-0000-0000-000000000002', '22222222-2222-2222-2222-222222222222', 1, 'CREATE', 'WAITING_APPROVAL', NULL, 173,
  'https://github.com/crosswatc-org/argocd/pull/173', now() - interval '1 hour', NULL, now() - interval '50 minutes')
ON CONFLICT DO NOTHING;

INSERT INTO operation (id, resource_id, revision_id, correlation_id, type, status, requested_by, requested_at) VALUES
 ('bbbbbbbb-0000-0000-0000-000000000001', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-0000-0000-0000-000000000001',
  'crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T', 'CREATE', 'SUCCEEDED', 'agnaldo', now() - interval '2 days'),
 ('bbbbbbbb-0000-0000-0000-000000000002', '22222222-2222-2222-2222-222222222222', 'aaaaaaaa-0000-0000-0000-000000000002',
  'crosswatc-req-01JCAB0000000000000000000A', 'CREATE', 'WAITING_APPROVAL', 'agnaldo', now() - interval '1 hour')
ON CONFLICT DO NOTHING;

INSERT INTO resource_projection (resource_id, stages, health, drift_state, last_reconcile_at) VALUES
 ('11111111-1111-1111-1111-111111111111',
  '[{"stage":"REQUEST","status":"SUCCEEDED","message":"Solicitação registrada no Portal"},
    {"stage":"PULL_REQUEST","status":"SUCCEEDED","message":"Pull Request #171 criado"},
    {"stage":"CI_POLICIES","status":"SUCCEEDED","message":"Validações aprovadas e mudança aprovada"},
    {"stage":"ARGO_CD","status":"SUCCEEDED","message":"Configuração sincronizada"},
    {"stage":"CROSSPLANE","status":"SUCCEEDED","message":"Infraestrutura composta"},
    {"stage":"CLOUD","status":"SUCCEEDED","message":"Recursos de cloud prontos"},
    {"stage":"OPERATIONAL","status":"SUCCEEDED","message":"Recurso operacional"}]',
  '{"crossplane":"ok","argocd":"ok","cloud":"ok","configuration":"ok","secrets":"ok"}', '{"detected":false}', now()),
 ('22222222-2222-2222-2222-222222222222',
  '[{"stage":"REQUEST","status":"SUCCEEDED","message":"Solicitação registrada no Portal"},
    {"stage":"PULL_REQUEST","status":"SUCCEEDED","message":"Pull Request #173 criado"},
    {"stage":"CI_POLICIES","status":"IN_PROGRESS","actionOwner":"approver","message":"Aguardando aprovação"},
    {"stage":"ARGO_CD","status":"PENDING"},{"stage":"CROSSPLANE","status":"PENDING"},
    {"stage":"CLOUD","status":"PENDING"},{"stage":"OPERATIONAL","status":"PENDING"}]',
  '{"crossplane":"unknown","argocd":"unknown","cloud":"unknown","configuration":"unknown","secrets":"ok"}', '{"detected":false}', now())
ON CONFLICT DO NOTHING;

INSERT INTO resource_topology (resource_id, nodes, edges, updated_at) VALUES
 ('11111111-1111-1111-1111-111111111111',
  '[{"id":"PostgreSQLInstance/team-payments/pgsql-payments-dev","type":"claim","kind":"PostgreSQLInstance","name":"pgsql-payments-dev","status":"ready","conditions":[{"type":"Synced","status":"True","reason":"ReconcileSuccess"},{"type":"Ready","status":"True","reason":"Available"}]},
    {"id":"XPostgreSQLInstance/xpostgresql-92a","type":"composite","kind":"XPostgreSQLInstance","name":"xpostgresql-92a","status":"ready","conditions":[{"type":"Synced","status":"True","reason":"ReconcileSuccess"},{"type":"Ready","status":"True","reason":"Available"}]},
    {"id":"FlexibleServer/postgresql-server","type":"managed-resource","kind":"FlexibleServer","name":"postgresql-server","status":"ready","provider":"azure","externalName":"pgsql-payments-dev-01","deletionPolicy":"Delete","conditions":[{"type":"Synced","status":"True","reason":"ReconcileSuccess"},{"type":"Ready","status":"True","reason":"Available"}]},
    {"id":"PrivateEndpoint/pe-pgsql","type":"managed-resource","kind":"PrivateEndpoint","name":"pe-pgsql","status":"not-ready","provider":"azure","deletionPolicy":"Delete","conditions":[{"type":"Synced","status":"True"},{"type":"Ready","status":"False","reason":"Creating"}]},
    {"id":"FlexibleServerDatabase/payments","type":"managed-resource","kind":"FlexibleServerDatabase","name":"payments","status":"error","provider":"azure","deletionPolicy":"Orphan","message":"AuthorizationFailed","conditions":[{"type":"Synced","status":"False","reason":"ReconcileError","message":"AuthorizationFailed: client does not have permission"},{"type":"Ready","status":"False"}]}]',
  '[{"source":"PostgreSQLInstance/team-payments/pgsql-payments-dev","target":"XPostgreSQLInstance/xpostgresql-92a"},
    {"source":"XPostgreSQLInstance/xpostgresql-92a","target":"FlexibleServer/postgresql-server"},
    {"source":"XPostgreSQLInstance/xpostgresql-92a","target":"PrivateEndpoint/pe-pgsql"},
    {"source":"XPostgreSQLInstance/xpostgresql-92a","target":"FlexibleServerDatabase/payments"}]', now())
ON CONFLICT DO NOTHING;

INSERT INTO tracker_event (id, dedupe_key, resource_id, revision_id, correlation_id, source, type, level, message, created_at) VALUES
 (gen_random_uuid(), 'seed:1', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-0000-0000-0000-000000000001', 'crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T', 'crosswatc', 'REQUEST.CREATED', 'info', 'Solicitação registrada', now() - interval '2 days'),
 (gen_random_uuid(), 'seed:2', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-0000-0000-0000-000000000001', 'crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T', 'github', 'GITOPS.PR.CREATED', 'info', 'Revisão 1: VALIDATING', now() - interval '2 days' + interval '1 minute'),
 (gen_random_uuid(), 'seed:3', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-0000-0000-0000-000000000001', 'crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T', 'github', 'GITOPS.PR.MERGED', 'info', 'Revisão 1: MERGED', now() - interval '2 days' + interval '4 minutes'),
 (gen_random_uuid(), 'seed:4', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-0000-0000-0000-000000000001', 'crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T', 'argocd', 'GITOPS.SYNC.SUCCEEDED', 'info', 'Revisão 1: RECONCILING', now() - interval '2 days' + interval '5 minutes'),
 (gen_random_uuid(), 'seed:5', '11111111-1111-1111-1111-111111111111', 'aaaaaaaa-0000-0000-0000-000000000001', 'crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T', 'crossplane', 'INFRA.MANAGED_RESOURCE.READY', 'info', 'Revisão 1: SUCCEEDED', now() - interval '2 days' + interval '9 minutes'),
 (gen_random_uuid(), 'seed:6', '11111111-1111-1111-1111-111111111111', NULL, 'crosswatc-req-01JCA9S8G4B2Z7QX3M5N6P8R9T', 'crosswatc', 'LIFECYCLE.OPERATIONAL', 'info', 'Recurso operacional', now() - interval '2 days' + interval '9 minutes')
ON CONFLICT DO NOTHING;
