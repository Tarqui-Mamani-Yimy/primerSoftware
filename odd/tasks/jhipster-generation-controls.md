# Controles de generación y descarga de JHipster

## Objetivo
Separar la generación del backend JHipster de la descarga del archivo ZIP ya generado.

## Problema
La interfaz combina ambas acciones, por lo que descargar puede volver a invocar la generación y no representa con claridad el estado ni el manifiesto del artefacto producido.

## Alcance autorizado
- `frontend/src/components/BackendGenerator/BackendGeneratorView.tsx`
- `frontend/src/i18n/es.ts`
- Este documento de tareas

## Restricciones
- Mantener interfaz y mensajes en español y accesibles.
- No modificar backend, móvil, CRDT/OT ni archivos ajenos.
- No usar mocks ni entidades hardcodeadas.
- El usuario ejecutará tests y builds completos; solo comprobaciones estructurales locales.
- TDD: habilitado por la configuración de sesión; no hay comando de prueba focalizado confirmado, por lo que se registra comprobación estructural.

## Estrategia
- Ruta: delegada; el alcance comprende dos archivos no triviales y la lectura previa necesaria.
- Entrega: `ask-on-risk`; estimación menor a 400 líneas.

## Tareas
- [x] JGC-01 — Adaptar el flujo de generación para conservar el ZIP exitoso, mostrar progreso, manifiesto o error saneado, y habilitar la descarga solo entonces.
  - Aceptación: generar llama una vez al endpoint; descargar no llama al endpoint y reutiliza el blob/archivo exactos recibidos.
  - Comprobación: lectura estructural de estados, manejador y atributos de accesibilidad.
  - Commit: `9cc4007` (`feat(frontend): separar generación y descarga JHipster`).
- [x] JGC-02 — Ajustar los textos en español para los dos controles y sus estados accesibles.
  - Aceptación: no quedan etiquetas que indiquen que generar descarga automáticamente.
  - Comprobación: lectura estructural de referencias de i18n.
  - Commit: `9cc4007` (`feat(frontend): separar generación y descarga JHipster`).

## Progreso
JGC-01 y JGC-02 implementadas. Comprobación estructural: el único llamado a `artifactApi.generate` está en el manejador de generación; el manejador de descarga recibe el artefacto almacenado y no invoca el endpoint. Los controles y estados accesibles referencian las nuevas claves de i18n. No se ejecutaron tests ni builds completos por instrucción del usuario.

## Próximo paso
Validación completa por parte del usuario; no se ejecutaron tests ni builds completos por instrucción explícita.
