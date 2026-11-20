# Lista de lanzamiento — estado real (tras las tareas 1–13)

Estado: ✅ hecho y comprobado · 🟡 hecho pero falta algo · 👤 solo puede hacerlo Haytham · ⛔ no se puede prometer.
Nada de esto se ha publicado ni empujado. Publicar de cara al exterior no se deshace limpiamente.

| # | Punto | Estado | Qué hay y qué falta |
|---|---|---|---|
| 1 | GIF arriba del README | 🟡 | `assets/demo.gif` (19 fotogramas, salida real de `scan`→`propose`→`replay` sobre `examples/`). Generado con ffmpeg porque `vhs` no estaba instalado; `assets/demo.tape` es la fuente para regrabarlo. 👤 aprobarlo. |
| 2 | Una línea | ✅ | Primera frase del README bajo el título. |
| 3 | Instalar en 1 comando | 🟡 | `install.sh` probado en local con una release falsa: instala con checksum correcto y se niega con el archivo alterado. `go install` y el `curl` solo funcionan cuando el repo sea público y exista la release `v0.1.0`. |
| 4 | Demo en vivo | ➖ | Previsto para v0.2 (no está en v0.1). |
| 5 | Licencia | ✅ | MIT (`LICENSE`, confirmada por Haytham). |
| 6 | Temas | 👤 | Al crear el repo: `claude-code`, `hooks`, `claude-md`, `golang`, `cli`, `developer-tools`. |
| 7 | Imagen social | 🟡 | `assets/social.svg` 1280×640. Falta convertirlo a PNG (no había `rsvg-convert`) y subirlo en Settings → Social preview. 👤 |
| 8 | CI en verde | 🟡 | Workflows escritos y con YAML válido; `actionlint`, `goreleaser` y `gitleaks` no estaban instalados, así que **no se han ejecutado**. Solo se sabrá tras el primer push. 👤 |
| 9 | Sin secretos en Git | 🟡 | Historial local: un solo autor falso (`dev@hookdistill.invalid`), sin cadenas tipo token fuera de los datos de prueba falsos (están en `.gitleaks.toml` como excepción). Gitleaks corre en CI. Antes del push, cambiar la identidad git del repo (ver abajo). |
| 10 | Good first issues | 👤 | 5 redactadas en `docs/launch/good-first-issues.md`; las publica Haytham con la etiqueta `good first issue`. |
| 11 | CONTRIBUTING | ✅ | `CONTRIBUTING.md`: compilar, probar, añadir una marca de corrección. |
| 12 | Plantillas y Discussions | 🟡 | Plantillas de bug e idea y `config.yml` hechos. 👤 Activar Discussions (Settings → Features); el enlace de `config.yml` apunta a ellas. |
| 13 | Release y changelog | 🟡 | `CHANGELOG.md` (0.1.0), `.goreleaser.yaml`, `release.yml`. 👤 Etiquetar `v0.1.0` y empujar la etiqueta. |
| 14 | Show HN | 👤 | Borrador en `docs/launch/show-hn.md`, sin cifras de estrellas ni de ahorro no medido. Lo publica Haytham, tras tener la release y el CI en verde. |
| 15 | Responder en <1 día | ⛔ | Claude no puede garantizarlo (solo actúa cuando se le invoca). Opción: una tarea programada que avise de incidencias nuevas. |
| 16 | Cero errores | ⛔ | Imposible de prometer. Lo que sí hay: tests en cada tarea, prueba de extremo a extremo con datos sintéticos, una ejecución sobre datos reales, y los límites declarados en el README. |

## Pasos de Haytham, en orden

1. **Identidad git:** el repo usa una identidad local falsa (`hookdistill dev <dev@hookdistill.invalid>`).
   Decide con qué nombre y correo quieres que aparezcan los commits (el correo real queda público en
   cada commit). Para cambiar los commits ya hechos hay que reescribir el historial: es decisión tuya.
2. **Rama:** la rama local se llama `master`; el README y el CI esperan `main` (el CI acepta las dos,
   el `curl` del README solo `main`).
3. **Crear el repo remoto** `abhaythambusiness-tech/hookdistill` (ese es el usuario de `gh` que está en
   la ruta del módulo de Go; si usas otra cuenta, hay que cambiarla en `go.mod` y los imports).
4. **Push**, mirar que el CI salga en verde y arreglar lo que falle.
5. **Temas, Discussions, imagen social** (PNG), good first issues.
6. **Etiquetar `v0.1.0`**; comprobar que la release trae binarios y `checksums.txt`, y probar `install.sh`
   contra la release real.
7. **Marcas de corrección:** repasar a mano las 85 frases de tu historial que `scan` toma por
   correcciones (el plan lo pedía; no lo hice para no leer tus mensajes privados) y afinar la lista.
8. **Show HN** y responder incidencias.
