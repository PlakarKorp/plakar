package io.plakar.poc.backup

import android.app.Activity
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.os.Environment
import android.provider.Settings
import android.widget.Button
import android.widget.EditText
import android.widget.ScrollView
import android.widget.TextView
import java.io.BufferedReader
import java.io.File
import java.io.InputStreamReader

/**
 * Proof of concept: drive the plakar CLI, shipped as a native library, from an
 * unrooted app holding All files access.
 *
 * The interesting parts are [plakarBinary] and [run] -- everything else is
 * scaffolding to make the thing pressable.
 */
class MainActivity : Activity() {

    private lateinit var storeField: EditText
    private lateinit var passphraseField: EditText
    private lateinit var logView: TextView
    private lateinit var logScroll: ScrollView

    /**
     * The extracted native library directory is the only location an app may
     * exec from since Android 10. `useLegacyPackaging` in build.gradle.kts is
     * what guarantees the file is actually unpacked here at install time.
     */
    private val plakarBinary: File
        get() = File(applicationInfo.nativeLibraryDir, "libplakar.so")

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        storeField = findViewById(R.id.store)
        passphraseField = findViewById(R.id.passphrase)
        logView = findViewById(R.id.log)
        logScroll = findViewById(R.id.logScroll)

        findViewById<Button>(R.id.grant).setOnClickListener { requestAllFilesAccess() }
        findViewById<Button>(R.id.version).setOnClickListener {
            run(listOf("version"))
        }
        findViewById<Button>(R.id.backup).setOnClickListener { startBackup() }
    }

    override fun onResume() {
        super.onResume()
        val granted = Environment.isExternalStorageManager()
        findViewById<TextView>(R.id.status).text =
            if (granted) getString(R.string.access_granted)
            else getString(R.string.access_missing)
    }

    private fun requestAllFilesAccess() {
        if (Environment.isExternalStorageManager()) {
            log("All files access already granted.")
            return
        }
        startActivity(
            Intent(
                Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION,
                Uri.parse("package:$packageName"),
            )
        )
    }

    private fun startBackup() {
        if (!Environment.isExternalStorageManager()) {
            log("Refusing to run: All files access has not been granted yet.")
            return
        }
        val store = storeField.text.toString().trim()
        if (store.isEmpty()) {
            log("Refusing to run: no store location given.")
            return
        }

        // Android/data and Android/obb are unreadable to us no matter what, and
        // walking them just produces a wall of permission errors; the thumbnail
        // cache is regenerable and pure churn.
        run(
            listOf(
                "at", store,
                "backup", "/sdcard",
                "-no-progress",
                "-ignore", "Android/data",
                "-ignore", "Android/obb",
                "-ignore", ".thumbnails",
            )
        )
    }

    /** Runs plakar with [args] on a background thread, streaming output to the log view. */
    private fun run(args: List<String>) {
        val binary = plakarBinary
        if (!binary.exists()) {
            log("${binary.path} is missing -- did build-plakar.sh run before assembling?")
            return
        }

        logView.text = ""
        log("$ plakar ${args.joinToString(" ")}")

        Thread {
            try {
                val pb = ProcessBuilder(listOf(binary.absolutePath) + args)
                pb.redirectErrorStream(true)

                // plakar resolves its config, cache and data directories from
                // the XDG variables, falling back to $HOME. Android sets none
                // of them, and on top of that Go's os/user needs $HOME to
                // return a usable current user at all -- so point the whole
                // lot at our private app directory.
                val home = filesDir.absolutePath
                pb.environment().apply {
                    put("HOME", home)
                    put("XDG_CONFIG_HOME", "$home/config")
                    put("XDG_CACHE_HOME", "$home/cache")
                    put("XDG_DATA_HOME", "$home/data")
                    put("TMPDIR", cacheDir.absolutePath)
                    val passphrase = passphraseField.text.toString()
                    if (passphrase.isNotEmpty()) put("PLAKAR_PASSPHRASE", passphrase)
                }

                val process = pb.start()
                BufferedReader(InputStreamReader(process.inputStream)).use { reader ->
                    while (true) {
                        val line = reader.readLine() ?: break
                        log(line)
                    }
                }
                log("--- exited ${process.waitFor()} ---")
            } catch (e: Exception) {
                log("failed to run plakar: $e")
            }
        }.start()
    }

    private fun log(line: String) = runOnUiThread {
        logView.append(line + "\n")
        logScroll.post { logScroll.fullScroll(ScrollView.FOCUS_DOWN) }
    }
}
