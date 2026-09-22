import 'package:flutter/material.dart';

import 'models.dart';
import 'strings.dart';

class ImageImportPreviewPanel extends StatelessWidget {
  const ImageImportPreviewPanel({
    super.key,
    required this.document,
    required this.onConfirm,
    required this.onCancel,
  });

  final UmlDocument document;
  final VoidCallback onConfirm;
  final VoidCallback onCancel;

  @override
  Widget build(BuildContext context) => Card(
        key: const Key('image-import.preview'),
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(AppStrings.imageImportPreview,
                  style: Theme.of(context).textTheme.titleMedium),
              Text('${document.classes.length} ${AppStrings.classesImported}'),
              Text(
                  '${document.relationships.length} ${AppStrings.relationshipsImported}'),
              const SizedBox(height: 8),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  TextButton(
                    key: const Key('image-import.cancel'),
                    onPressed: onCancel,
                    child: const Text(AppStrings.cancel),
                  ),
                  FilledButton(
                    key: const Key('image-import.confirm'),
                    onPressed: onConfirm,
                    child: const Text(AppStrings.imageImportConfirm),
                  ),
                ],
              ),
            ],
          ),
        ),
      );
}
