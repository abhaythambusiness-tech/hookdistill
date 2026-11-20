# hookdistill v0.1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Un binario Go que lee las sesiones locales de Claude Code, detecta correcciones repetidas y reglas de CLAUDE.md incumplidas, genera reglas, las reproduce contra el historial y actúa él mismo como hook.

**Architecture:** Un solo módulo con paquetes `internal/*` pequeños (transcript, rules, hook, detect, replay, install, report) y un `main` fino que despacha subcomandos. Las reglas son JSON lines; `hookdistill run` es el motor del hook, así que no se genera ningún script.

**Tech Stack:** Go 1.22 (el instalado es 1.22.2), solo biblioteca estándar, GitHub Actions, goreleaser, gitleaks.

**Spec:** `docs/superpowers/specs/2026-11-20-hookdistill-design.md` (aprobado 2026-11-20).

## Global Constraints

- Nombre `hookdistill`, licencia MIT, módulo `github.com/<login>/hookdistill` con `<login>` de `gh api user -q .login` (si `gh` no está autenticado, parar y preguntar a Haytham).
- Solo biblioteca estándar; sin red en ningún subcomando; el hook y los análisis son locales y deterministas (sin modelo).
- Regla JSON, un objeto por línea, campos exactos: `id`, `evento`, `herramienta`, `patron`, `accion` (`warn`|`block`), `mensaje`, `origen` (`correccion`|`CLAUDE.md`), `evidencia` (lista de `{session, seq}`).
- Archivo de reglas: `.claude/hookdistill.rules.json` (proyecto) y `~/.claude/hookdistill.rules.json` (global).
- Toda regla nace con `accion: "warn"`. Solo `promote` la sube a `block`, y solo si ningún golpe del replay está marcado como legítimo.
- Candidata de corrección: ≥3 apariciones en ≥2 sesiones.
- Coste de tokens: caracteres/4, siempre rotulado `estimated`; lo contado en transcripts se rotula `measured`.
- El informe redacta valores con forma de secreto antes de imprimir.
- CLI, README y mensajes en inglés (decisión del diseño); marcas de corrección en inglés y español.
- Comandos de test: `go vet ./...` y `go test ./...`. Cada commit termina con `Co-Authored-By: Claude Sonnet 5.5 <noreply@anthropic.com>`.

## Review Focus

1. Líneas del `.jsonl` corruptas, truncadas o con tipos desconocidos: se saltan, se cuentan y se informa; nunca pánico (Task 2).
2. Historial vacío o carpeta `~/.claude/projects` inexistente: mensaje claro y salida 0 (Task 8).
3. Archivo de reglas ilegible o con regex inválida en `run`: el hook **no bloquea** (falla abierto), avisa por stderr y sale 0 (Task 4).
4. `settings.json` con hooks previos, orden de claves o JSON inválido: se conserva todo, se hace copia de seguridad y se rechaza JSON inválido sin tocar nada (Task 10).
5. Secretos dentro de las sesiones (`sk-…`, `ghp_…`, `AKIA…`, `Bearer …`) aparecen en el informe o en la evidencia: se redactan (Task 8).

---

### Task 1: Scaffold y despacho de subcomandos

**Files:**
- Create: `go.mod`, `LICENSE` (MIT, titular: Haytham), `cmd/hookdistill/main.go`, `cmd/hookdistill/main_test.go`
- Modify: `.gitignore` (añadir `/hookdistill`)

**Interfaces:**
- Produces: `func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int` en `package main`; tabla `commands map[string]func(args []string, stdin io.Reader, stdout, stderr io.Writer) int` que las tareas 4, 8, 9 y 10 rellenan.

- [ ] **Step 1: Test que falla** — `TestRunUnknownCommand` (args `["nope"]` → código 2, stderr contiene `unknown command`), `TestRunVersion` (`["version"]` → código 0, stdout empieza por `hookdistill `), `TestRunNoArgsPrintsUsage` (sin args → código 2, stderr contiene `usage:`).
- [ ] **Step 2: Ejecutar** `go test ./cmd/...` → FAIL (`run` no definido).
- [ ] **Step 3: Implementar** `run` con la tabla `commands`; `version` imprime `hookdistill dev` (se sobrescribe con `-ldflags` en la Task 12). `git init` ya está hecho; `go mod init` con el módulo de Global Constraints.
- [ ] **Step 4: Ejecutar** `go vet ./... && go test ./...` → PASS.
- [ ] **Step 5: Commit** `feat: scaffold CLI dispatch and MIT license`.

### Task 2: Lector de transcripts

**Files:**
- Create: `internal/transcript/transcript.go`, `internal/transcript/transcript_test.go`, `internal/transcript/testdata/session_basic.jsonl`, `internal/transcript/testdata/session_corrupt.jsonl`

**Interfaces:**
- Produces:
  - `type ToolCall struct { Session string; Seq int; Time time.Time; Tool string; Input map[string]any; Sidechain bool; Project string }`
  - `type UserMsg struct { Session string; Seq int; Time time.Time; Text string; Project string }`
  - `type Session struct { ID, Project string; Calls []ToolCall; Msgs []UserMsg; Skipped int }`
  - `func LoadFile(path string) (Session, error)`; `func LoadAll(root string) ([]Session, error)` (recorre `root/*/*.jsonl`; `root` inexistente → `nil, nil`)
  - `func (c ToolCall) Subject() string` — `command` para `Bash`; `file_path` para `Read`, `Edit`, `MultiEdit`, `Write`; `""` en otro caso.

Formato real observado (2026-11-20): cada línea es un objeto con `type`, `sessionId`, `timestamp`, `isSidechain`; `type:"assistant"` lleva `message.content[]` con bloques `{"type":"tool_use","name","input"}`; `type:"user"` lleva `message.content` como cadena o como lista de bloques (`text`, `tool_result`). Hay muchos otros `type` (`attachment`, `api-request`, `system`, …) que se ignoran. `Seq` es el número de línea.

- [ ] **Step 1: Tests que fallan** — `TestLoadFileBasic` (fixture con 1 mensaje de usuario, 1 `tool_use` de Bash `git commit -am x` y 1 `tool_result` → 1 `UserMsg`, 1 `ToolCall`, el `tool_result` no cuenta como mensaje de usuario); `TestLoadFileSkipsCorruptAndUnknown` (fixture con una línea truncada, una con `type` desconocido y una con `content` de tipo inesperado → `Skipped == 2`, sin error, las líneas válidas se leen); `TestUserTextIgnoresInjectedBlocks` (un bloque `text` que empieza por `<` no es mensaje de usuario); `TestSubject` (Bash → comando; Edit → ruta; `Task` → `""`); `TestLoadAllMissingRoot` (`nil, nil`).
- [ ] **Step 2: Ejecutar** `go test ./internal/transcript/` → FAIL.
- [ ] **Step 3: Implementar** con `bufio.Scanner` de buffer ampliado a 16 MiB por línea; decodificar a tipos laxos (`json.RawMessage`) para que un campo inesperado no aborte; `Project` es el nombre de la carpeta padre.
- [ ] **Step 4: Ejecutar** `go vet ./... && go test ./...` → PASS.
- [ ] **Step 5: Commit** `feat: tolerant Claude Code transcript reader`.

### Task 3: Modelo de reglas

**Files:**
- Create: `internal/rules/rules.go`, `internal/rules/rules_test.go`

**Interfaces:**
- Produces:
  - `type Evidence struct { Session string \`json:"session"\`; Seq int \`json:"seq"\` }`
  - `type Rule struct { ID, Evento, Herramienta, Patron, Accion, Mensaje, Origen string; Evidencia []Evidence }` con las etiquetas JSON de Global Constraints
  - `func Load(path string) ([]Rule, error)` (archivo inexistente → `nil, nil`); `func Save(path string, rs []Rule) error` (JSON lines, escritura atómica con archivo temporal + rename)
  - `func (r Rule) Validate() error` (regex compila, `Accion` en `warn|block`, `Evento == "PreToolUse"`, `ID` no vacío)
  - `func (r Rule) Matches(tool, subject string) bool` (`Herramienta` vacía o `"*"` = cualquiera; `Patron` se compila con `regexp` de Go, que es RE2 y no tiene retroceso catastrófico)

- [ ] **Step 1: Tests que fallan** — `TestRoundTrip` (Save + Load devuelve lo mismo), `TestLoadMissingFile`, `TestValidateRejectsBadRegex`, `TestValidateRejectsBadAction`, `TestMatches` (tabla: `Bash` + `git commit -am` casa con patrón `git commit -am`; `Edit` no casa con regla de `Bash`; herramienta `*` casa con todo).
- [ ] **Step 2: Ejecutar** → FAIL.
- [ ] **Step 3: Implementar** las firmas anteriores.
- [ ] **Step 4: Ejecutar** `go vet ./... && go test ./...` → PASS.
- [ ] **Step 5: Commit** `feat: rule model with validation and atomic save`.

### Task 4: Motor del hook (`hookdistill run`)

**Files:**
- Create: `internal/hook/hook.go`, `internal/hook/hook_test.go`
- Modify: `cmd/hookdistill/main.go` (registrar `run`)

**Interfaces:**
- Consumes: `rules.Load`, `Rule.Validate`, `Rule.Matches` (Task 3).
- Produces: `func Run(stdin io.Reader, stdout, stderr io.Writer, ruleFiles []string) int`; entrada: JSON `{"tool_name","tool_input"}`; `func warnOutput(msg string) []byte` (único lugar que sabe el formato de aviso).

- [ ] **Step 0: Verificar el contrato vigente** de salida de PreToolUse en la documentación de Claude Code (context7). Contrato asumido por el diseño: bloquear = código 2 y mensaje por stderr; aviso = JSON por stdout con `permissionDecision: "ask"` y razón. Si la documentación difiere, ajustar solo `warnOutput` y dejar el cambio anotado en el test.
- [ ] **Step 1: Tests que fallan** — `TestRunBlocksMatchingBlockRule` (regla `block` + llamada que casa → código 2 y el mensaje de la regla en stderr); `TestRunWarnsMatchingWarnRule` (código 0 y stdout igual a `warnOutput(mensaje)`); `TestRunAllowsNonMatching` (código 0, stdout vacío); `TestRunFailsOpenOnBadRules` (archivo con JSON roto y otro con regex inválida → código 0, stderr con `hookdistill:` y sin bloquear); `TestRunFailsOpenOnBadInput` (stdin no JSON → código 0); `TestRunMergesProjectAndGlobalRules`.
- [ ] **Step 2: Ejecutar** → FAIL.
- [ ] **Step 3: Implementar** `Run`; ante cualquier error propio del motor devuelve 0 (un guardia roto nunca debe parar el trabajo del usuario). El subcomando `run` pasa `[".claude/hookdistill.rules.json", "<home>/.claude/hookdistill.rules.json"]`.
- [ ] **Step 4: Ejecutar** `go vet ./... && go test ./...` → PASS.
- [ ] **Step 5: Commit** `feat: hook engine that fails open`.

### Task 5: Detector de correcciones repetidas

**Files:**
- Create: `internal/detect/corrections.go`, `internal/detect/corrections_test.go`, `internal/detect/testdata/corrections_sessions.go` (constructor de sesiones en memoria para los tests)

**Interfaces:**
- Consumes: `transcript.Session`, `transcript.ToolCall.Subject`, `rules.Rule` (Tasks 2–3).
- Produces: `type Candidate struct { Rule rules.Rule; Count, Sessions int }`; `func Corrections(ss []transcript.Session) []Candidate`; `func signature(c transcript.ToolCall) string` (herramienta + primeras dos palabras del comando, o herramienta + directorio de `file_path`).

Marcas de corrección (insensibles a mayúsculas, al inicio del mensaje o tras puntuación): EN `no`, `stop`, `don't`, `do not`, `i said`, `again`, `never`, `wrong`, `revert`, `undo`; ES `no`, `para`, `no hagas`, `ya te dije`, `otra vez`, `nunca`, `mal`, `revierte`, `deshaz`.

- [ ] **Step 1: Tests que fallan** — `TestCorrectionsClusters` (3 correcciones tras `git commit -am` en 2 sesiones → 1 candidata con `Count 3`, `Sessions 2`, `Origen "correccion"`, `Accion "warn"`, patrón que casa con `git commit -am`); `TestCorrectionsBelowThreshold` (2 apariciones → ninguna); `TestCorrectionsSameSessionOnlyNotEnough` (3 apariciones en 1 sesión → ninguna); `TestCorrectionsSpanish` (`ya te dije que no` cuenta); `TestNoFalseMarkOnPlainNo` (`no hay problema, sigue` no es corrección: exige que la marca vaya en imperativo o negación del siguiente verbo — probar con 3 frases negativas y 3 positivas); `TestSidechainCallsIgnored`.
- [ ] **Step 2: Ejecutar** → FAIL.
- [ ] **Step 3: Implementar** `Corrections`: para cada mensaje de usuario con marca, tomar la última `ToolCall` no sidechain con `Seq` menor, agrupar por `signature`, filtrar por umbral, ordenar por `Count` descendente. La heurística de la marca se afina con los 538 transcripts reales en la Task 11, no aquí.
- [ ] **Step 4: Ejecutar** `go vet ./... && go test ./...` → PASS.
- [ ] **Step 5: Commit** `feat: detect repeated user corrections`.

### Task 6: Reglas de CLAUDE.md y coste estimado

**Files:**
- Create: `internal/detect/claudemd.go`, `internal/detect/claudemd_test.go`, `internal/detect/testdata/CLAUDE.md`

**Interfaces:**
- Consumes: `Candidate`, `rules.Rule`, `transcript.Session`, `Rule.Matches` (Tasks 2–5).
- Produces: `type NotConvertible struct { Line int; Text, Reason string }`; `func ClaudeMD(path string, ss []transcript.Session) ([]Candidate, []NotConvertible, error)`; `func EstimateTokens(chars, sessionsLoaded int) int` (= `chars/4*sessionsLoaded`).

Alcance de v1 (decisión mía, para no prometer lo que no hay): una línea es convertible solo si contiene una negación o prohibición (`never`, `don't`, `do not`, `must not`, `nunca`, `prohibido`, `no `) **y** un fragmento entre comillas invertidas (`` `git commit -am` ``, `` `rm -rf` ``); el fragmento, escapado con `regexp.QuoteMeta`, es el patrón sobre `Bash`. Todo lo demás es `NotConvertible` con motivo (`no backticked command or path`, `positive instruction`, `ambiguous`).

- [ ] **Step 1: Tests que fallan** — `TestClaudeMDConvertible` (línea «Never run `git push --force`» → candidata con patrón `git push --force`, `Origen "CLAUDE.md"`); `TestClaudeMDNotConvertible` (línea «Write clean code» → `NotConvertible` con `Reason`); `TestClaudeMDCountsViolations` (historial con 2 llamadas `Bash` que casan → `Count 2`); `TestClaudeMDMissingFile` (error claro); `TestEstimateTokens` (`EstimateTokens(400, 10) == 1000`).
- [ ] **Step 2: Ejecutar** → FAIL.
- [ ] **Step 3: Implementar** las firmas anteriores.
- [ ] **Step 4: Ejecutar** `go vet ./... && go test ./...` → PASS.
- [ ] **Step 5: Commit** `feat: map CLAUDE.md rules to patterns and estimate cost`.

### Task 7: Replay y promoción

**Files:**
- Create: `internal/replay/replay.go`, `internal/replay/replay_test.go`

**Interfaces:**
- Consumes: `rules.Rule`, `transcript.Session` (Tasks 2–3).
- Produces: `type Hit struct { ID, Session string; Seq int; Tool, Subject string }` (`ID` = primeros 8 caracteres de la sesión + `:` + `Seq`); `func Replay(r rules.Rule, ss []transcript.Session) []Hit`; `func Promote(r rules.Rule, hits []Hit, legit map[string]bool) (rules.Rule, error)` (devuelve la regla con `Accion "block"`; error si algún `Hit.ID` está en `legit`).

- [ ] **Step 1: Tests que fallan** — `TestReplayFindsHits` (regla `git commit -am`, historial con 2 coincidencias y 1 no → 2 golpes con IDs esperados); `TestReplayIgnoresSidechain`; `TestPromoteRefusesWhenLegit` (un golpe marcado legítimo → error y la regla original intacta); `TestPromoteOK` (sin marcas → `Accion == "block"`).
- [ ] **Step 2: Ejecutar** → FAIL.
- [ ] **Step 3: Implementar** las firmas anteriores.
- [ ] **Step 4: Ejecutar** `go vet ./... && go test ./...` → PASS.
- [ ] **Step 5: Commit** `feat: replay rules against history and gated promote`.

### Task 8: `scan` con redacción de secretos

**Files:**
- Create: `internal/report/report.go`, `internal/report/redact.go`, `internal/report/report_test.go`
- Modify: `cmd/hookdistill/main.go` (registrar `scan`)

**Interfaces:**
- Consumes: `transcript.LoadAll`, `detect.Corrections`, `detect.ClaudeMD`, `detect.EstimateTokens` (Tasks 2, 5, 6).
- Produces: `func Redact(s string) string`; `func Scan(root, claudeMDPath string, w io.Writer) error`. Salida: sesiones leídas y líneas saltadas, candidatas con cuenta y evidencia, reglas de CLAUDE.md `convertible / not convertible` (con motivo), tokens `estimated` y `measured` por separado.

- [ ] **Step 1: Tests que fallan** — `TestRedact` (tabla: `sk-ant-api03-abc…`, `ghp_` + 36 caracteres, `AKIA` + 16, `Bearer abc.def` → `[REDACTED]`; texto normal intacto); `TestScanEmptyHistory` (raíz inexistente → mensaje `no sessions found` y error `nil`); `TestScanRedactsSecrets` (sesión cuyo mensaje contiene un `ghp_…` → el texto del informe no lo contiene); `TestScanLabelsEstimates` (la salida contiene `estimated` junto al coste de CLAUDE.md); `TestScanReportsSkippedLines`.
- [ ] **Step 2: Ejecutar** → FAIL.
- [ ] **Step 3: Implementar** `Redact` con expresiones regulares y `Scan`; el subcomando `scan` acepta `--root` (por defecto `~/.claude/projects`) y `--claude-md` (por defecto `./CLAUDE.md`, si no existe solo omite esa sección).
- [ ] **Step 4: Ejecutar** `go vet ./... && go test ./...` → PASS.
- [ ] **Step 5: Commit** `feat: scan report with secret redaction`.

### Task 9: `propose`, `replay`, `promote`

**Files:**
- Modify: `cmd/hookdistill/main.go`
- Create: `cmd/hookdistill/propose_test.go`

**Interfaces:**
- Consumes: `detect.*`, `replay.*`, `rules.Load/Save` (Tasks 3, 5–7).
- Produces: subcomandos `propose [--root --claude-md --out]` (añade candidatas nuevas al archivo de reglas sin duplicar por `Patron` y sin tocar las existentes), `replay <id> [--root --legit hit,hit]` (lista golpes con contexto redactado y los guarda en `.claude/hookdistill.replay.json`), `promote <id> [--legit hit,hit]` (usa `replay.Promote`).

- [ ] **Step 1: Tests que fallan** — `TestProposeWritesWarnRules`, `TestProposeIsIdempotent` (dos ejecuciones no duplican), `TestProposeKeepsExistingRules`, `TestReplayPrintsHits`, `TestPromoteRefusesLegit`, `TestPromoteUnknownID` (error claro). Todos con `t.TempDir()` como proyecto y sesiones de fixture.
- [ ] **Step 2: Ejecutar** → FAIL.
- [ ] **Step 3: Implementar** los tres subcomandos; `ID` de regla = `c-<hash corto del patrón>` para que sea estable entre ejecuciones.
- [ ] **Step 4: Ejecutar** `go vet ./... && go test ./...` → PASS.
- [ ] **Step 5: Commit** `feat: propose, replay and promote commands`.

### Task 10: `install` y `uninstall`

**Files:**
- Create: `internal/install/install.go`, `internal/install/install_test.go`
- Modify: `cmd/hookdistill/main.go`

**Interfaces:**
- Produces: `func Install(settingsPath, binPath string) error`; `func Uninstall(settingsPath string) error`. Añaden/quitan una entrada `PreToolUse` con `matcher: "*"` y comando `"<binPath> run"` identificada por esa cadena; idempotentes; copia `settings.json.hookdistill.bak` antes de modificar; el subcomando acepta `--project` (usa `.claude/settings.json`) o, por defecto, `~/.claude/settings.json`.

- [ ] **Step 1: Tests que fallan** — `TestInstallCreatesFile` (no existe → se crea con una entrada); `TestInstallPreservesOtherHooksAndKeys` (settings con otro hook `PreToolUse` y `permissions` → siguen intactos); `TestInstallIdempotent` (dos veces → una entrada); `TestInstallRejectsInvalidJSON` (archivo roto → error y archivo sin cambios); `TestInstallWritesBackup`; `TestUninstallRemovesOnlyOurEntry`.
- [ ] **Step 2: Ejecutar** → FAIL.
- [ ] **Step 3: Implementar** leyendo a `map[string]any` y reescribiendo con `json.MarshalIndent`; escritura atómica.
- [ ] **Step 4: Ejecutar** `go vet ./... && go test ./...` → PASS.
- [ ] **Step 5: Commit** `feat: safe install and uninstall of the hook entry`.

### Task 11: Ejemplos anonimizados y prueba de extremo a extremo

**Files:**
- Create: `examples/sessions/` (≥6 sesiones sintéticas con tres correcciones repetidas, ninguna con datos reales), `examples/CLAUDE.md`, `cmd/hookdistill/e2e_test.go`

- [ ] **Step 1: Test que falla** — `TestE2EScanProposeReplay`: sobre `examples/` ejecuta `scan`, `propose`, `replay`; espera ≥3 reglas candidatas, ≥1 golpe en el replay y que la salida no contenga ninguna cadena de la lista de secretos sembrados en los ejemplos.
- [ ] **Step 2: Ejecutar** → FAIL (faltan ejemplos).
- [ ] **Step 3: Crear los ejemplos** escritos a mano; no copiar sesiones reales.
- [ ] **Step 4: Prueba sobre datos reales (manual, sin commit de datos):** `go run ./cmd/hookdistill scan` sobre `~/.claude/projects` de Haytham; anotar en `docs/superpowers/plans/` cuántas candidatas salen y cuántas marcas de corrección dan falsos positivos; afinar la lista de marcas de la Task 5 con ese resultado y repetir los tests. Criterio del diseño: ≥5 candidatas con evidencia.
- [ ] **Step 5: Ejecutar** `go vet ./... && go test ./...` → PASS y **Commit** `test: end-to-end run on synthetic examples`.

### Task 12: CI, release e instalador

**Files:**
- Create: `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.goreleaser.yaml`, `.gitleaks.toml`, `install.sh`

- [ ] **Step 1:** `ci.yml`: en push y PR, matriz `ubuntu`, `macos`, `windows`; pasos `go vet ./...`, `go test ./...`, build; trabajo aparte con gitleaks sobre el historial.
- [ ] **Step 2:** `.goreleaser.yaml`: binarios linux/darwin/windows × amd64/arm64, `checksums.txt`, `-ldflags "-X main.version={{.Version}}"` (ajustar `main.go` de la Task 1 para leer esa variable), changelog desde commits.
- [ ] **Step 3:** `release.yml`: al empujar un tag `v*` ejecuta goreleaser.
- [ ] **Step 4:** `install.sh`: descarga el binario de la última release, **verifica el checksum** antes de instalar, instala en `~/.local/bin`, no usa `sudo`.
- [ ] **Step 5: Verificar en local** (sin publicar nada): `go vet ./... && go test ./...`; `goreleaser check` si está instalado; `shellcheck install.sh` si está instalado. La comprobación «CI en verde» solo se puede hacer tras el primer push, que decide Haytham.
- [ ] **Step 6: Commit** `ci: workflows, goreleaser and checksum installer`.

### Task 13: Documentación y activos del repo

**Files:**
- Create: `README.md`, `CONTRIBUTING.md`, `CHANGELOG.md`, `.github/ISSUE_TEMPLATE/bug_report.yml`, `.github/ISSUE_TEMPLATE/feature_request.yml`, `.github/ISSUE_TEMPLATE/config.yml`, `assets/demo.tape`, `assets/social.svg`

- [ ] **Step 1: README (en inglés)**, en este orden: GIF arriba (`assets/demo.gif`), una línea («Turn the corrections you keep repeating to Claude Code into hooks, tested against your own history.»), instalación en un comando, demo de 3 comandos, tabla honesta de comparación (hookify, PRISM, rule2hook, con estrellas verificadas y fecha), límites (hooks y subagentes/MCP/modo `-p`; heurísticas v1; coste `estimated`), privacidad (local, sin red).
- [ ] **Step 2:** `CONTRIBUTING.md` (compilar, probar, cómo añadir una marca de corrección en otro idioma, estilo de commits); `CHANGELOG.md` con entrada `0.1.0`; plantillas de bug y de idea; `config.yml` que enlaza a Discussions.
- [ ] **Step 3:** `assets/demo.tape` (script de `vhs`) que graba `scan` → `propose` → `replay` sobre `examples/`; generar `assets/demo.gif` con `vhs assets/demo.tape` si `vhs` está instalado; si no, dejar el `.tape` y decirlo. `assets/social.svg` 1280×640; convertir a PNG con `rsvg-convert` si existe.
- [ ] **Step 4: Verificar:** el GIF existe y el README lo referencia arriba del todo; los enlaces relativos del README resuelven (`grep -o '](\./[^)]*' README.md` y comprobar cada ruta con `ls`).
- [ ] **Step 5: Commit** `docs: README, contributing, templates and assets`.

### Task 14: Paquete de lanzamiento (solo borradores)

**Files:**
- Create: `docs/launch/show-hn.md`, `docs/launch/good-first-issues.md`, `docs/launch/checklist.md`

- [ ] **Step 1:** `show-hn.md`: título y primer comentario, sin afirmaciones de estrellas ni de ahorro no medido; usa los números reales de la Task 11.
- [ ] **Step 2:** `good-first-issues.md`: 5 incidencias pequeñas redactadas (nuevas marcas de corrección en francés/alemán/portugués, más patrones de herramienta, ejemplo de regla para `Edit`, mensaje de error más claro, prueba en Windows).
- [ ] **Step 3:** `checklist.md`: los 16 puntos de la sección 5b del diseño con estado real tras las tareas 1–13 y los que **solo puede hacer Haytham**: crear el repo remoto, hacer push, activar Discussions, poner temas (`claude-code`, `hooks`, `claude-md`, `golang`, `cli`, `developer-tools`), etiquetar `v0.1.0`, publicar en Show HN, responder incidencias.
- [ ] **Step 4:** Nada de esto se publica ni se empuja desde este plan. **Commit** `docs: launch drafts`.

---

## Autorrevisión

- **Cobertura del diseño:** comandos `scan/propose/replay/install/uninstall/run` (Tasks 4, 8–10); detección de correcciones (5), de reglas de CLAUDE.md (6), coste rotulado (6, 8); replay y `block` solo sin falsos positivos (7, 9); privacidad y redacción (8); lista de lanzamiento (12–14); criterio de éxito sobre datos reales (11). `promote` no está en la lista de comandos del diseño: es la forma de cumplir su regla «pasa a `block` solo si el replay…», y queda anotada como decisión mía.
- **Consistencia de tipos:** `ToolCall.Subject()`, `Rule.Matches(tool, subject)`, `Candidate`, `Hit`, `Session.Skipped` se usan con los mismos nombres en todas las tareas.
- **No cubierto, a propósito:** subagentes y servidores MCP (declarado como límite en el README); detección con modelo (fuera de v1).
