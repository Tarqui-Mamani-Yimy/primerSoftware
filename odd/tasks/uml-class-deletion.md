# Eliminación de clase UML

## Objetivo

Permitir eliminar una clase/tabla UML desde el editor web, junto con todas sus
relaciones conectadas, solicitando confirmación y persistiendo el documento
resultante mediante el autosave existente.

## Alcance

- React web: inspector y estado del editor UML.
- Confirmación accesible en español con cantidad de relaciones afectadas.
- Eliminación en cascada por `sourceId`/`targetId`.
- Limpieza de selección e inspector.
- Pruebas focalizadas si el entorno las permite.
- Fuera de alcance: backend, Flutter, reuniones, CRDT/OT.

## Ruta delegada

`frontend/src/components/UmlCanvas/Inspector.tsx` →
`frontend/src/components/UmlCanvas/CanvasView.tsx` →
`frontend/src/App.tsx` →
`frontend/src/i18n/es.ts`.

## Checklist

- [x] Crear acción accesible para eliminar la clase.
- [x] Confirmar nombre y cantidad de relaciones antes de borrar.
- [x] Eliminar clase y relaciones conectadas.
- [x] Limpiar selección e inspector.
- [x] Programar autosave del documento válido.
- [x] Corregir/verificar sintaxis JSX existente en `CanvasView.tsx`.
- [x] Ejecutar verificación focalizada disponible.
- [x] Crear commit convencional sin `Co-Authored-By`.

## Verificación pendiente

- `cd frontend && npm run build`: OK; 1696 módulos transformados.
- `cd frontend && npm run lint`: pendiente por diagnósticos existentes en `App.tsx`, `api/diagramApi.ts` y `Sidebar.tsx`, no relacionados con esta tarea.
- Revisión manual pendiente: confirmación, cascada, selección y persistencia.
