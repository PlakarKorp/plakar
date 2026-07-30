plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "io.plakar.poc.backup"
    compileSdk = 35

    defaultConfig {
        applicationId = "io.plakar.poc.backup"
        // MANAGE_EXTERNAL_STORAGE ("All files access") is API 30+.
        minSdk = 30
        targetSdk = 35
        versionCode = 1
        versionName = "0.1-poc"

        ndk {
            abiFilters += "arm64-v8a"
        }
    }

    // Android 10+ enforces W^X on app-writable storage: the only place we are
    // allowed to exec from is the extracted native library directory. Modern
    // packaging stores libraries page-aligned inside the APK and maps them
    // straight from there, so nothing is ever written to disk and there is no
    // path to exec. Legacy packaging extracts at install time, which is what
    // gives us a real file to run.
    packaging {
        jniLibs {
            useLegacyPackaging = true
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        viewBinding = false
    }
}

dependencies {
    // Intentionally none: the PoC uses only the platform APIs so that the
    // build has nothing to resolve beyond the Android Gradle plugin itself.
}
