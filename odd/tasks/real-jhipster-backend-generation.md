# Tarea: Generación real de backend Java con JHipster (fase backend)

Estado: EN PROGRESO · Rama: `feat/relationship-reconfiguration` · Sin push.

## Objetivo

Reemplazar el generador Java hardcodeado del frontend por generación real desde el backend:
Go sigue siendo el runtime del servidor; JHipster (versión fijada, ejecutado vía `pnpm dlx` en un
directorio temporal aislado) genera un proyecto Java Spring Boot/JPA descargable (ZIP + manifest)
a partir del documento UML completo. El frontend NO se toca en esta fase (lo hará la fase web
posterior).

## Fases

- **Backend (esta tarea)**: `jdlgen` → servicio de generación → endpoint autenticado → ZIP + manifest.
- **Web (futura, fuera de alcance)**: el frontend deja de usar el mock y consume el endpoint.

## Restricciones

- Go es el runtime; JHipster genera el artefacto, nunca reemplaza Go.
- Instalaciones autorizadas EXCLUSIVAMENTE vía pnpm. Sin npm, sin instalación global, sin dependencia
  nueva en frontend.
- Pin de JHipster: `generator-jhipster@9.4.0` (latest estable del registry; binario `jhipster`;
  engines node `^22.18.0 || >=24.11.0`; entorno node v24.21.0 OK). Verificación de versión: `pnpm view`.
- No simular ni hardcodear contenido de dominio (Order/Customer/Payment/com.architect).
- Estrategias de herencia (SINGLE_TABLE / TABLE_PER_CLASS) y generalization/realization/dependency:
  se ocultan o deshabilitan honestamente (warn-and-skip en manifest/reporte), nunca emitir JDL inválido.
- Sin CRDT/OT, sin reuniones, sin archivos ajenos. Preservar untracked ajenos y cambios del owner
  (`backend/internal/realtime/ticket.go`, `backend/credenciales.md`, `backend/database/`,
  `frontend/pnpm-lock.yaml`, `frontend/pnpm-workspace.yaml`, otros `odd/tasks/*`).
- Usuario corre tests/builds completos; acá solo verificación focal del flujo de generación si el
  entorno permite. Commits convencionales, sin Co-Authored-By, sin push.

## Contrato del endpoint

```
POST /api/v1/projects/{projectId}/diagrams/{diagramId}/artifact   (auth Bearer, membership)
```

Body JSON:
```json
{
  "document": { /* DiagramDocument completo (mismo schema que PUT /diagrams/{id}) */ },
  "config": {             // opcional; defaults neutrales documentados
    "baseName": "UmlArchitect",          // sanitizado a identificador Java
    "packageName": "com.umlarchitect",   // sanitizado a package Java
    "buildTool": "maven",                // maven | gradle
    "authenticationType": "jwt"          // jwt
  }
}
```

Respuesta `200`: `application/zip`, `Content-Disposition: attachment; filename="<baseName>-jhipster-backend.zip"`.
El ZIP contiene una carpeta raíz `<baseName>/` con el proyecto generado y `manifest.json` en la raíz del ZIP.

Manifest (solo archivos que existen tras generar):
```json
{
  "generator": { "name": "generator-jhipster", "version": "9.4.0",
                 "invocation": "pnpm dlx --ignore-scripts generator-jhipster@9.4.0 jdl model.jdl --force --skip-install" },
  "generatedAt": "RFC3339 UTC",
  "baseName": "...", "packageName": "...",
  "entities": [], "relationships": [],
  "files": [], "fileCount": 0,
  "warnings": [], "skipped": []
}
```

Errores: `400` ValidationError (documento o config inválidos, mensajes claros), `401`, `403` (no member),
`404` (diagrama inexistente), `502` generation_error (fallo del generador con stderr sanitizado), `500` interno.
Timeout de generación: 10 min; timeout → `502` "generation timed out".

## Decisiones técnicas

- JDL con bloque `application { config { ... } }` embebido → un solo `jhipster jdl` genera app + entidades
  (sin paso separado `jhipster app`). `skipClient true` en JDL.
- Ejecución: `exec.Command` sin shell, args fijos, workdir = `os.MkdirTemp` aislado, cleanup con defer,
  nunca escribe dentro del repo.
- `jdlgen.Export` existente NO se modifica (CLI jdl-bundle existente y goldens intactos); se agrega
  `ExportArtifact(doc, Options)` que reutiliza el mapeo interno y añade el bloque de aplicación.
- Nuevas advertencias honestas: clase `isAbstract` (emitida como entidad normal), clase asociación
  (`isAssociationClass`, emitida como entidad normal con sus relaciones).
- Servicio inyecta un runner de comandos (interface) para testear sin pnpm real; la verificación focal
  real corre el flujo completo en temp.

## Hallazgos de verificación real (T-6, JHipster 9.4.0)

Verificación focal ejecutada con `pnpm dlx` en `/tmp/opencode/jhipster-check` el 20-sep-2026.
Todos los fixes fueron aplicados al código y se validó el shape completo (app + entidades + enum +
relación OneToMany → `Alpha.java` con `BigDecimal total`, `Color color`, `Set<Beta> betas`; changelogs
Liquibase; `enumeration/Color.java`):

1. **pnpm 12 gate de build scripts**: `ERR_PNPM_IGNORED_BUILDS` (unrs-resolver@1.12.2) aborta `pnpm dlx`
   por defecto; `pnpm-workspace.yaml` con `onlyBuiltDependencies` NO aplica al proyecto efímero de dlx.
   Fix: `pnpm dlx --ignore-scripts ...` — JHipster 9.4.0 funciona sin ese postinstall (binarios
   prebuilt). Actualizado en `invocation()` + test.
2. **JDL gramática v9**: `baseName "UmlArchitect"` y `packageName "com.umlarchitect"` (con comillas) son
   ERROR de parseo ("A name is expected..."). Fix: emitir SIN comillas. Actualizado en
   `renderApplicationBlock` + test.
3. **JHipster 9 exige `entities` dentro del bloque application**: sin bloque app → error "The JDL object
   and its application's name are mandatory"; con app pero entidad fuera del bloque → la entidad se
   descarta en silencio (ni warning). Fix: `renderApplicationBlock(o, entities)` emite
   `  entities Alpha, Beta` dentro del bloque, derivado de `rep.Entities` (misma fuente que el JDL).
   Enums y relaciones siguen fuera del bloque (válido, verificado).

## Causa raíz del fallo real (T-7, reserved words del lexer JDL 9.4.0)

El usuario reportó que el endpoint devolvía solo la COLA del stack trace (sin causa). Verificación real
con el JDL de un diagrama de boundaries reveló la causa raíz: **campos con nombres reservados del lexer
JDL**. Un atributo UML llamado `required` (validation token), `unique`, `baseName` (application config
key), `readOnly` (option token), `min` (min-max token), etc. se tokeniza como token de gramática y NO como
NAME: el parser de chevrotain muere con `MismatchedTokenException: Found an invalid token '}', at line:
27 and column: 1.` — y el endpoint solo mostraba el tail del stderr.

Investigation: vocabulario de tokens extraído del binario pinned 9.4.0 (`dist/lib/jdl/core/parsing/lexer/
{lexer,option-tokens,validation-tokens,minmax-tokens,relationship-type-tokens}.js` +
`built-in-options/tokens/{application-tokens,deployment-tokens}.js`) → mapa `jdlReservedWords` embebido en
jdlgen. También `dist/lib/jdl/core/parsing/validator.js`:
- `ENTITY_NAME_PATTERN = /^[A-Z][A-Za-z0-9]*$/`, fields `ALPHANUMERIC = /^[A-Za-z][A-Za-z0-9]*$/`
  (`validation-patterns.js`) → **el sufijo `_` no es válido en nombres JDL**.

Fixes aplicados (naming determinístico, sin `_`):
- `jdlSafeIdentifier`: filtra a `[A-Za-z0-9]`, descarta dígitos iniciales, input vacío/dígitos → "Unnamed".
- `FieldName`: lowerCamel → + `"2"` si es Java reserved (`class`, `int`) o JDL reserved (`required`,
  `unique`, `baseName`, `readOnly`, `min`, …). `EntityName`: UpperCamel → + `"2"` si es JDL reserved
  (`OneToOne`, `ManyToMany`, …; UpperCamel nunca es keyword Java).
- `EnsureUnique`: colisiones → `base2`, `base3`, … (antes `base_2`).
- Warnings honestas: "JDL reserved word (would break the JHipster JDL parser)" vs "Java identifier
  sanitization" vs "name collision after sanitization"; relación g1/generalization actualizada.
- Error del endpoint accionable: `GenerationFailure{Cause,Log}`; `classifyRunnerFailure` distingue
  prerequisito faltante (pnpm no en PATH), timeout, rechazo de parseo JHipster (primera línea con
  `MismatchedTokenException`/`NoViableAltException`/`ERROR! ERROR!`), `ERR_PNPM`, genérico; `sanitizeLog`
  recorta head 400/tail 800 y enmascara `<workdir>`, `<home>`, `<tmp>` y paths absolutos. 502 sin leaks.

Verificación end-to-end (20-sep-2026): harness temporal con `required`, `unique`, `baseName`, `readOnly`,
`min`, clase `OneToOne` y `Order Item` → JDL con `required2`/`unique2`/`baseName2`/`readOnly2`/`min2`/
entidad `OneToOne2`/`OrderItem` + warnings → `pnpm dlx --ignore-scripts generator-jhipster@9.4.0 jdl
model.jdl --force --skip-install` → **exit 0, "Congratulations"**, `Order.java` con `private String
required2;` y `OneToOne2.java` con `private String baseName2;`. Harness temporal eliminado. Goldens edge
regenerados y diff inspeccionado (solo `_`→`2` y renames coherentes).

## Checklist

- [x] T-1 documento de tracking creado (este archivo) — commit docs-only inmediato.
- [x] T-2 exploración: router/service/auth/store + goldens de jdlgen + contrato mock frontend (solo referencia) + versión JHipster fijada en registry.
- [x] T-3 jdlgen: `ExportArtifact` + bloque application (con `entities`) + validación de config + warnings abstract/assoc + tests.
- [x] T-4 servicio de generación aislado: temp dir, runner inyectado, manifest, ZIP + tests.
- [x] T-5 httpapi: endpoint POST + auth + membership + mapeo de errores + tests.
- [x] T-6 verificación focal real ejecutada (`pnpm dlx --ignore-scripts generator-jhipster@9.4.0`); 3 fixes aplicados y revalidados end-to-end.
- [x] T-7 causa raíz del fallo real: reserved words del lexer JDL (investigación en binario pinned 9.4.0);
      naming determinístico sin `_` (válido para los validators), warnings honestas, clasificación y
      sanitización del error del endpoint, goldens edge regenerados; verificación end-to-end exit 0.
- [x] T-8 fixes de deuda pre-existente al HEAD: `--ignore-scripts` en `Run()` (alinea código/test/manifest con
      el comando verificado), import faltante en `artifact_test.go`, contrato de rutas (12 rutas REST; las
      WS/ticket viven en `RealtimeRoutes()`, no en `Routes()`).
- [x] T-9 commits en work-units + memoria Engram + cierre del doc (este archivo) + reporte final.
- [ ] T-10 reporte final al usuario: causa raíz, fixes, comandos verificados, checks pendientes.

## Ruta de delegación (evidencia)

- [x] T-2 mapeo: delegación intentada (explore agent) → NO disponible: "OpenCode's free tier can only be used
  from within OpenCode" (error de provider). Mapeo inline con lecturas acotadas; evidencia registrada acá.
- [x] T-3..T-5 escritura: delegación intentada → misma indisponibilidad de sub-agents; writer inline con
  comandos acotados y `gofmt -l` como check de formato (sin go build/test: los corre el usuario). Evidencia: fallo de provider reproducido en el intento T-2.
- [x] T-6 verificación: inline (bash), autorizada por el usuario como verificación focal; fixes aplicados inline.