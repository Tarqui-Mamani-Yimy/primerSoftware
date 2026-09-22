import 'package:flutter/services.dart';

class PickedImage {
  const PickedImage({required this.bytes, required this.mimeType});

  final Uint8List bytes;
  final String mimeType;
}

class AndroidImagePicker {
  const AndroidImagePicker(
      [this.channel = const MethodChannel('uml/image_picker')]);

  final MethodChannel channel;

  Future<PickedImage?> pick() async {
    final result = await channel.invokeMethod<dynamic>('pickImage');
    if (result == null) return null;
    final map = Map<Object?, Object?>.from(result as Map);
    final bytes = map['bytes'];
    final mimeType = map['mimeType'];
    if (bytes is! Uint8List || mimeType is! String) {
      throw const FormatException('Invalid image picker response');
    }
    return PickedImage(bytes: bytes, mimeType: mimeType);
  }
}
