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
                 "invocation": "pnpm dlx generator-jhipster@9.4.0 jdl model.jdl --force --skip-install" },
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

## Checklist

- [x] T-1 documento de tracking creado (este archivo) — commit docs-only inmediato.
- [ ] T-2 exploración: router/service/auth/store + goldens de jdlgen + contrato mock frontend (solo referencia) + versión JHipster fijada en registry.
- [ ] T-3 jdlgen: `ExportArtifact` + bloque application + validación de config + warnings abstract/assoc + tests.
- [ ] T-4 servicio de generación aislado: temp dir, runner inyectado, manifest, ZIP + tests.
- [ ] T-5 httpapi: endpoint POST + auth + membership + mapeo de errores + tests.
- [ ] T-6 verificación focal real (`pnpm dlx generator-jhipster@9.4.0 ...`) si el entorno permite; fixes.
- [ ] T-7 reporte final: endpoint/contrato, versión exacta JHipster, comandos pnpm, commits, checks pendientes.

## Ruta de delegación (evidencia)

- [ ] Registrar por tarea: inline o delegada (trigger: mapeo 4+ archivos, writer 2+ archivos no triviales).
- [ ] Sub-agentes: chequear disponibilidad del tool `task`; si no está disponible, inline con evidencia acá.