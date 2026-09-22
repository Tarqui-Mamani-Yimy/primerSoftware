import 'package:flutter/material.dart';

class VoiceCommandPreviewPanel extends StatelessWidget {
  const VoiceCommandPreviewPanel({
    super.key,
    required this.transcript,
    required this.description,
    required this.onConfirm,
    required this.onCancel,
  });

  final String transcript;
  final String description;
  final VoidCallback onConfirm;
  final VoidCallback onCancel;

  @override
  Widget build(BuildContext context) => Card(
        key: const Key('workspace.voice.preview'),
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Text('Vista previa del cambio'),
              const SizedBox(height: 4),
              Text(transcript),
              const SizedBox(height: 4),
              Text(description),
              const SizedBox(height: 8),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  TextButton(
                    key: const Key('workspace.voice.cancel'),
                    onPressed: onCancel,
                    child: const Text('Cancelar'),
                  ),
                  const SizedBox(width: 8),
                  FilledButton(
                    key: const Key('workspace.voice.confirm'),
                    onPressed: onConfirm,
                    child: const Text('Confirmar cambio'),
                  ),
                ],
              ),
            ],
          ),
        ),
      );
}
