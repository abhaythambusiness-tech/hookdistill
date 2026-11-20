---
origen: mixto
status: aprobado-2026-11-20 (nombre hookdistill y licencia MIT confirmados por Haytham)
fecha: 2026-11-20
nombre: hookdistill
---

# hookdistill — diseño

> **En cristiano:** Claude Code a veces se salta lo que escribes en CLAUDE.md, y tú acabas repitiendo las mismas correcciones. `hookdistill` lee tus sesiones guardadas, encuentra las reglas que Claude incumple y las correcciones que repites, y las convierte en un «guardia» automático (un hook) que no se puede saltar. Antes de activarlo, lo prueba contra tu historial para demostrar que no bloquea cosas legítimas. Es un único programa, sin instalar nada más, y nada sale de tu ordenador.

## 1. Intención y porqué

- **Problema (datos propios, `foros-claude-code`, 437 posts de 30 días):** 21 quejas de «se salta las reglas/CLAUDE.md», 34 de acciones destructivas no pedidas, 58 de «dice hecho sin pruebas». Casos concretos: `git commit -am` empujando trabajo ajeno pese a una regla en memoria; «ignora prompts, reglas y skills»; «rompe las mismas reglas 7 veces en 7 días».
- **Por qué hooks:** las instrucciones de CLAUDE.md son peticiones que el modelo puede ignorar; los hooks y reglas de denegación los aplica Claude Code siempre. Las cifras de cumplimiento que circulan (~80 % frente a 100 %) son anecdóticas, no medidas.
- **Por qué ahora:** el parque de herramientas existe pero ninguna cierra el ciclo «evidencia en tus sesiones → hook → prueba contra tu historial».
- **Objetivo de Haytham:** repo para otros usuarios que gane estrellas en GitHub. No es herramienta interna de NeoZeler.

## 2. Validación de competencia (before-you-build, 2026-11-20)

| Herramienta | Qué hace | Estado | Hueco que deja |
|---|---|---|---|
| [claudecode-rule2hook](https://github.com/zxdxjtu/claudecode-rule2hook) | Texto de reglas → hooks | 402★, último push 2025-07-01 (parada) | No mira sesiones; no prueba el hook |
| hookify (plugin oficial, `claude-plugins-official`) | Reglas en markdown; sin argumentos analiza la conversación reciente y propone reglas | Oficial; estrellas no comprobadas | Solo la conversación actual, no todo el historial; usa el modelo (gasta tokens); sin medición |
| [PRISM](https://github.com/jakeefr/prism) | Lee `~/.claude/projects`, mide coste de CLAUDE.md, detecta reglas ignoradas, recomienda diffs | 21★, Python, push 2026-06-07 | Recomienda texto, no genera hook; requiere Python |
| cctx-cli (PyPI 1.24.4) | Diagnóstico de sesiones, parches para CLAUDE.md | Estrellas no comprobadas | Igual: parches de texto, no hooks |
| claude-enforce (npm) | Genera hooks desde reglas, `--dry-run` | No comprobado | Solo texto de reglas |
| skill `self-review` / `mine-transcripts` | Detecta correcciones repetidas (palabras tipo «no», «stop») | Skills sueltas | Dependen del modelo; sin hook |

**Veredicto: GO, estrecho.** Nadie hace las tres cosas juntas: (1) minar *todo* el historial, (2) emitir un hook ejecutable, (3) **reproducirlo contra el historial** para mostrar cuántas llamadas habría bloqueado y revisar los falsos positivos. El hueco es la repetición con prueba, no la conversión de texto.

**Riesgos de mercado, sin maquillar**
- Los competidores directos tienen pocas estrellas (21 y 402): el nicho existe pero no es masivo. Las estrellas dependerán del lanzamiento y de una demo que impresione, no del tamaño del nicho.
- **Riesgo de plataforma:** Anthropic ya tiene `/insights` y hookify; hay un issue abierto pidiendo que `/insights` verifique la configuración. Puede absorber la idea. Mitigación: ser local, determinista, sin modelo, y publicar rápido.
- El formato de los transcripts `.jsonl` no está documentado como contrato estable.
- Los hooks no cubren subagentes, MCP ni modo `-p` de forma fiable (según una fuente; sin verificar). Hay que decirlo en el README.

## 3. Metas, no-metas, anti-metas

**Metas**
- Un binario Go sin dependencias, multiplataforma, offline.
- Informe en terminal con números: reglas incumplidas, correcciones repetidas, tokens estimados.
- Generar reglas ejecutables y **reproducirlas** contra el historial antes de activarlas.
- Instalar el hook con un comando y poder desinstalarlo.

**No-metas (v1)**
- No llama a ningún modelo (sin red). Opcional en versiones futuras.
- No reescribe CLAUDE.md.
- No intenta convertir reglas que no sean deterministas («escribe código limpio»): las lista como *no convertibles* y dice por qué.

**Anti-metas (si pasa esto, fracasó aunque compile)**
- Un hook generado que bloquea trabajo legítimo del usuario.
- Cualquier byte de transcript sale de la máquina.
- Prometer ahorro de tokens sin número o presentarlo como medido cuando es estimado.
- Que el README parezca otro «rule2hook» más.

## 4. Enfoques considerados

| | Qué es | A favor | En contra |
|---|---|---|---|
| A | Solo análisis estático de CLAUDE.md → hooks | Rápido de hacer | Choca de frente con rule2hook, claude-enforce y hookify; sin diferencia |
| **B (elegido)** | **Binario Go: mina sesiones + reproduce + ejecuta el hook él mismo** | Único; determinista; sin tokens; demo potente | Hay que parsear un formato no estable |
| C | Plugin/skill de Claude que use el modelo | Distribución fácil | No determinista, gasta tokens (contradice el objetivo) y compite con hookify oficial |

**Elegido por Haytham: B** (propuesto por mí; aprobado al decir «minar las sesiones» y «valida la competencia y escribe el diseño»). Un envoltorio de plugin puede venir después, solo para distribución.

## 5. Diseño (decisiones de Claude, a revisar)

**Comandos**
- `hookdistill scan` — recorre `~/.claude/projects/**/*.jsonl` y los `CLAUDE.md`; imprime el informe.
- `hookdistill propose` — escribe reglas candidatas en `.claude/hookdistill.rules.json` (por proyecto) para revisión.
- `hookdistill replay <regla>` — aplica la regla a cada llamada de herramienta del historial; lista las que habría bloqueado, con contexto, para marcar falsos positivos.
- `hookdistill install` / `uninstall` — añade/quita una entrada `hookdistill run` en `settings.json` (PreToolUse).
- `hookdistill run` — es el hook: lee el JSON por stdin, compara con las reglas, sale con código 2 y mensaje si bloquea. **Sin generar scripts**: el mismo binario es el motor, así el hook es auditable y no hay código generado que revisar.

**Detección**
1. *Correcciones repetidas:* mensajes del usuario justo después de una llamada de herramienta que contienen marcas de corrección (EN/ES: «no», «stop», «don't», «I said», «otra vez», «no hagas»). Se agrupan por firma de la llamada anterior (herramienta + primer token del comando o patrón de ruta). Candidata si aparece ≥3 veces en ≥2 sesiones.
2. *Reglas de CLAUDE.md incumplidas:* líneas con forma determinista («nunca/never/prohibido X», «antes de X hacer Y»). Se extrae un patrón sobre `tool_name` + `tool_input` y se busca en el historial. Las líneas sin patrón posible se cuentan como *no convertibles*.
3. *Coste (estimado, así rotulado):* tokens ≈ caracteres/4 de la regla × sesiones donde se carga; más turnos gastados tras cada corrección. El informe separa «medido en transcript» de «estimado».

**Formato de regla** (JSON, una línea por regla): `id`, `evento`, `herramienta`, `patron`, `accion` (`warn` | `block`), `mensaje`, `origen` (corrección | CLAUDE.md), `evidencia` (enlaces a sesión y turno).

**Seguridad por defecto:** toda regla nace en `warn`. Pasa a `block` solo si el usuario lo pide **y** `replay` mostró 0 bloqueos de llamadas que él marcó como legítimas.

**Privacidad:** solo lectura de `~/.claude`; sin red; el informe redacta valores que parezcan secretos antes de imprimir.

**Distribución y estrellas (plan, sin garantías)**
- README con GIF de 20 s: `scan` sobre sesiones reales → regla → `replay` con 0 falsos positivos.
- Tabla honesta de comparación con hookify, PRISM, rule2hook (la de arriba).
- `go install`, binarios con goreleaser y script de una línea.
- Lanzamiento: Show HN + r/ClaudeAI + issue de referencia en los repos competidores solo si aporta, sin spam.
- Datos de ejemplo anonimizados en `examples/` para que cualquiera pruebe sin exponer sus sesiones.

## 5b. Lista de lanzamiento (16 puntos de Haytham, objetivo 10K estrellas)

La cifra de 10K es una ambición, no un compromiso: ningún competidor directo pasa de 402★. Se mide por pasos (primero 100★ en 30 días).

| # | Punto | Cómo se cumple | Quién | Cuándo |
|---|---|---|---|---|
| 1 | GIF arriba del README | Grabar `scan` → `propose` → `replay` sobre datos de `examples/` (no sesiones reales) | Claude graba, Haytham aprueba | v0.1 |
| 2 | Una línea | «Convierte las correcciones que repites a Claude Code en hooks, probados contra tu historial.» | Claude | v0.1 |
| 3 | Instalar en 1 comando | `go install` + script `curl \| sh` que verifica checksum | Claude | v0.1 |
| 4 | Demo en vivo | Página estática con un informe de ejemplo (sin servidor; los datos son locales por diseño) | Claude | v0.2 |
| 5 | Licencia | MIT, **a confirmar por Haytham** | Haytham decide | antes del primer push |
| 6 | Temas | `claude-code`, `hooks`, `claude-md`, `golang`, `cli`, `developer-tools` | Claude propone | al crear el repo |
| 7 | Imagen social | 1280×640 | Claude | v0.1 |
| 8 | CI en verde | GitHub Actions: `go vet`, `go test`, build en Linux/macOS/Windows | Claude | v0.1 |
| 9 | Sin secretos en Git | Repo nuevo, sin historial previo; `guard-secretos` antes de cada commit; gitleaks en CI | Claude | siempre |
| 10 | Good first issues | 5 incidencias pequeñas etiquetadas (p. ej. nuevas marcas de corrección en otros idiomas) | Claude redacta, Haytham publica | v0.1 |
| 11 | CONTRIBUTING.md | Cómo compilar, probar, añadir una heurística | Claude | v0.1 |
| 12 | Plantillas y Discussions | Plantillas de bug y de idea; activar Discussions | Claude escribe, Haytham activa | v0.1 |
| 13 | Release y changelog | Tag `v0.1.0`, goreleaser, CHANGELOG.md | Claude prepara, Haytham etiqueta | v0.1 |
| 14 | Show HN | Texto preparado por Claude; **lo publica Haytham** desde su cuenta | Haytham | tras v0.1 estable |
| 15 | Responder en <1 día | **No lo puede garantizar Claude** (no actúa sin que se le invoque). Opción: tarea programada que avise de incidencias nuevas | Haytham | tras publicar |
| 16 | Cero errores | Imposible de prometer. En su lugar: tests, `replay` de la propia herramienta sobre sí misma, verificación antes de cada release, y declarar los límites en el README | Claude | siempre |

**Orden:** nada se publica ni se hace push a GitHub sin que Haytham lo apruebe en ese momento; publicar es de cara al exterior y no se deshace limpiamente.

## 6. Criterio de éxito

- **Técnico (v0.1):** sobre las 538 sesiones locales de Haytham encuentra ≥5 reglas candidatas con evidencia y el `replay` de al menos una da 0 falsos positivos revisados a mano.
- **Producto:** un usuario ajeno reporta un error real evitado por un hook generado.
- **Estrellas:** objetivo de Haytham; no se puede prometer. Referencia: los competidores directos están en 21 y 402.

## 7. Deja al constructor

Estructura de paquetes, heurística exacta de marcas de corrección (afinar con datos), esquema JSON definitivo, tolerancia a versiones del `.jsonl`, matriz de plataformas, formato exacto del contrato de salida del hook para `warn` (verificar en la documentación vigente de Claude Code al implementar).

## 8. Revisión como constructor (hecha por Claude, sin subagente)

- **Respondido por el contexto:** lenguaje (Go, regla global), público (otros usuarios), local y sin modelo.
- **Decisión menor, mía:** reglas en JSON y no en markdown como hookify, para poder reproducirlas y validarlas sin modelo.
- **Nombre (cerrado 2026-11-20):** `hookwise` descartado por colisión con [vishnujayvel/hookwise](https://github.com/vishnujayvel/hookwise) (framework de hooks para Claude Code en Go, 8★, paquete npm 1.3.0). Elegido `hookdistill`: libre en GitHub, npm y PyPI el 2026-11-20.
- **Idioma (decisión mía):** documentación interna en español; CLI, README y repo en inglés, porque el público objetivo es global. Las marcas de corrección se detectan en inglés y español.
