# Kafui — Main Specification

This specification describes, in vendor-neutral, black-box requirements, the complete feature set of a cli-based management client for Apache Kafka. It is intended as the source of truth for implementing the application from scratch. Each feature area is specified in its own subfolder; implementation agents SHALL treat every referenced spec as part of this specification.

## Product Summary

The application is a terminal tui interactive terminal that connects to one or more Kafka clusters and lets operators inspect and manage brokers, topics, messages, consumer groups, schemas, connectors, streaming SQL, ACLs, and quotas. It supports pluggable authentication, fine-grained role-based access control, audit logging, metrics collection, and runtime configuration through the UI.

## Requirement Conventions

- Each feature spec uses `### Requirement:` blocks with SHALL language and `#### Scenario:` blocks (WHEN/THEN) describing observable behavior.
- Requirements are technology-agnostic: they constrain behavior, not implementation language, framework, or architecture.
- Kafka-domain terminology (broker, topic, partition, consumer group, offset, ACL) is used as defined by Apache Kafka.

## Feature Index

| # | Feature | Spec | Scope |
|---|---------|------|-------|
| 1 | Cluster Management & Dashboard | [cluster-management/spec.md](cluster-management/spec.md) | Multi-cluster configuration, cluster overview dashboard, health status, background statistics collection, per-cluster capability detection and read-only mode |
| 2 | Broker Management | [brokers/spec.md](brokers/spec.md) | Broker listing and details, broker configuration viewing/editing, log directories, partition distribution/skew, per-broker metrics |
| 3 | Topic Management | [topics/spec.md](topics/spec.md) | Topic list/search/sort, create/clone/edit/delete/recreate topics, partitions and replication changes, purge messages, topic configs, topic analysis (message statistics scan) |
| 4 | Message Browsing & Producing | [messages/spec.md](messages/spec.md) | Browsing with seek modes and cursors, live tailing, string and programmable smart filters, producing messages, serialization/deserialization formats, pluggable serde extension point, data masking, export |
| 5 | Consumer Groups & Offsets | [consumer-groups/spec.md](consumer-groups/spec.md) | Group list/search/sort with lag, group details and member assignments, delete groups/offsets, offset reset (earliest/latest/timestamp/specific) |
| 6 | Schema Registry | [schema-registry/spec.md](schema-registry/spec.md) | Subject listing/search, schema versions and diffing, register/delete schemas, AVRO/JSON/PROTOBUF types, global and per-subject compatibility levels, compatibility checks |
| 7 | Kafka Connect | [kafka-connect/spec.md](kafka-connect/spec.md) | Multiple Connect clusters, connector listing/search, connector lifecycle (create/update/delete/pause/resume/stop/restart), task management, plugin listing and config validation, offset reset |
| 8 | Streaming SQL (ksqlDB) | [ksql/spec.md](ksql/spec.md) | ksqlDB server configuration, listing streams/tables, executing statements with parameters, live streaming query results, error reporting |
| 9 | ACLs & Client Quotas | [acls-and-quotas/spec.md](acls-and-quotas/spec.md) | ACL listing/filtering, create/delete bindings, convenience flows for consumer/producer/stream-app ACL sets, CSV export/sync, client quota listing and upsert/delete |
| 10 | Authentication, RBAC & Audit | [auth-rbac-audit/spec.md](auth-rbac-audit/spec.md) | Pluggable auth (disabled/form/LDAP/OAuth2-OIDC), role model bound to identity-provider subjects, per-resource permission enforcement, permission-aware UI, audit logging of user actions |
| 11 | Application Configuration | [application-config/spec.md](application-config/spec.md) | Declarative cluster configuration, runtime config viewing/editing through the UI (setup wizard), connectivity validation before apply, file uploads, secret redaction, app info/health endpoints |
| 12 | Metrics & Monitoring | [metrics-and-monitoring/spec.md](metrics-and-monitoring/spec.md) | Broker metrics collection (JMX/Prometheus scraping), cluster-level aggregation, Prometheus exposition endpoints, time-series graphs with predefined parameterized queries |
| 13 | Web UI Shell & Cross-Cutting UX | [ui-shell/spec.md](ui-shell/spec.md) | SPA serving, navigation tree with feature-conditional sections, deep-linkable routing, theming, notifications, confirmation dialogs, shared table/editor patterns, version notice |

## Cross-Cutting Constraints

These apply across all feature areas and take precedence over silence in individual specs:

- **Authorization everywhere**: Every operation on every resource SHALL be checked against the RBAC permission model (feature 10). Listings SHALL be filtered to resources the user may view; forbidden actions SHALL be rejected by the backend and hidden or disabled in the UI.
- **Audit everywhere**: Every state-changing operation SHALL produce an audit record when auditing is enabled (feature 10).
- **Read-only mode**: When a cluster is configured read-only, all state-changing operations against it SHALL be rejected (features 1, 11).
- **Feature-conditional UI**: Sections for optional integrations (schema registry, connect, ksqlDB, ACL support, metrics) SHALL appear only when the corresponding integration is configured/supported for the selected cluster (features 1, 13).
- **Secret handling**: Credentials and other secrets SHALL never be returned in plain text by any endpoint that displays configuration (features 7, 11).
- **Destructive-action confirmation**: The UI SHALL require explicit confirmation before any destructive operation (feature 13).

## How to Use This Specification

Implementation agents SHOULD implement features in roughly the index order: 1, 11 and 13 form the foundation (connectivity, configuration, shell); 2–5 are the core Kafka workflows; 6–9 are optional integrations; 10 and 12 span the whole application and can be layered in once their touch points exist.
