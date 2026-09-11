import com.google.protobuf.gradle.*

plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
    id("org.jetbrains.kotlin.plugin.compose")
    id("com.google.protobuf")
    id("com.google.devtools.ksp")
}

val prepareSwiftExtractor by tasks.registering(Exec::class) {
    workingDir(rootProject.projectDir.resolve("../.."))
    commandLine("bash", "scripts/build-swift-extractor.sh")
    inputs.files(fileTree("../../../mobile/swift-graphql/Sources"), file("../../../mobile/swift-graphql/Package.swift"))
    outputs.file("../../../mobile/swift-graphql/.build/debug/extract-graphql")
}

val generateContracts by tasks.registering(Exec::class) {
    dependsOn(":graphql-tools:installDist")
    dependsOn(prepareSwiftExtractor)
    workingDir(rootProject.projectDir.resolve("../.."))
    commandLine("go", "run", "./cmd/generate")
    inputs.files(fileTree("src/main") { include("**/*.kt", "**/*.graphql") })
    inputs.files(fileTree("../../../mobile/ios/Sources"), fileTree("../../../app") { include("**/*.graphql") })
    inputs.files(fileTree("../../../internal/compiler"), fileTree("../../../internal/plan"), fileTree("../../../cmd/generate"))
    inputs.files(file("../../../go.mod"), file("../../../go.sum"), file("../../../schema.graphql"), file("../../../proto/backend.proto"))
    outputs.dir("../../../generated")
}

ksp { arg("fragmentModels", rootProject.file("../../generated/native-models.json").absolutePath) }
tasks.matching { it.name.startsWith("ksp") }.configureEach {
    dependsOn(generateContracts)
    inputs.file("../../../generated/native-models.json")
}

val connectGenerator by configurations.creating
val prepareConnectGenerator by tasks.registering(Copy::class) {
    from(connectGenerator)
    into(layout.buildDirectory.dir("codegen"))
    rename { "connect-generator.jar" }
}

android {
    namespace = "dev.fragmentrpc"
    compileSdk = 35
    defaultConfig {
        applicationId = "dev.fragmentrpc.poc.android"
        minSdk = 26
        targetSdk = 35
        versionCode = 1
        versionName = "0.1"
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }
    buildFeatures { compose = true }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions { jvmTarget = "17" }
    sourceSets["main"].proto {
        srcDir("../../../generated")
        include("app.proto")
    }
    packaging { resources.excludes += "/META-INF/{AL2.0,LGPL2.1}" }
}

protobuf {
    protoc { artifact = "com.google.protobuf:protoc:4.34.0" }
    plugins { create("connect") { path = rootProject.file("protoc-gen-connect-kotlin").absolutePath } }
    generateProtoTasks {
        all().forEach { task ->
            task.dependsOn(prepareConnectGenerator)
            task.dependsOn(generateContracts)
            task.builtins {
                create("java") { option("lite") }
                create("kotlin") { option("lite") }
            }
            task.plugins { create("connect") }
        }
    }
}

dependencies {
    implementation(project(":graphql-annotations"))
    ksp(project(":graphql-tools"))
    connectGenerator("com.connectrpc:protoc-gen-connect-kotlin:0.8.0") { isTransitive = false }
    implementation(platform("androidx.compose:compose-bom:2025.06.00"))
    implementation("androidx.activity:activity-compose:1.10.1")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.ui:ui")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.9.1")
    implementation("com.connectrpc:connect-kotlin-okhttp:0.8.0")
    implementation("com.connectrpc:connect-kotlin-google-javalite-ext:0.8.0")
    implementation("com.google.protobuf:protobuf-javalite:4.34.0")
    implementation("com.google.protobuf:protobuf-kotlin-lite:4.34.0")
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.10.2")
    androidTestImplementation(platform("androidx.compose:compose-bom:2025.06.00"))
    androidTestImplementation("androidx.test.ext:junit:1.2.1")
    androidTestImplementation("androidx.test:runner:1.6.2")
    androidTestImplementation("androidx.compose.ui:ui-test-junit4")
    debugImplementation("androidx.compose.ui:ui-test-manifest")
}
