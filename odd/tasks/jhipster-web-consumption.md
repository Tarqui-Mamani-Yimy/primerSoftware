# Tarea: Consumo web del endpoint de generación JHipster (fase web)

Estado: EN PROGRESO · Rama: `feat/relationship-reconfiguration` · Sin push.

## Objetivo

El frontend deja de usar el generador Java hardcodeado (`frontend/src/data/codeGenerator.ts`) y
consume el endpoint real `POST /api/v1/projects/{projectId}/diagrams/{id}/artifact` implementado en
la fase backend (commits 962a53e/1c6b68a/a90d931). La vista BackendGenerator pasa de preview de mock
a cockpit real: config → generación → descarga del ZIP + resumen desde `manifest.json`.

## Alcance

- `frontend/src/api/diagramApi.ts`: nuevo `artifactApi.generate(...)` (blob con auth; parsea
  Content-Disposition para el filename; errores como ApiError con status/message).
- `frontend/src/components/BackendGenerator/BackendGeneratorView.tsx`: reescritura visual fiel al
  cockpit oscuro (mismos colores/fonts/material symbols) con formulario de config, estados
  idle/generating/error/success, descarga real y resumen del manifest (entities, fileCount, warnings,
  skipped) vía JSZip.
- `frontend/src/App.tsx`: quitar `strategy` state y `generateAllCodeFiles`; export 'zip' del Header →
  generación real con defaults; quitar rama 'sql' del mock.
- `frontend/src/components/Header.tsx`: sacar el ítem 'sql' del menú de exportación; relabel zip.
- Borrar `frontend/src/data/codeGenerator.ts` y los tipos `CodeFile`/`JpaStrategy` (quedan sin uso).
- `frontend/src/i18n/es.ts`: claves de generación nuevas; limpiar claves del mock.

## Contrato consumido

- `POST {VITE_API_BASE_URL || http://localhost:8080/api/v1}/projects/{projectId}/diagrams/{id}/artifact`
- Body: `{ "document": UMLDiagramDocument, "config": { baseName, packageName, buildTool, authenticationType } }`
  (config opcional; defaults backend: UmlArchitect/com.umlarchitect/maven/jwt).
- 200 → `application/zip` + `Content-Disposition: attachment; filename="<baseName>-jhipster-backend.zip"`;
  dentro, carpeta raíz `<baseName>/` + `manifest.json` (generator/entities/relationships/warnings/skipped/files).
- Errores: 400 (documento/config inválidos, message claro), 401, 403, 404, 502 (fallo de generación
  con stderr sanitizado). Timeout backend: 10 min (502).

## Restricciones

- Sin npm; sin dependencias nuevas de frontend (JSZip ya está).
- Verificación permitida: lecturas + grep estáticos. Usuario corre `tsc`/build/tests.
- Commits convencionales, sin Co-Authored-By, sin push. Preservar archivos ajenos
  (backend/realtime/ticket.go, backend/credenciales.md, backend/database/, frontend/pnpm-*,
  otros odd/tasks/*). No tocar el backend en esta fase.
- Delegación intentada en T-1 → NO disponible ("OpenCode's free tier can only be used from within
  OpenCode", error de provider, reproducido en fase backend y en este intento). Evidencia acá.

## Checklist

- [x] T-1 exploración del área (api client, types, App wiring, Header menu, i18n) + evidencia delegación.
- [x] T-2 api client: `artifactApi.generate` + parseo de filename + errores (diagramApi.ts: requestFile + ArtifactConfig/BackendArtifactResult; tipos CodeFile/JpaStrategy fuera de types.ts).
- [x] T-3 vista BackendGenerator: cockpit real (config/estados/descarga/manifest) + claves es.generator nuevas.
- [x] T-4 wiring App.tsx (quit strategy/generateAllCodeFiles/JSZip; export 'zip' real vía refs; view con projectId/diagramId/document) + Header.tsx (sin 'sql', relabel zip) + borrado codeGenerator.ts + i18n.
- [x] T-5 verificación estática: grep de referencias muertas limpio (JpaStrategy/CodeFile/generateAllCodeFiles/export.sql/generator.artifactTree/strategy/JSZip-en-App/data-imports); es.ts íntegro; diagramApi sin tipos duplicados.
- [ ] T-6 reporte final (usuario corre tsc/build).

## Rutas y evidencia

- Ruta: inline para todas las tareas (sin delegación). Evidencia del bloqueo de sub-agents en T-1 (error de provider: "OpenCode's free tier can only be used from within OpenCode"); mismo error reproducido en fase backend. El intento de delegación en esta fase usó el agente `explore` estándar → no disponibilidad de sub-agentes en este runtime.
- WU planeados: 1) api client + types; 2) vista + i18n; 3) wiring + borrado del mock; 4) doc de tracking. Sin push.