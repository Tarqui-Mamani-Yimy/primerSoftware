import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart';

import 'api.dart';
import 'models.dart';
import 'strings.dart';
import 'voice_transcription_service.dart';
import 'voice_command_flow.dart';
import 'voice_command_preview.dart';
import 'voice_commands.dart';
import 'uml_mutations.dart';
import 'manual_uml_controls.dart';
import 'manual_uml_dialogs.dart';
import 'manual_dialog_guard.dart';
import 'realtime_service.dart';
import 'artifact_error.dart';
import 'artifact_service.dart';
import 'image_import_flow.dart';
import 'image_import_preview.dart';
import 'image_picker_channel.dart';
import 'voice_command_help.dart';

void main() => runApp(UmlArchitectApp(api: ApiClient()));

class UmlArchitectApp extends StatelessWidget {
  const UmlArchitectApp({super.key, required this.api});
  final ApiClient api;

  @override
  Widget build(BuildContext context) => MaterialApp(
        title: AppStrings.appName,
        theme: ThemeData(
            colorScheme: ColorScheme.fromSeed(seedColor: Colors.indigo),
            useMaterial3: true),
        home: LoginPage(api: api),
      );
}

class LoginPage extends StatefulWidget {
  const LoginPage({super.key, required this.api});
  final ApiClient api;
  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final email = TextEditingController();
  final password = TextEditingController();
  bool busy = false;
  String? error;

  @override
  void dispose() {
    email.dispose();
    password.dispose();
    super.dispose();
  }

  Future<void> submit() async {
    setState(() {
      busy = true;
      error = null;
    });
    try {
      await widget.api.login(email.text.trim(), password.text);
      if (mounted)
        Navigator.of(context).pushReplacement(
            MaterialPageRoute(builder: (_) => ProjectsPage(api: widget.api)));
    } catch (e) {
      if (mounted) setState(() => error = e.toString());
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
        body: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    const Icon(Icons.account_tree, size: 64),
                    const SizedBox(height: 16),
                    Text(AppStrings.appName,
                        style: Theme.of(context).textTheme.headlineMedium,
                        textAlign: TextAlign.center),
                    const SizedBox(height: 32),
                    TextField(
                        controller: email,
                        keyboardType: TextInputType.emailAddress,
                        decoration:
                            const InputDecoration(labelText: AppStrings.email)),
                    const SizedBox(height: 12),
                    TextField(
                        controller: password,
                        obscureText: true,
                        decoration: const InputDecoration(
                            labelText: AppStrings.password,
                            hintText: AppStrings.passwordPlaceholder)),
                    if (error != null)
                      Padding(
                          padding: const EdgeInsets.only(top: 12),
                          child: Text(error!,
                              style: TextStyle(
                                  color: Theme.of(context).colorScheme.error))),
                    const SizedBox(height: 20),
                    FilledButton(
                        onPressed: busy ? null : submit,
                        child: busy
                            ? const CircularProgressIndicator(
                                semanticsLabel: AppStrings.loggingIn)
                            : const Text(AppStrings.login)),
                  ]),
            ),
          ),
        ),
      );
}

class ProjectsPage extends StatefulWidget {
  const ProjectsPage({super.key, required this.api});
  final ApiClient api;
  @override
  State<ProjectsPage> createState() => _ProjectsPageState();
}

class _ProjectsPageState extends State<ProjectsPage> {
  late Future<List<Project>> future;
  String? actionError;
  Project? createdProject;
  bool actionBusy = false;

  @override
  void initState() {
    super.initState();
    future = widget.api.projects();
  }

  Future<void> refreshProjects() async {
    setState(() {
      future = widget.api.projects();
      actionError = null;
    });
    await future;
  }

  String projectError(Object error, {required bool joining}) {
    if (error is ApiException) {
      if (error.statusCode == 401) return AppStrings.sessionExpired;
      if (joining && error.statusCode == 409)
        return AppStrings.projectAlreadyJoined;
      if (joining && (error.statusCode == 404 || error.statusCode == 400))
        return AppStrings.projectJoinError;
      if (!joining && error.statusCode == 400)
        return AppStrings.projectCreatedError;
    }
    return joining
        ? AppStrings.projectJoinError
        : AppStrings.projectCreatedError;
  }

  Future<void> createProject() async {
    final name = TextEditingController();
    final description = TextEditingController();
    final formKey = GlobalKey<FormState>();
    final input = await showDialog<List<String>>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text(AppStrings.createProject),
        content: Form(
          key: formKey,
          child: SingleChildScrollView(
            child: Column(mainAxisSize: MainAxisSize.min, children: [
              Text(AppStrings.createProjectHelp),
              const SizedBox(height: 16),
              TextFormField(
                controller: name,
                autofocus: true,
                textInputAction: TextInputAction.next,
                decoration: const InputDecoration(
                    labelText: AppStrings.projectName,
                    hintText: AppStrings.projectNamePlaceholder),
                validator: (value) => value == null || value.trim().isEmpty
                    ? AppStrings.projectNameRequired
                    : null,
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: description,
                maxLines: 3,
                decoration: const InputDecoration(
                    labelText: AppStrings.projectDescription,
                    hintText: AppStrings.projectDescriptionPlaceholder),
              ),
            ]),
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text(AppStrings.cancel)),
          FilledButton(
              onPressed: () {
                if (formKey.currentState!.validate())
                  Navigator.pop(
                      context, [name.text.trim(), description.text.trim()]);
              },
              child: const Text(AppStrings.create)),
        ],
      ),
    );
    name.dispose();
    description.dispose();
    if (input == null || !mounted) return;
    setState(() {
      actionBusy = true;
      actionError = null;
    });
    try {
      final project =
          await widget.api.createProject(input[0], description: input[1]);
      if (!mounted) return;
      setState(() => createdProject = project);
      await refreshProjects();
    } catch (error) {
      if (mounted)
        setState(() => actionError = projectError(error, joining: false));
    } finally {
      if (mounted) setState(() => actionBusy = false);
    }
  }

  Future<void> joinProject() async {
    final code = TextEditingController();
    final formKey = GlobalKey<FormState>();
    final input = await showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text(AppStrings.joinProject),
        content: Form(
          key: formKey,
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            const Text(AppStrings.joinProjectHelp),
            const SizedBox(height: 16),
            TextFormField(
              controller: code,
              autofocus: true,
              textCapitalization: TextCapitalization.characters,
              decoration: const InputDecoration(
                  labelText: AppStrings.classroomCode,
                  hintText: AppStrings.classroomCodePlaceholder),
              validator: (value) => value == null || value.trim().isEmpty
                  ? AppStrings.classroomCodeRequired
                  : null,
            ),
          ]),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text(AppStrings.cancel)),
          FilledButton(
              onPressed: () {
                if (formKey.currentState!.validate())
                  Navigator.pop(context, code.text.trim().toUpperCase());
              },
              child: const Text(AppStrings.join)),
        ],
      ),
    );
    code.dispose();
    if (input == null || !mounted) return;
    setState(() {
      actionBusy = true;
      actionError = null;
    });
    try {
      await widget.api.joinProject(input);
      await refreshProjects();
    } catch (error) {
      if (mounted)
        setState(() => actionError = projectError(error, joining: true));
    } finally {
      if (mounted) setState(() => actionBusy = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text(AppStrings.assignedProjects)),
        body: FutureBuilder<List<Project>>(
          future: future,
          builder: (context, snapshot) {
            if (snapshot.connectionState != ConnectionState.done)
              return const Center(child: CircularProgressIndicator());
            if (snapshot.hasError)
              return Center(
                  child: Text(
                      '${AppStrings.couldNotLoadProjects}: ${snapshot.error}'));
            final projects = snapshot.data ?? [];
            return ListView(
              padding: const EdgeInsets.all(12),
              children: [
                if (actionError != null)
                  Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: Text(actionError!,
                          semanticsLabel: actionError,
                          style: TextStyle(
                              color: Theme.of(context).colorScheme.error))),
                if (createdProject != null)
                  Card(
                    child: ListTile(
                      leading: const Icon(Icons.check_circle),
                      title: const Text(AppStrings.projectCreated),
                      subtitle: Text(
                          '${AppStrings.shareClassroomCode}\n${createdProject!.accessCode ?? ''}',
                          semanticsLabel:
                              '${AppStrings.shareClassroomCode} ${createdProject!.accessCode ?? ''}'),
                    ),
                  ),
                Row(children: [
                  Expanded(
                      child: FilledButton.icon(
                          onPressed: actionBusy ? null : createProject,
                          icon: const Icon(Icons.add),
                          label: const Text(AppStrings.newProject))),
                  const SizedBox(width: 12),
                  Expanded(
                      child: OutlinedButton.icon(
                          onPressed: actionBusy ? null : joinProject,
                          icon: const Icon(Icons.group_add),
                          label: const Text(AppStrings.joinProject))),
                ]),
                const SizedBox(height: 16),
                if (projects.isEmpty)
                  const Padding(
                      padding: EdgeInsets.all(24),
                      child: Center(child: Text(AppStrings.noProjects)))
                else
                  ...projects.map(
                    (project) => Card(
                      child: ListTile(
                        title: Text(project.name),
                        subtitle: Text(project.description),
                        trailing: const Icon(Icons.chevron_right),
                        onTap: () => Navigator.of(context).push(
                          MaterialPageRoute(
                            builder: (_) => DiagramsPage(
                              api: widget.api,
                              project: project,
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
              ],
            );
          },
        ),
      );
}

class DiagramsPage extends StatefulWidget {
  const DiagramsPage({super.key, required this.api, required this.project});
  final ApiClient api;
  final Project project;
  @override
  State<DiagramsPage> createState() => _DiagramsPageState();
}

class _DiagramsPageState extends State<DiagramsPage> {
  late Future<List<DiagramSummary>> future;
  @override
  void initState() {
    super.initState();
    future = widget.api.diagrams(widget.project.id);
  }

  Future<void> create() async {
    final doc = await widget.api.createDiagram(
        widget.project.id, UmlDocument(name: AppStrings.newDiagram));
    if (mounted)
      Navigator.of(context).push(MaterialPageRoute(
          builder: (_) => WorkspacePage(
              api: widget.api, project: widget.project, document: doc)));
  }

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(
            title: Text(widget.project.name,
                semanticsLabel:
                    '${AppStrings.diagrams}: ${widget.project.name}')),
        floatingActionButton: FloatingActionButton(
            onPressed: create, child: const Icon(Icons.add)),
        body: FutureBuilder<List<DiagramSummary>>(
          future: future,
          builder: (context, snapshot) {
            if (snapshot.connectionState != ConnectionState.done)
              return const Center(child: CircularProgressIndicator());
            if (snapshot.hasError)
              return Center(
                  child: Text(
                      '${AppStrings.couldNotLoadDiagrams}: ${snapshot.error}'));
            final diagrams = snapshot.data ?? [];
            if (diagrams.isEmpty)
              return const Center(child: Text(AppStrings.noDiagrams));
            return ListView(
                children: diagrams
                    .map((diagram) => ListTile(
                        title: Text(diagram.name),
                        trailing: const Icon(Icons.chevron_right),
                        onTap: () async {
                          final doc = await widget.api
                              .diagram(widget.project.id, diagram.id);
                          if (mounted)
                            Navigator.of(context).push(MaterialPageRoute(
                                builder: (_) => WorkspacePage(
                                    api: widget.api,
                                    project: widget.project,
                                    document: doc)));
                        }))
                    .toList());
          },
        ),
      );
}

class WorkspacePage extends StatefulWidget {
  const WorkspacePage(
      {super.key,
      required this.api,
      required this.project,
      required this.document});
  final ApiClient api;
  final Project project;
  final UmlDocument document;
  @override
  State<WorkspacePage> createState() => _WorkspacePageState();
}

class _WorkspacePageState extends State<WorkspacePage>
    with WidgetsBindingObserver {
  late UmlDocument document;
  late TextEditingController name;
  bool saving = false;
  String saveStatus = AppStrings.saved;
  Timer? autosaveTimer;
  bool dirty = false;
  final VoiceTranscriptionService voiceService = VoiceTranscriptionService();
  bool voiceBusy = false;
  bool voiceRecording = false;
  String voiceStatus = '';
  String voiceText = '';
  VoiceMutationPreview? voicePreview;
  UmlDocument? lastVoiceSnapshot;
  DiagramRealtimeService? realtime;
  StreamSubscription<RealtimeEvent>? realtimeSubscription;
  final List<RealtimeMember> collaborators = [];
  bool realtimeBusy = false;
  String realtimeStatus = AppStrings.realtimeConnecting;
  File? generatedArtifact;
  bool generatingArtifact = false;
  DateTime? lastTouched;
  // Conflict UI: when the server returns 409 we hold the snapshot here and
  // give the user a choice between overwriting (keep mine) and discarding
  // (reload) their local edit.
  UmlDocument? conflictRemote;
  int documentGeneration = 0;
  final AndroidImagePicker imagePicker = const AndroidImagePicker();
  bool imageImportBusy = false;
  String imageImportStatus = '';
  ImageImportPreview? imagePreview;

  @override
  void initState() {
    super.initState();
    document = widget.document;
    name = TextEditingController(text: document.name);
    WidgetsBinding.instance.addObserver(this);
    unawaited(connectRealtime());
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    autosaveTimer?.cancel();
    name.dispose();
    unawaited(voiceService.dispose());
    realtimeSubscription?.cancel();
    unawaited(realtime?.dispose());
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    // Coming back from the background is a common cause of a stale realtime
    // connection (the OS may have suspended networking); trigger an
    // immediate reconnect instead of waiting for the next backoff attempt.
    if (state == AppLifecycleState.resumed) {
      unawaited(realtime?.reconnectNow());
    }
  }

  Future<void> toggleVoiceTranscription() async {
    if (voiceBusy) return;
    if (!voiceRecording) {
      setState(() {
        voiceBusy = true;
        voiceStatus = AppStrings.preparingVoiceModel;
        voiceText = '';
        voicePreview = null;
      });
      try {
        await voiceService.startRecording(onDownloadProgress: (progress) {
          if (!mounted) return;
          setState(() => voiceStatus = progress == null
              ? AppStrings.downloadingVoiceModel
              : '${AppStrings.downloadingVoiceModel} ${(progress * 100).round()}%');
        });
        if (mounted)
          setState(() {
            voiceRecording = true;
            voiceStatus = AppStrings.recordingVoice;
          });
      } on VoicePermissionException {
        if (mounted)
          setState(() => voiceStatus = AppStrings.voicePermissionDenied);
      } catch (_) {
        if (mounted) setState(() => voiceStatus = AppStrings.voiceModelFailed);
      } finally {
        if (mounted) setState(() => voiceBusy = false);
      }

      return;
    }

    setState(() {
      voiceBusy = true;
      voiceStatus = AppStrings.transcribingVoice;
    });
    try {
      final text = await voiceService.stopAndTranscribe();
      if (mounted)
        setState(() {
          voiceRecording = false;
          voiceText = text;
          voiceStatus =
              text.isEmpty ? AppStrings.voiceEmpty : AppStrings.voiceReady;
          voicePreview = text.isEmpty ? null : _previewTranscript(text);
        });
    } catch (_) {
      if (mounted)
        setState(() {
          voiceRecording = false;
          voiceStatus = AppStrings.voiceFailed;
        });
    } finally {
      if (mounted) setState(() => voiceBusy = false);
    }
  }

  VoiceMutationPreview? _previewTranscript(String transcript) {
    final command = parseVoiceCommand(transcript);
    if (command == null) {
      voiceStatus = AppStrings.voiceCommandInvalid;
      return null;
    }
    if (command.kind == VoiceCommandKind.undoVoiceCommand) {
      final snapshot = lastVoiceSnapshot;
      if (snapshot == null) {
        voiceStatus = AppStrings.voiceCommandUndoUnavailable;
        return null;
      }
      voiceStatus = AppStrings.voiceReady;
      return VoiceMutationPreview(
        command: command,
        document: cloneUmlDocument(snapshot),
      );
    }
    final preview = previewVoiceCommand(document, command);
    voiceStatus = preview == null
        ? AppStrings.voiceCommandInvalid
        : AppStrings.voiceReady;
    return preview;
  }

  void cancelVoicePreview() {
    setState(() {
      voicePreview = null;
      voiceStatus = AppStrings.voiceCommandCancelled;
    });
  }

  void confirmVoicePreview() {
    final preview = voicePreview;
    if (preview == null ||
        !canConfirmVoicePreview(preview, hasConflict: conflictRemote != null)) {
      return;
    }
    setState(() {
      if (preview.command.kind == VoiceCommandKind.undoVoiceCommand) {
        document = cloneUmlDocument(preview.document);
        lastVoiceSnapshot = null;
      } else {
        lastVoiceSnapshot = cloneUmlDocument(document);
        document = cloneUmlDocument(preview.document);
      }
      voicePreview = null;
      voiceStatus = AppStrings.voiceCommandApplied;
      documentGeneration++;
    });
    scheduleAutosave();
  }

  String _voicePreviewDescription(VoiceCommand command) {
    switch (command.kind) {
      case VoiceCommandKind.createClass:
        return 'Crear clase ${command.name}';
      case VoiceCommandKind.addAttribute:
        return 'Agregar atributo ${command.name} a ${command.className}';
      case VoiceCommandKind.addMethod:
        return 'Agregar método ${command.name} a ${command.className}';
      case VoiceCommandKind.createRelationship:
        return 'Relacionar ${command.sourceName} con ${command.targetName}';
      case VoiceCommandKind.undoVoiceCommand:
        return AppStrings.voiceCommandUndoPreview;
    }
  }

  void _invalidateVoiceUndo() {
    lastVoiceSnapshot = null;
    voicePreview = invalidateVoicePreview(voicePreview);
    documentGeneration++;
  }

  Future<void> importImage() async {
    if (imageImportBusy || document.id == null) return;
    final diagramId = document.id;
    final generation = documentGeneration;
    setState(() {
      imageImportBusy = true;
      imageImportStatus = AppStrings.importingImage;
      imagePreview = null;
    });
    try {
      final picked = await imagePicker.pick();
      if (picked == null || !mounted) return;
      final imported = await widget.api.importDiagramImage(
        widget.project.id,
        diagramId!,
        image: picked.bytes,
        mimeType: picked.mimeType,
      );
      if (!mounted ||
          document.id != diagramId ||
          generation != documentGeneration ||
          conflictRemote != null) {
        return;
      }
      setState(() {
        imagePreview = ImageImportPreview.create(
          imported: imported,
          active: document,
          generation: generation,
        );
        imageImportStatus = AppStrings.imageImportPreview;
      });
    } catch (_) {
      if (mounted) {
        setState(() => imageImportStatus = AppStrings.imageImportFailed);
      }
    } finally {
      if (mounted) setState(() => imageImportBusy = false);
    }
  }

  void cancelImagePreview() {
    setState(() {
      imagePreview = null;
      imageImportStatus = AppStrings.voiceCommandCancelled;
    });
  }

  void confirmImagePreview() {
    final preview = imagePreview;
    if (preview == null ||
        !canConfirmImageImport(
          preview,
          active: document,
          generation: documentGeneration,
          hasConflict: conflictRemote != null,
        )) {
      return;
    }
    _invalidateVoiceUndo();
    setState(() {
      document = preview.documentFor(document);
      imagePreview = null;
      imageImportStatus = AppStrings.voiceCommandApplied;
    });
    scheduleAutosave();
  }

  Future<void> connectRealtime() async {
    final id = document.id;
    if (id == null) return;
    realtime = DiagramRealtimeService(
        api: widget.api, projectId: widget.project.id, diagramId: id);
    realtimeSubscription = realtime!.events.listen((event) {
      if (!mounted) return;
      if (event is RealtimeSnapshot) {
        _invalidateVoiceUndo();
        setState(() {
          if (dirty) {
            conflictRemote = event.document;
            realtimeStatus = AppStrings.realtimeConflict;
          } else {
            document = event.document;
            documentGeneration++;
            name.text = event.document.name;
            document.reviewNumber = event.reviewNumber;
            document.version = event.version;
            realtimeStatus = AppStrings.realtimeConnected;
          }
          collaborators
            ..clear()
            ..addAll(event.members);
        });
      } else if (event is RealtimePresenceChanged) {
        setState(() {
          collaborators
              .removeWhere((member) => member.userId == event.member.userId);
          if (event.joined) collaborators.add(event.member);
        });
      } else if (event is RealtimeDiagramChanged &&
          event.reviewNumber > document.reviewNumber) {
        unawaited(_applyRemoteChange(event));
      } else if (event is RealtimeErrorEvent) {
        setState(() => realtimeStatus = AppStrings.realtimeDisconnected);
      } else if (event is RealtimeConnectionChanged) {
        setState(() {
          switch (event.state) {
            case RealtimeConnectionState.connected:
              realtimeStatus = AppStrings.realtimeConnected;
              break;
            case RealtimeConnectionState.reconnecting:
              realtimeStatus = AppStrings.realtimeReconnecting;
              break;
            case RealtimeConnectionState.disconnected:
              realtimeStatus = AppStrings.realtimeDisconnected;
              break;
          }
        });
      }
    });
    // DiagramRealtimeService.connect() never throws: connection failures and
    // subsequent automatic retries are reported through the RealtimeConnectionChanged
    // events handled above, which is what keeps realtimeStatus accurate.
    setState(() => realtimeBusy = true);
    try {
      await realtime!.connect();
    } finally {
      if (mounted) setState(() => realtimeBusy = false);
    }
  }

  Future<void> _applyRemoteChange(RealtimeDiagramChanged event) async {
    try {
      final remote = await widget.api.diagram(widget.project.id, document.id!);
      if (!mounted || remote.reviewNumber <= document.reviewNumber) return;
      _invalidateVoiceUndo();
      setState(() {
        if (dirty) {
          conflictRemote = remote;
          realtimeStatus = AppStrings.realtimeConflict;
        } else {
          document = remote;
          documentGeneration++;
          name.text = remote.name;
          realtimeStatus = AppStrings.realtimeRemoteChanged;
        }
      });
    } catch (_) {
      if (mounted) setState(() => realtimeStatus = AppStrings.realtimeConflict);
    }
  }

  Future<void> generateBackend() async {
    if (document.id == null || generatingArtifact) return;
    setState(() => generatingArtifact = true);
    try {
      final bytes = await widget.api
          .generateArtifact(widget.project.id, document.id!, document);
      final file = await ArtifactService().save(bytes);
      if (mounted)
        setState(() {
          generatedArtifact = file;
          saveStatus = AppStrings.backendReady;
        });
    } catch (error) {
      if (mounted) {
        setState(() => saveStatus = artifactFailureMessage(error));
      }
    } finally {
      if (mounted) setState(() => generatingArtifact = false);
    }
  }

  Future<void> shareBackend() async {
    final file = generatedArtifact;
    if (file != null) await ArtifactService().share(file);
  }

  Future<void> deleteClass(UmlClass umlClass) async {
    final dialogToken = _manualDialogToken();
    final count = document.relationships
        .where((relationship) =>
            relationship.sourceId == umlClass.id ||
            relationship.targetId == umlClass.id)
        .length;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text(AppStrings.deleteClass),
        content: Text(
            '${AppStrings.deleteClassConfirm}\n${AppStrings.deleteClassRelations}: $count'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text(AppStrings.cancel)),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text(AppStrings.deleteClassConfirmAction)),
        ],
      ),
    );
    if (confirmed != true || !_manualDialogStillCurrent(dialogToken)) return;
    _applyManualDocument(
      deleteUmlClass(document, classId: umlClass.id),
    );
  }

  // flushOnExit is called when the user pops or the OS starts tearing down
  // the page. We try a single best-effort PUT so the working document isn't
  // lost when the user backs out mid-edit.
  Future<void> flushOnExit() async {
    if (!dirty || document.id == null) return;
    autosaveTimer?.cancel();
    document.name =
        name.text.trim().isEmpty ? 'Untitled diagram' : name.text.trim();
    try {
      document = await widget.api
          .updateDiagram(widget.project.id, document.id!, document);
      dirty = false;
    } catch (_) {
      // Swallow the failure: the entry banner already explains what was lost.
    }
  }

  void scheduleAutosave() {
    if (document.id == null) return;
    autosaveTimer?.cancel();
    setState(() => saveStatus = AppStrings.savingSoon);
    autosaveTimer = Timer(const Duration(milliseconds: 700), save);
  }

  Future<void> save() async {
    if (document.id == null) return;
    document.name =
        name.text.trim().isEmpty ? 'Untitled diagram' : name.text.trim();
    if (saving) return;
    setState(() {
      saving = true;
      saveStatus = AppStrings.saving;
    });
    dirty = true;
    try {
      document = await widget.api
          .updateDiagram(widget.project.id, document.id!, document);
      dirty = false;
      if (mounted) setState(() => saveStatus = AppStrings.saved);
    } on ApiException catch (error) {
      if (error.statusCode == 409) {
        if (mounted)
          setState(() {
            saveStatus = AppStrings.persistenceConflict;
            conflictRemote = error.current ?? document;
          });
        return;
      }
      if (mounted) setState(() => saveStatus = AppStrings.saveFailed);
    } catch (_) {
      // One best-effort retry on transient failures: the next timer will pick
      // up from where we left off and the working document is preserved.
      await Future<void>.delayed(const Duration(milliseconds: 600));
      if (!dirty || document.id == null) return;
      try {
        document = await widget.api
            .updateDiagram(widget.project.id, document.id!, document);
        dirty = false;
        if (mounted) setState(() => saveStatus = AppStrings.saved);
      } catch (_) {
        if (mounted) setState(() => saveStatus = AppStrings.saveFailed);
      }
    } finally {
      if (mounted) setState(() => saving = false);
    }
  }

  Future<void> createCheckpoint(String message) async {
    if (document.id == null) return;
    autosaveTimer?.cancel();
    setState(() {
      saving = true;
      saveStatus = AppStrings.checkpointBusy;
    });
    document.name =
        name.text.trim().isEmpty ? 'Untitled diagram' : name.text.trim();
    try {
      final created = await widget.api.checkpointDiagram(widget.project.id,
          document.id!, document, message.isEmpty ? null : message);
      document.version = created.document?.version ?? (document.version + 1);
      document.reviewNumber = created.reviewNumber;
      dirty = false;
      if (mounted) setState(() => saveStatus = AppStrings.checkpointSucceeded);
    } on ApiException catch (error) {
      if (error.statusCode == 409) {
        if (mounted)
          setState(() {
            saveStatus = AppStrings.checkpointConflict;
            conflictRemote = error.current ?? document;
          });
        return;
      }
      if (mounted) setState(() => saveStatus = AppStrings.saveFailed);
    } catch (_) {
      if (mounted) setState(() => saveStatus = AppStrings.saveFailed);
    } finally {
      if (mounted) setState(() => saving = false);
    }
  }

  Future<void> resolveConflict({required bool keepMine}) async {
    final remote = conflictRemote;
    if (remote == null) return;
    if (keepMine) {
      _invalidateVoiceUndo();
      // Accept the cost: bump our baseline to the server's version, then let
      // autosave resume. Subsequent PUTs will succeed but the latest
      // checkpoint will be overwritten; the user is on the hook for that.
      document.version = remote.version;
      document.reviewNumber = remote.reviewNumber;
      documentGeneration++;
      conflictRemote = null;
      dirty = false;
      scheduleAutosave();
    } else {
      _invalidateVoiceUndo();
      // Discard local edits: re-apply the server document verbatim and clear
      // the dirty flag.
      setState(() {
        document = remote;
        documentGeneration++;
        name.text = remote.name;
        conflictRemote = null;
        saveStatus = AppStrings.saved;
      });
      dirty = false;
      await showVersions();
    }
  }

  void addClass() {
    _invalidateVoiceUndo();
    final existingNames = document.classes.map((item) => item.name).toSet();
    var className = 'NewClass';
    var suffix = 2;
    while (existingNames.contains(className)) {
      className = 'NewClass$suffix';
      suffix++;
    }
    setState(() {
      document.classes.add(UmlClass(
          id: DateTime.now().microsecondsSinceEpoch.toString(),
          name: className));
      documentGeneration++;
    });
    scheduleAutosave();
  }

  bool _manualChangeAllowed() {
    if (conflictRemote == null) return true;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text(AppStrings.persistenceConflict)),
    );
    return false;
  }

  ManualDialogToken _manualDialogToken() => ManualDialogToken(
        diagramId: document.id,
        version: document.version,
        reviewNumber: document.reviewNumber,
        generation: documentGeneration,
      );

  bool _manualDialogStillCurrent(ManualDialogToken token) =>
      mounted &&
      conflictRemote == null &&
      token.matches(
        diagramId: document.id,
        version: document.version,
        reviewNumber: document.reviewNumber,
        generation: documentGeneration,
      );

  void _applyManualDocument(UmlDocument? updated) {
    if (updated == null || !_manualChangeAllowed()) return;
    _invalidateVoiceUndo();
    setState(() {
      document = updated;
      documentGeneration++;
    });
    scheduleAutosave();
  }

  String _newManualId(String prefix, String value) =>
      '$prefix-${DateTime.now().microsecondsSinceEpoch}-$value';

  Future<void> addAttributeManually(UmlClass umlClass) async {
    final dialogToken = _manualDialogToken();
    final form = await showAttributeForm(context);
    if (form == null || !_manualDialogStillCurrent(dialogToken)) return;
    _applyManualDocument(
      addUmlAttribute(
        document,
        classId: umlClass.id,
        attributeId: _newManualId('attribute', form.name),
        attributeName: form.name,
        attributeType: form.type,
        visibility: form.visibility,
        isPk: form.isPk,
      ),
    );
  }

  Future<void> editAttributeManually(
      UmlClass umlClass, UmlAttribute attribute) async {
    final dialogToken = _manualDialogToken();
    final form = await showAttributeForm(context, initial: attribute);
    if (form == null || !_manualDialogStillCurrent(dialogToken)) return;
    _applyManualDocument(
      updateUmlAttribute(
        document,
        classId: umlClass.id,
        attributeId: attribute.id,
        name: form.name,
        type: form.type,
        visibility: form.visibility,
        isPk: form.isPk,
      ),
    );
  }

  Future<void> deleteAttributeManually(
      UmlClass umlClass, UmlAttribute attribute) async {
    final dialogToken = _manualDialogToken();
    if (!await _confirmManualDelete()) return;
    if (!_manualDialogStillCurrent(dialogToken)) return;
    _applyManualDocument(
      deleteUmlAttribute(
        document,
        classId: umlClass.id,
        attributeId: attribute.id,
      ),
    );
  }

  Future<void> addMethodManually(UmlClass umlClass) async {
    final dialogToken = _manualDialogToken();
    final form = await showMethodForm(context);
    if (form == null || !_manualDialogStillCurrent(dialogToken)) return;
    _applyManualDocument(
      addUmlMethod(
        document,
        classId: umlClass.id,
        methodId: _newManualId('method', form.name),
        methodName: form.name,
        returnType: form.returnType,
        visibility: form.visibility,
        isAbstract: form.isAbstract,
      ),
    );
  }

  Future<void> editMethodManually(UmlClass umlClass, UmlMethod method) async {
    final dialogToken = _manualDialogToken();
    final form = await showMethodForm(context, initial: method);
    if (form == null || !_manualDialogStillCurrent(dialogToken)) return;
    _applyManualDocument(
      updateUmlMethod(
        document,
        classId: umlClass.id,
        methodId: method.id,
        name: form.name,
        returnType: form.returnType,
        visibility: form.visibility,
        isAbstract: form.isAbstract,
      ),
    );
  }

  Future<void> deleteMethodManually(UmlClass umlClass, UmlMethod method) async {
    final dialogToken = _manualDialogToken();
    if (!await _confirmManualDelete()) return;
    if (!_manualDialogStillCurrent(dialogToken)) return;
    _applyManualDocument(
      deleteUmlMethod(
        document,
        classId: umlClass.id,
        methodId: method.id,
      ),
    );
  }

  Future<void> addRelationshipManually() async {
    if (document.classes.isEmpty) return;
    final dialogToken = _manualDialogToken();
    final form = await showRelationshipForm(context, classes: document.classes);
    if (form == null || !_manualDialogStillCurrent(dialogToken)) return;
    UmlClass? source;
    UmlClass? target;
    for (final item in document.classes) {
      if (item.id == form.sourceId) source = item;
      if (item.id == form.targetId) target = item;
    }
    if (source == null || target == null) return;
    _applyManualDocument(
      createUmlRelationship(
        document,
        relationshipId: _newManualId('relationship', form.type),
        sourceClassId: source.id,
        targetClassId: target.id,
        type: form.type,
        sourceMultiplicity: form.sourceMultiplicity,
        targetMultiplicity: form.targetMultiplicity,
        label: form.label,
      ),
    );
  }

  Future<void> editRelationshipManually(UmlRelationship relationship) async {
    final dialogToken = _manualDialogToken();
    final form = await showRelationshipForm(
      context,
      classes: document.classes,
      initial: relationship,
    );
    if (form == null || !_manualDialogStillCurrent(dialogToken)) return;
    UmlClass? source;
    UmlClass? target;
    for (final item in document.classes) {
      if (item.id == form.sourceId) source = item;
      if (item.id == form.targetId) target = item;
    }
    if (source == null || target == null) return;
    _applyManualDocument(
      updateUmlRelationship(
        document,
        relationshipId: relationship.id,
        type: form.type,
        sourceClassId: source.id,
        targetClassId: target.id,
        sourceMultiplicity: form.sourceMultiplicity,
        targetMultiplicity: form.targetMultiplicity,
        label: form.label,
      ),
    );
  }

  Future<void> deleteRelationshipManually(UmlRelationship relationship) async {
    final dialogToken = _manualDialogToken();
    if (!await _confirmManualDelete()) return;
    if (!_manualDialogStillCurrent(dialogToken)) return;
    _applyManualDocument(
      deleteUmlRelationship(document, relationshipId: relationship.id),
    );
  }

  Future<bool> _confirmManualDelete() async =>
      await showDialog<bool>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text(AppStrings.delete),
          content: const Text(AppStrings.confirmDeleteMember),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text(AppStrings.cancel),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text(AppStrings.delete),
            ),
          ],
        ),
      ) ??
      false;

  Future<void> showVersions() async {
    if (document.id == null) return;
    final versions = await _safeLoadVersions();
    if (!mounted || versions == null) return;
    await showModalBottomSheet<void>(
      context: context,
      builder: (context) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          padding: const EdgeInsets.all(16),
          children: [
            Text(AppStrings.versionHistory,
                style: Theme.of(context).textTheme.titleLarge),
            if (versions.isEmpty)
              const ListTile(title: Text(AppStrings.noVersions)),
            ...versions.map((version) {
              final timestamp = version.createdAt ?? '-';
              final author =
                  version.createdBy.isEmpty ? 'anónimo' : version.createdBy;
              final note = version.message ?? '';
              return ListTile(
                title: Text('Versión ${version.versionNumber} · $author'),
                subtitle: Text('$timestamp${note.isEmpty ? '' : ' — $note'}'),
                onTap: () async {
                  Navigator.of(context).pop();
                  await _restoreVersion(version.versionNumber);
                },
              );
            }),
          ],
        ),
      ),
    );
  }

  Future<List<DiagramVersion>?> _safeLoadVersions() async {
    try {
      return await widget.api.versions(widget.project.id, document.id!);
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${AppStrings.couldNotLoadVersions}: $error')),
        );
      }
      return null;
    }
  }

  Future<void> _restoreVersion(int versionNumber) async {
    try {
      final restored = await widget.api
          .restore(widget.project.id, document.id!, versionNumber);
      if (!mounted) return;
      _invalidateVoiceUndo();
      setState(() {
        document = restored;
        documentGeneration++;
        name.text = restored.name;
        saveStatus = AppStrings.restored;
        dirty = false;
      });
    } on ApiException catch (error) {
      if (mounted && error.statusCode == 409) {
        setState(() {
          saveStatus = AppStrings.checkpointConflict;
          conflictRemote = error.current ?? document;
        });
        return;
      }
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${AppStrings.restoreFailed}: $error')),
        );
      }
    }
  }

  Future<void> promptCheckpoint() async {
    final controller = TextEditingController();
    final formKey = GlobalKey<FormState>();
    final busy = !mounted ? false : saving;
    final result = await showDialog<String?>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text(AppStrings.checkpointTitle),
        content: Form(
          key: formKey,
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            const Text(AppStrings.checkpointHelp),
            const SizedBox(height: 16),
            TextFormField(
              controller: controller,
              autofocus: true,
              maxLines: 3,
              decoration: const InputDecoration(
                  labelText: AppStrings.checkpointMessageLabel,
                  hintText: AppStrings.checkpointMessagePlaceholder),
            ),
          ]),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, null),
              child: const Text(AppStrings.cancel)),
          FilledButton(
              onPressed:
                  busy ? null : () => Navigator.pop(context, controller.text),
              child: const Text(AppStrings.checkpointSubmit)),
        ],
      ),
    );
    if (result == null) {
      controller.dispose();
      return;
    }
    final message = result;
    controller.dispose();
    await createCheckpoint(message);
  }

  void confirmExit() async {
    if (!dirty || document.id == null) {
      Navigator.of(context).pop();
      return;
    }
    final discardConfirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text(AppStrings.exitChangesLost),
        content: const Text(AppStrings.exitWithoutFlush),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: const Text(AppStrings.keepEditing)),
          FilledButton(
              onPressed: () => Navigator.pop(context, true),
              child: const Text(AppStrings.discard)),
        ],
      ),
    );
    if (discardConfirmed == true) {
      Navigator.of(context).pop();
      return;
    }
    await flushOnExit();
    if (mounted) Navigator.of(context).pop();
  }

  @override
  Widget build(BuildContext context) => PopScope(
        canPop: false,
        onPopInvokedWithResult: (didPop, _) async {
          if (didPop) return;
          if (!dirty || document.id == null) {
            Navigator.of(context).pop();
            return;
          }
          final discard = await showDialog<bool>(
            context: context,
            builder: (context) => AlertDialog(
              title: const Text(AppStrings.exitChangesLost),
              content: const Text(AppStrings.exitWithoutFlush),
              actions: [
                TextButton(
                    onPressed: () => Navigator.pop(context, false),
                    child: const Text(AppStrings.keepEditing)),
                FilledButton(
                    onPressed: () => Navigator.pop(context, true),
                    child: const Text(AppStrings.discard)),
              ],
            ),
          );
          if (discard == true) {
            // Even on discard, attempt a last good-faith flush so the user's
            // edits survive if they just need to back out of one diagram to
            // a different one. The dialog confirms they understood the risk.
            await flushOnExit();
            if (mounted) Navigator.of(context).pop();
            return;
          }
          await flushOnExit();
        },
        child: Scaffold(
          appBar: AppBar(title: const Text(AppStrings.workspace), actions: [
            IconButton(
                onPressed: saving ? null : showVersions,
                icon: const Icon(Icons.history),
                tooltip: AppStrings.versionHistory),
            IconButton(
                onPressed: saving ? null : promptCheckpoint,
                icon: const Icon(Icons.bookmark_add),
                tooltip: AppStrings.checkpoint),
            IconButton(
                onPressed: saving ? null : save,
                icon: saving
                    ? const CircularProgressIndicator(
                        semanticsLabel: AppStrings.saving)
                    : const Icon(Icons.save),
                tooltip: AppStrings.save),
          ]),
          body: Stack(children: [
            ListView(padding: const EdgeInsets.all(16), children: [
              TextField(
                controller: name,
                onChanged: (_) {
                  _invalidateVoiceUndo();
                  scheduleAutosave();
                },
                decoration:
                    const InputDecoration(labelText: AppStrings.diagramName),
              ),
              Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: Text(saveStatus, key: const Key('workspace.status'))),
              Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: Text(realtimeStatus,
                      key: const Key('workspace.realtime.status'))),
              if (collaborators.isNotEmpty)
                Card(
                    child: ListTile(
                  leading: const Icon(Icons.people),
                  title: const Text(AppStrings.collaborators),
                  subtitle: Text(collaborators
                      .map((member) => member.displayName)
                      .join(', ')),
                )),
              Row(children: [
                Expanded(
                    child: FilledButton.icon(
                  onPressed: generatingArtifact ? null : generateBackend,
                  icon: generatingArtifact
                      ? const SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator())
                      : const Icon(Icons.archive),
                  label: Text(generatingArtifact
                      ? AppStrings.generatingBackend
                      : AppStrings.generateBackend),
                )),
                if (generatedArtifact != null) ...[
                  const SizedBox(width: 8),
                  IconButton(
                      onPressed: shareBackend,
                      icon: const Icon(Icons.share),
                      tooltip: AppStrings.shareBackend),
                ],
              ]),
              const SizedBox(height: 16),
              FilledButton.icon(
                key: const Key('image-import.open'),
                onPressed: imageImportBusy ? null : importImage,
                icon: imageImportBusy
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator())
                    : const Icon(Icons.image_search),
                label: Text(imageImportBusy
                    ? AppStrings.importingImage
                    : AppStrings.importImage),
              ),
              if (imageImportStatus.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: Text(imageImportStatus,
                      key: const Key('image-import.status')),
                ),
              if (imagePreview != null)
                Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: ImageImportPreviewPanel(
                    document: imagePreview!.document,
                    onConfirm: confirmImagePreview,
                    onCancel: cancelImagePreview,
                  ),
                ),
              const SizedBox(height: 16),
              Card(
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        Text(AppStrings.voiceTranscription,
                            style: Theme.of(context).textTheme.titleMedium),
                        const SizedBox(height: 4),
                        const Text(AppStrings.voiceTranscriptionHelp),
                        if (voiceStatus.isNotEmpty)
                          Padding(
                              padding: const EdgeInsets.only(top: 8),
                              child: Text(voiceStatus,
                                  key: const Key('workspace.voice.status'))),
                        if (voiceText.isNotEmpty)
                          Padding(
                            padding: const EdgeInsets.only(top: 8),
                            child: Semantics(
                              label: AppStrings.voiceTranscription,
                              value: voiceText,
                              child: InputDecorator(
                                decoration: const InputDecoration(
                                    labelText: 'Texto transcripto'),
                                child: SelectableText(voiceText),
                              ),
                            ),
                          ),
                        if (voicePreview != null)
                          VoiceCommandPreviewPanel(
                            transcript: voiceText,
                            description:
                                _voicePreviewDescription(voicePreview!.command),
                            onConfirm: confirmVoicePreview,
                            onCancel: cancelVoicePreview,
                          ),
                        const SizedBox(height: 8),
                        Row(
                          children: [
                            Expanded(
                              child: FilledButton.icon(
                                onPressed:
                                    voiceBusy ? null : toggleVoiceTranscription,
                                icon: Icon(
                                    voiceRecording ? Icons.stop : Icons.mic),
                                label: Text(voiceRecording
                                    ? AppStrings.stopAndTranscribe
                                    : AppStrings.recordVoice),
                              ),
                            ),
                            const SizedBox(width: 8),
                            const VoiceCommandHelpButton(),
                          ],
                        ),
                      ]),
                ),
              ),
              const SizedBox(height: 8),
              FilledButton.icon(
                  onPressed: addClass,
                  icon: const Icon(Icons.add),
                  label: const Text(AppStrings.addClass)),
              const SizedBox(height: 8),
              ...document.classes.map((umlClass) => Card(
                  child: Padding(
                      padding: const EdgeInsets.all(12),
                      child: Column(children: [
                        TextFormField(
                          decoration: const InputDecoration(
                              labelText: AppStrings.className,
                              prefixIcon: Icon(Icons.class_)),
                          initialValue: umlClass.name,
                          onChanged: (value) {
                            _invalidateVoiceUndo();
                            umlClass.name = value;
                            scheduleAutosave();
                          },
                        ),
                        SwitchListTile(
                          dense: true,
                          contentPadding: EdgeInsets.zero,
                          title: const Text(AppStrings.associationClass,
                              style: TextStyle(fontSize: 13)),
                          value: umlClass.isAssociationClass,
                          onChanged: (value) {
                            _invalidateVoiceUndo();
                            umlClass.isAssociationClass = value;
                            scheduleAutosave();
                          },
                        ),
                        Align(
                          alignment: Alignment.centerRight,
                          child: TextButton.icon(
                            onPressed: () => deleteClass(umlClass),
                            icon: const Icon(Icons.delete_outline),
                            label: const Text(AppStrings.deleteClass),
                          ),
                        ),
                      ])))),
              const SizedBox(height: 16),
              ManualUmlControls(
                document: document,
                onAddAttribute: addAttributeManually,
                onEditAttribute: editAttributeManually,
                onDeleteAttribute: deleteAttributeManually,
                onAddMethod: addMethodManually,
                onEditMethod: editMethodManually,
                onDeleteMethod: deleteMethodManually,
                onAddRelationship: addRelationshipManually,
                onEditRelationship: editRelationshipManually,
                onDeleteRelationship: deleteRelationshipManually,
              ),
              if (document.classes.isEmpty)
                const Padding(
                    padding: EdgeInsets.all(24),
                    child: Text(AppStrings.addClassHint)),
            ]),
            if (conflictRemote != null)
              Positioned(
                left: 12,
                right: 12,
                bottom: 12,
                child: Material(
                  color: Colors.transparent,
                  child: Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                        color: const Color(0xFFB00020),
                        borderRadius: BorderRadius.circular(8)),
                    child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text(AppStrings.checkpointConflict,
                              style: const TextStyle(
                                  color: Colors.white,
                                  fontWeight: FontWeight.bold)),
                          const SizedBox(height: 6),
                          Text(
                              'Local: ${document.name} (v${document.version}) — Remoto: ${conflictRemote!.name} (v${conflictRemote!.version})',
                              style: const TextStyle(color: Colors.white)),
                          const SizedBox(height: 8),
                          Row(
                              mainAxisAlignment: MainAxisAlignment.end,
                              children: [
                                TextButton(
                                    onPressed: () =>
                                        resolveConflict(keepMine: true),
                                    child: Text(
                                        AppStrings.checkpointConflictKeepMine,
                                        style: const TextStyle(
                                            color: Colors.white))),
                                FilledButton(
                                    onPressed: () =>
                                        resolveConflict(keepMine: false),
                                    child: const Text(
                                        AppStrings.checkpointConflictReload)),
                              ]),
                        ]),
                  ),
                ),
              ),
          ]),
        ),
      );
}
