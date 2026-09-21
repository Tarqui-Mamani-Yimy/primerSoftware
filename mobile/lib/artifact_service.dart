import 'dart:io';
import 'dart:typed_data';

import 'package:path_provider/path_provider.dart';
import 'package:share_plus/share_plus.dart';

class ArtifactService {
  Future<File> save(Uint8List bytes, {String fileName = 'jhipster-backend.zip'}) async {
    if (!Platform.isAndroid) throw UnsupportedError('La descarga de artefactos solo está disponible en Android.');
    final directory = await getApplicationDocumentsDirectory();
    final file = File('${directory.path}/$fileName');
    await file.writeAsBytes(bytes, flush: true);
    return file;
  }

  Future<void> share(File file) => Share.shareXFiles([XFile(file.path)], text: 'Backend JHipster generado');
}
