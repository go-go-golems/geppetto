---
Title: SQLite Summary Query Output
Ticket: HIST-2026-03-14-TO-2026-06-14
Status: active
Topics:
    - git-history
    - docmgr
    - research
DocType: reference
Intent: long-term
Owners:
    - manuel
RelatedFiles: []
ExternalSources: []
Summary: "Rendered output of scripts/02_summary_queries.sql against various/history.sqlite."
LastUpdated: 2026-06-14T12:50:00-04:00
WhatFor: "Quick inspection of the SQLite aggregate query results used by the history report."
WhenToUse: "Use when validating or reusing the raw SQL summaries behind the generated report."
---

# Overall git activity
| commits | merge_commits | non_merge_commits | files_changed | insertions | deletions | first_day  |  last_day  |
|---------|---------------|-------------------|---------------|------------|-----------|------------|------------|
| 525     | 59            | 466               | 4702          | 513044     | 85013     | 2026-03-01 | 2026-06-06 |
\n# Commits by month and category
|  month  |   category    | commits | files_changed | insertions | deletions |
|---------|---------------|---------|---------------|------------|-----------|
| 2026-03 | maintenance   | 71      | 689           | 22064      | 24507     |
| 2026-03 | docs/research | 56      | 280           | 28402      | 2910      |
| 2026-03 | features      | 31      | 261           | 29832      | 230       |
| 2026-03 | merge         | 18      | 737           | 76466      | 23246     |
| 2026-03 | fixes         | 15      | 42            | 2425       | 110       |
| 2026-03 | dependencies  | 9       | 18            | 93         | 103       |
| 2026-04 | merge         | 9       | 72            | 4658       | 727       |
| 2026-04 | maintenance   | 8       | 36            | 2102       | 627       |
| 2026-04 | docs/research | 8       | 23            | 476        | 155       |
| 2026-04 | dependencies  | 7       | 13            | 121        | 53        |
| 2026-04 | fixes         | 4       | 15            | 293        | 21        |
| 2026-04 | features      | 3       | 11            | 1839       | 12        |
| 2026-04 | tests         | 1       | 9             | 106        | 11        |
| 2026-05 | docs/research | 60      | 253           | 63604      | 616       |
| 2026-05 | maintenance   | 59      | 400           | 15616      | 7207      |
| 2026-05 | merge         | 26      | 618           | 100019     | 5483      |
| 2026-05 | fixes         | 22      | 99            | 3016       | 374       |
| 2026-05 | features      | 19      | 177           | 18405      | 414       |
| 2026-05 | tests         | 11      | 25            | 2138       | 164       |
| 2026-05 | dependencies  | 10      | 31            | 222        | 184       |
| 2026-06 | docs/research | 28      | 129           | 45035      | 327       |
| 2026-06 | features      | 16      | 120           | 12397      | 319       |
| 2026-06 | maintenance   | 13      | 186           | 8252       | 9057      |
| 2026-06 | merge         | 6       | 326           | 69399      | 7748      |
| 2026-06 | fixes         | 6       | 22            | 558        | 30        |
| 2026-06 | tests         | 5       | 103           | 5485       | 358       |
| 2026-06 | dependencies  | 4       | 7             | 21         | 20        |
\n# Top touched paths
|                             path                             | commits | insertions | deletions |
|--------------------------------------------------------------|---------|------------|-----------|
| go.mod                                                       | 60      | 239        | 203       |
| go.sum                                                       | 57      | 551        | 431       |
| pkg/steps/ai/openai/engine_openai.go                         | 37      | 804        | 736       |
| pkg/steps/ai/openai_responses/engine.go                      | 35      | 551        | 2411      |
| pkg/js/modules/geppetto/module_test.go                       | 32      | 1702       | 4844      |
| ttmp/vocabulary.yaml                                         | 31      | 124        | 4         |
| pkg/steps/ai/openai_responses/engine_test.go                 | 30      | 2160       | 212       |
| pkg/doc/types/geppetto.d.ts                                  | 29      | 725        | 969       |
| pkg/js/modules/geppetto/module.go                            | 29      | 359        | 343       |
| pkg/js/modules/geppetto/spec/geppetto.d.ts.tmpl              | 27      | 723        | 811       |
| pkg/steps/ai/openai_responses/streaming.go                   | 26      | 2982       | 2506      |
| Makefile                                                     | 25      | 170        | 93        |
| pkg/steps/ai/gemini/engine_gemini.go                         | 24      | 415        | 997       |
| pkg/js/modules/geppetto/api_engines.go                       | 24      | 393        | 901       |
| pkg/steps/ai/openai_responses/helpers.go                     | 23      | 701        | 253       |
| ttmp/2026/05/08/GP-EVENT-VOCABULARY--split-provider-run-and- | 22      | 2664       | 8         |
| text-segment-event-vocabulary/reference/01-investigation-dia |         |            |           |
| ry.md                                                        |         |            |           |
| pkg/doc/topics/13-js-api-reference.md                        | 22      | 1101       | 1443      |
| ttmp/2026/05/08/GP-EVENT-VOCABULARY--split-provider-run-and- | 22      | 106        | 2         |
| text-segment-event-vocabulary/changelog.md                   |         |            |           |
| ttmp/2026/05/08/GP-EVENT-VOCABULARY--split-provider-run-and- | 21      | 1685       | 369       |
| text-segment-event-vocabulary/tasks.md                       |         |            |           |
| pkg/steps/ai/claude/engine_claude.go                         | 21      | 227        | 95        |
| examples/js/geppetto/README.md                               | 19      | 226        | 120       |
| pkg/js/modules/geppetto/provider/provider_test.go            | 19      | 1282       | 228       |
| pkg/doc/topics/14-js-api-user-guide.md                       | 18      | 735        | 927       |
| pkg/js/modules/geppetto/api_profiles.go                      | 15      | 80         | 1024      |
| ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-reg | 15      | 330        | 0         |
| istry-loading-and-remove-parseengineprofileregistrysourceent |         |            |           |
| ries/changelog.md                                            |         |            |           |
| pkg/steps/ai/openai/helpers.go                               | 15      | 318        | 180       |
| pkg/steps/ai/openai_responses/observability.go               | 15      | 596        | 120       |
| pkg/doc/topics/00-docs-index.md                              | 15      | 51         | 35        |
| pkg/steps/ai/claude/content-block-merger_test.go             | 14      | 883        | 101       |
| ttmp/2026/03/19/GP-50-REGISTRY-LOADING-CLEANUP--clean-up-reg | 14      | 90         | 16        |
| istry-loading-and-remove-parseengineprofileregistrysourceent |         |            |           |
| ries/tasks.md                                                |         |            |           |
\n# Docmgr ticket count by month
|  month  | tickets | docs | words  |
|---------|---------|------|--------|
| 2026-03 | 24      | 163  | 262517 |
| 2026-04 | 1       | 7    | 11395  |
| 2026-05 | 17      | 123  | 285173 |
| 2026-06 | 10      | 90   | 281620 |
\n# Largest docmgr tickets by written words
|                  ticket                   | ticket_day |                            title                             | docs | words  |
|-------------------------------------------|------------|--------------------------------------------------------------|------|--------|
| GP-RESPONSES-REPLAY                       | 2026-05-06 | Audit Responses API reasoning parsing and replay schema      | 10   | 144467 |
| 2026-06-05-geppetto-gemini-api-polish     | 2026-06-05 | Geppetto Gemini API Polish for Gemini 3 Flash                | 19   | 103895 |
| GP-GOJA-API-2026-06-01                    | 2026-06-01 | Review and redesign Geppetto go-go-goja API and JavaScript b | 8    | 49417  |
|                                           |            | indings                                                      |      |        |
| 2026-06-05-geppetto-provider-gap-audit    | 2026-06-05 | Geppetto Provider Gap Audit                                  | 16   | 49050  |
| GP-GOJA-STREAM-EVENTS-2026-06-01          | 2026-06-01 | Design Geppetto JS streaming events via go-go-goja event emi | 9    | 38206  |
|                                           |            | tter                                                         |      |        |
| GP-50-REGISTRY-LOADING-CLEANUP            | 2026-03-19 | Clean up registry loading and remove ParseEngineProfileRegis | 13   | 27616  |
|                                           |            | trySourceEntries                                             |      |        |
| GP-49-ENGINE-PROFILES                     | 2026-03-18 | reintroduce engine profiles and separate them from app runti | 7    | 22516  |
|                                           |            | me configuration                                             |      |        |
| GP-OBSERVABILITY                          | 2026-05-07 | Add Geppetto provider and event observability hooks for high | 8    | 21472  |
|                                           |            | -frequency inference debugging                               |      |        |
| GP-40-OPINIONATED-GO-APIS                 | 2026-03-17 | Opinionated Go APIs for Geppetto Runner Scaffolding          | 7    | 19703  |
| GP-EVENT-VOCABULARY                       | 2026-05-08 | Split provider, run, and text segment event vocabulary       | 9    | 19200  |
| GP-55-HTTP-PROXY                          | 2026-03-27 | Add HTTP proxy flags to Geppetto and Pinocchio               | 9    | 18174  |
| GP-41-REMOVE-PROFILE-OVERRIDES            | 2026-03-17 | Remove request-level profile override functionality from Gep | 8    | 17727  |
|                                           |            | petto profile resolution                                     |      |        |
| 2026-06-05-geppetto-llm-proxy-image-input | 2026-06-05 | Geppetto and llm-proxy Image Input Support                   | 8    | 14121  |
| GP-33                                     | 2026-03-15 | Extract scoped DB tool pattern into reusable geppetto packag | 6    | 13505  |
|                                           |            | e                                                            |      |        |
| GEP-EMBPROF-001                           | 2026-05-23 | Embedding Profiles for Geppetto and Pinocchio Registries     | 6    | 13273  |
| GP-CODE-REVIEW                            | 2026-05-07 | Code review and cleanup guide for Geppetto observability and | 7    | 13234  |
|                                           |            |  recent runtime integration                                  |      |        |
| GP-56-OPEN-RESPONSES                      | 2026-03-27 | Add Open Responses support to Geppetto with raw reasoning tr | 7    | 12807  |
|                                           |            | aces and semantic streaming                                  |      |        |
| GP-45-REMOVE-LEGACY-UNUSED-FUNCTIONALITY  | 2026-03-17 | Remove legacy and unused functionality from geppetto         | 6    | 12397  |
| GP-43-REMOVE-STEPSETTINGSPATCH            | 2026-03-17 | Remove StepSettingsPatch from Geppetto profile runtime and m | 7    | 11411  |
|                                           |            | ove final StepSettings resolution to callers                 |      |        |
| GP-53-OPENAI-RESPONSES-MULTIMODAL-MEDIA   | 2026-04-22 | Add multimodal media support to Geppetto openai-responses    | 7    | 11395  |
\n# Recent notable commits
| author_day |  short_sha   |   category    |                          subject                           |
|------------|--------------|---------------|------------------------------------------------------------|
| 2026-06-06 | 1ad8be2bf14a | merge         | Merge pull request #372 from wesen/bug/store-runtime-owner |
| 2026-06-06 | d091f5ff50dc | fixes         | embeddings: preserve embedding-local API settings          |
| 2026-06-06 | 66b3167bd328 | dependencies  | :arrow_up: Bump depenencies                                |
| 2026-06-06 | a8ef7d85aae6 | merge         | Merge pull request #371 from go-go-golems/task/llm-proxy   |
| 2026-06-06 | 712318b7fde6 | features      | geppetto: expose embeddings to JavaScript                  |
| 2026-06-06 | dd6f735c0423 | fixes         | Gemini: preserve function call thought signatures          |
| 2026-06-05 | bd293e52cb7d | maintenance   | Claude: wrap unsigned reasoning blocks                     |
| 2026-06-05 | 5a0da10d5921 | docs/research | Docs: close image input ticket                             |
| 2026-06-05 | 0bfce53115d9 | tests         | Image input: add smoke coverage                            |
| 2026-06-05 | bc33c233e89b | maintenance   | Gemini: map inline image input                             |
| 2026-06-05 | c63ae387ff93 | maintenance   | Image input: normalize provider mappings                   |
| 2026-06-05 | dcb3fa1746a0 | features      | Image input: add normalization helper                      |
| 2026-06-05 | b46598b9c604 | docs/research | Docs: plan image input support                             |
| 2026-06-05 | 3ed5a67125e6 | tests         | Gemini: modernize SDK path and smoke coverage              |
| 2026-06-05 | e4099e5cf061 | docs/research | Docs: create Gemini API polish guide                       |
| 2026-06-05 | d5412db204c6 | docs/research | Docs: audit provider gap matrix                            |
| 2026-06-05 | e21863f6ab52 | docs/research | Docs: create provider gap audit guide                      |
| 2026-06-05 | 1522d7bd881e | features      | Support Claude extended thinking streams                   |
| 2026-06-05 | 7043076d0658 | maintenance   | Force Claude engine streaming requests                     |
| 2026-06-05 | 1f2f057db446 | merge         | Merge pull request #370 from wesen/bug/store-runtime-owner |
| 2026-06-05 | 2d35c00bb619 | merge         | Merge something?                                           |
| 2026-06-04 | 2535c87421b3 | merge         | Merge pull request #369 from wesen/task/goja-runtime-flags |
| 2026-06-04 | 59b0c1266b16 | tests         | Preserve session filters in geppetto latest turn lookup    |
| 2026-06-04 | de32fe472ee8 | maintenance   | Avoid dynamic SQL in geppetto turn listing                 |
| 2026-06-04 | 1dacfd6edf77 | fixes         | Fix geppetto provider lint issues                          |
| 2026-06-04 | c85b3a7fa0b4 | dependencies  | :arrow_up: Bump depenencies                                |
| 2026-06-04 | 4c975f1bd6a3 | fixes         | Fix geppetto profile agent build nil API panic             |
| 2026-06-04 | 5aaa8748532e | maintenance   | Register geppetto provider resource closers                |
| 2026-06-04 | d89b75b23269 | features      | Add geppetto host service contributions                    |
| 2026-06-04 | 67a8571b565d | features      | Add geppetto xgoja turn store flags                        |
| 2026-06-04 | 6f0bc2d25117 | maintenance   | Simplify geppetto xgoja provider config                    |
| 2026-06-02 | 154f2b330075 | dependencies  | :arrow_up: Bump go mod                                     |
| 2026-06-02 | ec2cb1d914b4 | merge         | Merge pull request #367 from wesen/task/geppetto-js        |
| 2026-06-02 | 931d54c09f98 | docs/research | Diary: record CI async test fix                            |
| 2026-06-02 | 5acbd86763ea | tests         | Fix async turn store test race                             |
| 2026-06-02 | a2c6883afd4a | docs/research | Document JS goTool host registry behavior                  |
| 2026-06-02 | c96ad9226ca4 | docs/research | Diary: record PR review feedback                           |
| 2026-06-02 | 7db41813193f | features      | Address JS API review feedback                             |
| 2026-06-02 | 55c854cb03f8 | docs/research | Diary: record push validation follow-up                    |
| 2026-06-02 | 426bdff63dc0 | dependencies  | Bump Go toolchain for govulncheck                          |
