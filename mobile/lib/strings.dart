class AppStrings {
  static const appName = 'AI UML Architect';
  static const email = 'Correo electrónico';
  static const password = 'Contraseña';
  static const passwordPlaceholder = 'Tu contraseña';
  static const login = 'Iniciar sesión';
  static const loggingIn = 'Ingresando…';
  static const assignedProjects = 'Proyectos asignados';
  static const noProjects = 'No hay proyectos asignados';
  static const couldNotLoadProjects = 'No se pudieron cargar los proyectos';
  static const newProject = 'Nuevo proyecto';
  static const createProject = 'Crear proyecto';
  static const createProjectHelp = 'Creá un proyecto para comenzar a trabajar.';
  static const projectName = 'Nombre del proyecto';
  static const projectNamePlaceholder = 'Ej.: Ingeniería de software';
  static const projectDescription = 'Descripción (opcional)';
  static const projectDescriptionPlaceholder =
      'Describí el objetivo del proyecto';
  static const joinProject = 'Unirse a un proyecto';
  static const joinProjectHelp =
      'Ingresá el código Classroom que te compartieron.';
  static const classroomCode = 'Código Classroom';
  static const classroomCodePlaceholder = 'Ej.: X7K2P9';
  static const join = 'Unirse';
  static const cancel = 'Cancelar';
  static const create = 'Crear';
  static const projectNameRequired = 'Ingresá un nombre para el proyecto.';
  static const classroomCodeRequired = 'Ingresá un código Classroom.';
  static const projectCreated = 'Proyecto creado';
  static const shareClassroomCode =
      'Compartí este código para invitar integrantes:';
  static const projectCreatedError =
      'No se pudo crear el proyecto. Revisá los datos e intentá nuevamente.';
  static const projectJoinError =
      'No se pudo unir al proyecto. Verificá el código e intentá nuevamente.';
  static const projectAlreadyJoined = 'Ya pertenecés a este proyecto.';
  static const sessionExpired = 'Tu sesión expiró. Iniciá sesión nuevamente.';
  static const operationFailed =
      'No se pudo completar la operación. Intentá nuevamente.';
  static const voiceTranscription = 'Transcripción de voz';
  static const voiceTranscriptionHelp =
      'La voz se procesa localmente y no modifica el diagrama.';
  static const recordVoice = 'Grabar voz';
  static const stopAndTranscribe = 'Detener y transcribir';
  static const preparingVoiceModel = 'Preparando modelo local…';
  static const downloadingVoiceModel = 'Descargando modelo local…';
  static const recordingVoice = 'Grabando…';
  static const transcribingVoice = 'Transcribiendo en el dispositivo…';
  static const voiceReady = 'Texto generado. Revisalo antes de usarlo.';
  static const voiceEmpty = 'No se detectó texto en la grabación.';
  static const voiceFailed = 'No se pudo transcribir la grabación.';
  static const voiceModelFailed = 'No se pudo preparar el modelo local.';
  static const voicePermissionDenied =
      'Se necesita permiso de micrófono para grabar.';
  static const voiceCommandPreview = 'Vista previa del cambio';
  static const voiceCommandConfirm = 'Confirmar cambio';
  static const voiceCommandCancel = 'Cancelar comando';
  static const voiceCommandInvalid = 'No se reconoció un comando UML válido.';
  static const voiceCommandCancelled =
      'Comando cancelado; el diagrama no cambió.';
  static const voiceCommandApplied = 'Comando aplicado; guardando el diagrama…';
  static const voiceCommandUndoUnavailable =
      'No hay un cambio de voz confirmado para deshacer.';
  static const voiceCommandUndoPreview =
      'Restaurar el último cambio confirmado por voz';
  static const diagrams = 'Diagramas';
  static const couldNotLoadDiagrams = 'No se pudieron cargar los diagramas';
  static const noDiagrams = 'Todavía no hay diagramas. Creá uno.';
  static const newDiagram = 'Nuevo diagrama';
  static const workspace = 'Espacio de trabajo';
  static const diagramName = 'Nombre del diagrama';
  static const addClass = 'Agregar clase';
  static const className = 'Nombre de la clase';
  static const associationClass = 'Clase de asociación';
  static const addClassHint = 'Agregá una clase para comenzar a modelar.';
  static const versionHistory = 'Historial de versiones';
  static const noVersions = 'Todavía no hay versiones';
  static const save = 'Guardar';
  static const saved = 'Guardado';
  static const savingSoon = 'Se guardará pronto…';
  static const saving = 'Guardando…';
  static const saveFailed = 'Error al guardar';
  static const restored = 'Restaurado';
  static const couldNotLoadVersions = 'No se pudieron cargar las versiones';
  static const restoreFailed = 'No se pudo restaurar';
  static const exitChangesLost =
      'Hay cambios sin sincronizar. ¿Salir sin guardar?';
  static const discard = 'Salir sin guardar';
  static const keepEditing = 'Seguir editando';
  static const exitWithoutFlush =
      'Hay cambios sin sincronizar y estamos cerrando la pantalla.';
  static const pendingFlushOnExit =
      'Intentaremos enviar el último estado antes de salir.';
  static const checkpoint = 'Crear checkpoint';
  static const checkpointTitle = 'Crear un checkpoint';
  static const checkpointHelp =
      'Un checkpoint etiqueta el estado actual y lo agrega al historial con tu nombre.';
  static const checkpointMessageLabel = 'Mensaje (opcional)';
  static const checkpointMessagePlaceholder =
      'Ej.: corregí la relación de checkout';
  static const checkpointSubmit = 'Crear checkpoint';
  static const checkpointBusy = 'Creando el checkpoint…';
  static const checkpointSucceeded = 'Checkpoint creado';
  static const checkpointConflict =
      'Otra persona creó un checkpoint antes. Revisá el estado y decidí qué hacer.';
  static const checkpointConflictKeepMine = 'Sobrescribir igualmente';
  static const checkpointConflictReload = 'Recargar y descartar mi edición';
  static const persistenceConflict = 'Versión desactualizada';
  static const syncOffline = 'Sin conexión';
  static const deleteClass = 'Eliminar clase';
  static const deleteClassConfirm =
      '¿Eliminar esta clase y sus relaciones conectadas?';
  static const deleteClassRelations = 'Se eliminarán relaciones conectadas';
  static const deleteClassConfirmAction = 'Eliminar';
  static const realtimeConnecting = 'Conectando colaboradores…';
  static const realtimeConnected = 'Colaboración realtime activa';
  static const realtimeConflict =
      'Hay una edición remota mientras tenés cambios locales.';
  static const realtimeRemoteChanged = 'Otra persona actualizó este diagrama.';
  static const realtimeDisconnected = 'Colaboración realtime desconectada';
  static const collaborators = 'Colaboradores en línea';
  static const generateBackend = 'Generar backend JHipster';
  static const generatingBackend = 'Generando backend…';
  static const backendReady = 'Backend generado y listo para compartir.';
  static const shareBackend = 'Compartir ZIP';
  static const backendFailed = 'No se pudo generar el backend.';
}
