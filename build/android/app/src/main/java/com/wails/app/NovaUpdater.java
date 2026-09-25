package com.wails.app;

import android.app.Activity;
import android.content.Intent;
import android.net.Uri;
import android.os.Build;
import android.provider.Settings;
import android.util.Log;
import android.webkit.JavascriptInterface;

import androidx.core.content.FileProvider;

import java.io.File;
import java.io.IOException;

/**
 * Exposed to the page as window.NovaAndroid. Nova's Go side downloads and
 * verifies an update APK into cache/updates/; this hands it to the system
 * package installer. Android itself refuses the install unless the APK is
 * signed with the same key as the installed app.
 */
public class NovaUpdater {
    private static final String TAG = "NovaUpdater";
    private final Activity activity;

    public NovaUpdater(Activity activity) {
        this.activity = activity;
    }

    /** Whether Android lets Nova start the package installer. */
    @JavascriptInterface
    public boolean canInstall() {
        return Build.VERSION.SDK_INT < Build.VERSION_CODES.O
                || activity.getPackageManager().canRequestPackageInstalls();
    }

    /**
     * Starts installing the APK at path. Returns "ok", "permission" when the
     * user first has to allow installs from Nova (the settings screen is
     * opened), or an error message.
     */
    @JavascriptInterface
    public String installApk(String path) {
        try {
            File updates = new File(activity.getCacheDir(), "updates").getCanonicalFile();
            File apk = new File(path).getCanonicalFile();
            // Only install what Nova downloaded itself.
            if (!apk.getPath().startsWith(updates.getPath() + File.separator)
                    || !apk.getName().endsWith(".apk") || !apk.isFile()) {
                return "not a downloaded update";
            }
            if (!canInstall()) {
                Intent settings = new Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
                        Uri.parse("package:" + activity.getPackageName()));
                settings.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
                activity.startActivity(settings);
                return "permission";
            }
            Uri uri = FileProvider.getUriForFile(activity, activity.getPackageName() + ".fileprovider", apk);
            Intent install = new Intent(Intent.ACTION_VIEW);
            install.setDataAndType(uri, "application/vnd.android.package-archive");
            install.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_ACTIVITY_NEW_TASK);
            activity.startActivity(install);
            return "ok";
        } catch (IOException | RuntimeException e) {
            Log.e(TAG, "installApk failed", e);
            return e.getMessage() != null ? e.getMessage() : "could not start the installer";
        }
    }
}
