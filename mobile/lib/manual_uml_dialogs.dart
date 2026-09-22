import 'package:flutter/material.dart';

import 'models.dart';
import 'strings.dart';

class AttributeForm {
  const AttributeForm({
    required this.name,
    required this.type,
    required this.visibility,
    required this.isPk,
  });
  final String name;
  final String type;
  final String visibility;
  final bool isPk;
}

class MethodForm {
  const MethodForm({
    required this.name,
    required this.returnType,
    required this.visibility,
    required this.isAbstract,
  });
  final String name;
  final String returnType;
  final String visibility;
  final bool isAbstract;
}

class RelationshipForm {
  const RelationshipForm({
    required this.sourceId,
    required this.targetId,
    required this.type,
    required this.sourceMultiplicity,
    required this.targetMultiplicity,
    required this.label,
  });
  final String sourceId;
  final String targetId;
  final String type;
  final String sourceMultiplicity;
  final String targetMultiplicity;
  final String label;
}

Future<AttributeForm?> showAttributeForm(
  BuildContext context, {
  UmlAttribute? initial,
}) async {
  final name = TextEditingController(text: initial?.name ?? '');
  final type = TextEditingController(text: initial?.type ?? 'String');
  var visibility = initial?.visibility ?? '+';
  var isPk = initial?.isPk ?? false;
  final result = await showDialog<AttributeForm>(
    context: context,
    builder: (context) => StatefulBuilder(
      builder: (context, setState) => AlertDialog(
        title: Text(initial == null ? AppStrings.add : AppStrings.edit),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                  controller: name,
                  decoration: const InputDecoration(
                      labelText: AppStrings.attributeName)),
              TextField(
                  controller: type,
                  decoration: const InputDecoration(
                      labelText: AppStrings.attributeType)),
              DropdownButtonFormField<String>(
                value: visibility,
                decoration: const InputDecoration(labelText: 'Visibilidad'),
                items: const [
                  DropdownMenuItem(value: '+', child: Text('+ pública')),
                  DropdownMenuItem(value: '-', child: Text('- privada')),
                  DropdownMenuItem(value: '#', child: Text('# protegida')),
                ],
                onChanged: (value) => setState(() => visibility = value ?? '+'),
              ),
              CheckboxListTile(
                contentPadding: EdgeInsets.zero,
                title: const Text('Clave primaria'),
                value: isPk,
                onChanged: (value) => setState(() => isPk = value ?? false),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text(AppStrings.cancel)),
          FilledButton(
            onPressed: () => Navigator.pop(
              context,
              AttributeForm(
                  name: name.text,
                  type: type.text,
                  visibility: visibility,
                  isPk: isPk),
            ),
            child: const Text(AppStrings.create),
          ),
        ],
      ),
    ),
  );
  name.dispose();
  type.dispose();
  return result;
}

Future<MethodForm?> showMethodForm(
  BuildContext context, {
  UmlMethod? initial,
}) async {
  final methodName = TextEditingController(text: initial?.name ?? '');
  final returnType = TextEditingController(text: initial?.returnType ?? 'void');
  var visibility = initial?.visibility ?? '+';
  var isAbstract = initial?.isAbstract ?? false;
  final result = await showDialog<MethodForm>(
    context: context,
    builder: (context) => StatefulBuilder(
      builder: (context, setState) => AlertDialog(
        title: Text(initial == null ? AppStrings.add : AppStrings.edit),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                  controller: methodName,
                  decoration:
                      const InputDecoration(labelText: AppStrings.methodName)),
              TextField(
                  controller: returnType,
                  decoration:
                      const InputDecoration(labelText: AppStrings.returnType)),
              DropdownButtonFormField<String>(
                value: visibility,
                decoration: const InputDecoration(labelText: 'Visibilidad'),
                items: const [
                  DropdownMenuItem(value: '+', child: Text('+ pública')),
                  DropdownMenuItem(value: '-', child: Text('- privada')),
                  DropdownMenuItem(value: '#', child: Text('# protegida')),
                ],
                onChanged: (value) => setState(() => visibility = value ?? '+'),
              ),
              CheckboxListTile(
                contentPadding: EdgeInsets.zero,
                title: const Text('Abstracto'),
                value: isAbstract,
                onChanged: (value) =>
                    setState(() => isAbstract = value ?? false),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text(AppStrings.cancel)),
          FilledButton(
            onPressed: () => Navigator.pop(
              context,
              MethodForm(
                  name: methodName.text,
                  returnType: returnType.text,
                  visibility: visibility,
                  isAbstract: isAbstract),
            ),
            child: const Text(AppStrings.create),
          ),
        ],
      ),
    ),
  );
  methodName.dispose();
  returnType.dispose();
  return result;
}

Future<RelationshipForm?> showRelationshipForm(
  BuildContext context, {
  required List<UmlClass> classes,
  UmlRelationship? initial,
}) async {
  final source = TextEditingController(text: initial?.sourceMultiplicity ?? '');
  final target = TextEditingController(text: initial?.targetMultiplicity ?? '');
  final label = TextEditingController(text: initial?.label ?? '');
  var sourceId = initial?.sourceId ?? classes.first.id;
  var targetId = initial?.targetId ?? classes.first.id;
  var type = initial?.type ?? 'association';
  final result = await showDialog<RelationshipForm>(
    context: context,
    builder: (context) => StatefulBuilder(
      builder: (context, setState) => AlertDialog(
        title: Text(
            initial == null ? AppStrings.addRelationship : AppStrings.edit),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              DropdownButtonFormField<String>(
                value: sourceId,
                decoration:
                    const InputDecoration(labelText: AppStrings.sourceClass),
                items: classes
                    .map((item) => DropdownMenuItem(
                        value: item.id, child: Text(item.name)))
                    .toList(),
                onChanged: (value) =>
                    setState(() => sourceId = value ?? sourceId),
              ),
              DropdownButtonFormField<String>(
                value: targetId,
                decoration:
                    const InputDecoration(labelText: AppStrings.targetClass),
                items: classes
                    .map((item) => DropdownMenuItem(
                        value: item.id, child: Text(item.name)))
                    .toList(),
                onChanged: (value) =>
                    setState(() => targetId = value ?? targetId),
              ),
              DropdownButtonFormField<String>(
                value: type,
                decoration: const InputDecoration(
                    labelText: AppStrings.relationshipType),
                items: const [
                  'association',
                  'aggregation',
                  'composition',
                  'generalization',
                  'realization',
                  'dependency'
                ]
                    .map((item) =>
                        DropdownMenuItem(value: item, child: Text(item)))
                    .toList(),
                onChanged: (value) => setState(() => type = value ?? type),
              ),
              TextField(
                  controller: source,
                  decoration: const InputDecoration(
                      labelText: AppStrings.sourceMultiplicity)),
              TextField(
                  controller: target,
                  decoration: const InputDecoration(
                      labelText: AppStrings.targetMultiplicity)),
              TextField(
                  controller: label,
                  decoration: const InputDecoration(
                      labelText: AppStrings.relationshipLabel)),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text(AppStrings.cancel)),
          FilledButton(
            onPressed: () => Navigator.pop(
              context,
              RelationshipForm(
                sourceId: sourceId,
                targetId: targetId,
                type: type,
                sourceMultiplicity: source.text,
                targetMultiplicity: target.text,
                label: label.text,
              ),
            ),
            child: const Text(AppStrings.create),
          ),
        ],
      ),
    ),
  );
  source.dispose();
  target.dispose();
  label.dispose();
  return result;
}
