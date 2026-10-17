CREATE TABLE tenant_clusters (id text PRIMARY KEY, name text NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
ALTER TABLE tenants ADD COLUMN cluster_id text REFERENCES tenant_clusters(id);
CREATE INDEX tenants_cluster_idx ON tenants(cluster_id);
