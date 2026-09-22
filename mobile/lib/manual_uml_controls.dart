import 'package:flutter/material.dart';

import 'models.dart';
import 'strings.dart';

class ManualUmlControls extends StatelessWidget {
  const ManualUmlControls({
    super.key,
    required this.document,
    required this.onAddAttribute,
    required this.onEditAttribute,
    required this.onDeleteAttribute,
    required this.onAddMethod,
    required this.onEditMethod,
    required this.onDeleteMethod,
    required this.onAddRelationship,
    required this.onEditRelationship,
    required this.onDeleteRelationship,
  });

  final UmlDocument document;
  final ValueChanged<UmlClass> onAddAttribute;
  final void Function(UmlClass, UmlAttribute) onEditAttribute;
  final void Function(UmlClass, UmlAttribute) onDeleteAttribute;
  final ValueChanged<UmlClass> onAddMethod;
  final void Function(UmlClass, UmlMethod) onEditMethod;
  final void Function(UmlClass, UmlMethod) onDeleteMethod;
  final VoidCallback onAddRelationship;
  final ValueChanged<UmlRelationship> onEditRelationship;
  final ValueChanged<UmlRelationship> onDeleteRelationship;

  @override
  Widget build(BuildContext context) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          ...document.classes.map(
            (umlClass) => Card(
              key: ValueKey('manual.class.${umlClass.id}'),
              child: ExpansionTile(
                title: Text(umlClass.name),
                subtitle: Text(
                  '${umlClass.attributes.length} atributos · '
                  '${umlClass.methods.length} métodos',
                ),
                children: [
                  _memberSection(
                    context,
                    title: AppStrings.attributes,
                    icon: Icons.data_object,
                    onAdd: () => onAddAttribute(umlClass),
                    children: umlClass.attributes
                        .map(
                          (attribute) => ListTile(
                            dense: true,
                            title: Text(
                              '${attribute.visibility}${attribute.name}: ${attribute.type}',
                            ),
                            trailing: _actions(
                              onEdit: () =>
                                  onEditAttribute(umlClass, attribute),
                              onDelete: () =>
                                  onDeleteAttribute(umlClass, attribute),
                            ),
                          ),
                        )
                        .toList(),
                  ),
                  _memberSection(
                    context,
                    title: AppStrings.methods,
                    icon: Icons.functions,
                    onAdd: () => onAddMethod(umlClass),
                    children: umlClass.methods
                        .map(
                          (method) => ListTile(
                            dense: true,
                            title: Text(
                              '${method.visibility}${method.name}(): '
                              '${method.returnType}',
                            ),
                            trailing: _actions(
                              onEdit: () => onEditMethod(umlClass, method),
                              onDelete: () => onDeleteMethod(umlClass, method),
                            ),
                          ),
                        )
                        .toList(),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              Text(
                AppStrings.relationships,
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const Spacer(),
              OutlinedButton.icon(
                key: const Key('manual.relationship.add'),
                onPressed: onAddRelationship,
                icon: const Icon(Icons.add_link),
                label: const Text(AppStrings.addRelationship),
              ),
            ],
          ),
          if (document.relationships.isEmpty)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 12),
              child: Text(AppStrings.noRelationships),
            ),
          ...document.relationships.map(
            (relationship) => Card(
              key: ValueKey('manual.relationship.${relationship.id}'),
              child: ListTile(
                title: Text(
                  '${_className(relationship.sourceId)} → '
                  '${_className(relationship.targetId)}',
                ),
                subtitle: Text(
                  [
                    relationship.type,
                    if (relationship.sourceMultiplicity != null)
                      'origen ${relationship.sourceMultiplicity}',
                    if (relationship.targetMultiplicity != null)
                      'destino ${relationship.targetMultiplicity}',
                    if (relationship.label != null) relationship.label!,
                  ].join(' · '),
                ),
                trailing: _actions(
                  onEdit: () => onEditRelationship(relationship),
                  onDelete: () => onDeleteRelationship(relationship),
                ),
              ),
            ),
          ),
        ],
      );

  Widget _memberSection(
    BuildContext context, {
    required String title,
    required IconData icon,
    required VoidCallback onAdd,
    required List<Widget> children,
  }) =>
      Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          ListTile(
            dense: true,
            leading: Icon(icon),
            title: Text(title),
            trailing: IconButton(
              key: Key('manual.${title.toLowerCase()}.add'),
              onPressed: onAdd,
              icon: const Icon(Icons.add),
              tooltip: AppStrings.add,
            ),
          ),
          if (children.isEmpty)
            const Padding(
              padding: EdgeInsets.only(left: 16, bottom: 8),
              child: Text(AppStrings.noMembers),
            )
          else
            ...children,
        ],
      );

  Widget _actions({
    required VoidCallback onEdit,
    required VoidCallback onDelete,
  }) =>
      Wrap(
        spacing: 4,
        children: [
          IconButton(
            onPressed: onEdit,
            icon: const Icon(Icons.edit_outlined),
            tooltip: AppStrings.edit,
          ),
          IconButton(
            onPressed: onDelete,
            icon: const Icon(Icons.delete_outline),
            tooltip: AppStrings.delete,
          ),
        ],
      );

  String _className(String id) {
    for (final umlClass in document.classes) {
      if (umlClass.id == id) return umlClass.name;
    }
    return id;
  }
}
