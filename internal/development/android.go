package development

// Android export for Godot: Eclipse Temurin JDK 17 and the Android SDK
// packages Godot's APK export signs and aligns with. Pinned by the checksums
// their publishers give (Adoptium's SHA-256; the SHA-1 in Google's
// repository index, here as SHA-256 of the same verified archives). Google
// publishes Linux build-tools for x86_64 only; elsewhere the step is skipped
// and verify reports it missing. The archives are unpacked with Python's
// zipfile, keeping each file's recorded mode (they hold no symlinks), and
// renamed into the SDK layout sdkmanager would write.
const JDKVersion = "17.0.11+9"
const AndroidBuildTools = "35.0.0"
const AndroidPlatform = "android-35"
const AndroidPlatformTools = "36.0.2"

const androidInstall = `RUN set -eu; \
    case "$(dpkg --print-architecture)" in \
      amd64) ;; \
      *) echo 'Google publishes Linux Android build-tools for x86_64 only; Android export is not installed' >&2; exit 0 ;; \
    esac; \
    curl --fail --show-error --silent --location --retry 3 --connect-timeout 15 --max-time 900 \
      "https://github.com/adoptium/temurin17-binaries/releases/download/jdk-17.0.11%2B9/OpenJDK17U-jdk_x64_linux_hotspot_17.0.11_9.tar.gz" -o /tmp/repokit-jdk.tar.gz; \
    printf '%s  %s\n' aa7fb6bb342319d227a838af5c363bfa1b4a670c209372f9e6585bd79da6220c /tmp/repokit-jdk.tar.gz | sha256sum -c -; \
    test ! -e /opt/jdk-17.0.11+9; \
    tar -xzf /tmp/repokit-jdk.tar.gz -C /opt; rm /tmp/repokit-jdk.tar.gz; \
    /opt/jdk-17.0.11+9/bin/java -version 2>&1 | grep -F '"17.0.11"'; \
    sdk=/opt/android-sdk; install -d "$sdk/build-tools" "$sdk/platforms"; \
    build_tools="build-tools_r35_linux.zip bd3a4966912eb8b30ed0d00b0cda6b6543b949d5ffe00bea54c04c81e1561d88 android-15 build-tools/35.0.0"; \
    platform="platform-35_r02.zip 0988cacad01b38a18a47bac14a0695f246bc76c1b06c0eeb8eb0dc825ab0c8e0 android-35 platforms/android-35"; \
    platform_tools="platform-tools_r36.0.2-linux.zip 3afdea91441815ab41254193df0343d92c1b1c0d0237165c3a345c8af8891c31 platform-tools platform-tools"; \
    for pkg in "$build_tools" "$platform" "$platform_tools"; do \
      set -- $pkg; \
      curl --fail --show-error --silent --location --retry 3 --connect-timeout 15 --max-time 900 \
        "https://dl.google.com/android/repository/$1" -o /tmp/repokit-android.zip; \
      printf '%s  %s\n' "$2" /tmp/repokit-android.zip | sha256sum -c -; \
      rm -rf /tmp/repokit-android; \
      python3 -c 'import os, sys, zipfile; z = zipfile.ZipFile(sys.argv[1]); [os.chmod(z.extract(i, sys.argv[2]), (i.external_attr >> 16) & 0o777 or 0o644) for i in z.infolist() if not i.is_dir()]' \
        /tmp/repokit-android.zip /tmp/repokit-android; \
      mv "/tmp/repokit-android/$3" "$sdk/$4"; \
      rm -rf /tmp/repokit-android /tmp/repokit-android.zip; \
    done; \
    grep -Fx 'Pkg.Revision=35.0.0' "$sdk/build-tools/35.0.0/source.properties"; \
    grep -Fx 'Pkg.Revision=2' "$sdk/platforms/android-35/source.properties"; \
    grep -Fx 'Pkg.Revision=36.0.2' "$sdk/platform-tools/source.properties"; \
    PATH="/opt/jdk-17.0.11+9/bin:$PATH" "$sdk/build-tools/35.0.0/apksigner" --version; \
    chmod -R a+rX "$sdk"; \
    for tool in java javac keytool jarsigner; do ln -s "/opt/jdk-17.0.11+9/bin/$tool" "/usr/local/bin/$tool"; done; \
    for tool in apksigner zipalign aapt2; do ln -s "$sdk/build-tools/35.0.0/$tool" "/usr/local/bin/$tool"; done; \
    ln -s "$sdk/platform-tools/adb" /usr/local/bin/adb; \
    apksigner --version; java -version 2>&1 | grep -F '"17.0.11"'
# Worker terminals are login shells, which reset PATH: the tools are linked
# into /usr/local/bin, and Godot's export reads these locations.
ENV JAVA_HOME=/opt/jdk-17.0.11+9 \
    ANDROID_HOME=/opt/android-sdk \
    ANDROID_SDK_ROOT=/opt/android-sdk
`
