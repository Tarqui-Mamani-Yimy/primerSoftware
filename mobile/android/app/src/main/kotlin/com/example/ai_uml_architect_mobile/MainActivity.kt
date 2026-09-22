package com.example.ai_uml_architect_mobile

import android.app.Activity
import android.content.Intent
import android.net.Uri
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private val channelName = "uml/image_picker"
    private val requestCode = 7001
    private var pendingResult: MethodChannel.Result? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, channelName)
            .setMethodCallHandler { call, result ->
                if (call.method != "pickImage") {
                    result.notImplemented()
                    return@setMethodCallHandler
                }
                if (pendingResult != null) {
                    result.error("BUSY", "Image picker is already open", null)
                    return@setMethodCallHandler
                }
                pendingResult = result
                startActivityForResult(
                    Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
                        addCategory(Intent.CATEGORY_OPENABLE)
                        type = "image/*"
                    },
                    requestCode
                )
            }
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode != this.requestCode) return
        val result = pendingResult ?: return
        pendingResult = null
        if (resultCode != Activity.RESULT_OK || data?.data == null) {
            result.success(null)
            return
        }
        val uri: Uri = data.data!!
        val resolver = contentResolver
        val mime = resolver.getType(uri)
        if (mime != "image/png" && mime != "image/jpeg" && mime != "image/webp") {
            result.error("UNSUPPORTED_TYPE", "Only PNG, JPEG, or WEBP images are supported", null)
            return
        }
        try {
            val bytes = resolver.openInputStream(uri)?.use { stream ->
                val output = java.io.ByteArrayOutputStream()
                val buffer = ByteArray(8192)
                var total = 0
                while (true) {
                    val count = stream.read(buffer)
                    if (count < 0) break
                    total += count
                    if (total > 10 * 1024 * 1024) {
                        result.error("TOO_LARGE", "Image exceeds 10 MB limit", null)
                        return@use null
                    }
                    output.write(buffer, 0, count)
                }
                output.toByteArray()
            }
            if (bytes == null) return
            result.success(mapOf("bytes" to bytes, "mimeType" to mime))
        } catch (error: Exception) {
            result.error("READ_FAILED", "Could not read selected image", error.message)
        }
    }
}
