# Architecture

GoSX Studio is the reusable website authoring layer that sits beside `gosx-cms` and `gosx-admin`.

## Library Roles

`gosx-cms` owns content management:

- content schemas
- media records
- lifecycle state
- content storage
- generated assets and their provenance

`gosx-admin` owns operator back-office surfaces:

- CRUD/resource management
- dashboards
- internal workflows
- resource actions
- operational reporting

`gosx-studio` owns website customization and authoring:

- page canvas
- visual inspectors
- no-code configuration controls
- section and block layout tools
- flow authoring
- publish readiness
- showcase plugins

## Composition Model

Host applications configure Studio. Editors use Studio.

The host application supplies:

- content adapters
- permission adapters
- server actions
- route bindings
- product copy
- feature flags
- design tokens

Studio supplies:

- `.gsx` surfaces for visible editor UI
- GoSX engines for heavy client-side interactions
- GoSX islands for focused reactive controls
- extension points for plugins
- common authoring language for non-technical operators

## Noni Proving Ground

Noni's Mud Relics remains the reference implementation while Studio is extracted. The proving-ground app should keep using operator-facing language such as "Website editor". The reusable package can keep internal names such as GoSX Studio for package boundaries, contracts, and docs.

The extraction should move from concrete Noni components toward stable Studio contracts without flattening CMS, Admin, and Studio into one package.
