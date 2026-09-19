import 'dart:async';

import 'package:flutter/material.dart';

import 'api.dart';
import 'models.dart';
import 'strings.dart';

void main() => runApp(UmlArchitectApp(api: ApiClient()));

class UmlArchitectApp extends StatelessWidget {
  const UmlArchitectApp({super.key, required this.api});
  final ApiClient api;

  @override
  Widget build(BuildContext context) => MaterialApp(
        title: AppStrings.appName,
        theme: ThemeData(colorScheme: ColorScheme.fromSeed(seedColor: Colors.indigo), useMaterial3: true),
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
  void dispose() { email.dispose(); password.dispose(); super.dispose(); }

  Future<void> submit() async {
    setState(() { busy = true; error = null; });
    try {
      await widget.api.login(email.text.trim(), password.text);
      if (mounted) Navigator.of(context).pushReplacement(MaterialPageRoute(builder: (_) => ProjectsPage(api: widget.api)));
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
              child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
                const Icon(Icons.account_tree, size: 64),
                const SizedBox(height: 16),
                Text(AppStrings.appName, style: Theme.of(context).textTheme.headlineMedium, textAlign: TextAlign.center),
                const SizedBox(height: 32),
                TextField(controller: email, keyboardType: TextInputType.emailAddress, decoration: const InputDecoration(labelText: AppStrings.email)),
                const SizedBox(height: 12),
                TextField(controller: password, obscureText: true, decoration: const InputDecoration(labelText: AppStrings.password, hintText: AppStrings.passwordPlaceholder)),
                if (error != null) Padding(padding: const EdgeInsets.only(top: 12), child: Text(error!, style: TextStyle(color: Theme.of(context).colorScheme.error))),
                const SizedBox(height: 20),
                FilledButton(onPressed: busy ? null : submit, child: busy ? const CircularProgressIndicator(semanticsLabel: AppStrings.loggingIn) : const Text(AppStrings.login)),
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
  @override
  void initState() { super.initState(); future = widget.api.projects(); }
  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text(AppStrings.assignedProjects)),
        body: FutureBuilder<List<Project>>(
          future: future,
          builder: (context, snapshot) {
            if (snapshot.connectionState != ConnectionState.done) return const Center(child: CircularProgressIndicator());
            if (snapshot.hasError) return Center(child: Text('${AppStrings.couldNotLoadProjects}: ${snapshot.error}'));
            final projects = snapshot.data ?? [];
            if (projects.isEmpty) return const Center(child: Text(AppStrings.noProjects));
            return ListView.builder(
              padding: const EdgeInsets.all(12),
              itemCount: projects.length,
              itemBuilder: (_, index) {
                final project = projects[index];
                return Card(child: ListTile(title: Text(project.name), subtitle: Text(project.description), trailing: const Icon(Icons.chevron_right),
                  onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => DiagramsPage(api: widget.api, project: project)))));
              },
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
  void initState() { super.initState(); future = widget.api.diagrams(widget.project.id); }
  Future<void> create() async {
    final doc = await widget.api.createDiagram(widget.project.id, UmlDocument(name: AppStrings.newDiagram));
    if (mounted) Navigator.of(context).push(MaterialPageRoute(builder: (_) => WorkspacePage(api: widget.api, project: widget.project, document: doc)));
  }
  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: Text(widget.project.name, semanticsLabel: '${AppStrings.diagrams}: ${widget.project.name}')),
        floatingActionButton: FloatingActionButton(onPressed: create, child: const Icon(Icons.add)),
        body: FutureBuilder<List<DiagramSummary>>(
          future: future,
          builder: (context, snapshot) {
            if (snapshot.connectionState != ConnectionState.done) return const Center(child: CircularProgressIndicator());
            if (snapshot.hasError) return Center(child: Text('${AppStrings.couldNotLoadDiagrams}: ${snapshot.error}'));
            final diagrams = snapshot.data ?? [];
            if (diagrams.isEmpty) return const Center(child: Text(AppStrings.noDiagrams));
            return ListView(children: diagrams.map((diagram) => ListTile(title: Text(diagram.name), trailing: const Icon(Icons.chevron_right),
              onTap: () async {
                final doc = await widget.api.diagram(widget.project.id, diagram.id);
                if (mounted) Navigator.of(context).push(MaterialPageRoute(builder: (_) => WorkspacePage(api: widget.api, project: widget.project, document: doc)));
              })).toList());
          },
        ),
      );
}

class WorkspacePage extends StatefulWidget {
  const WorkspacePage({super.key, required this.api, required this.project, required this.document});
  final ApiClient api;
  final Project project;
  final UmlDocument document;
  @override
  State<WorkspacePage> createState() => _WorkspacePageState();
}

class _WorkspacePageState extends State<WorkspacePage> {
  late UmlDocument document;
  late TextEditingController name;
  bool saving = false;
  String saveStatus = AppStrings.saved;
  Timer? autosaveTimer;

  @override
  void initState() { super.initState(); document = widget.document; name = TextEditingController(text: document.name); }

  @override
  void dispose() { autosaveTimer?.cancel(); name.dispose(); super.dispose(); }

  void scheduleAutosave() {
    autosaveTimer?.cancel();
    setState(() => saveStatus = AppStrings.savingSoon);
    autosaveTimer = Timer(const Duration(milliseconds: 700), save);
  }

  Future<void> save() async {
    document.name = name.text.trim().isEmpty ? 'Untitled diagram' : name.text.trim();
    if (document.id == null || saving) return;
    setState(() { saving = true; saveStatus = AppStrings.saving; });
    try {
      document = await widget.api.updateDiagram(widget.project.id, document.id!, document);
      if (mounted) setState(() => saveStatus = AppStrings.saved);
    } catch (e) {
      if (mounted) setState(() => saveStatus = AppStrings.saveFailed);
    } finally { if (mounted) setState(() => saving = false); }
  }

  void addClass() {
    setState(() => document.classes.add(UmlClass(id: DateTime.now().microsecondsSinceEpoch.toString(), name: 'NewClass')));
    scheduleAutosave();
  }

  Future<void> showVersions() async {
    if (document.id == null) return;
    late final List<DiagramVersion> versions;
    try {
      versions = await widget.api.versions(widget.project.id, document.id!);
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${AppStrings.couldNotLoadVersions}: $error')),
        );
      }
      return;
    }
    if (!mounted) return;
    await showModalBottomSheet<void>(
      context: context,
      builder: (context) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          padding: const EdgeInsets.all(16),
          children: [
            Text(AppStrings.versionHistory, style: Theme.of(context).textTheme.titleLarge),
            if (versions.isEmpty) const ListTile(title: Text(AppStrings.noVersions)),
            ...versions.map((version) => ListTile(
              title: Text('Version ${version.versionNumber}'),
              subtitle: version.createdAt == null ? null : Text(version.createdAt!),
              onTap: () async {
                try {
                  final restored = await widget.api.restore(widget.project.id, document.id!, version.versionNumber);
                  if (!mounted) return;
                  setState(() {
                    document = restored;
                    name.text = restored.name;
                    saveStatus = AppStrings.restored;
                  });
                  Navigator.of(context).pop();
                } catch (error) {
                  if (mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(content: Text('${AppStrings.restoreFailed}: $error')),
                    );
                  }
                }
              },
            )),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text(AppStrings.workspace), actions: [
          IconButton(onPressed: saving ? null : showVersions, icon: const Icon(Icons.history), tooltip: AppStrings.versionHistory),
          IconButton(onPressed: saving ? null : save, icon: saving ? const CircularProgressIndicator(semanticsLabel: AppStrings.saving) : const Icon(Icons.save), tooltip: AppStrings.save),
        ]),
        body: ListView(padding: const EdgeInsets.all(16), children: [
          TextField(controller: name, onChanged: (_) => scheduleAutosave(), decoration: const InputDecoration(labelText: AppStrings.diagramName)),
          Padding(padding: const EdgeInsets.only(top: 8), child: Text(saveStatus)),
          const SizedBox(height: 16),
          FilledButton.icon(onPressed: addClass, icon: const Icon(Icons.add), label: const Text(AppStrings.addClass)),
          const SizedBox(height: 8),
          ...document.classes.map((umlClass) => Card(child: Padding(padding: const EdgeInsets.all(12), child: TextFormField(
            decoration: const InputDecoration(labelText: AppStrings.className, prefixIcon: Icon(Icons.class_)),
            initialValue: umlClass.name,
            onChanged: (value) { umlClass.name = value; scheduleAutosave(); },
          )))),
          if (document.classes.isEmpty) const Padding(padding: EdgeInsets.all(24), child: Text(AppStrings.addClassHint)),
        ]),
      );
}
